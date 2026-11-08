package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/michaelginalick/recipe-app/internal/dto"
)

type stubProber struct {
	resp dto.ProbeResponse
}

func (s stubProber) Probe(_ context.Context, _ string) dto.ProbeResponse {
	return s.resp
}

func TestProbeHandler_RecipeFound(t *testing.T) {
	resp := dto.ProbeResponse{
		Scraped: true,
		Recipe:  &dto.ProbeRecipe{Name: "Chocolate Cake"},
	}

	body := `{"url":"https://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubProber{resp: resp}).ServeHTTP(rr, req)

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

	ProbeHandler(stubProber{resp: dto.ProbeResponse{}}).ServeHTTP(rr, req)

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

func TestProbeHandler_ServiceError(t *testing.T) {
	resp := dto.ProbeResponse{Error: "request timed out"}

	body := `{"url":"https://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubProber{resp: resp}).ServeHTTP(rr, req)

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

func TestProbeHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()

	ProbeHandler(stubProber{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestProbeHandler_MissingURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubProber{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestProbeHandler_InvalidScheme(t *testing.T) {
	body := `{"url":"ftp://example.com/recipe/"}`
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ProbeHandler(stubProber{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
