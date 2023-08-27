package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"fahy.xyz/livetrack/db"

	"github.com/gorilla/mux"
)

const numberOfDates = 5

type Handler struct {
	manager *db.Manager

	logger  *slog.Logger
	metrics handlerMetrics
}

type handlerMetrics interface{}

type emptyHandlerMetrics struct{}

func NewHandler(manager *db.Manager, logger *slog.Logger, metrics handlerMetrics) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
		metrics: metrics,
	}
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Route triggered", "method", "GET", "route", "[/ping]")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{\"status\": \"pong\"}"))
}

func (h *Handler) GetDatesWithCount(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Route triggered", "method", "GET", "route", "[/dates]")
	dates, counts, err := h.manager.GetDatesWithCount(context.Background(), numberOfDates)
	if err != nil {
		h.logger.Error("Error retrieving dates", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-type", "application/json")
	if err := json.NewEncoder(w).Encode(
		struct {
			Dates  []time.Time `json:"dates"`
			Counts []int       `json:"counts"`
		}{
			Dates:  dates,
			Counts: counts,
		}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) GetPilots(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Route triggered", "method", "GET", "route", "[/pilots]")
	pilots, err := h.manager.GetAllPilots(context.Background())
	if err != nil {
		h.logger.Error("Error retrieving pilots", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-type", "application/json")
	if err := json.NewEncoder(w).Encode(pilots); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) GetTracksOfDay(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Route triggered", "method", "GET", "route", "[/tracks/{date}]")
	date, err := time.Parse("2006-01-02", mux.Vars(r)["date"])
	if err != nil {
		h.logger.Error("Error retrieving parameter", "parameter", "date")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	tracks, err := h.manager.GetAllTracksOfDay(context.Background(), date)
	if err != nil {
		h.logger.Error("Error retrieving pilots", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-type", "application/json")
	if err := json.NewEncoder(w).Encode(tracks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
