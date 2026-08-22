package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                string
	DatabasePath        string
	SyncIntervalMinutes int
	AllowedOrigins      []string
	APIKey              string
	// Env selects the deployment profile: "development" (default) or "production".
	// In production an empty APIKey is a fatal configuration error (#56).
	Env string
	// TrustedProxies lists proxy IPs/CIDRs whose X-Forwarded-For/X-Real-IP
	// headers are trusted when resolving the real client IP for rate
	// limiting. Empty means only loopback peers are trusted (#57).
	TrustedProxies []string
	// LogFormat selects the slog output format: "text" (default) or "json".
	LogFormat string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "analytics.db"
	}
	syncInterval := 60
	if s := os.Getenv("SYNC_INTERVAL_MINUTES"); s != "" {
		if val, err := strconv.Atoi(s); err == nil && val > 0 {
			syncInterval = val
		}
	}
	apiKey := os.Getenv("API_KEY")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	// Trim and lowercase so " Production " still selects the strict profile;
	// unknown profiles are rejected by Validate instead of silently behaving
	// like development.
	env = strings.ToLower(strings.TrimSpace(env))
	switch env {
	case "dev":
		env = "development"
	case "prod":
		env = "production"
	}

	logFormat := strings.ToLower(os.Getenv("LOG_FORMAT"))
	if logFormat != "json" {
		logFormat = "text"
	}

	allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsStr != "" {
		for _, o := range strings.Split(allowedOriginsStr, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	} else {
		allowedOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	return &Config{
		Port:                port,
		DatabasePath:        dbPath,
		SyncIntervalMinutes: syncInterval,
		AllowedOrigins:      allowedOrigins,
		APIKey:              apiKey,
		Env:                 env,
		TrustedProxies:      parseTrustedProxies(os.Getenv("TRUSTED_PROXY_CIDRS")),
		LogFormat:           logFormat,
	}
}

// Validate applies deployment-profile checks. In production, running without
// an API key would silently disable auth on every endpoint, so fail fast
// instead of starting with an unauthenticated API (#56).
func (c *Config) Validate() error {
	switch c.Env {
	case "development", "production":
	default:
		return fmt.Errorf("%w (got %q)", ErrInvalidEnv, c.Env)
	}
	if c.IsProduction() && c.APIKey == "" {
		return ErrMissingAPIKey
	}
	return nil
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

// parseTrustedProxies parses a comma-separated list of IPs or CIDRs. Entries
// that are plain IPs are normalized to /32 (or /128 for IPv6). Invalid
// entries are reported and skipped so one typo cannot disable the proxy trust
// configuration silently.
func parseTrustedProxies(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var trusted []string
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				slog.Warn("ignoring invalid TRUSTED_PROXY_CIDRS entry", "entry", entry, "error", err)
				continue
			}
			trusted = append(trusted, entry)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			slog.Warn("ignoring invalid TRUSTED_PROXY_CIDRS entry", "entry", entry)
			continue
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		trusted = append(trusted, entry+"/"+strconv.Itoa(bits))
	}
	return trusted
}
