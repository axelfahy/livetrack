package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"codnect.io/chrono"
	"fahy.xyz/livetrack/internal/bot"
	"fahy.xyz/livetrack/internal/db"
	"fahy.xyz/livetrack/internal/metrics"
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
	// Behaviour settings
	FetchInterval     time.Duration `envconfig:"FETCH_INTERVAL"     default:"4m"                          desc:"The interval between two fetches"`
	Organization      string        `envconfig:"ORGANIZATION"       required:"true"                       desc:"The organization of the pilots to retrieve"`
	LivetrackEndpoint string        `envconfig:"LIVETRACK_ENDPOINT" default:"https://livetrack.fahy.xyz/" desc:"The livetrack endpoint for the messages"`
	// Telegram config
	TelegramChannel string `envconfig:"TELEGRAM_CHANNEL" required:"true" desc:"The telegram channel to use"`
	TelegramToken   string `envconfig:"TELEGRAM_TOKEN"   required:"true" desc:"The telegram token to use"`
	// Metrics
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"bot" desc:"The Prometheus subsystem for the metrics"`
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
		logger.Error("running livetrack-bot", "error", err)
		os.Exit(1)
	}
}

//nolint:cyclop,maintidx // To be refactored.
func run(env envConfig, logger *slog.Logger) error {
	logger.Info("Livetrack bot is initializing...",
		"version", version.Version,
		"revision", version.Revision,
		"build_date", version.BuildDate,
		"os", version.GoOS,
		"os_arch", version.GoArch,
		"go_version", version.GoVersion,
	)

	logger = logger.With("component", "bot")
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

		if err := httpServer.Shutdown(ctx); err != nil { //nolint:contextcheck,lll // This is a bug https://github.com/kkHAIKE/contextcheck/issues/2
			return fmt.Errorf("shutting down HTTP server: %w", err)
		}

		return nil
	})

	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		env.PostgresUser, env.PostgresPassword,
		env.PostgresHost, env.PostgresPort, env.PostgresDBName,
	)
	logger.Info("Connecting to database", "URL", databaseURL)

	manager, err := db.NewManager(ctx, databaseURL, logger.With("component", "manager"), promMetrics)
	if err != nil {
		return fmt.Errorf("starting DB manager: %w", err)
	}

	logger.Debug("DB manager initialized")

	pilots, err := manager.GetPilotsFromOrg(ctx, env.Organization)
	if err != nil {
		return fmt.Errorf("retrieving pilots: %w", err)
	}

	bot, err := bot.New(env.TelegramChannel, env.TelegramToken, logger.With("component", "telegram-bot"), promMetrics)
	if err != nil {
		return fmt.Errorf("starting telegram bot: %w", err)
	}

	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		logger.Info("Removing all telegram messages", "time", time.Now())

		if err = bot.DeleteMessages(); err != nil {
			logger.Error("Removing messages", "error", err)
		}
		// Reload the pilots in case we have new ones and reset the tracks.
		pilots, err = manager.GetPilotsFromOrg(ctx, env.Organization)
		if err != nil {
			logger.Error("Retrieving pilots", "error", err)

			return
		}
	}, "0 0 0 * * *")

	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		now := time.Now()
		logger.Info("Retrieving tracks", "time", now)

		for i := range pilots {
			since := now.Truncate(time.Hour * 24)
			if pilots[i].Points != nil {
				since = pilots[i].Points[len(pilots[i].Points)-1].DateTime
			}

			logger.Info("Retrieving", "pilot", pilots[i], "since", since)

			points, err := manager.GetTrackSince(ctx, pilots[i].ID, since)
			if err != nil {
				logger.Error("Retrieving track for pilot", "pilot", pilots[i], "error", err)

				return
			}

			logger.Debug("Retrieved", "points", points)

			// If no point registered, send the start message.
			if len(points) > 0 && len(pilots[i].Points) == 0 {
				err = bot.SendMessage(fmt.Sprintf(
					"*%s* started tracking at %s\n%s",
					pilots[i].Name,
					points[0].DateTime.Format(time.RFC822),
					pilots[i].GetLivetrackURL(env.LivetrackEndpoint),
				))
				if err != nil {
					logger.Error("Sending message", "pilot", pilots[i], "error", err)

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

					sbbURL, err := pilots[i].GetSbbItinerary(point.Latitude, point.Longitude)
					if err != nil {
						logger.Error("Retrieving SBB itinerary", "pilot", pilots[i], "error", err)
					} else {
						sbbItinerary = fmt.Sprintf("[Back with SBB](%s)", sbbURL)
					}

					msg = fmt.Sprintf(
						"*%s* sent OK at %s\nFlight time: %s\nDistance ALL/TO: %.2f/%.2f km\n%s\n%s\n%s",
						pilots[i].Name,
						point.DateTime.Format(time.RFC822),
						pilots[i].GetFlightTime(),
						pilots[i].GetCumulativeDistance(),
						pilots[i].GetTakeOffDistance(),
						pilots[i].GetLivetrackURL(env.LivetrackEndpoint),
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
					if err = bot.SendMessage(msg); err != nil {
						logger.Error("Sending message", "msg", msg, "error", err)

						return
					}
				}
			}
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
