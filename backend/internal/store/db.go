package store

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

func NewDB(dbPath string) (*sql.DB, error) {
	dsn := dbPath
	if dbPath != ":memory:" {
		// File-backed databases opt into immediate transaction locks. BEGIN
		// IMMEDIATE acquires the SQLite write lock up front, which serializes
		// writers (two server processes running migrations, or a migration
		// racing a sync write) instead of letting them deadlock at upgrade
		// time and fail with SQLITE_BUSY.
		dsn = appendDSNParam(dsn, "_txlock=immediate")
	}

	db, err := sql.Open("sqlite", dsn)
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

func appendDSNParam(dsn, param string) string {
	if strings.Contains(dsn, "?") {
		return dsn + "&" + param
	}
	return dsn + "?" + param
}
