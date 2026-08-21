package store

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

func NewDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// SQLite uses a single-writer concurrency model. Set max open/idle connections to 1.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Enable WAL (Write-Ahead Logging) and busy timeout for file-based SQLite databases
	if dbPath != ":memory:" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			return nil, err
		}
		if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
			return nil, err
		}
	}

	return db, nil
}
