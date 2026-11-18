package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/michaelginalick/recipe-app/internal/domain"
	"github.com/michaelginalick/recipe-app/internal/dto"
)

type trieInserter interface {
	Insert(id, name string, ingredients []string)
}

type sqlWriter struct {
	db   *sql.DB
	trie trieInserter
}

func NewSQLWriter(db *sql.DB, trie trieInserter) *sqlWriter {
	return &sqlWriter{db: db, trie: trie}
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

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	if w.trie != nil {
		ingNames := make([]string, len(r.Ingredients))
		for i, ing := range r.Ingredients {
			ingNames[i] = ing.Name
		}
		w.trie.Insert(fmt.Sprintf("%d", id), r.Name, ingNames)
	}

	return id, nil
}
