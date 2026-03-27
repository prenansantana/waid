package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	migrations "github.com/prenansantana/waid/migrations"
)

// RunMigrations creates the schema_migrations table if needed and executes any
// pending up migrations in filename order.
func RunMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("store: read migrations dir: %w", err)
	}

	// Collect only .up.sql files and sort them.
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&count); err != nil {
			return fmt.Errorf("store: check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		data, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", name, err)
		}

		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("store: exec migration %s: %w", name, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations(name) VALUES (?)`, name); err != nil {
			return fmt.Errorf("store: record migration %s: %w", name, err)
		}
	}

	return nil
}
