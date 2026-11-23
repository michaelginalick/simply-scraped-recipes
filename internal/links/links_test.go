package links

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

type fakeParser struct {
	recipe domain.Recipe
}

func (f fakeParser) Parse(_ io.Reader) domain.Recipe { return f.recipe }

func newClientWithParser(t *testing.T, body string, status int, p RecipeParser) *LinkClient {
	t.Helper()
	return NewLinkClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}), p)
}

// roundTripFunc lets a plain func satisfy http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newClientServing(t *testing.T, body string, status int) *LinkClient {
	t.Helper()
	return NewLinkClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}), nil)
}

const recipePage = `<!DOCTYPE html>
<html>
<body>
  <span class="wprm-recipe-name">Chocolate Cake</span>
  <li class="wprm-recipe-ingredient">
    <span class="wprm-recipe-ingredient-amount">2</span>
    <span class="wprm-recipe-ingredient-unit">cups</span>
    <span class="wprm-recipe-ingredient-name">flour</span>
  </li>
  <div class="wprm-recipe-instruction-text">Mix and bake.</div>

  <a href="/chocolate-cake/">Chocolate Cake</a>
  <a href="/banana-bread/">Banana Bread</a>
  <a href="https://other.com/recipe/">External</a>
  <a href="/de/schokoladenkuchen/">German</a>
  <a href="/recipe/comments/1/">Comment</a>
  <a href="/search?q=cake">Has query</a>
</body>
</html>`

func TestExtractRecipeLinks_ParsesRecipe(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, recipePage)
	}))
	defer srv.Close()

	client := NewLinkClient(nil, nil)
	_, recipe, err := client.ExtractRecipeLinks(context.Background(), srv.URL, srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("ExtractRecipeLinks: %v", err)
	}

	if recipe.Name != "Chocolate Cake" {
		t.Errorf("got name %q, want %q", recipe.Name, "Chocolate Cake")
	}

	if len(recipe.Ingredients) != 1 {
		t.Errorf("got %d ingredients, want 1", len(recipe.Ingredients))
	}

	if len(recipe.Instructions) != 1 {
		t.Errorf("got %d instructions, want 1", len(recipe.Instructions))
	}
}

func TestExtractRecipeLinks_FiltersLinks(t *testing.T) {
	t.Parallel()
	client := newClientServing(t, recipePage, http.StatusOK)

	urls, _, err := client.ExtractRecipeLinks(context.Background(), "https://example.com/page/", "example.com")
	if err != nil {
		t.Fatalf("ExtractRecipeLinks: %v", err)
	}

	for _, u := range urls {
		if u.Host != "example.com" {
			t.Errorf("expected only example.com links, got %s", u)
		}

		if strings.Contains(u.Path, "/de/") {
			t.Errorf("should have filtered /de/ path: %s", u)
		}

		if strings.Contains(u.Path, "comments") {
			t.Errorf("should have filtered comments path: %s", u)
		}

		if u.RawQuery != "" {
			t.Errorf("should have filtered links with query params: %s", u)
		}
	}

	if len(urls) != 2 {
		t.Errorf("got %d urls, want 2 (/chocolate-cake/ and /banana-bread/)", len(urls))
	}
}

func TestExtractRecipeLinks_NonOKStatus(t *testing.T) {
	t.Parallel()
	client := newClientServing(t, "", http.StatusForbidden)

	_, _, err := client.ExtractRecipeLinks(context.Background(), "https://example.com/", "example.com")
	if err == nil {
		t.Error("expected error for non-200 status, got nil")
	}
}

func TestExtractRecipeLinks_EmptyPage(t *testing.T) {
	t.Parallel()
	client := newClientServing(t, `<html><body><p>Nothing here.</p></body></html>`, http.StatusOK)

	urls, recipe, err := client.ExtractRecipeLinks(context.Background(), "https://example.com/", "example.com")
	if err != nil {
		t.Fatalf("ExtractRecipeLinks: %v", err)
	}

	if len(urls) != 0 {
		t.Errorf("got %d urls, want 0", len(urls))
	}

	if recipe.Name != "" {
		t.Errorf("expected empty recipe name, got %q", recipe.Name)
	}
}

func TestExtractRecipeLinks_SetsUserAgent(t *testing.T) {
	t.Parallel()
	var gotUA string

	client := NewLinkClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotUA = r.Header.Get("User-Agent")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`<html><body></body></html>`)),
			Header:     make(http.Header),
		}, nil
	}), nil)

	_, _, err := client.ExtractRecipeLinks(context.Background(), "https://example.com/", "example.com")
	if err != nil {
		t.Error(fmt.Printf("unexpected error %v", err))
	}

	if gotUA == "" {
		t.Error("expected User-Agent header to be set, got empty string")
	}

	if strings.Contains(gotUA, "Go-http-client") {
		t.Errorf("User-Agent should not be the default Go client, got %q", gotUA)
	}
}

func TestNewLinkClient_DefaultTransport(t *testing.T) {
	t.Parallel()
	if NewLinkClient(nil, nil) == nil {
		t.Error("expected non-nil LinkClient")
	}
}

func TestExtractRecipeLinks_UsesFakeParser(t *testing.T) {
	t.Parallel()
	want := domain.Recipe{Name: "Injected Recipe"}
	client := newClientWithParser(t,
		`<html><body><a href="/page/">link</a></body></html>`,
		http.StatusOK,
		fakeParser{recipe: want},
	)

	_, got, err := client.ExtractRecipeLinks(context.Background(), "https://example.com/", "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Name != want.Name {
		t.Errorf("got recipe name %q, want %q", got.Name, want.Name)
	}
}
