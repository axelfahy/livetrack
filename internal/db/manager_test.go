package db_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"fahy.xyz/livetrack/internal/db"
	"fahy.xyz/livetrack/internal/model"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	logger  = slog.New(slog.Default().Handler())
	manager *db.Manager
)

type emptyMetrics struct{}

func (m emptyMetrics) PilotRetrieved() {}
func (m emptyMetrics) TrackRetrieved() {}
func (m emptyMetrics) TrackWritten()   {}

func TestMain(m *testing.M) {
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		logger.Error("Could not construct pool", "error", err)
		os.Exit(1)
	}

	err = pool.Client.Ping()
	if err != nil {
		logger.Error("Could not connect to Docker", "error", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Could not get current directory", "error", err)
		os.Exit(1)
	}

	// Pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "17",
		Env: []string{
			"POSTGRES_DB=postgres",
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"PGDATE=/pgdata",
			"listen_addresses = '*'",
			"log_statement=all",
		},
		Mounts: []string{
			cwd + "/../../tools/setup/init_test.sql:/docker-entrypoint-initdb.d/init.sql",
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		logger.Error("Could not start resource", "error", err)
		os.Exit(1)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	// Using defautl database postgres for tests to avoid having to create one separately.
	databaseURL := fmt.Sprintf("postgres://postgres:postgres@%s/postgres?sslmode=disable", hostAndPort)

	logger := slog.New(slog.Default().Handler())
	ctx := context.Background()

	if err = resource.Expire(120); err != nil { // Tell docker to hard kill the container in 120 seconds
		os.Exit(1)
	}

	// Exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		manager, err = db.NewManager(ctx, databaseURL, logger, &emptyMetrics{})
		if err != nil {
			return fmt.Errorf("creating manager: %w", err)
		}

		if err = manager.Ping(ctx); err != nil {
			return fmt.Errorf("pinging: %w", err)
		}

		return nil
	}); err != nil {
		logger.Error("Could not connect to docker", "error", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		logger.Error("Could not purge resource", "error", err)
		os.Exit(1)
	}

	manager.Close()

	os.Exit(code)
}

func TestManager_GetAllPilots(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	pilots, err := manager.GetAllPilots(ctx)
	require.NoError(t, err)
	assert.Len(t, pilots, 4)
	assert.Equal(t, "Bix", pilots[0].Name)
	assert.Equal(t, "Ferrix", pilots[0].Home)
}

func TestManager_GetPilotID(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	pilotID, err := manager.GetPilotID(ctx, "Luthen")
	require.NoError(t, err)
	assert.Equal(t, "0RKUQmnYcUhGflhlrrsm9jthBJo2WjNOq", pilotID)
}

func TestManager_GetPilotsFromOrg(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	pilots, err := manager.GetPilotsFromOrg(ctx, "empire")
	require.NoError(t, err)
	assert.Len(t, pilots, 1)
	assert.Equal(t, "Moff", pilots[0].Name)
}

func TestManager_WriteTrack(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	points := []model.Point{
		{
			DateTime:   time.Date(2023, time.Month(8), 22, 8, 0, 0, 0, time.UTC),
			Latitude:   46.45669,
			Longitude:  6.88411,
			Altitude:   479,
			MsgType:    "UNLIMITED-TRACK",
			MsgContent: "",
		},
		{
			DateTime:   time.Date(2023, time.Month(8), 22, 8, 5, 0, 0, time.UTC),
			Latitude:   46.45549,
			Longitude:  6.8854,
			Altitude:   0,
			MsgType:    "OK",
			MsgContent: "Pilot has landed safely",
		},
	}
	err := manager.WriteTrack(ctx, "0Z7eRKM9rCcrima9ic2qqvNFjDjgf87fG", points)
	require.NoError(t, err)

	// Insert with no point.
	err = manager.WriteTrack(ctx, "0RKUQmnYcUhGflhlrrsm9jthBJo2WjNOq", []model.Point{})
	require.NoError(t, err)

	// Retrieve the track of the given day.
	pointsA, err := manager.GetTrackOfDay(ctx, "0Z7eRKM9rCcrima9ic2qqvNFjDjgf87fG", time.Date(2023, time.Month(8), 22, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Len(t, pointsA, 2)

	// Retrieve when no track.
	pointsB, err := manager.GetTrackOfDay(ctx, "0RKUQmnYcUhGflhlrrsm9jthBJo2WjNOq", time.Date(2023, time.Month(8), 23, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Empty(t, pointsB)

	// Retrieve the track since.
	pointsC, err := manager.GetTrackSince(ctx, "0Z7eRKM9rCcrima9ic2qqvNFjDjgf87fG", time.Date(2023, time.Month(8), 22, 8, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Len(t, pointsC, 1)
}
