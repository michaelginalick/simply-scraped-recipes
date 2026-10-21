package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	d "github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
)

type stubReader struct {
	recipe  d.Recipe
	results []d.Recipe
	err     error
}

func (s stubReader) GetRecipe(_ context.Context, _ string) (d.Recipe, error) {
	return s.recipe, s.err
}

func (s stubReader) SearchRecipe(_ context.Context, _ string) ([]d.Recipe, error) {
	return s.results, s.err
}

type stubWriter struct {
	id  int64
	err error
}

func (s stubWriter) CreateRecipe(_ context.Context, _ dto.CreateRecipeRequest) (int64, error) {
	return s.id, s.err
}

func TestGetHandler_OK(t *testing.T) {
	rec := d.Recipe{
		Name:         "Pasta",
		Ingredients:  []d.Ingredient{{Name: "noodles"}},
		Instructions: []d.Instruction{{Step: "Boil water."}},
	}

	req := httptest.NewRequest(http.MethodGet, "/recipe/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()

	GetHandler(stubReader{recipe: rec}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var body envelope[dto.RecipeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Data.Name != "Pasta" {
		t.Errorf("got name %q, want %q", body.Data.Name, "Pasta")
	}
}

func TestGetHandler_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/recipe/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	GetHandler(stubReader{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/recipe/99", nil)
	req.SetPathValue("id", "99")
	rr := httptest.NewRecorder()

	GetHandler(stubReader{err: d.ErrNotFound}).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestGetHandler_InternalError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/recipe/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()

	GetHandler(stubReader{err: errors.New("db error")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestCreateHandler_Created(t *testing.T) {
	body := `{"name":"Pasta","ingredients":[{"name":"noodles"}],"instructions":[{"step":"Boil."}]}`
	req := httptest.NewRequest(http.MethodPost, "/recipe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	CreateHandler(stubWriter{id: 1}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusCreated)
	}
}

func TestCreateHandler_MissingName(t *testing.T) {
	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/recipe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	CreateHandler(stubWriter{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestCreateHandler_AlreadyExists(t *testing.T) {
	body := `{"name":"Pasta"}`
	req := httptest.NewRequest(http.MethodPost, "/recipe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	CreateHandler(stubWriter{err: d.ErrAlreadyExists}).ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusConflict)
	}
}

func TestCreateHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/recipe", strings.NewReader(`{bad json`))
	rr := httptest.NewRecorder()

	CreateHandler(stubWriter{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSearchHandler_OK(t *testing.T) {
	results := []d.Recipe{
		{ID: "1", Name: "Pasta", Ingredients: []d.Ingredient{{Name: "noodles"}}},
	}

	req := httptest.NewRequest(http.MethodGet, "/search?q=pasta", nil)
	rr := httptest.NewRecorder()

	SearchHandler(stubReader{results: results}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var body envelope[[]dto.RecipeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Data) != 1 {
		t.Fatalf("got %d results, want 1", len(body.Data))
	}

	if body.Data[0].Name != "Pasta" {
		t.Errorf("got name %q, want %q", body.Data[0].Name, "Pasta")
	}
}

func TestSearchHandler_MissingQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rr := httptest.NewRecorder()

	SearchHandler(stubReader{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSearchHandler_QuerySanitized(t *testing.T) {
	// A query that is entirely special chars becomes empty after sanitization → 400
	req := httptest.NewRequest(http.MethodGet, `/search?q="*;`, nil)
	rr := httptest.NewRecorder()

	SearchHandler(stubReader{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSanitizeQuery(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`chocolate`, `chocolate`},
		{`"chocolate"`, `chocolate`},
		{`chocolate*`, `chocolate`},
		{`select from recipes;`, `select from recipes`},
		{`(cake)`, `cake`},
		{`  spaces  `, `spaces`},
	}

	for _, c := range cases {
		got := sanitizeQuery(c.input)
		if got != c.want {
			t.Errorf("sanitizeQuery(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
