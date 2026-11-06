package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/dto"
	"github.com/michaelginalick/recipe-app/pkg/links"
	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

type recipeScraper interface {
	Probe(ctx context.Context, rawURL string) (r.Recipe, error)
}

type ProbeService struct {
	scraper recipeScraper
}

func NewProbeService(scraper recipeScraper) ProbeService {
	return ProbeService{scraper: scraper}
}

func (p ProbeService) Probe(ctx context.Context, rawURL string) dto.ProbeResponse {
	recipe, err := p.scraper.Probe(ctx, rawURL)

	switch {
	case errors.Is(err, links.ErrProbeTimeout):
		return dto.ProbeResponse{Error: "request timed out"}
	case errors.Is(err, links.ErrTooManyRequests):
		return dto.ProbeResponse{Error: "rate limited by target site"}
	case err != nil:
		slog.Error("probe url", "url", rawURL, "error", err)
		return dto.ProbeResponse{Error: err.Error()}
	case recipe.Name != "":
		return dto.ProbeResponse{
			Scraped: true,
			Recipe:  &dto.ProbeRecipe{Name: recipe.Name},
		}
	default:
		return dto.ProbeResponse{}
	}
}
