package config

import "os"

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
	return &Config{
		Port:                port,
		DatabasePath:        dbPath,
		SyncIntervalMinutes: 60,
	}
}
