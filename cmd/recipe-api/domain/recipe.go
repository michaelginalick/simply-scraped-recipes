package domain

import (
	"errors"
)

var (
	ErrNotFound      = errors.New("recipe not found")
	ErrAlreadyExists = errors.New("recipe already exists")
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