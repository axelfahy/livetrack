package trackfetcher

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fahy.xyz/livetrack/cmd"
	"fahy.xyz/livetrack/db"
	"fahy.xyz/livetrack/fetcher"
	"fahy.xyz/livetrack/metrics"

	"github.com/kelseyhightower/envconfig"
	"github.com/procyon-projects/chrono"
)

type fetcherEnvConfig struct {
	// Fetchers
	SpotBaseUrl string `envconfig:"SPOT_BASE_URL" default:"https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/"`
	// Behaviour settings
	FetchInterval time.Duration `envconfig:"FETCH_INTERVAL" default:"4m"`
}

func Main(ctx context.Context) {
	var env fetcherEnvConfig
	logger, ok := ctx.Value(cmd.LogKey).(*slog.Logger)
	if !ok {
		panic("error retrieving logger from context")
	}
	logger = logger.With("component", "fetcher")

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

	if err := envconfig.Process("", &env); err != nil {
		logger.Error("error checking env variables", "error", err)
		return
	}
	logger.Info("Fetcher configuration", "env", env)

	pilots, err := manager.GetAllPilots(ctx)
	if err != nil {
		logger.Error("Error retrieving pilots", "error", err)
		return
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownChan)

	spotFetcher := fetcher.NewSpotFetcher(env.SpotBaseUrl, logger.With("component", "spot-fetcher"), metrics)

	taskScheduler := chrono.NewDefaultTaskScheduler()

	// Reload the pilots list each day.
	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		logger.Info("Reloading pilots", "time", time.Now())
		pilots, err = manager.GetAllPilots(ctx)
		if err != nil {
			logger.Error("Error retrieving pilots", "error", err)
			return
		}
	}, "0 0 0 * * *")

	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		logger.Info("Fetching tracker sources", "time", time.Now())
		for _, pilot := range pilots {
			points, err := spotFetcher.Fetch(pilot.Id)
			if err != nil {
				logger.Error("Error retrieving tracker for spot", "id", pilot.Id, "error", err)
				return
			}
			logger.Debug("Fetched", "points", points)

			if len(points) > 0 {
				if err = manager.WriteTrack(ctx, pilot.Id, points); err != nil {
					logger.Error("Error writing track", "id", pilot.Id, "track", points, "error", err)
				}
			}
		}
	}, env.FetchInterval)

	if err == nil {
		logger.Info("Task has been scheduled successfully.")
	}

	select {
	case <-shutdownChan:
		logger.Info("Shutdown signal received, exiting...")
		shutdownSchedulerChan := taskScheduler.Shutdown()
		<-shutdownSchedulerChan
		cancel()
		break
	case <-ctx.Done():
		logger.Info("Group context is done, exiting...")
		shutdownSchedulerChan := taskScheduler.Shutdown()
		<-shutdownSchedulerChan
		cancel()
		break
	}

	err = ctx.Err()
	if err != nil && err != context.Canceled {
		logger.Error("Got an error from the error group context", "err", err)
	} else {
		logger.Info("Shutdown properly completed")
	}
}
