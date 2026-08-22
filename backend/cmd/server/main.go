package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/handler"
	"ecommerce-analytics/internal/scheduler"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func main() {
	cfg := config.Load()

	setupLogger(cfg)

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	db, err := store.NewDB(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		slog.Info("closing database connection")
		if err := db.Close(); err != nil {
			slog.Error("error closing database", "error", err)
		} else {
			slog.Info("database connection closed cleanly")
		}
	}()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(rootCtx); err != nil {
		slog.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	connectors := connector.BuildConnectors(nil, cfg.IsProduction())

	syncSvc := service.NewSyncService(repo, connectors)
	analyticsSvc := service.NewAnalyticsService(repo)

	// Perform initial sync
	slog.Info("performing initial multi-channel sync")
	if err := syncSvc.SyncAll(rootCtx, time.Now().AddDate(0, 0, -30), time.Now()); err != nil {
		if errors.Is(err, service.ErrSyncInProgress) {
			slog.Warn("initial sync skipped: another sync is already running")
		} else {
			slog.Warn("initial sync failed", "error", err)
		}
	}

	// Start background scheduler with cancellable context
	scheduler.Start(rootCtx, syncSvc, cfg.SyncIntervalMinutes)

	apiHandler := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, cfg)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           apiHandler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Channel to listen for errors from the server goroutine
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("analytics engine server listening", "port", cfg.Port, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Listen for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-sigChan:
		slog.Info("received shutdown signal, initiating graceful shutdown", "signal", sig.String())
	}

	// Cancel background scheduler context
	rootCancel()
	slog.Info("background scheduler stopped")

	// Create context with timeout for draining active HTTP requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown encountered an error", "error", err)
	} else {
		slog.Info("HTTP server stopped gracefully")
	}

	slog.Info("shutdown complete")
}

// setupLogger installs the process-wide slog logger: JSON for machine
// ingestion in deployments, human-readable text otherwise (#39).
func setupLogger(cfg *config.Config) {
	var handler slog.Handler
	handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(handler))

	fmt.Fprintf(os.Stderr, "logger initialized (format=%s)\n", cfg.LogFormat)
}
