package domain

import (
	"errors"
)

var (
	ErrNotFound         = errors.New("recipe not found")
	ErrAlreadyExists    = errors.New("recipe already exists")
	ErrJobNotFound      = errors.New("job not found")
	ErrJobAlreadyExists = errors.New("job already exists for this url")
	ErrProbeTimeout     = errors.New("probe timed out")
	ErrTooManyRequests  = errors.New("too many requests 429")
)

type Recipe struct {
	ID           string
	Name         string
	Ingredients  []Ingredient
	Instructions []Instruction
}

type Ingredient struct {
	Name string
}

type Instruction struct {
	Step string
}
