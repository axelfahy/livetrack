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

type pilotData struct {
	PilotData string `json:"pilotData"`
}

func (h *Handler) getTracksOfDay(ctx context.Context, date string) (pilotData, error) {
	url, err := url.JoinPath(h.endpoint, "/tracks/"+date)
	if err != nil {
		return pilotData{}, fmt.Errorf("parsing URL: %w", err)
	}

	h.logger.Info("[GET]", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return pilotData{}, fmt.Errorf("getting tracks: %w", err)
	}

	req.Header.Set("User-Agent", "Wget/1.13.4 (linux-gnu)")

	resp, err := h.client.Do(req)
	if err != nil {
		return pilotData{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data := make(map[string][]model.Point)
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return pilotData{}, fmt.Errorf("parsing tracks: %w", err)
	}

	// Filter out empty tracks.
	for pilot, points := range data {
		if len(points) == 0 {
			delete(data, pilot)
		}
	}
	h.logger.Debug("Tracks", "data", data)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return pilotData{}, fmt.Errorf("encoding response: %w", err)
	}

	return pilotData{PilotData: string(jsonData)}, nil
}

// home retrieves the track of the current day.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("[/]")

	today := time.Now().Format("2006-01-02")

	tmplData, err := h.getTracksOfDay(r.Context(), today)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", today, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := h.template.ExecuteTemplate(w, "index.html", tmplData); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

// getDates retrieves the last 5 dates having tracks.
//
// The date are templates as options in a selected box.
// The first entry is "Today" even if there is no entry for the current day.
func (h *Handler) GetDates(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("[/dates]")

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
	h.logger.Info("Get dates", "dates", dates, "selected", selectedDate)

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

	w.Header().Set("Content-Type", "text/html")

	if err := h.template.ExecuteTemplate(w, "options.html", options); err != nil {
		h.logger.ErrorContext(r.Context(), "Executing template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) GetTracks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date := vars["date"]

	tmplData, err := h.getTracksOfDay(r.Context(), date)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Retrieving tracks", "date", date, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := h.template.ExecuteTemplate(w, "index.html", tmplData); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}
