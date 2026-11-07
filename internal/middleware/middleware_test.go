package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain_ExecutionOrder(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	h := Chain{mw1, mw2}.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(want) {
		t.Fatalf("got order %v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Errorf("position %d: got %q, want %q", i, order[i], want[i])
		}
	}
}

func TestLogging_CapturesStatus(t *testing.T) {
	h := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusTeapot {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusTeapot)
	}
}

func TestJSONContentType(t *testing.T) {
	h := JSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("got Content-Type %q, want %q", ct, "application/json")
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	h := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
