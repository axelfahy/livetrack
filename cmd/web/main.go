package web

import (
	"context"
	"fmt"
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

// access control and  CORS middleware
func accessControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS,PUT")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")
		fmt.Print("in access control")
		fmt.Print(next)

		if r.Method == "OPTIONS" {
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

	router.HandleFunc("/ping", handler.Ping).Methods(http.MethodGet)

	router.HandleFunc("/dates", handler.GetDatesWithCount).Methods(http.MethodGet)
	router.HandleFunc("/pilots", handler.GetPilots).Methods(http.MethodGet)
	router.HandleFunc("/tracks/{date}", handler.GetTracksOfDay).Methods(http.MethodGet)

	//router.Use(accessControlMiddleware)

	//router.PathPrefix("/").Handler(frontendHandler)
	logger.Info("Livetrack web module initialized")
	defer logger.Info("Closing...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
