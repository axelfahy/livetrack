package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fahy.xyz/livetrack/internal/db"
	"fahy.xyz/livetrack/internal/metrics"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sourcegraph/conc/pool"
)

type envConfig struct {
	// Logging
	Port     string         `envconfig:"PORT"      default:"8080" desc:"The port for the HTTP server"`
	LogLevel *slog.LevelVar `envconfig:"LOG_LEVEL" default:"info" desc:"The log level"`
	// Postgres config
	PostgresHost     string `envconfig:"POSTGRES_HOST"     default:"localhost" desc:"The postgres host"`
	PostgresPort     int    `envconfig:"POSTGRES_PORT"     default:"5432"      desc:"The postgres port"`
	PostgresDBName   string `envconfig:"POSTGRES_DB_NAME"  default:"tracking"  desc:"The postgres database name"`
	PostgresUser     string `envconfig:"POSTGRES_USER"     required:"true"     desc:"The postgres user"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" required:"true"     desc:"The postgres password"`
	// Metrics
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"api" desc:"The Prometheus subsystem for the metrics"`
}

const (
	defaultReadTimeout    = 5 * time.Second
	defaultWriteTimeout   = 10 * time.Second
	defaultIdleTimeout    = 30 * time.Second
	serverShutdownTimeout = 10 * time.Second
)

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		slog.Default().Error("Processing env var", "error", err)
		os.Exit(1)
	}

	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: env.LogLevel})

	logger := slog.New(handler)

	if err := run(env, logger); err != nil {
		logger.Error("running livetrack-api", "error", err)
		os.Exit(1)
	}
}

func run(env envConfig, logger *slog.Logger) error {
	logger.Info("Livetrack api is initializing...",
		"version", version.Version,
		"revision", version.Revision,
		"build_date", version.BuildDate,
		"os", version.GoOS,
		"os_arch", version.GoArch,
		"go_version", version.GoVersion,
	)

	logger = logger.With("component", "api")
	logger.Info("Configuration", "env", env)

	promMetrics, promReg, err := metrics.NewPrometheusMetrics(env.MetricsSubsystem)
	if err != nil {
		return fmt.Errorf("creating Prometheus metrics: %w", err)
	}

	mux := mux.NewRouter()
	mux.Handle(metrics.Path, promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
	logger.Debug("Metrics initialized")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ctxPool := pool.New().
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	httpServer := http.Server{
		Addr:         fmt.Sprint(":", env.Port),
		Handler:      mux,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	logger.Debug("HTTP server initialized")

	ctxPool.Go(func(_ context.Context) error {
		if err := httpServer.ListenAndServe(); err != nil {
			return fmt.Errorf("HTTP server crashed: %w", err)
		}

		return nil
	})

	ctxPool.Go(func(ctx context.Context) error {
		<-ctx.Done()
		logger.Debug("Context is finished")

		ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down HTTP server: %w", err)
		}

		return nil
	})

	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", env.PostgresUser, env.PostgresPassword, env.PostgresHost, env.PostgresPort, env.PostgresDBName)
	logger.Info("Connecting to database", "URL", databaseUrl)

	manager, err := db.NewManager(ctx, databaseUrl, logger.With("component", "manager"), promMetrics)
	if err != nil {
		return fmt.Errorf("starting DB manager: %w", err)
	}

	logger.Debug("DB manager initialized")

	handler := NewHandler(manager, logger.With("component", "handler"), promMetrics)
	apiRouter := mux.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/ping", handler.Ping).Methods(http.MethodGet)

	apiRouter.HandleFunc("/dates", handler.GetDatesWithCount).Methods(http.MethodGet)
	apiRouter.HandleFunc("/pilots", handler.GetPilots).Methods(http.MethodGet)
	apiRouter.HandleFunc("/tracks/{date}", handler.GetTracksOfDay).Methods(http.MethodGet)

	logger.Info("Livetrack api module initialized")

	if err = ctxPool.Wait(); err != nil {
		return fmt.Errorf("goroutine pool: %w", err)
	}

	return nil
}
