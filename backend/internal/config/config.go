package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                string
	DatabasePath        string
	SyncIntervalMinutes int
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
	return &Config{
		Port:                port,
		DatabasePath:        dbPath,
		SyncIntervalMinutes: syncInterval,
	}
}
