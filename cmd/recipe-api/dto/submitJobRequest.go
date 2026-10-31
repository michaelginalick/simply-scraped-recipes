package dto

import (
	"errors"
	"net/url"
)

type SubmitJobRequest struct {
	URL string `json:"url"`
}

type SubmitJobResponse struct {
	ID string `json:"id"`
}

func (r SubmitJobRequest) Validate() error {
	if r.URL == "" {
		return errors.New("url is required")
	}

	u, err := url.Parse(r.URL)
	if err != nil {
		return errors.New("url is invalid")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url must use http or https")
	}

	if u.Host == "" {
		return errors.New("url must include a host")
	}

	return nil
}
