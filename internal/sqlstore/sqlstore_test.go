package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/michaelginalick/recipe-app/internal/domain"
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

func seedRecipe(t *testing.T, db *sql.DB, r domain.Recipe) int64 {
	t.Helper()

	w := NewSQLWriter(db)
	if _, err := w.Save(context.Background(), r, ""); err != nil {
		t.Fatalf("seed recipe: %v", err)
	}

	var id int64
	if err := db.QueryRow(`SELECT id FROM recipes WHERE name = ?`, r.Name).Scan(&id); err != nil {
		t.Fatalf("seed recipe get id: %v", err)
	}

	return id
}

func TestSQLWriter_Save(t *testing.T) {
	db := newTestDB(t)
	w := NewSQLWriter(db)

	r := domain.Recipe{
		Name:         "Pasta",
		Ingredients:  []domain.Ingredient{{Name: "noodles"}, {Name: "salt"}},
		Instructions: []domain.Instruction{{Step: "Boil water."}, {Step: "Cook pasta."}},
	}

	isNew, err := w.Save(context.Background(), r, "https://example.com/pasta")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !isNew {
		t.Error("got isNew=false, want true for first insert")
	}
}

func TestSQLWriter_Save_Duplicate(t *testing.T) {
	db := newTestDB(t)
	w := NewSQLWriter(db)

	r := domain.Recipe{Name: "Pasta"}

	if _, err := w.Save(context.Background(), r, ""); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	isNew, err := w.Save(context.Background(), r, "")
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if isNew {
		t.Error("got isNew=true, want false for duplicate")
	}
}

