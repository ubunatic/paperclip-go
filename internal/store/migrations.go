package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrate creates the schema_migrations tracking table and applies any
// SQL files in the migrations/ directory that have not yet been applied,
// in lexicographic (filename) order.
func (s *Store) migrate() error {
	// Bootstrap: ensure the tracking table exists before running any migrations.
	_, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL UNIQUE,
		applied_at TEXT    NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	// Load the set of already-applied migration names.
	rows, err := s.DB.Query(`SELECT name FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("querying schema_migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scanning schema_migrations.name: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating schema_migrations: %w", err)
	}

	// Collect and sort SQL files.
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading embedded migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if applied[e.Name()] {
			continue
		}
		data, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", e.Name(), err)
		}
		if err := applyMigration(s.DB, e.Name(), string(data)); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(db *sql.DB, name, sqlText string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning migration tx %s: %w", name, err)
	}
	if _, err := tx.Exec(sqlText); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("executing migration %s: %w", name, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations(name, applied_at) VALUES (?, ?)`,
		name, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("recording migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("committing migration %s: %w", name, err)
	}
	return nil
}
