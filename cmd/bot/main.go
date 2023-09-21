package telegrambot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"fahy.xyz/livetrack/bot"
	"fahy.xyz/livetrack/cmd"
	"fahy.xyz/livetrack/db"
	"fahy.xyz/livetrack/metrics"

	"github.com/kelseyhightower/envconfig"
	"github.com/procyon-projects/chrono"
)

type botEnvConfig struct {
	// Behaviour settings
	FetchInterval time.Duration `envconfig:"FETCH_INTERVAL" default:"4m"`
	// Organization to filter the pilots to retrieve
	Organization string `envconfig:"ORGANIZATION" required:"true"`
	// Telegram config
	TelegramChannel string `envconfig:"TELEGRAM_CHANNEL" required:"true"`
	TelegramToken   string `envconfig:"TELEGRAM_TOKEN" required:"true"`
}

func Main(ctx context.Context) {
	var env botEnvConfig
	logger, ok := ctx.Value(cmd.LogKey).(*slog.Logger)
	if !ok {
		panic("error retrieving logger from context")
	}
	logger = logger.With("component", "bot")

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
		logger.Error("Error checking env variables", "error", err)
		return
	}
	logger.Info("Bot configuration", "env", env)

	pilots, err := manager.GetPilotsFromOrg(ctx, env.Organization)
	if err != nil {
		logger.Error("Error retrieving pilots", "error", err)
		return
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownChan)

	bot, err := bot.New(env.TelegramChannel, env.TelegramToken, logger.With("component", "telegram-bot"), metrics)
	if err != nil {
		logger.Error("Error starting telegram bot", "error", err)
		return
	}

	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		logger.Info("Removing all telegram messages", "time", time.Now())
		if err = bot.DeleteMessages(); err != nil {
			logger.Error("Error removing messages", "error", err)
		}
		// Reload the pilots in case we have new ones and reset the tracks.
		pilots, err = manager.GetPilotsFromOrg(ctx, env.Organization)
		if err != nil {
			logger.Error("Error retrieving pilots", "error", err)
			return
		}
	}, "0 0 0 * * *")

	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		now := time.Now()
		logger.Info("Retrieving tracks", "time", now)
		for i := 0; i < len(pilots); i++ {
			since := now.Truncate(time.Hour * 24)
			if pilots[i].Points != nil {
				since = pilots[i].Points[len(pilots[i].Points)-1].DateTime
			}
			logger.Info("Retrieving", "pilot", pilots[i], "since", since)
			points, err := manager.GetTrackSince(ctx, pilots[i].ID, since)
			if err != nil {
				logger.Error("Error retrieving track for pilot", "pilot", pilots[i], "error", err)
				return
			}
			logger.Debug("Retrieved", "points", points)

			// If no point registered, send the start message.
			if len(points) > 0 && len(pilots[i].Points) == 0 {
				err = bot.SendMessage(fmt.Sprintf(
					"*%s* started tracking at %s",
					pilots[i].Name,
					points[0].DateTime.Format(time.RFC822),
				))
				if err != nil {
					logger.Error("Error sending message", "pilot", pilots[i], "error", err)
					return
				}
			}

			for _, point := range points {
				if slices.Contains(pilots[i].Points, point) {
					break
				}
				pilots[i].Points = append(pilots[i].Points, point)
				msg := ""
				switch point.MsgType {
				case "OK":
					sbbItinerary := "No SBB itinerary"
					sbbUrl, err := pilots[i].GetSbbItinerary(point.Latitude, point.Longitude)
					if err != nil {
						logger.Error("Error retrieving SBB itinerary", "pilot", pilots[i], "error", err)
					} else {
						sbbItinerary = fmt.Sprintf("[Back with SBB](%s)", sbbUrl)
					}
					msg = fmt.Sprintf(
						"*%s* sent OK at %s\nFlight time: %s\nDistance ALL/TO: %.2f/%.2f km\n%s\n%s",
						pilots[i].Name,
						point.DateTime.Format(time.RFC822),
						pilots[i].GetFlightTime(),
						pilots[i].GetCumulativeDistance(),
						pilots[i].GetTakeOffDistance(),
						point.GetItineraryURL(),
						sbbItinerary,
					)
				case "HELP", "MOVE", "CUSTOM":
					msg = fmt.Sprintf(
						"*%s* sent %s!!!",
						pilots[i].Name,
						point.MsgContent,
					)
				case "START":
					msg = fmt.Sprintf(
						"*%s* started tracking again at %s",
						pilots[i].Name,
						point.DateTime,
					)
				case "OFF":
					msg = fmt.Sprintf(
						"*%s* turned the tracking off at %s",
						pilots[i].Name,
						point.DateTime,
					)
				default:
					logger.Warn("Message type unknown", "point", point)
				}
				if msg != "" {
					err = bot.SendMessage(msg)
					if err != nil {
						logger.Error("Error sending message", "msg", msg, "error", err)
						return
					}
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
