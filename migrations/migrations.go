package migrations

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed 001_initial.sql
var m001 string

//go:embed 002_add_indexes_and_scrape_jobs.sql
var m002 string

// SCHEMA is the full combined schema, kept for callers that apply it directly.
var SCHEMA = m001 + "\n" + m002

type migration struct {
	version int
	sql     string
}

var all = []migration{
	{1, m001},
	{2, m002},
}

// Run creates the schema_migrations table if needed, bootstraps pre-migration
// databases, then applies any unapplied migrations in version order.
func Run(db *sql.DB) error {
	if err := ensureTable(db); err != nil {
		return err
	}
	if err := bootstrap(db); err != nil {
		return err
	}
	for _, m := range all {
		applied, err := isApplied(db, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := apply(db, m); err != nil {
			return err
		}
	}
	return nil
}

func ensureTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

// bootstrap seeds migration 1 as already applied when the DB was created
// before the migration system existed (recipes table present, no recorded migrations).
func bootstrap(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var recipesExists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type='table' AND name='recipes')`).
		Scan(&recipesExists)
	if err != nil {
		return err
	}
	if !recipesExists {
		return nil
	}

	_, err = db.Exec(`INSERT OR IGNORE INTO schema_migrations (version) VALUES (1)`)
	return err
}

func isApplied(db *sql.DB, version int) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version).
		Scan(&exists)
	return exists, err
}

func apply(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("migration %d: begin: %w", m.version, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.sql); err != nil {
		return fmt.Errorf("migration %d: %w", m.version, err)
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
		return fmt.Errorf("migration %d: record: %w", m.version, err)
	}
	return tx.Commit()
}
