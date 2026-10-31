package application_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/michaelginalick/recipe-app/pkg/application"
)

func TestRun_Completes(t *testing.T) {
	// Three pages: root links to two children, children link back to root.
	// Once all three are visited and the only discovered link (root) is already
	// seen, pending drops to zero and Run() must return.
	pages := map[string]string{
		"/": `<html><body>
			<a href="/page1">Page 1</a>
			<a href="/page2">Page 2</a>
		</body></html>`,
		"/page1": `<html><body><a href="/">Home</a></body></html>`,
		"/page2": `<html><body><a href="/">Home</a></body></html>`,
	}

	var visited atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		visited.Add(1)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)

	a, err := application.New(
		application.WithSourceLink(server.URL+"/"),
		application.WithThreadCount(1),
		application.WithRootHost(u.Host),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		a.Run()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not complete within 5 seconds")
	}

	if n := visited.Load(); n != 3 {
		t.Errorf("visited %d pages, want 3", n)
	}
}
