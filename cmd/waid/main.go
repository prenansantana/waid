package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prenansantana/waid/internal/api"
	"github.com/prenansantana/waid/internal/config"
	internalnats "github.com/prenansantana/waid/internal/nats"
	"github.com/prenansantana/waid/internal/notifier"
	"github.com/prenansantana/waid/internal/resolver"
	"github.com/prenansantana/waid/internal/store"
	_ "modernc.org/sqlite"
)

// Build-time variables set by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate API key is set and meets minimum security requirements.
	if cfg.Server.APIKey == "" {
		fmt.Fprintln(os.Stderr, "FATAL: WAID_SERVER_API_KEY is required. Set it via environment variable or waid.yaml.")
		fmt.Fprintln(os.Stderr, "  Example: export WAID_SERVER_API_KEY=$(openssl rand -hex 32)")
		os.Exit(1)
	}
	if len(cfg.Server.APIKey) < 16 {
		fmt.Fprintln(os.Stderr, "FATAL: WAID_SERVER_API_KEY must be at least 16 characters.")
		os.Exit(1)
	}

	logger := newLogger(cfg.Logging)
	slog.SetDefault(logger)

	logger.Info("starting waid",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("date", date),
		slog.String("db_driver", cfg.Database.Driver),
		slog.Int("port", cfg.Server.Port),
	)

	// Initialize store (runs migrations for SQLite automatically).
	st, err := store.New(cfg.Database.Driver, cfg.Database.URL)
	if err != nil {
		logger.Error("failed to initialize store", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer st.Close()

	// Initialize NATS (embedded or external).
	natsClient, err := internalnats.NewNATS(cfg.NATS, logger)
	if err != nil {
		logger.Error("failed to initialize NATS", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize resolver.
	res := resolver.New(st, cfg.Resolver.AutoCreate, cfg.Resolver.DefaultCountry, logger)

	// Initialize webhook store using the same database.
	ws, err := newWebhookStore(cfg.Database.Driver, cfg.Database.URL)
	if err != nil {
		logger.Error("failed to initialize webhook store", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer ws.Close()

	// Initialize notifier.
	ntf := notifier.NewNotifier(ws, logger)

	// Create API server with all dependencies.
	apiSrv := api.NewWithWebhookStore(cfg, st, res, ws, ntf, natsClient, logger)
	apiSrv.SetVersion(version)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      apiSrv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			errCh <- err
		}
	}()

	select {
	case <-stop:
	case <-errCh:
	}
	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := natsClient.Close(); err != nil {
		logger.Warn("NATS close error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}

// newWebhookStore opens a *sql.DB connection for the webhook store.
// For SQLite it reuses the same file; for Postgres it opens a stdlib connection.
func newWebhookStore(driver, url string) (notifier.WebhookStore, error) {
	switch driver {
	case "sqlite":
		db, err := sql.Open("sqlite", url)
		if err != nil {
			return nil, fmt.Errorf("webhook store: open sqlite: %w", err)
		}
		return notifier.NewSQLiteWebhookStore(db), nil
	case "postgres":
		db, err := sql.Open("pgx", url)
		if err != nil {
			return nil, fmt.Errorf("webhook store: open postgres: %w", err)
		}
		return notifier.NewPostgresWebhookStore(db), nil
	default:
		return nil, fmt.Errorf("webhook store: unsupported driver: %s", driver)
	}
}

// newLogger constructs a structured slog.Logger based on the logging config.
func newLogger(cfg config.LoggingConfig) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
