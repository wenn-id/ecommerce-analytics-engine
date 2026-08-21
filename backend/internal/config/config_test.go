package config_test

import (
	"os"
	"testing"

	"ecommerce-analytics/internal/config"
)

func TestConfigLoadDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("SYNC_INTERVAL_MINUTES")

	cfg := config.Load()
	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "analytics.db" {
		t.Errorf("expected db analytics.db, got %s", cfg.DatabasePath)
	}
	if cfg.SyncIntervalMinutes != 60 {
		t.Errorf("expected sync interval 60, got %d", cfg.SyncIntervalMinutes)
	}
}

func TestConfigLoadCustom(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DB_PATH", "custom.db")
	os.Setenv("SYNC_INTERVAL_MINUTES", "30")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_PATH")
		os.Unsetenv("SYNC_INTERVAL_MINUTES")
	}()

	cfg := config.Load()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "custom.db" {
		t.Errorf("expected db custom.db, got %s", cfg.DatabasePath)
	}
	if cfg.SyncIntervalMinutes != 30 {
		t.Errorf("expected sync interval 30, got %d", cfg.SyncIntervalMinutes)
	}
}
