package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	d "github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
)

type Writer interface {
	CreateRecipe(ctx context.Context, r dto.CreateRecipeRequest) (int64, error)
}

type Reader interface {
	GetRecipe(ctx context.Context, id string) (d.Recipe, error)
	SearchRecipe(ctx context.Context, queryString string) ([]d.Recipe, error)
}

// ftsSpecial matches characters that could alter FTS5 query semantics or cause parse errors.
// e.g. input: `"chocolate" OR *cake^;` → stripped to `chocolate  OR cake `
var ftsSpecial = regexp.MustCompile(`["*^():+\-;]`)

func sanitizeQuery(q string) string {
	return strings.TrimSpace(ftsSpecial.ReplaceAllString(q, " "))
}

func GetHandler(reader Reader) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		rec, err := reader.GetRecipe(r.Context(), id)
		if err != nil {
			if errors.Is(err, d.ErrNotFound) {
				http.Error(w, "recipe not found", http.StatusNotFound)
				return
			}

			slog.Error("get recipe", "id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		rr := dto.NewRecipeResponse(id, rec)
		if err := json.NewEncoder(w).Encode(envelope[dto.RecipeResponse]{Data: rr}); err != nil {
			return
		}
	})
}

func CreateHandler(writer Writer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req dto.CreateRecipeRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if err := req.Validate(); err != nil {
			errStr := fmt.Sprintf("%v", err)
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

		_, err := writer.CreateRecipe(r.Context(), req)
		if err != nil {
			if errors.Is(err, d.ErrAlreadyExists) {
				http.Error(w, "recipe already exists", http.StatusConflict)
				return
			}

			slog.Error("create recipe", "name", req.Name, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}

func SearchHandler(reader Reader) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := sanitizeQuery(r.URL.Query().Get("q"))

		if q == "" {
			http.Error(w, "missing q", http.StatusBadRequest)
			return
		}

		results, err := reader.SearchRecipe(r.Context(), q)
		if err != nil {
			slog.Error("search recipes", "query", q, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		resp := dto.NewRecipesResponse(results)

		if err := json.NewEncoder(w).Encode(envelope[[]dto.RecipeResponse]{Data: resp}); err != nil {
			return
		}
	})
}
