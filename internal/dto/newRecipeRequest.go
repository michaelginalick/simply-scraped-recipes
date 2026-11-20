package dto

import (
	"errors"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

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

func (r CreateRecipeRequest) ToDomain() domain.Recipe {
	recipe := domain.Recipe{Name: r.Name}
	for _, ing := range r.Ingredients {
		recipe.Ingredients = append(recipe.Ingredients, domain.Ingredient{Name: ing.Name})
	}
	for _, inst := range r.Instructions {
		recipe.Instructions = append(recipe.Instructions, domain.Instruction{Step: inst.Step})
	}
	return recipe
}
