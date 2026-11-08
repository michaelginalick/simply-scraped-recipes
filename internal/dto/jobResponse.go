package dto

import (
	"time"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

type JobResponse struct {
	ID          string  `json:"id"`
	URL         string  `json:"url"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	ErrorMsg    *string `json:"error_msg,omitempty"`
}

func NewJobResponse(j domain.Job) JobResponse {
	resp := JobResponse{
		ID:        j.ID,
		URL:       j.URL,
		Status:    j.Status.String(),
		CreatedAt: j.CreatedAt.Format(time.RFC3339),
		ErrorMsg:  j.ErrorMsg,
	}
	if j.StartedAt != nil {
		s := j.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if j.CompletedAt != nil {
		s := j.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &s
	}
	return resp
}
