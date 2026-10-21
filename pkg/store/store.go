package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

type Store struct {
	db *sql.DB
}

func New(path, schema string) (Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return Store{}, fmt.Errorf("open db: %w", err)
	}

	// SQLite supports only one writer at a time; a single connection avoids locking errors.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return Store{}, fmt.Errorf("ping db: %w", err)
	}

	if err := ApplySchema(db, schema); err != nil {
		db.Close()
		return Store{}, err
	}

	return Store{db: db}, nil
}

// ApplySchema runs the provided schema SQL against db.
// Safe to call on an existing database (all statements use IF NOT EXISTS).
func ApplySchema(db *sql.DB, schema string) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	return nil
}

// Save inserts the recipe and returns true if it was newly inserted.
// A duplicate name is silently ignored via the UNIQUE constraint.
func (s Store) Save(recipe r.Recipe, sourceURL string) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT OR IGNORE INTO recipes (name, source_url) VALUES (?, ?)`, recipe.Name, sourceURL)
	if err != nil {
		return false, err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return false, tx.Commit()
	}

	id, err := res.LastInsertId()
	if err != nil {
		return false, err
	}

	for _, ing := range recipe.Ingredients {
		if _, err := tx.Exec(`INSERT INTO ingredients (recipe_id, name) VALUES (?, ?)`, id, ing.Name); err != nil {
			return false, err
		}
	}

	for i, inst := range recipe.Instructions {
		if _, err := tx.Exec(`INSERT INTO instructions (recipe_id, step_num, step) VALUES (?, ?, ?)`, id, i+1, inst.Step); err != nil {
			return false, err
		}
	}

	return true, tx.Commit()
}

func (s Store) Close() error {
	return s.db.Close()
}
