package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/michaelginalick/recipe-app/pkg/links"
	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

type recipeScraper interface {
	Probe(ctx context.Context, rawURL string) (r.Recipe, error)
}

type ProbeResult struct {
	Scraped          bool
	Name             string
	Error            string
}

type ProbeService struct {
	scraper recipeScraper
}

func NewProbeService(scraper recipeScraper) ProbeService {
	return ProbeService{scraper: scraper}
}

func (p ProbeService) Probe(ctx context.Context, rawURL string) ProbeResult {
	recipe, err := p.scraper.Probe(ctx, rawURL)

	switch {
	case errors.Is(err, links.ErrProbeTimeout):
		return ProbeResult{Error: "request timed out"}
	case errors.Is(err, links.ErrTooManyRequests):
		return ProbeResult{Error: "rate limited by target site"}
	case err != nil:
		slog.Error("probe url", "url", rawURL, "error", err)
		return ProbeResult{Error: err.Error()}
	case recipe.Name != "":
		return ProbeResult{
			Scraped:          true,
			Name:             recipe.Name,
		}
	default:
		return ProbeResult{}
	}
}
