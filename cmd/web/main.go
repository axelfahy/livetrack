package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fahy.xyz/livetrack/internal/metrics"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sourcegraph/conc/pool"
)

type envConfig struct {
	// Logging
	Port     string         `envconfig:"PORT"      default:"3000" desc:"The port for the web interface"`
	LogLevel *slog.LevelVar `envconfig:"LOG_LEVEL" default:"info" desc:"The log level"`
	// Metrics
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"web" desc:"The Prometheus subsystem for the metrics"`

	APIEndpoint string `envconfig:"API_ENDPOINT" default:"https://livetrack.fahy.xyz/api/" desc:"The endpoint to retrieve the tracks"`
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
		logger.Error("running livetrack-web", "error", err)
		os.Exit(1)
	}
}

func run(env envConfig, logger *slog.Logger) error {
	logger.Info("Livetrack web is initializing...",
		"version", version.Version,
		"revision", version.Revision,
		"build_date", version.BuildDate,
		"os", version.GoOS,
		"os_arch", version.GoArch,
		"go_version", version.GoVersion,
	)

	logger = logger.With("component", "web")
	logger.Info("Configuration", "env", env)

	promMetrics, promReg, err := metrics.NewPrometheusMetrics(env.MetricsSubsystem)
	if err != nil {
		return fmt.Errorf("creating Prometheus metrics: %w", err)
	}

	router := mux.NewRouter()
	router.Handle(metrics.Path, promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ctxPool := pool.New().
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	httpServer := http.Server{
		Addr:         fmt.Sprint(":", env.Port),
		Handler:      router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	logger.Debug("HTTP server initialized")

	ctxPool.Go(func(ctx context.Context) error {
		httpServer.BaseContext = func(_ net.Listener) context.Context { return ctx }

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

		if err := httpServer.Shutdown(ctx); err != nil { //nolint:contextcheck,lll // This is a bug https://github.com/kkHAIKE/contextcheck/issues/2
			return fmt.Errorf("shutting down HTTP server: %w", err)
		}

		return nil
	})

	handler := NewHandler(env.APIEndpoint, logger.With("component", "handler"), promMetrics)

	router.HandleFunc("/", handler.Home)
	router.HandleFunc("/dates", handler.GetDates)
	router.HandleFunc("/tracks/{date}", handler.GetTracks)

	logger.Info("Livetrack web tracking initialized")

	if err = ctxPool.Wait(); err != nil {
		return fmt.Errorf("goroutine pool: %w", err)
	}

	return nil
}
