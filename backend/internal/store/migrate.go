package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
)

// migrationsFS holds the versioned SQL migrations. Files are named
// NNNN_description.up.sql / NNNN_description.down.sql; only up migrations are
// applied automatically. Add a new numbered file to change the schema — never
// edit an applied migration (#39).
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

var migrationNameRe = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

// Migrate applies every pending up-migration from fsys in version order.
// Applied versions are recorded in the schema_migrations table, making the
// process idempotent and safe to re-run on every startup.
//
// The whole run (version read + applies) executes inside a single write
// transaction. Combined with the _txlock=immediate DSN (see db.go), two
// server processes cannot both observe the same baseline and race to apply
// the same migration: the second blocks at BEGIN until the first commits,
// then sees the applied versions and exits cleanly.
func Migrate(ctx context.Context, db *sql.DB, fsys embed.FS) error {
	entries, err := fsys.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var versions []int
	files := make(map[int]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := migrationNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err != nil {
			return fmt.Errorf("parse migration version from %q: %w", e.Name(), err)
		}
		if _, dup := files[v]; dup {
			return fmt.Errorf("duplicate migration version %d", v)
		}
		versions = append(versions, v)
		files[v] = e.Name()
	}
	sort.Ints(versions)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var applied int
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&applied); err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}

	for _, v := range versions {
		if v <= applied {
			continue
		}
		body, err := fsys.ReadFile("migrations/" + files[v])
		if err != nil {
			return fmt.Errorf("read migration %s: %w", files[v], err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", files[v], err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", v); err != nil {
			return fmt.Errorf("record migration %s: %w", files[v], err)
		}
		slog.Info("database migration applied", "version", v, "file", files[v])
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}
