package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
)

// ingredientSep and instructionSep are ASCII control characters that cannot
// appear in recipe text, used as GROUP_CONCAT delimiters in recipe.sql.
const (
	ingredientSep  = rune(31) // ASCII unit separator
	instructionSep = rune(30) // ASCII record separator
)

type dbRecipe struct {
	name         string
	ingredients  []string
	instructions []string
}

type sqlReader struct {
	db *sql.DB
}

func NewSQLReader(db *sql.DB) *sqlReader {
	return &sqlReader{db: db}
}

func (r sqlReader) GetRecipe(ctx context.Context, id string) (domain.Recipe, error) {
	var name string
	var ings, insts sql.NullString

	err := r.db.QueryRowContext(ctx, getRecipeQuery, sql.Named("id", id)).Scan(&name, &ings, &insts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Recipe{}, domain.ErrNotFound
		}
		return domain.Recipe{}, fmt.Errorf("get recipe: %w", err)
	}

	row := dbRecipe{name: name}
	if ings.Valid {
		row.ingredients = strings.Split(ings.String, string(ingredientSep))
	}
	if insts.Valid {
		row.instructions = strings.Split(insts.String, string(instructionSep))
	}

	// domainRecord := domain.NewDomainRecord(row)

	return row.toDomain(), nil
}

func (r sqlReader) SearchRecipe(ctx context.Context, queryString string) ([]domain.Recipe, error) {
	rows, err := r.db.QueryContext(ctx, searchRecipeQuery, sql.Named("query", strings.ToLower(queryString)))
	if err != nil {
		return nil, fmt.Errorf("search recipes: %w", err)
	}
	defer rows.Close()

	var result []domain.Recipe

	for rows.Next() {
		var id int64
		var name string
		var ings, insts sql.NullString

		if err := rows.Scan(&id, &name, &ings, &insts); err != nil {
			return nil, fmt.Errorf("scan recipe: %w", err)
		}

		row := dbRecipe{name: name}
		if ings.Valid {
			row.ingredients = strings.Split(ings.String, string(ingredientSep))
		}
		if insts.Valid {
			row.instructions = strings.Split(insts.String, string(instructionSep))
		}

		rec := row.toDomain()
		rec.ID = fmt.Sprintf("%d", id)
		result = append(result, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes: %w", err)
	}

	return result, nil
}

func (r sqlReader) GetJob(ctx context.Context, id string) (domain.Job, error) {
	var job domain.Job
	var idInt int64
	var statusStr string
	var startedAt, completedAt sql.NullTime
	var errorMsg sql.NullString

	err := r.db.QueryRowContext(ctx, getJobQuery, sql.Named("id", id)).
		Scan(&idInt, &job.URL, &job.RootHost, &statusStr,
			&job.CreatedAt, &startedAt, &completedAt, &errorMsg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Job{}, domain.ErrJobNotFound
		}
		return domain.Job{}, fmt.Errorf("get job: %w", err)
	}

	job.ID = fmt.Sprintf("%d", idInt)

	job.Status, err = domain.ParseJobStatus(statusStr)
	if err != nil {
		return domain.Job{}, fmt.Errorf("get job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if errorMsg.Valid {
		job.ErrorMsg = &errorMsg.String
	}

	return job, nil
}

func (d dbRecipe) toDomain() domain.Recipe {
	r := domain.Recipe{Name: d.name}

	for _, ing := range d.ingredients {
		r.Ingredients = append(r.Ingredients, domain.Ingredient{Name: ing})
	}

	for _, inst := range d.instructions {
		r.Instructions = append(r.Instructions, domain.Instruction{Step: inst})
	}

	return r
}
