package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"fahy.xyz/livetrack/internal/db"
	"github.com/gorilla/mux"
)

const numberOfDates = 5

type Handler struct {
	manager *db.Manager

	logger  *slog.Logger
	metrics handlerMetrics
}

type handlerMetrics interface{}

func NewHandler(manager *db.Manager, logger *slog.Logger, metrics handlerMetrics) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
		metrics: metrics,
	}
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "Route triggered", "method", "GET", "route", "[/ping]")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("{\"status\": \"pong\"}")); err != nil {
		h.logger.Error("Error pinging", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (h *Handler) GetDatesWithCount(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "Route triggered", "method", "GET", "route", "[/dates]")

	dates, counts, err := h.manager.GetDatesWithCount(r.Context(), numberOfDates)
	if err != nil {
		h.logger.Error("Error retrieving dates", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
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

		return
	}
}

func (h *Handler) GetPilots(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "Route triggered", "method", "GET", "route", "[/pilots]")

	pilots, err := h.manager.GetAllPilots(r.Context())
	if err != nil {
		h.logger.Error("Error retrieving pilots", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.NewEncoder(w).Encode(pilots); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (h *Handler) GetTracksOfDay(w http.ResponseWriter, r *http.Request) {
	h.logger.DebugContext(r.Context(), "Route triggered", "method", "GET", "route", "[/tracks/{date}]")

	date, err := time.Parse("2006-01-02", mux.Vars(r)["date"])
	if err != nil {
		h.logger.Error("Error retrieving parameter", "parameter", "date")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	h.logger.InfoContext(r.Context(), "Route triggered", "method", "GET", "route", "[/tracks/{date}]", "date", date)

	tracks, err := h.manager.GetAllTracksOfDay(r.Context(), date)
	if err != nil {
		h.logger.Error("Error retrieving pilots", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.NewEncoder(w).Encode(tracks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (h *Handler) GetTrackOfDayForPilot(w http.ResponseWriter, r *http.Request) {
	h.logger.DebugContext(r.Context(), "Route triggered", "method", "GET", "route", "[/track/{date}/{pilot}]")
	date, err := time.Parse("2006-01-02", mux.Vars(r)["date"])
	if err != nil {
		h.logger.Error("Error retrieving parameter", "parameter", "date")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	pilot := mux.Vars(r)["pilot"]
	h.logger.InfoContext(r.Context(), "Route triggered", "method", "GET", "route", "[/track/{date}/{pilot}]", "pilot", pilot, "date", date)

	pilotID, err := h.manager.GetPilotID(r.Context(), pilot)
	if err != nil {
		h.logger.Error("Error retrieving pilot ID", "pilot", pilot)

		code := http.StatusInternalServerError
		if errors.Is(err, db.ErrPilotNotFound) {
			code = http.StatusNotFound
		}

		http.Error(w, err.Error(), code)

		return
	}

	tracks, err := h.manager.GetTrackOfDay(r.Context(), pilotID, date)
	if err != nil {
		h.logger.Error("Error retrieving pilot's track", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.NewEncoder(w).Encode(tracks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}
