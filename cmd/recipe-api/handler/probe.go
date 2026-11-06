package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
)

type Prober interface {
	Probe(ctx context.Context, rawURL string) dto.ProbeResponse
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

		json.NewEncoder(w).Encode(envelope[dto.ProbeResponse]{Data: svc.Probe(r.Context(), req.URL)})
	}
}
