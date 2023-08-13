package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sqooba/go-common/version"

	"fahy.xyz/livetrack/cmd"
	telegrambot "fahy.xyz/livetrack/cmd/bot"
	fetcher "fahy.xyz/livetrack/cmd/fetcher"
	"fahy.xyz/livetrack/db"
	"fahy.xyz/livetrack/metrics"
)

type mainEnvConfig struct {
	// Logging
	Port     string `envconfig:"PORT" default:"8080"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"debug"`

	// Postgres config
	// User and password must be store in a `.env` file
	PostgresHost     string `envconfig:"POSTGRES_HOST" default:"localhost"`
	PostgresPort     int    `envconfig:"POSTGRES_PORT" default:"5432"`
	PostgresDBName   string `envconfig:"POSTGRES_DB_NAME" default:"tracking"`
	PostgresUser     string `envconfig:"POSTGRES_User" required:"true"`
	PostgresPassword string `envconfig:"POSTGRES_User" required:"true"`

	// Metrics settings
	MetricsNamespace string `envconfig:"METRICS_NAMESPACE" default:"livetrack"`
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"main"`
	MetricsPath      string `envconfig:"METRICS_PATH" default:"/metrics"`
}

func main() {
	logLevel := &slog.LevelVar{}
	opts := slog.HandlerOptions{
		Level: logLevel,
	}
	handler := slog.NewJSONHandler(os.Stdout, &opts)
	logger := slog.New(handler)

	now := time.Now()

	logger.Info("Livetrack is initializing...",
		slog.String("version", version.Version),
		slog.String("commit", version.GitCommit),
		slog.String("build-date", version.BuildDate),
		slog.String("os-arch", version.OsArch),
		slog.Time("time", now),
	)

	var env mainEnvConfig
	if err := envconfig.Process("", &env); err != nil {
		logger.Error("Failed to process env var", "error", err)
		return
	}
	logger.Info("Main configuration", "env", env)

	flag.Parse()

	switch strings.ToLower(env.LogLevel) {
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "error":
		logLevel.Set(slog.LevelError)
	case "warn":
		logLevel.Set(slog.LevelError)
	default:
		slog.Warn("Log level not valid, setting info level")
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownChan)

	metrics := metrics.InitPrometheus(env.MetricsNamespace, env.MetricsSubsystem)

	mux := http.NewServeMux()

	// Setup metrics endpoint
	mux.Handle(env.MetricsPath, promhttp.Handler())

	s := http.Server{Addr: fmt.Sprint(":", env.Port), Handler: mux}
	go func() {
		logger.Error("listen and serve", s.ListenAndServe())
		return
	}()
	logger.Debug("Server http initialized", "port", env.Port)

	ctx := context.Background()

	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", env.PostgresUser, env.PostgresPassword, env.PostgresHost, env.PostgresPort, env.PostgresDBName)
	logger.Info("Connecting to database", "url", databaseUrl)
	manager, err := db.NewManager(ctx, databaseUrl, logger, metrics)
	if err != nil {
		logger.Error("Error starting DB manager", "error", err)
		return
	}

	// Prepare and run specific command
	ctx = context.WithValue(ctx, cmd.LogKey, logger)
	ctx = context.WithValue(ctx, cmd.ManagerKey, manager)
	ctx = context.WithValue(ctx, cmd.MetricsKey, metrics)
	ctx = context.WithValue(ctx, cmd.MuxKey, mux)

	// Inferring command to use using first the env variable and then the CLI argument.
	command := ""
	if flag.Arg(0) != "" {
		command = flag.Arg(0)
	}

	logger.Info("Starting", "command", command)

	switch command {
	case "bot":
		telegrambot.Main(ctx)
	case "fetcher":
		fetcher.Main(ctx)
	default:
		logger.Error("invalid command", "command", command)
		return
	}
}
