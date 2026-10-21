package dto

import "errors"

type CreateRecipeRequest struct {
	Name         string               `json:"name"`
	Ingredients  []IngredientRequest  `json:"ingredients"`
	Instructions []InstructionRequest `json:"instructions"`
}

type IngredientRequest struct {
	Name string `json:"name"`
}

type InstructionRequest struct {
	Step string `json:"step"`
}

func (r CreateRecipeRequest) Validate() error {
	if r.Name == "" {
		return errors.New("must include name")
	}
	return nil
}
