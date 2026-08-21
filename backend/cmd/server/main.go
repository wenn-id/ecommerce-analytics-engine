package main

import (
	"context"
	"log"
	"net/http"
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
	defer db.Close()

	ctx := context.Background()
	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
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
	_ = syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -30), time.Now())

	// Start background scheduler
	scheduler.Start(ctx, syncSvc, cfg.SyncIntervalMinutes)

	apiHandler := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: apiHandler,
	}

	log.Printf("Analytics Engine Server listening on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
