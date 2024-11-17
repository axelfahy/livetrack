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
	"fahy.xyz/livetrack/internal/fetcher"
	"fahy.xyz/livetrack/internal/metrics"
	"fahy.xyz/livetrack/internal/model"

	"github.com/kelseyhightower/envconfig"
	"github.com/procyon-projects/chrono"
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
	// Fetchers
	SpotBaseUrl   string `envconfig:"SPOT_BASE_URL"   default:"https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/" desc:"The base URL for the SPOT tracking"`
	GarminBaseUrl string `envconfig:"GARMIN_BASE_URL" default:"https://share.garmin.com/Feed/Share/"                                        desc:"The base URL for the garmin tracking"`
	// Behaviour settings
	FetchInterval time.Duration `envconfig:"FETCH_INTERVAL" default:"4m" desc:"The interval between two fetches"`
	// Metrics
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"fetcher" desc:"The Prometheus subsystem for the metrics"`
}

const (
	garminTracker = "garmin"
	spotTracker   = "spot"

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
		logger.Error("running livetrack-fetcher", "error", err)
		os.Exit(1)
	}
}

func run(env envConfig, logger *slog.Logger) error {
	logger.Info("Livetrack fetcher is initializing...",
		"version", version.Version,
		"revision", version.Revision,
		"build_date", version.BuildDate,
		"os", version.GoOS,
		"os_arch", version.GoArch,
		"go_version", version.GoVersion,
	)

	logger = logger.With("component", "fetcher")
	logger.Info("Configuration", "env", env)

	promMetrics, promReg, err := metrics.NewPrometheusMetrics(env.MetricsSubsystem)
	if err != nil {
		return fmt.Errorf("creating Prometheus metrics: %w", err)
	}

	mux := http.NewServeMux()
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

	pilots, err := manager.GetAllPilots(ctx)
	if err != nil {
		return fmt.Errorf("retrieving pilots: %w", err)
	}

	garminFetcher := fetcher.NewGarminFetcher(env.GarminBaseUrl, logger.With("component", "garmin-fetcher"), promMetrics)
	spotFetcher := fetcher.NewSpotFetcher(env.SpotBaseUrl, logger.With("component", "spot-fetcher"), promMetrics)

	taskScheduler := chrono.NewDefaultTaskScheduler()

	// Reload the pilots list each day.
	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		logger.Info("Reloading pilots", "time", time.Now())

		pilots, err = manager.GetAllPilots(ctx)
		if err != nil {
			logger.Error("Retrieving pilots", "error", err)
			return
		}
	}, "0 0 0 * * *")

	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		logger.Info("Fetching tracker sources", "time", time.Now())

		for _, pilot := range pilots {
			var points []model.Point

			switch pilot.TrackerType {
			case garminTracker:
				points, err = garminFetcher.Fetch(pilot.ID)
				if err != nil {
					logger.Error("Retrieving tracker for garmin", "ID", pilot.ID, "error", err)
					continue
				}
			case spotTracker:
				points, err = spotFetcher.Fetch(pilot.ID)
				if err != nil {
					logger.Error("Retrieving tracker for spot", "ID", pilot.ID, "error", err)
					continue
				}
			default:
				logger.Error("Unknown tracker", "pilot", pilot)
				continue
			}

			logger.Debug("Fetched", "points", points)

			if len(points) > 0 {
				if err = manager.WriteTrack(ctx, pilot.ID, points); err != nil {
					logger.Error("Writing track", "ID", pilot.ID, "track", points, "error", err)
				}
			}

			time.Sleep(5 * time.Second)
		}
	}, env.FetchInterval)

	if err == nil {
		logger.Info("Task has been scheduled successfully")
	}

	if err = ctxPool.Wait(); err != nil {
		return fmt.Errorf("goroutine pool: %w", err)
	}

	return nil
}
