package service

import (
	"context"
	"errors"
	"testing"

	"github.com/michaelginalick/recipe-app/internal/dto"
	"github.com/michaelginalick/recipe-app/internal/links"
	r "github.com/michaelginalick/recipe-app/internal/recipes"
)

type stubScraper struct {
	recipe r.Recipe
	err    error
}

func (s stubScraper) Probe(_ context.Context, _ string) (r.Recipe, error) {
	return s.recipe, s.err
}

func TestProbeService(t *testing.T) {
	fullRecipe := r.Recipe{
		Name:         "Chocolate Cake",
		Ingredients:  []r.Ingredient{{Name: "flour"}, {Name: "sugar"}},
		Instructions: []r.Instruction{{Step: "Mix."}, {Step: "Bake."}},
	}

	cases := []struct {
		name        string
		scraper     stubScraper
		wantScraped bool
		wantRecipe  *dto.ProbeRecipe
		wantError   string
	}{
		{
			name:        "recipe found",
			scraper:     stubScraper{recipe: fullRecipe},
			wantScraped: true,
			wantRecipe:  &dto.ProbeRecipe{Name: "Chocolate Cake"},
		},
		{
			name:    "no recipe markup",
			scraper: stubScraper{recipe: r.Recipe{}},
		},
		{
			name:      "timeout",
			scraper:   stubScraper{err: links.ErrProbeTimeout},
			wantError: "request timed out",
		},
		{
			name:      "rate limited",
			scraper:   stubScraper{err: links.ErrTooManyRequests},
			wantError: "rate limited by target site",
		},
		{
			name:      "fetch error",
			scraper:   stubScraper{err: errors.New("connection refused")},
			wantError: "connection refused",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := NewProbeService(tc.scraper).Probe(context.Background(), "https://example.com/recipe/")

			if resp.Scraped != tc.wantScraped {
				t.Errorf("got Scraped=%v, want %v", resp.Scraped, tc.wantScraped)
			}

			if tc.wantRecipe != nil {
				if resp.Recipe == nil {
					t.Fatal("expected Recipe to be set, got nil")
				}
				if resp.Recipe.Name != tc.wantRecipe.Name {
					t.Errorf("got name %q, want %q", resp.Recipe.Name, tc.wantRecipe.Name)
				}
			} else if resp.Recipe != nil {
				t.Errorf("expected Recipe to be nil, got %+v", resp.Recipe)
			}

			if resp.Error != tc.wantError {
				t.Errorf("got error %q, want %q", resp.Error, tc.wantError)
			}
		})
	}
}