func TestSQLReader_GetRecipe(t *testing.T) {
	db := newTestDB(t)

	r := domain.Recipe{
		Name:         "Avocado Toast",
		Ingredients:  []domain.Ingredient{{Name: "avocado"}, {Name: "bread"}},
		Instructions: []domain.Instruction{{Step: "Toast bread."}, {Step: "Mash avocado."}},
	}
	id := seedRecipe(t, db, r)

	reader := NewSQLReader(db)
	got, err := reader.GetRecipe(context.Background(), fmt.Sprintf("%d", id))
	if err != nil {
		t.Fatalf("GetRecipe: %v", err)
	}

	if got.Name != r.Name {
		t.Errorf("got name %q, want %q", got.Name, r.Name)
	}

	if len(got.Ingredients) != len(r.Ingredients) {
		t.Errorf("got %d ingredients, want %d", len(got.Ingredients), len(r.Ingredients))
	}

	if len(got.Instructions) != len(r.Instructions) {
		t.Errorf("got %d instructions, want %d", len(got.Instructions), len(r.Instructions))
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

	seedRecipe(t, db, domain.Recipe{
		Name:        "Chocolate Cake",
		Ingredients: []domain.Ingredient{{Name: "chocolate"}, {Name: "flour"}},
	})
	seedRecipe(t, db, domain.Recipe{
		Name:        "Banana Bread",
		Ingredients: []domain.Ingredient{{Name: "banana"}, {Name: "flour"}},
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

	seedRecipe(t, db, domain.Recipe{
		Name:        "Lentil Soup",
		Ingredients: []domain.Ingredient{{Name: "lentils"}},
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

func seedJob(t *testing.T, js *sqlJobStore, rawURL, rootHost string) string {
	t.Helper()
	id, err := js.InsertJob(context.Background(), rawURL, rootHost)
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return id
}

func TestSQLJobStore_InsertJob(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	id, err := js.InsertJob(context.Background(), "https://example.com", "example.com")
	if err != nil {
		t.Fatalf("InsertJob: %v", err)
	}
	if id == "" {
		t.Error("got empty id, want non-empty")
	}
}

func TestSQLJobStore_InsertJob_Duplicate(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	seedJob(t, js, "https://example.com", "example.com")

	_, err := js.InsertJob(context.Background(), "https://example.com", "example.com")
	if !errors.Is(err, domain.ErrJobAlreadyExists) {
		t.Errorf("got %v, want ErrJobAlreadyExists", err)
	}
}

func TestSQLReader_GetJob(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	js := NewSQLJobStore(db)
	reader := NewSQLReader(db)

	id := seedJob(t, js, "https://example.com/recipe", "example.com")

	got, err := reader.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}

	if got.ID != id {
		t.Errorf("got id %q, want %q", got.ID, id)
	}

	if got.URL != "https://example.com/recipe" {
		t.Errorf("got url %q, want %q", got.URL, "https://example.com/recipe")
	}

	if got.Status != domain.JobStatusPending {
		t.Errorf("got status %v, want %v", got.Status, domain.JobStatusPending)
	}
}

func TestSQLReader_GetJob_NotFound(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	reader := NewSQLReader(db)

	_, err := reader.GetJob(context.Background(), "999")
	if !errors.Is(err, domain.ErrJobNotFound) {
		t.Errorf("got %v, want ErrJobNotFound", err)
	}
}

func TestSQLJobStore_PromoteAndGetJob_Empty(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	job, err := js.PromoteAndGetJob(context.Background(), 2)
	if err != nil {
		t.Fatalf("PromoteAndGetJob: %v", err)
	}
	if job != nil {
		t.Errorf("got job %v, want nil", job)
	}
}

func TestSQLJobStore_PromoteAndGetJob_Promotes(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	inserted := seedJob(t, js, "https://example.com/recipe", "example.com")

	job, err := js.PromoteAndGetJob(context.Background(), 1)
	if err != nil {
		t.Fatalf("PromoteAndGetJob: %v", err)
	}
	if job == nil {
		t.Fatal("got nil, want a promoted job")
	}
	if job.ID != inserted {
		t.Errorf("got id %q, want %q", job.ID, inserted)
	}

	got, err := NewSQLReader(db).GetJob(context.Background(), inserted)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Status != domain.JobStatusRunning {
		t.Errorf("got status %v, want %v", got.Status, domain.JobStatusRunning)
	}
}

func TestSQLJobStore_PromoteAndGetJob_AtCapacity(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	seedJob(t, js, "https://example.com/a", "example.com")
	seedJob(t, js, "https://example.com/b", "example.com")

	// Promote first job, filling the single slot.
	if _, err := js.PromoteAndGetJob(context.Background(), 1); err != nil {
		t.Fatalf("first PromoteAndGetJob: %v", err)
	}

	// Second job should not be promoted since maxJobs=1 is now at capacity.
	job, err := js.PromoteAndGetJob(context.Background(), 1)
	if err != nil {
		t.Fatalf("second PromoteAndGetJob: %v", err)
	}
	if job != nil {
		t.Errorf("got job %v, want nil (at capacity)", job)
	}
}

func TestSQLJobStore_CompleteJob_Done(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	id := seedJob(t, js, "https://example.com/recipe", "example.com")

	if err := js.CompleteJob(context.Background(), id, ""); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}

	got, err := NewSQLReader(db).GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Status != domain.JobStatusDone {
		t.Errorf("got status %v, want %v", got.Status, domain.JobStatusDone)
	}
	if got.ErrorMsg != nil {
		t.Errorf("got error_msg %v, want nil", got.ErrorMsg)
	}
	if got.CompletedAt == nil {
		t.Error("got nil completed_at, want a timestamp")
	}
}

func TestSQLJobStore_CompleteJob_Error(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	id := seedJob(t, js, "https://example.com/recipe", "example.com")

	if err := js.CompleteJob(context.Background(), id, "something went wrong"); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}

	got, err := NewSQLReader(db).GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Status != domain.JobStatusError {
		t.Errorf("got status %v, want %v", got.Status, domain.JobStatusError)
	}
	if got.ErrorMsg == nil || *got.ErrorMsg != "something went wrong" {
		t.Errorf("got error_msg %v, want %q", got.ErrorMsg, "something went wrong")
	}
}

func TestSQLJobStore_RecoverJobs(t *testing.T) {
	db := newTestDB(t)
	js := NewSQLJobStore(db)

	id1 := seedJob(t, js, "https://example.com/a", "example.com")
	id2 := seedJob(t, js, "https://example.com/b", "example.com")

	// Promote both jobs to running.
	if _, err := js.PromoteAndGetJob(context.Background(), 2); err != nil {
		t.Fatalf("promote first job: %v", err)
	}
	if _, err := js.PromoteAndGetJob(context.Background(), 2); err != nil {
		t.Fatalf("promote second job: %v", err)
	}

	if err := js.RecoverJobs(context.Background()); err != nil {
		t.Fatalf("RecoverJobs: %v", err)
	}

	for _, id := range []string{id1, id2} {
		got, err := NewSQLReader(db).GetJob(context.Background(), id)
		if err != nil {
			t.Fatalf("GetJob(%q): %v", id, err)
		}
		if got.Status != domain.JobStatusError {
			t.Errorf("job %q: got status %v, want %v", id, got.Status, domain.JobStatusError)
		}
	}
}
