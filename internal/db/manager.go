package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fahy.xyz/livetrack/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

const errDuplicateKey = "23505"

type Manager struct {
	client  *pgxpool.Pool
	logger  *slog.Logger
	metrics managerMetrics
}

type managerMetrics interface {
	PilotRetrieved()
	TrackRetrieved()
	TrackWritten()
}

func NewManager(
	ctx context.Context,
	databaseURL string,
	logger *slog.Logger,
	metrics managerMetrics,
) (*Manager, error) {
	conn, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("creating pgxpool: %w", err)
	}

	if err = conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging: %w", err)
	}

	manager := &Manager{
		client:  conn,
		logger:  logger,
		metrics: metrics,
	}
	manager.logger.Info("Connected to database", "manager", manager)

	return manager, nil
}

func (m *Manager) Ping(ctx context.Context) error {
	if err := m.client.Ping(ctx); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	return nil
}

func (m *Manager) Close() {
	m.client.Close()
}

func (m *Manager) GetAllPilots(ctx context.Context) ([]model.Pilot, error) {
	rows, err := m.client.Query(ctx, "SELECT id, name, home, orgs, tracker_type FROM pilot")
	if err != nil {
		return nil, fmt.Errorf("querying pilots: %w", err)
	}

	defer rows.Close()

	pilots, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[model.Pilot])
	if err != nil {
		return nil, fmt.Errorf("collecting rows: %w", err)
	}

	m.logger.Debug("Pilots retrieved", "pilots", pilots)
	m.metrics.PilotRetrieved()

	return pilots, nil
}

// GetDatesWithCount returns the recent dates (n=limit) with the number of flights for the day.
func (m *Manager) GetDatesWithCount(ctx context.Context, limit int) ([]time.Time, []int, error) {
	rows, err := m.client.Query(
		ctx,
		`SELECT COUNT(DISTINCT pilot_id), unix_time::date 
		 FROM track 
		 GROUP BY unix_time::date 
		 ORDER BY unix_time 
		 DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("querying dates with count: %w", err)
	}
	defer rows.Close()

	dates := []time.Time{}
	counts := []int{}

	for rows.Next() {
		var flightDate time.Time

		var count int
		if err = rows.Scan(&count, &flightDate); err != nil {
			return nil, nil, fmt.Errorf("scanning row: %w", err)
		}

		dates = append(dates, flightDate)
		counts = append(counts, count)
	}

	m.logger.Debug("Dates retrieved", "dates", dates, "counts", counts)

	return dates, counts, nil
}

func (m *Manager) GetPilotsFromOrg(ctx context.Context, org string) ([]model.Pilot, error) {
	rows, err := m.client.Query(ctx, "SELECT id, name, home, orgs, tracker_type FROM pilot WHERE $1=ANY(orgs)", org)
	if err != nil {
		return nil, fmt.Errorf("querying pilots from org %s: %w", org, err)
	}

	defer rows.Close()

	pilots, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[model.Pilot])
	if err != nil {
		return nil, fmt.Errorf("collecting rows: %w", err)
	}

	m.logger.Debug("Pilots retrieved", "pilots", pilots, "org", org)
	m.metrics.PilotRetrieved()

	return pilots, nil
}

func (m *Manager) WriteTrack(ctx context.Context, pilotID string, track []model.Point) error {
	m.logger.Debug("Inserting track", "pilot", pilotID, "track", track)

	for _, point := range track {
		_, err := m.client.Exec(
			ctx,
			`INSERT INTO track (pilot_id, unix_time, latitude, longitude, altitude, msg_type, msg_content)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			pilotID,
			point.DateTime,
			point.Latitude,
			point.Longitude,
			point.Altitude,
			point.MsgType,
			point.MsgContent,
		)
		if err, ok := err.(*pq.Error); ok {
			m.logger.Error("Error writing track", "pilotID", pilotID, "error", err.Code)

			if err.Code != errDuplicateKey {
				return fmt.Errorf("writing track: %w", err)
			}
		}
	}

	m.logger.Debug("Track written", "pilotID", pilotID, "track", track)
	m.metrics.TrackWritten()

	return nil
}

// GetAllTracksOfDay returns all the tracks of the day.
//
// The key of the map returned is the name of the pilot.
func (m *Manager) GetAllTracksOfDay(ctx context.Context, date time.Time) (map[string][]model.Point, error) {
	tracks := make(map[string][]model.Point)

	pilots, err := m.GetAllPilots(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting all pilots: %w", err)
	}

	for _, pilot := range pilots {
		points, err := m.GetTrackOfDay(ctx, pilot.ID, date)
		if err != nil {
			return nil, fmt.Errorf("getting track of day: %w", err)
		}

		tracks[pilot.Name] = points
	}

	return tracks, nil
}

// GetTrackOfDay returns the track of the pilot for the given day.
func (m *Manager) GetTrackOfDay(ctx context.Context, pilotID string, date time.Time) ([]model.Point, error) {
	day := date.Format("2006-01-02")
	m.logger.Debug("Retrieving track", "pilot", pilotID, "day", day)

	rows, err := m.client.Query(
		ctx,
		`SELECT unix_time, latitude, longitude, altitude, msg_type, msg_content 
		 FROM track 
		 WHERE pilot_id = $1 AND DATE(unix_time) = $2 
		 ORDER BY unix_time`,
		pilotID,
		day,
	)
	if err != nil {
		return nil, fmt.Errorf("querying track of day: %w", err)
	}

	defer rows.Close()

	points, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[model.Point])
	if err != nil {
		return nil, fmt.Errorf("collecting rows: %w", err)
	}

	m.logger.Debug("Track retrieved", "pilot", pilotID, "points", points)
	m.metrics.TrackRetrieved()
	// TODO: computed stats (flight time, takeoffdist, cumdist, avgspeed, legspeed, legdist
	return points, nil
}

// GetTrackSince returns the track of the pilot since the given date.
//
// If a point occurred at the since time, it is not returned.
func (m *Manager) GetTrackSince(ctx context.Context, pilotID string, since time.Time) ([]model.Point, error) {
	m.logger.Debug("Retrieving track", "pilot", pilotID, "since", since)

	rows, err := m.client.Query(
		ctx,
		`SELECT unix_time, latitude, longitude, altitude, msg_type, msg_content 
		 FROM track 
		 WHERE pilot_id = $1 AND unix_time > $2 
		 ORDER BY unix_time`,
		pilotID,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("querying track since %s: %w", since, err)
	}

	defer rows.Close()

	points, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[model.Point])
	if err != nil {
		return nil, fmt.Errorf("collecting rows: %w", err)
	}

	m.logger.Debug("Track retrieved", "pilot", pilotID, "points", points)

	return points, nil
}
