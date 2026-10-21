package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
)

type sqlWriter struct {
	db *sql.DB
}

func NewSQLWriter(db *sql.DB) *sqlWriter {
	return &sqlWriter{db: db}
}

func (w *sqlWriter) CreateRecipe(ctx context.Context, r dto.CreateRecipeRequest) (int64, error) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, createRecipeQuery, sql.Named("name", r.Name))
	if err != nil {
		return 0, fmt.Errorf("insert recipe: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return 0, domain.ErrAlreadyExists
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	for _, ing := range r.Ingredients {
		_, err := tx.ExecContext(ctx, createIngredientQuery,
			sql.Named("recipe_id", id),
			sql.Named("name", ing.Name),
		)
		if err != nil {
			return 0, fmt.Errorf("insert ingredient: %w", err)
		}
	}

	for i, inst := range r.Instructions {
		_, err := tx.ExecContext(ctx, createInstructionQuery,
			sql.Named("recipe_id", id),
			sql.Named("step_num", i+1),
			sql.Named("step", inst.Step),
		)
		if err != nil {
			return 0, fmt.Errorf("insert instruction: %w", err)
		}
	}

	return id, tx.Commit()
}
