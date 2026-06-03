package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/gorilla/mux"

	"fahy.xyz/livetrack/internal/model"
)

const (
	timeout = 10 * time.Second
)

var errUnexpectedStatusCode = errors.New("unexpected status code")

var (
	ErrInvalidDate  = errors.New("invalid date format")
	ErrInvalidPilot = errors.New("invalid pilot name")
)

var (
	dateRegex  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	pilotRegex = regexp.MustCompile(`^[^\s]+$`)
)

func validateDate(date string) error {
	if !dateRegex.MatchString(date) {
		return ErrInvalidDate
	}

	return nil
}

func validatePilot(pilot string) error {
	if !pilotRegex.MatchString(pilot) {
		return ErrInvalidPilot
	}

	return nil
}

type handlerMetrics any

type templateData struct {
	Tracks         template.JS
	EventsEndpoint string
}

type Handler struct {
	apiEndpoint string
	sseEndpoint string
	client      *http.Client
	template    *template.Template
	logger      *slog.Logger
	metrics     handlerMetrics
}

//go:embed views/*
var views embed.FS

func NewHandler(apiEndpoint, sseEndpoint string, logger *slog.Logger, metrics handlerMetrics) *Handler {
	tViews := template.Must(template.ParseFS(views, "views/*"))

	client := &http.Client{
		Timeout: timeout,
	}

	return &Handler{
		apiEndpoint: apiEndpoint,
		sseEndpoint: sseEndpoint,
		client:      client,
		template:    tViews,
		logger:      logger,
		metrics:     metrics,
	}
}

// Home retrieves the track of the current day.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "[/]")

	today := time.Now().Format("2006-01-02")

	pilot := r.URL.Query().Get("pilot")
	if pilot == "" {
		h.logger.Debug("No pilot specified, showing all tracks.")
	}

	var (
		tracks template.JS
		err    error
	)

	//nolint:nestif // Too many errors to check; to be refactored.
	if pilot != "" {
		tracks, err = h.getTrackOfDayForPilot(r.Context(), today, pilot)
		if err != nil {
			if errors.Is(err, ErrInvalidDate) || errors.Is(err, ErrInvalidPilot) {
				h.logger.ErrorContext(r.Context(), "Retrieving track", "date", today, "pilot", pilot, "error", err)
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			h.logger.ErrorContext(r.Context(), "Retrieving track", "date", today, "pilot", pilot, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	} else {
		tracks, err = h.getTracksOfDay(r.Context(), today)
		if err != nil {
			if errors.Is(err, ErrInvalidDate) {
				h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", today, "error", err)
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", today, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}

	h.logger.DebugContext(r.Context(), "Tracks", "date", "today", "tracks", tracks)
	data := templateData{
		Tracks:         tracks,
		EventsEndpoint: h.sseEndpoint,
	}

	if err := h.template.ExecuteTemplate(w, "index.html", data); err != nil {
		h.logger.ErrorContext(r.Context(), "Executing template", "error", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

// GetDates retrieves the last 5 dates having tracks.
//
// The first entry is "Today" even if there is no entry for the current day.
func (h *Handler) GetDates(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "[/dates]")

	reqURL, err := url.JoinPath(h.apiEndpoint, "/dates")
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Parsing url", "error", err)
		http.Error(w, "error parsing url", http.StatusInternalServerError)

		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Creating request", "error", err)
		http.Error(w, "error creating request", http.StatusInternalServerError)

		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Retrieving dates", "error", err)
		http.Error(w, "error retrieving dates", http.StatusInternalServerError)

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.logger.ErrorContext(r.Context(), "Unexpected status from API", "status", resp.StatusCode, "body", string(body))
		http.Error(w, "error retrieving dates", http.StatusInternalServerError)

		return
	}

	dates := struct {
		Dates  []time.Time `json:"dates"`
		Counts []int       `json:"counts"`
	}{}

	if err = json.NewDecoder(resp.Body).Decode(&dates); err != nil {
		h.logger.ErrorContext(r.Context(), "Parsing dates", "error", err)
		http.Error(w, "error parsing dates", http.StatusInternalServerError)

		return
	}

	today := time.Now().Format("2006-01-02")

	result := []string{today}

	for _, date := range dates.Dates {
		dateFmt := date.Format("2006-01-02")
		if dateFmt != today {
			result = append(result, dateFmt)
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.ErrorContext(r.Context(), "Encoding dates", "error", err)
		http.Error(w, "error encoding dates", http.StatusInternalServerError)
	}
}

// GetTracks retrieves the track of the given date.
func (h *Handler) GetTracks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date := vars["date"]
	h.logger.InfoContext(r.Context(), fmt.Sprintf("[/tracks/%s]", date))

	tracks, err := h.getTracksOfDay(r.Context(), date)
	if err != nil {
		if errors.Is(err, ErrInvalidDate) {
			h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", date, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", date, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	h.logger.DebugContext(r.Context(), "Tracks", "date", date, "tracks", tracks)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(tracks))
}

// getTracksOfDay retrieves the tracks of the given day.
//
// Pilots without points are removed from the output.
// The structure is marshalled and returned as a `template.JS`.
func (h *Handler) getTracksOfDay(ctx context.Context, date string) (template.JS, error) {
	if err := validateDate(date); err != nil {
		return "", fmt.Errorf("validating date: %w", err)
	}

	reqURL, err := url.JoinPath(h.apiEndpoint, "/tracks/"+date)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	h.logger.Info("[GET]", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("unable to read body: %w", err)
		}

		return "", fmt.Errorf("%s %w: %d", string(body), errUnexpectedStatusCode, resp.StatusCode)
	}

	data := make(map[string][]model.Point)
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("parsing tracks: %w", err)
	}

	// Filter out empty tracks.
	for pilot, points := range data {
		if len(points) == 0 {
			delete(data, pilot)
		}
	}

	h.logger.DebugContext(ctx, "Tracks", "data", data)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshalling tracks: %w", err)
	}

	return template.JS(jsonData), nil //nolint:gosec // G203: JSON from json.Marshal is safe in JS context
}

// getTrackOfDayForPilot retrieves the pilot's track for the given day.
func (h *Handler) getTrackOfDayForPilot(ctx context.Context, date, pilot string) (template.JS, error) {
	if err := validateDate(date); err != nil {
		return "", fmt.Errorf("validating date: %w", err)
	}

	if err := validatePilot(pilot); err != nil {
		return "", fmt.Errorf("validating pilot: %w", err)
	}

	reqURL, err := url.JoinPath(h.apiEndpoint, "/track/"+date+"/"+pilot)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("%s %w: %d", string(body), errUnexpectedStatusCode, resp.StatusCode)
	}

	points := []model.Point{}
	if err := json.NewDecoder(resp.Body).Decode(&points); err != nil {
		return "", fmt.Errorf("parsing tracks: %w", err)
	}

	data := map[string][]model.Point{pilot: points}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshalling tracks: %w", err)
	}

	return template.JS(jsonData), nil //nolint:gosec // G203: JSON from json.Marshal is safe in JS context
}
