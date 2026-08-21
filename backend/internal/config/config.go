package config

import (
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
	}
}
