package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
	"modernc.org/sqlite"
	lib "modernc.org/sqlite/lib"
)

type sqlJobStore struct {
	db *sql.DB
}

func NewSQLJobStore(db *sql.DB) *sqlJobStore {
	return &sqlJobStore{db: db}
}

func (s *sqlJobStore) InsertJob(ctx context.Context, rawURL, rootHost string) (string, error) {
	result, err := s.db.ExecContext(ctx, insertJobQuery,
		sql.Named("url", rawURL),
		sql.Named("root_host", rootHost),
	)
	if err != nil {
		var sqlErr *sqlite.Error
		if errors.As(err, &sqlErr) && sqlErr.Code() == lib.SQLITE_CONSTRAINT_UNIQUE {
			return "", domain.ErrJobAlreadyExists
		}
		return "", fmt.Errorf("insert job: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("insert job: last insert id: %w", err)
	}
	return fmt.Sprintf("%d", id), nil
}

func (s *sqlJobStore) PromoteAndGetJob(ctx context.Context, maxJobs int) (*domain.Job, error) {
	var job domain.Job
	var idInt int64
	err := s.db.QueryRowContext(ctx, promoteAndGetJobQuery, sql.Named("max_jobs", maxJobs)).
		Scan(&idInt, &job.URL, &job.RootHost)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("promote job: %w", err)
	}
	job.ID = fmt.Sprintf("%d", idInt)
	return &job, nil
}

func (s *sqlJobStore) CompleteJob(ctx context.Context, id, errMsg string) error {
	status := domain.JobStatusDone
	var msg *string
	if errMsg != "" {
		status = domain.JobStatusError
		msg = &errMsg
	}
	_, err := s.db.ExecContext(ctx, completeJobQuery,
		sql.Named("id", id),
		sql.Named("status", status.String()),
		sql.Named("error_msg", msg),
	)
	return err
}

func (s *sqlJobStore) RecoverJobs(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, recoverJobsQuery)
	return err
}

func (s *sqlJobStore) GetJob(ctx context.Context, id string) (domain.Job, error) {
	var job domain.Job
	var idInt int64
	var statusStr string
	var startedAt, completedAt sql.NullTime
	var errorMsg sql.NullString

	err := s.db.QueryRowContext(ctx, getJobQuery, sql.Named("id", id)).
		Scan(&idInt, &job.URL, &job.RootHost, &statusStr,
			&job.CreatedAt, &startedAt, &completedAt, &errorMsg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Job{}, domain.ErrJobNotFound
		}
		return domain.Job{}, fmt.Errorf("get job: %w", err)
	}

	job.ID = fmt.Sprintf("%d", idInt)

	job.Status, err = domain.ParseJobStatus(statusStr)
	if err != nil {
		return domain.Job{}, fmt.Errorf("get job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if errorMsg.Valid {
		job.ErrorMsg = &errorMsg.String
	}

	return job, nil
}
