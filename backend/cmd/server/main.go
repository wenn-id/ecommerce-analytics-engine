package main

import (
	"context"
	"errors"
	"log"
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

	db, err := store.NewDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		log.Println("Closing database connection...")
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		} else {
			log.Println("Database connection closed cleanly.")
		}
	}()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(rootCtx); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	connectors := []connector.PlatformConnector{
		connector.NewMetaConnector(),
		connector.NewTikTokConnector(),
		connector.NewShopeeConnector(),
	}

	syncSvc := service.NewSyncService(repo, connectors)
	analyticsSvc := service.NewAnalyticsService(repo)

	// Perform initial sync
	log.Println("Performing initial multi-channel sync...")
	_ = syncSvc.SyncAll(rootCtx, time.Now().AddDate(0, 0, -30), time.Now())

	// Start background scheduler with cancellable context
	scheduler.Start(rootCtx, syncSvc, cfg.SyncIntervalMinutes)

	apiHandler := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: apiHandler,
	}

	// Channel to listen for errors from the server goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Analytics Engine Server listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Listen for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received shutdown signal (%v). Initiating graceful shutdown...", sig)
	}

	// Cancel background scheduler context
	rootCancel()
	log.Println("Background scheduler stopped.")

	// Create context with timeout for draining active HTTP requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown encountered an error: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully.")
	}

	log.Println("Shutdown complete.")
}
