package application

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/michaelginalick/recipe-app/pkg/links"
	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

type linkFetcher interface {
	ExtractRecipeLinks(url, scopedHost string) ([]*url.URL, r.Recipe, error)
}

// RecipeStore persists recipes; a duplicate name is a no-op.
type RecipeStore interface {
	Save(r.Recipe, string) (bool, error)
}

const (
	DefaultThreadCount int = 3
	MaxThreads         int = 10
	MinThreads         int = 1
)

type Application struct {
	source      *url.URL
	worklist    chan []*url.URL
	unseenLinks chan *url.URL
	seenLinks   map[string]bool
	store       RecipeStore
	threadCount int
	rootHost    string
	linkClient  linkFetcher
	sync.Mutex
}

type Option func(*Application) error

func WithSourceLink(link string) Option {
	return func(app *Application) error {
		u, err := url.Parse(link)
		if err != nil {
			return err
		}
		app.source = u
		return nil
	}
}

func WithThreadCount(num int) Option {
	return func(app *Application) error {
		if num > MaxThreads || num < MinThreads {
			return fmt.Errorf("thread count must be between %d and %d", MinThreads, MaxThreads)
		}
		app.threadCount = num
		return nil
	}
}

func WithRootHost(root string) Option {
	return func(app *Application) error {
		if root == "" {
			return fmt.Errorf("host cannot be empty")
		}
		app.rootHost = root
		return nil
	}
}

func WithStore(s RecipeStore) Option {
	return func(app *Application) error {
		app.store = s
		return nil
	}
}

func New(opts ...Option) (*Application, error) {
	app := &Application{
		worklist:    make(chan []*url.URL),
		unseenLinks: make(chan *url.URL),
		seenLinks:   make(map[string]bool),
		linkClient:  links.NewLinkClient(nil),
		threadCount: DefaultThreadCount,
	}

	for _, opt := range opts {
		if err := opt(app); err != nil {
			return &Application{}, fmt.Errorf("option failed %w", err)
		}
	}

	return app, nil
}

func (app *Application) startWorkers(ctx context.Context, wg *sync.WaitGroup) {
	for range app.threadCount {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case link, ok := <-app.unseenLinks:
					if !ok {
						return
					}
					foundLinks, recipe := app.crawl(link)
					app.saveRecipe(recipe, link.String())

					// Send without blocking the worker so it can keep reading
					// from unseenLinks while processWorkList is busy.
					go func() {
						select {
						case <-ctx.Done():
						case app.worklist <- foundLinks:
						}
					}()
				}
			}
		})
	}
}

// processWorkList fans out unseen URLs from each crawl result to workers.
// pending tracks how many crawl results are still in flight; when it reaches
// zero every reachable URL has been visited and the function returns.
func (app *Application) processWorkList(ctx context.Context, pending *atomic.Int64) {
	for list := range app.worklist {
		for _, u := range list {
			if app.seenURLs(u.String()) {
				continue
			}

			pending.Add(1)
			select {
			case <-ctx.Done():
				return
			case app.unseenLinks <- u:
			}
		}

		if pending.Add(-1) == 0 {
			return
		}
	}
}

func (app *Application) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var pending atomic.Int64

	pending.Store(1) // account for the seed batch
	go func() { app.worklist <- []*url.URL{app.source} }()

	app.startWorkers(ctx, &wg)

	app.processWorkList(ctx, &pending)
	// cancel() via defer stops all workers; wg.Wait() ensures they exit cleanly.
	wg.Wait()
}

func (app *Application) crawl(u *url.URL) ([]*url.URL, r.Recipe) {
	urls, r, err := app.linkClient.ExtractRecipeLinks(u.String(), app.rootHost)
	if err != nil {
		slog.Error("error parsing link", "error", err)
	}
	return urls, r
}

func (app *Application) seenURLs(urlStr string) bool {
	app.Lock()
	defer app.Unlock()

	if app.seenLinks[urlStr] {
		return true
	}
	app.seenLinks[urlStr] = true
	return false
}

func (app *Application) saveRecipe(recipe r.Recipe, sourceURL string) {
	if app.store == nil {
		return
	}

	if recipe.Name == "" {
		slog.Debug("no recipe found", "url", sourceURL)
		return
	}

	isNew, err := app.store.Save(recipe, sourceURL)
	if err != nil {
		slog.Error("error saving recipe", "error", err)
		return
	}

	if isNew {
		slog.Info("saved recipe", "name", recipe.Name)
	}
}
