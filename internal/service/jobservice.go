package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	app "github.com/michaelginalick/recipe-app/internal/application"
	pkgstore "github.com/michaelginalick/recipe-app/internal/store"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

const defaultScrapeThreads = 2

type jobStore interface {
	InsertJob(ctx context.Context, rawURL, rootHost string) (string, error)
	PromoteAndGetJob(ctx context.Context, maxJobs int) (*domain.Job, error)
	CompleteJob(ctx context.Context, id, errMsg string) error
	RecoverJobs(ctx context.Context) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
}

type jobService struct {
	store       jobStore
	recipeStore pkgstore.Store
	dispatchCh  chan struct{}
	maxJobs     int
	wg          sync.WaitGroup
}

func NewJobService(store jobStore, recipeStore pkgstore.Store, maxJobs int) *jobService {
	return &jobService{
		store:       store,
		recipeStore: recipeStore,
		dispatchCh:  make(chan struct{}, 1),
		maxJobs:     maxJobs,
	}
}

func (js *jobService) Submit(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	id, err := js.store.InsertJob(ctx, rawURL, u.Host)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (js *jobService) GetJob(ctx context.Context, id string) (domain.Job, error) {
	return js.store.GetJob(ctx, id)
}

// Stop waits for the ticker goroutine to exit via ctx cancellation,
// then waits for all in-flight jobs to complete.
func (js *jobService) Stop() {
	js.wg.Wait()
}

func (js *jobService) Start(ctx context.Context) {
	js.dispatch(ctx)
}

func (js *jobService) dispatch(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				js.tryPromote(ctx)
			}
		}
	}()
}

func (js *jobService) tryPromote(ctx context.Context) {
	for {
		job, err := js.store.PromoteAndGetJob(ctx, js.maxJobs)
		if err != nil {
			slog.Error("promote job", "error", err)
			return
		}
		if job == nil {
			return
		}
		js.wg.Add(1)
		go func(j domain.Job) {
			defer js.wg.Done()
			js.runJob(j)
		}(*job)
	}
}

func (js *jobService) runJob(job domain.Job) {
	var setupErr error
	defer func() {
		var errMsg string
		if p := recover(); p != nil {
			errMsg = fmt.Sprintf("panic: %v", p)
		} else if setupErr != nil {
			errMsg = setupErr.Error()
		}
		if err := js.store.CompleteJob(context.Background(), job.ID, errMsg); err != nil {
			slog.Error("complete job", "id", job.ID, "error", err)
		}
	}()

	a, err := app.New(
		app.WithSourceLink(job.URL),
		app.WithThreadCount(defaultScrapeThreads),
		app.WithRootHost(job.RootHost),
		app.WithStore(js.recipeStore),
	)
	if err != nil {
		setupErr = err
		return
	}
	a.Run()
}
