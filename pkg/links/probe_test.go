package links

import (
	"context"
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

func assertErr(t *testing.T, err error, want error, wantNonNil bool) {
	t.Helper()
	switch {
	case want != nil && err != want:
		t.Errorf("got err %v, want %v", err, want)
	case wantNonNil && err == nil:
		t.Error("expected non-nil error, got nil")
	case want == nil && !wantNonNil && err != nil:
		t.Errorf("unexpected error: %v", err)
	}
}

func assertRecipe(t *testing.T, recipe r.Recipe, wantName string, wantIngredients, wantInstructions int) {
	t.Helper()
	if recipe.Name != wantName {
		t.Errorf("got name %q, want %q", recipe.Name, wantName)
	}
	if len(recipe.Ingredients) != wantIngredients {
		t.Errorf("got %d ingredients, want %d", len(recipe.Ingredients), wantIngredients)
	}
	if len(recipe.Instructions) != wantInstructions {
		t.Errorf("got %d instructions, want %d", len(recipe.Instructions), wantInstructions)
	}
}

func TestProbe(t *testing.T) {
	cases := []struct {
		name             string
		client           *LinkClient
		wantErr          error
		wantErrNonNil    bool
		wantRecipeName   string
		wantIngredients  int
		wantInstructions int
	}{
		{
			name:             "parses recipe from WP markup",
			client:           newClientServing(t, recipePage, http.StatusOK),
			wantRecipeName:   "Chocolate Cake",
			wantIngredients:  1,
			wantInstructions: 1,
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
			name:          "non-200 status returns error",
			client:        newClientServing(t, "", http.StatusNotFound),
			wantErrNonNil: true,
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
			assertErr(t, err, tc.wantErr, tc.wantErrNonNil)
			assertRecipe(t, recipe, tc.wantRecipeName, tc.wantIngredients, tc.wantInstructions)
		})
	}
}
