package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

type sqlWriter struct {
	db *sql.DB
}

func NewSQLWriter(db *sql.DB) *sqlWriter {
	return &sqlWriter{db: db}
}

func (w *sqlWriter) Save(ctx context.Context, r domain.Recipe, sourceURL string) (bool, error) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, createRecipeQuery,
		sql.Named("name", r.Name),
		sql.Named("source_url", sourceURL),
	)
	if err != nil {
		return false, fmt.Errorf("insert recipe: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return false, tx.Commit()
	}

	id, err := result.LastInsertId()
	if err != nil {
		return false, fmt.Errorf("get last insert id: %w", err)
	}

	for _, ing := range r.Ingredients {
		if _, err := tx.ExecContext(ctx, createIngredientQuery,
			sql.Named("recipe_id", id),
			sql.Named("name", ing.Name),
		); err != nil {
			return false, fmt.Errorf("insert ingredient: %w", err)
		}
	}

	for i, inst := range r.Instructions {
		if _, err := tx.ExecContext(ctx, createInstructionQuery,
			sql.Named("recipe_id", id),
			sql.Named("step_num", i+1),
			sql.Named("step", inst.Step),
		); err != nil {
			return false, fmt.Errorf("insert instruction: %w", err)
		}
	}

	return true, tx.Commit()
}
