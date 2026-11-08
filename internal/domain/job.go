package domain

import (
	"fmt"
	"time"
)

type JobStatus int

const (
	JobStatusPending JobStatus = iota
	JobStatusRunning
	JobStatusDone
	JobStatusError
)

func (s JobStatus) String() string {
	switch s {
	case JobStatusPending:
		return "pending"
	case JobStatusRunning:
		return "running"
	case JobStatusDone:
		return "done"
	case JobStatusError:
		return "error"
	default:
		panic(fmt.Sprintf("unknown job status: %d", s))
	}
}

func ParseJobStatus(s string) (JobStatus, error) {
	switch s {
	case "pending":
		return JobStatusPending, nil
	case "running":
		return JobStatusRunning, nil
	case "done":
		return JobStatusDone, nil
	case "error":
		return JobStatusError, nil
	default:
		return 0, fmt.Errorf("unknown job status: %q", s)
	}
}

type Job struct {
	ID          string
	URL         string
	RootHost    string
	Status      JobStatus
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ErrorMsg    *string
}
