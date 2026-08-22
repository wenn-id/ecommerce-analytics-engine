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
func Migrate(ctx context.Context, db *sql.DB, fsys embed.FS) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

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

	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&applied); err != nil {
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
		if err := applyMigration(ctx, db, v, files[v], string(body)); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, version int, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for migration %s: %w", name, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	slog.Info("database migration applied", "version", version, "file", name)
	return nil
}
