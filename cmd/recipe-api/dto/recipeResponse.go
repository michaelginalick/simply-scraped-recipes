package dto

import (
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/domain"
)

type RecipeResponse struct {
	ID           string                `json:"id,omitempty"`
	Name         string                `json:"name"`
	Ingredients  []IngredientResponse  `json:"ingredients"`
	Instructions []InstructionResponse `json:"instructions"`
}

type IngredientResponse struct {
	Name string `json:"name"`
}

type InstructionResponse struct {
	Step string `json:"step"`
}

func NewRecipeResponse(id string, rr domain.Recipe) RecipeResponse {
	resp := RecipeResponse{
		ID:   id,
		Name: rr.Name,
	}

	for _, ing := range rr.Ingredients {
		resp.Ingredients = append(resp.Ingredients, IngredientResponse{Name: ing.Name})
	}

	for _, inst := range rr.Instructions {
		resp.Instructions = append(resp.Instructions, InstructionResponse{Step: inst.Step})
	}

	return resp
}

func NewRecipesResponse(rr []domain.Recipe) []RecipeResponse {
	resp := make([]RecipeResponse, len(rr))
	for i, rec := range rr {
		resp[i] = NewRecipeResponse(rec.ID, rec)
	}
	return resp
}
