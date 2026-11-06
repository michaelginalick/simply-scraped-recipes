package links

import (
	"context"
	"errors"
	"net/http"
	"testing"

	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

type mockTimeoutErr struct{}

func (mockTimeoutErr) Error() string   { return "mock timeout" }
func (mockTimeoutErr) Timeout() bool   { return true }
func (mockTimeoutErr) Temporary() bool { return false }

func newTimeoutClient() *LinkClient {
	return NewLinkClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, mockTimeoutErr{}
	}))
}

// errAny signals that any non-nil error satisfies the test case.
var errAny = errors.New("any error")

func assertProbeErr(t *testing.T, err, want error) {
	t.Helper()
	if want == errAny {
		if err == nil {
			t.Error("expected non-nil error, got nil")
		}
	} else if err != want {
		t.Errorf("got err %v, want %v", err, want)
	}
}

func assertProbeRecipe(t *testing.T, got r.Recipe, want *r.Recipe) {
	t.Helper()
	if want != nil {
		if got.Name != want.Name {
			t.Errorf("got name %q, want %q", got.Name, want.Name)
		}
		if len(got.Ingredients) != len(want.Ingredients) {
			t.Errorf("got %d ingredients, want %d", len(got.Ingredients), len(want.Ingredients))
		}
		if len(got.Instructions) != len(want.Instructions) {
			t.Errorf("got %d instructions, want %d", len(got.Instructions), len(want.Instructions))
		}
	} else if got.Name != "" {
		t.Errorf("expected empty recipe, got name %q", got.Name)
	}
}

func TestProbe(t *testing.T) {
	parsedRecipe := &r.Recipe{
		Name:         "Chocolate Cake",
		Ingredients:  []r.Ingredient{{Name: "2 cups flour"}},
		Instructions: []r.Instruction{{Step: "Mix and bake."}},
	}

	cases := []struct {
		name       string
		client     *LinkClient
		wantErr    error
		wantRecipe *r.Recipe
	}{
		{
			name:       "parses recipe from WP markup",
			client:     newClientServing(t, recipePage, http.StatusOK),
			wantRecipe: parsedRecipe,
		},
		{
			name:   "no recipe markup returns empty recipe",
			client: newClientServing(t, `<html><body><p>No recipe here.</p></body></html>`, http.StatusOK),
		},
		{
			name:    "429 returns ErrTooManyRequests",
			client:  newClientServing(t, "", http.StatusTooManyRequests),
			wantErr: ErrTooManyRequests,
		},
		{
			name:    "non-200 status returns error",
			client:  newClientServing(t, "", http.StatusNotFound),
			wantErr: errAny,
		},
		{
			name:    "timeout returns ErrProbeTimeout",
			client:  newTimeoutClient(),
			wantErr: ErrProbeTimeout,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recipe, err := tc.client.Probe(context.Background(), "https://example.com/recipe/")
			assertProbeErr(t, err, tc.wantErr)
			assertProbeRecipe(t, recipe, tc.wantRecipe)
		})
	}
}
