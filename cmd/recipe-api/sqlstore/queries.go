package sqlstore

import (
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed recipe.sql
var recipeSQL string

var (
	getRecipeQuery         = mustQuery(recipeSQL, "GetRecipe")
	createRecipeQuery      = mustQuery(recipeSQL, "CreateRecipe")
	createIngredientQuery  = mustQuery(recipeSQL, "CreateIngredient")
	createInstructionQuery = mustQuery(recipeSQL, "CreateInstruction")
	searchRecipeQuery      = mustQuery(recipeSQL, "SearchRecipe")
	schemaExistsQuery      = mustQuery(recipeSQL, "SchemaExists")
)

func SchemaExists(db *sql.DB) (bool, error) {
	var exists bool
	if err := db.QueryRow(schemaExistsQuery).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// mustQuery extracts a named query from a SQL file using -- name: <Name> headers.
// Panics at startup if the query is not found.
func mustQuery(sql, name string) string {
	for block := range strings.SplitSeq("\n"+sql, "\n-- name: ") {
		n, body, ok := strings.Cut(strings.TrimSpace(block), "\n")
		if ok && strings.TrimSpace(n) == name {
			return strings.TrimSpace(body)
		}
	}
	panic("query not found: " + name)
}
