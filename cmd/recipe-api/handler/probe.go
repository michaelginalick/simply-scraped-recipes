package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
	srv "github.com/michaelginalick/recipe-app/cmd/recipe-api/service"
)

type Prober interface {
	Probe(ctx context.Context, rawURL string) srv.ProbeResult
}

func ProbeHandler(svc Prober) http.HandlerFunc {
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

		result := svc.Probe(r.Context(), req.URL)

		resp := dto.ProbeResponse{
			Scraped: result.Scraped,
			Error:   result.Error,
		}
		if result.Scraped {
			resp.Recipe = &dto.ProbeRecipe{
				Name:             result.Name,
			}
		}

		json.NewEncoder(w).Encode(envelope[dto.ProbeResponse]{Data: resp})
	}
}
