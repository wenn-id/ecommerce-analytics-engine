package config_test

import (
	"testing"

	"ecommerce-analytics/internal/config"
)

func TestConfigLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("SYNC_INTERVAL_MINUTES", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("API_KEY", "")

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
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected default allowed origins, got %v", cfg.AllowedOrigins)
	}
	if cfg.APIKey != "" {
		t.Errorf("expected empty APIKey, got %s", cfg.APIKey)
	}
}

func TestConfigLoadCustom(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "custom.db")
	t.Setenv("SYNC_INTERVAL_MINUTES", "30")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")
	t.Setenv("API_KEY", "secret-test-key")

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
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "https://app.example.com" || cfg.AllowedOrigins[1] != "https://admin.example.com" {
		t.Errorf("expected custom allowed origins, got %v", cfg.AllowedOrigins)
	}
	if cfg.APIKey != "secret-test-key" {
		t.Errorf("expected APIKey secret-test-key, got %s", cfg.APIKey)
	}
}
