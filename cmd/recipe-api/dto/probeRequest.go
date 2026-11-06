package dto

import (
	"errors"
	"net/url"
)

type ProbeRequest struct {
	URL string `json:"url"`
}

func (r ProbeRequest) Validate() error {
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

type ProbeRecipe struct {
	Name             string `json:"name"`
	IngredientCount  int    `json:"ingredient_count"`
	InstructionCount int    `json:"instruction_count"`
}

type ProbeResponse struct {
	Scraped bool         `json:"scraped"`
	Recipe  *ProbeRecipe `json:"recipe,omitempty"`
	Error   string       `json:"error,omitempty"`
}
