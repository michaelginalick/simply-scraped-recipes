package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	app "github.com/michaelginalick/recipe-app/internal/application"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

const defaultScrapeThreads = 2

type jobStore interface {
	InsertJob(ctx context.Context, rawURL, rootHost string) (string, error)
	PromoteAndGetJob(ctx context.Context, maxJobs int) (*domain.Job, error)
	CompleteJob(ctx context.Context, id, errMsg string) error
	RecoverJobs(ctx context.Context) error
}

// RecipeStore persists recipes; a duplicate name is a no-op.
type RecipeStore interface {
	Save(ctx context.Context, r domain.Recipe, sourceURL string) (bool, error)
}

type jobService struct {
	store       jobStore
	recipeStore RecipeStore
	dispatchCh  chan struct{}
	maxJobs     int
	wg          sync.WaitGroup
}

func NewJobService(store jobStore, recipeStore RecipeStore, maxJobs int) *jobService {
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
	slog.Info("job submitted", "id", id, "url", rawURL)

	// non-blocking send - if a dispatch is already pending, skip
	select {
	case js.dispatchCh <- struct{}{}:
	default:
	}
	return id, nil
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
			case <-ticker.C: // sweep for jobs missed during downtime
				js.tryPromote(ctx)
			case <-js.dispatchCh: // immediate trigger on new submission
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
		slog.Info("job started", "id", job.ID, "url", job.URL)
		js.wg.Add(1)
		go func(j domain.Job) {
			defer js.wg.Done()
			js.runJob(j)
		}(*job)
	}
}

func (js *jobService) runJob(job domain.Job) {
	log := slog.With("id", job.ID, "url", job.URL)
	log.Info("job running")
	var setupErr error
	defer func() {
		var errMsg string
		if p := recover(); p != nil {
			errMsg = fmt.Sprintf("panic: %v", p)
		} else if setupErr != nil {
			errMsg = setupErr.Error()
		}
		if errMsg == "" {
			log.Info("job completed")
		}
		if err := js.store.CompleteJob(context.Background(), job.ID, errMsg); err != nil {
			log.Error("complete job", "error", err)
		}
	}()

	a, err := app.New(
		app.WithSourceLink(job.URL),
		app.WithThreadCount(defaultScrapeThreads),
		app.WithStore(js.recipeStore),
	)
	if err != nil {
		setupErr = err
		return
	}
	a.Run()
}
