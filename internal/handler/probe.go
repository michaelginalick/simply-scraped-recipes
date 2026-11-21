package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/michaelginalick/recipe-app/internal/domain"
	"github.com/michaelginalick/recipe-app/internal/dto"
)

type RecipeScraper interface {
	Probe(ctx context.Context, rawURL string) (domain.Recipe, error)
}

func ProbeHandler(scraper RecipeScraper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req dto.ProbeRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if err := req.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		recipe, err := scraper.Probe(r.Context(), req.URL)
		json.NewEncoder(w).Encode(envelope[dto.ProbeResponse]{Data: mapProbeResponse(recipe, err)})
	}
}

func mapProbeResponse(recipe domain.Recipe, err error) dto.ProbeResponse {
	switch {
	case errors.Is(err, domain.ErrProbeTimeout):
		return dto.ProbeResponse{Error: "request timed out"}
	case errors.Is(err, domain.ErrTooManyRequests):
		return dto.ProbeResponse{Error: "rate limited by target site"}
	case err != nil:
		slog.Error("probe url", "error", err)
		return dto.ProbeResponse{Error: err.Error()}
	case recipe.Name != "":
		return dto.ProbeResponse{Scraped: true, Recipe: &dto.ProbeRecipe{Name: recipe.Name}}
	default:
		return dto.ProbeResponse{}
	}
}
