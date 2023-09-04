package web

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"fahy.xyz/livetrack/cmd"
	"fahy.xyz/livetrack/db"
	"fahy.xyz/livetrack/metrics"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
)

type webEnvConfig struct {
	// TODO
}

// enableCORS handles access control and  CORS middleware
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Main(ctx context.Context) {
	var env webEnvConfig
	logger, ok := ctx.Value(cmd.LogKey).(*slog.Logger)
	if !ok {
		panic("error retrieving logger from context")
	}
	logger = logger.With("component", "web")

	metrics, ok := ctx.Value(cmd.MetricsKey).(*metrics.Metrics)
	if !ok {
		logger.Error("Error retrieving metrics from context")
		return
	}

	manager, ok := ctx.Value(cmd.ManagerKey).(*db.Manager)
	if !ok {
		logger.Error("Error retrieving DB manager from context")
		return
	}

	router, ok := ctx.Value(cmd.MuxKey).(*mux.Router)
	if !ok {
		logger.Error("Error retrieving mux from context")
		return
	}

	if err := envconfig.Process("", &env); err != nil {
		logger.Error("error checking env variables", "error", err)
		return
	}
	logger.Info("Web configuration", "env", env)

	handler := NewHandler(manager, logger.With("component", "handler"), metrics)
	apiRouter := router.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/ping", handler.Ping).Methods(http.MethodGet)

	apiRouter.HandleFunc("/dates", handler.GetDatesWithCount).Methods(http.MethodGet)
	apiRouter.HandleFunc("/pilots", handler.GetPilots).Methods(http.MethodGet)
	apiRouter.HandleFunc("/tracks/{date}", handler.GetTracksOfDay).Methods(http.MethodGet)

	// router.Use(enableCORS)

	// router.PathPrefix("/").Handler(frontendHandler)
	logger.Info("Livetrack web module initialized")
	defer logger.Info("Closing...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
