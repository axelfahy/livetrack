package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"fahy.xyz/livetrack/internal/model"
	"github.com/gorilla/mux"
)

const (
	timeout = 10 * time.Second
)

type handlerMetrics interface{}

type Handler struct {
	endpoint string
	client   *http.Client
	template *template.Template
	logger   *slog.Logger
	metrics  handlerMetrics
}

// Option represents a single date option for the select element.
type Option struct {
	Date     string
	Label    string
	Selected bool
}

//go:embed views/*
var views embed.FS

func NewHandler(endpoint string, logger *slog.Logger, metrics handlerMetrics) *Handler {
	tViews := template.Must(template.ParseFS(views, "views/*"))

	client := &http.Client{
		Timeout: timeout,
	}

	return &Handler{
		endpoint: endpoint,
		client:   client,
		template: tViews,
		logger:   logger,
		metrics:  metrics,
	}
}

// Home retrieves the track of the current day.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "[/]")

	today := time.Now().Format("2006-01-02")

	jsonData, err := h.getTracksOfDay(r.Context(), today)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", today, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	h.logger.DebugContext(r.Context(), "Tracks", "date", "today", "json", jsonData)

	if err := h.template.ExecuteTemplate(w, "index.html", jsonData); err != nil {
		h.logger.ErrorContext(r.Context(), "Executing template", "error", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

// GetDates retrieves the last 5 dates having tracks.
//
// The date are templates as options in a selected box.
// The first entry is "Today" even if there is no entry for the current day.
func (h *Handler) GetDates(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "[/dates]")

	url, err := url.JoinPath(h.endpoint, "/dates")
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Parsing url", "error", err)
		http.Error(w, "error parsing url", http.StatusInternalServerError)
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Creating request", "error", err)
		http.Error(w, "error creating request", http.StatusInternalServerError)
	}

	req.Header.Set("User-Agent", "Wget/1.13.4 (linux-gnu)")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Retrieving dates", "error", err)
		http.Error(w, "error retrieving dates", http.StatusInternalServerError)
	}

	defer resp.Body.Close()

	dates := struct {
		Dates  []time.Time `json:"dates"`
		Counts []int       `json:"counts"`
	}{}

	if err = json.NewDecoder(resp.Body).Decode(&dates); err != nil {
		h.logger.ErrorContext(r.Context(), "Parsing dates", "error", err)
		http.Error(w, "error parsing dates", http.StatusInternalServerError)
	}

	today := time.Now().Format("2006-01-02")

	selectedDate := r.URL.Query().Get("date")
	h.logger.InfoContext(r.Context(), "Get dates", "dates", dates, "selected", selectedDate)

	options := []Option{
		{Date: today, Label: "Today", Selected: selectedDate == today || selectedDate == ""},
	}

	for _, date := range dates.Dates {
		dateFmt := date.Format("2006-01-02")
		if dateFmt != today {
			options = append(options, Option{
				Date:     dateFmt,
				Label:    dateFmt,
				Selected: dateFmt == selectedDate,
			})
		}
	}

	if err := h.template.ExecuteTemplate(w, "options.html", options); err != nil {
		h.logger.ErrorContext(r.Context(), "Executing template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetTracks retrieves the track of the given date.
func (h *Handler) GetTracks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date := vars["date"]
	h.logger.InfoContext(r.Context(), fmt.Sprintf("[/tracks/%s]", date))

	jsonData, err := h.getTracksOfDay(r.Context(), date)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", date, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	h.logger.DebugContext(r.Context(), "Tracks", "date", date, "json", jsonData)

	if err := h.template.ExecuteTemplate(w, "index.html", jsonData); err != nil {
		h.logger.ErrorContext(r.Context(), "Executing template", "error", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

// getTracksOfDay retrieves the tracks of the given day.
//
// Pilots without points are removed from the output.
// It structure is Marshalled and return as a string.
func (h *Handler) getTracksOfDay(ctx context.Context, date string) (string, error) {
	url, err := url.JoinPath(h.endpoint, "/tracks/"+date)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	h.logger.Info("[GET]", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("getting tracks: %w", err)
	}

	req.Header.Set("User-Agent", "Wget/1.13.4 (linux-gnu)")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	h.logger.Info("body", "body", resp.Body)

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

	// Convert the JSON data back to a string
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshalling tracks: %w", err)
	}

	return string(jsonData), nil
}
