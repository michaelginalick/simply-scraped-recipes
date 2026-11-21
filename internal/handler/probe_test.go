package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/michaelginalick/recipe-app/internal/domain"
	"github.com/michaelginalick/recipe-app/internal/dto"
)

type stubRecipeScraper struct {
	recipe domain.Recipe
	err    error
}

func (s stubRecipeScraper) Probe(_ context.Context, _ string) (domain.Recipe, error) {
	return s.recipe, s.err
}

func TestProbeHandler_Timeout(t *testing.T) {
	body := `{"url":"https://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{err: domain.ErrProbeTimeout}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var got envelope[dto.ProbeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Data.Error != "request timed out" {
		t.Errorf("got error %q, want %q", got.Data.Error, "request timed out")
	}
}

func TestProbeHandler_RecipeFound(t *testing.T) {
	body := `{"url":"https://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{recipe: domain.Recipe{Name: "Chocolate Cake"}}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var got envelope[dto.ProbeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !got.Data.Scraped {
		t.Error("expected Scraped to be true")
	}

	if got.Data.Recipe == nil || got.Data.Recipe.Name != "Chocolate Cake" {
		t.Errorf("unexpected recipe: %+v", got.Data.Recipe)
	}
}

func TestProbeHandler_NoRecipeFound(t *testing.T) {
	body := `{"url":"https://example.com/page/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var got envelope[dto.ProbeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Data.Scraped {
		t.Error("expected Scraped to be false")
	}
}

func TestProbeHandler_RateLimited(t *testing.T) {
	body := `{"url":"https://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{err: domain.ErrTooManyRequests}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var got envelope[dto.ProbeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Data.Error != "rate limited by target site" {
		t.Errorf("got error %q, want %q", got.Data.Error, "rate limited by target site")
	}
}

func TestProbeHandler_FetchError(t *testing.T) {
	body := `{"url":"https://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{err: errors.New("connection refused")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}

	var got envelope[dto.ProbeResponse]
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Data.Error != "connection refused" {
		t.Errorf("got error %q, want %q", got.Data.Error, "connection refused")
	}
}

func TestProbeHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestProbeHandler_MissingURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestProbeHandler_InvalidScheme(t *testing.T) {
	body := `{"url":"ftp://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubRecipeScraper{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
