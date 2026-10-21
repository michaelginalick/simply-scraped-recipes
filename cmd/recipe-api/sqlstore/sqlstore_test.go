package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
	"github.com/michaelginalick/recipe-app/migrations"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec(migrations.SCHEMA); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func seedRecipe(t *testing.T, db *sql.DB, r dto.CreateRecipeRequest) int64 {
	t.Helper()

	w := NewSQLWriter(db)
	id, err := w.CreateRecipe(context.Background(), r)
	if err != nil {
		t.Fatalf("seed recipe: %v", err)
	}

	return id
}

func TestSQLWriter_CreateRecipe(t *testing.T) {
	db := newTestDB(t)
	w := NewSQLWriter(db)

	req := dto.CreateRecipeRequest{
		Name:         "Pasta",
		Ingredients:  []dto.IngredientRequest{{Name: "noodles"}, {Name: "salt"}},
		Instructions: []dto.InstructionRequest{{Step: "Boil water."}, {Step: "Cook pasta."}},
	}

	id, err := w.CreateRecipe(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateRecipe: %v", err)
	}

	if id <= 0 {
		t.Errorf("got id %d, want > 0", id)
	}
}

func TestSQLWriter_CreateRecipe_Duplicate(t *testing.T) {
	db := newTestDB(t)
	w := NewSQLWriter(db)

	req := dto.CreateRecipeRequest{Name: "Pasta"}

	if _, err := w.CreateRecipe(context.Background(), req); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err := w.CreateRecipe(context.Background(), req)
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("got %v, want ErrAlreadyExists", err)
	}
}

func TestSQLReader_GetRecipe(t *testing.T) {
	db := newTestDB(t)

	req := dto.CreateRecipeRequest{
		Name:         "Avocado Toast",
		Ingredients:  []dto.IngredientRequest{{Name: "avocado"}, {Name: "bread"}},
		Instructions: []dto.InstructionRequest{{Step: "Toast bread."}, {Step: "Mash avocado."}},
	}
	id := seedRecipe(t, db, req)

	reader := NewSQLReader(db)
	got, err := reader.GetRecipe(context.Background(), fmt.Sprintf("%d", id))
	if err != nil {
		t.Fatalf("GetRecipe: %v", err)
	}

	if got.Name != req.Name {
		t.Errorf("got name %q, want %q", got.Name, req.Name)
	}

	if len(got.Ingredients) != len(req.Ingredients) {
		t.Errorf("got %d ingredients, want %d", len(got.Ingredients), len(req.Ingredients))
	}

	if len(got.Instructions) != len(req.Instructions) {
		t.Errorf("got %d instructions, want %d", len(got.Instructions), len(req.Instructions))
	}
}

func TestSQLReader_GetRecipe_NotFound(t *testing.T) {
	db := newTestDB(t)
	reader := NewSQLReader(db)

	_, err := reader.GetRecipe(context.Background(), "999")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestSQLReader_SearchRecipe(t *testing.T) {
	db := newTestDB(t)

	seedRecipe(t, db, dto.CreateRecipeRequest{
		Name:        "Chocolate Cake",
		Ingredients: []dto.IngredientRequest{{Name: "chocolate"}, {Name: "flour"}},
	})
	seedRecipe(t, db, dto.CreateRecipeRequest{
		Name:        "Banana Bread",
		Ingredients: []dto.IngredientRequest{{Name: "banana"}, {Name: "flour"}},
	})

	reader := NewSQLReader(db)
	results, err := reader.SearchRecipe(context.Background(), "chocolate")
	if err != nil {
		t.Fatalf("SearchRecipe: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	if results[0].Name != "Chocolate Cake" {
		t.Errorf("got %q, want %q", results[0].Name, "Chocolate Cake")
	}
}

func TestSQLReader_SearchRecipe_CaseInsensitive(t *testing.T) {
	db := newTestDB(t)

	seedRecipe(t, db, dto.CreateRecipeRequest{
		Name:        "Lentil Soup",
		Ingredients: []dto.IngredientRequest{{Name: "lentils"}},
	})

	reader := NewSQLReader(db)

	for _, q := range []string{"lentil", "Lentil", "LENTIL"} {
		results, err := reader.SearchRecipe(context.Background(), q)
		if err != nil {
			t.Fatalf("SearchRecipe(%q): %v", q, err)
		}

		if len(results) != 1 {
			t.Errorf("query %q: got %d results, want 1", q, len(results))
		}
	}
}

func TestSQLReader_SearchRecipe_NoResults(t *testing.T) {
	db := newTestDB(t)

	reader := NewSQLReader(db)
	results, err := reader.SearchRecipe(context.Background(), "unicorn")
	if err != nil {
		t.Fatalf("SearchRecipe: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
