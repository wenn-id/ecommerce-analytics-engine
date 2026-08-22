package config_test

import (
	"errors"
	"testing"

	"ecommerce-analytics/internal/config"
)

func TestConfigLoadNewFields(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_FORMAT", "JSON")
	t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.1/8, 172.16.0.0/12 ,10.0.0.7,not-a-cidr")

	cfg := config.Load()
	if cfg.Env != "production" {
		t.Errorf("expected env production, got %s", cfg.Env)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected log format json (case-insensitive), got %s", cfg.LogFormat)
	}
	// Plain IPs are normalized to /32 and invalid entries dropped.
	want := []string{"127.0.0.1/8", "172.16.0.0/12", "10.0.0.7/32"}
	if len(cfg.TrustedProxies) != len(want) {
		t.Fatalf("expected %d trusted proxies, got %v", len(want), cfg.TrustedProxies)
	}
	for i := range want {
		if cfg.TrustedProxies[i] != want[i] {
			t.Errorf("trusted proxy %d: expected %s, got %s", i, want[i], cfg.TrustedProxies[i])
		}
	}
}

func TestConfigLoadDefaultsNewFields(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_FORMAT", "")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")

	cfg := config.Load()
	if cfg.Env != "development" {
		t.Errorf("expected default env development, got %s", cfg.Env)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected default log format text, got %s", cfg.LogFormat)
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Errorf("expected no trusted proxies by default, got %v", cfg.TrustedProxies)
	}
}

func TestConfigValidateProductionRequiresAPIKey(t *testing.T) {
	cfg := &config.Config{Env: "production", APIKey: ""}
	if err := cfg.Validate(); !errors.Is(err, config.ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}

	cfg = &config.Config{Env: "production", APIKey: "secret"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	// Development keeps keyless mode available for local demos.
	cfg = &config.Config{Env: "development", APIKey: ""}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected keyless development config to be valid, got %v", err)
	}
}

func TestConfigEnvNormalization(t *testing.T) {
	// Whitespace and aliases must not silently downgrade the strict profile.
	for _, raw := range []string{" production ", "PRODUCTION", "Prod"} {
		t.Setenv("APP_ENV", raw)
		if cfg := config.Load(); !cfg.IsProduction() {
			t.Errorf("APP_ENV=%q should select the production profile, got %q", raw, cfg.Env)
		}
	}
	t.Setenv("APP_ENV", " dev ")
	if cfg := config.Load(); cfg.Env != "development" {
		t.Errorf("APP_ENV=' dev ' should normalize to development, got %q", cfg.Env)
	}
}

func TestConfigValidateRejectsUnknownEnv(t *testing.T) {
	// A typo like "poduction" must fail loudly, not run with lax development
	// behavior.
	cfg := &config.Config{Env: "poduction"}
	if err := cfg.Validate(); !errors.Is(err, config.ErrInvalidEnv) {
		t.Fatalf("expected ErrInvalidEnv, got %v", err)
	}
}
