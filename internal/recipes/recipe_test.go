package recipes

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parse(t *testing.T, s string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc
}

func TestParseRecipe_Name(t *testing.T) {
	doc := parse(t, `<html><body>
		<span class="wprm-recipe-name">Avocado Toast</span>
	</body></html>`)

	r := ParseRecipe(doc)

	if r.Name != "Avocado Toast" {
		t.Errorf("got %q, want %q", r.Name, "Avocado Toast")
	}
}

func TestParseRecipe_Ingredients(t *testing.T) {
	doc := parse(t, `<html><body>
		<li class="wprm-recipe-ingredient">
			<span class="wprm-recipe-ingredient-amount">2</span>
			<span class="wprm-recipe-ingredient-unit">cups</span>
			<span class="wprm-recipe-ingredient-name">flour</span>
		</li>
		<li class="wprm-recipe-ingredient">
			<span class="wprm-recipe-ingredient-amount">1</span>
			<span class="wprm-recipe-ingredient-name">egg</span>
		</li>
	</body></html>`)

	r := ParseRecipe(doc)

	if len(r.Ingredients) != 2 {
		t.Fatalf("got %d ingredients, want 2", len(r.Ingredients))
	}

	if r.Ingredients[0].Name != "2 cups flour" {
		t.Errorf("got %q, want %q", r.Ingredients[0].Name, "2 cups flour")
	}

	if r.Ingredients[1].Name != "1 egg" {
		t.Errorf("got %q, want %q", r.Ingredients[1].Name, "1 egg")
	}
}

func TestParseRecipe_SkipsSystem2(t *testing.T) {
	doc := parse(t, `<html><body>
		<li class="wprm-recipe-ingredient">
			<span class="wprm-recipe-ingredient-unit-system-1">
				<span class="wprm-recipe-ingredient-amount">1</span>
				<span class="wprm-recipe-ingredient-unit">cup</span>
			</span>
			<span class="wprm-recipe-ingredient-unit-system-2">
				<span class="wprm-recipe-ingredient-amount">240</span>
				<span class="wprm-recipe-ingredient-unit">ml</span>
			</span>
			<span class="wprm-recipe-ingredient-name">water</span>
		</li>
	</body></html>`)

	r := ParseRecipe(doc)

	if len(r.Ingredients) != 1 {
		t.Fatalf("got %d ingredients, want 1", len(r.Ingredients))
	}

	if r.Ingredients[0].Name != "1 cup water" {
		t.Errorf("got %q, want %q", r.Ingredients[0].Name, "1 cup water")
	}
}

func TestParseRecipe_Instructions(t *testing.T) {
	doc := parse(t, `<html><body>
		<div class="wprm-recipe-instruction-text">Preheat oven to 350F.</div>
		<div class="wprm-recipe-instruction-text">Mix dry ingredients.</div>
	</body></html>`)

	r := ParseRecipe(doc)

	if len(r.Instructions) != 2 {
		t.Fatalf("got %d instructions, want 2", len(r.Instructions))
	}

	if r.Instructions[0].Step != "Preheat oven to 350F." {
		t.Errorf("got %q, want %q", r.Instructions[0].Step, "Preheat oven to 350F.")
	}
}

func TestParseRecipe_Empty(t *testing.T) {
	doc := parse(t, `<html><body><p>No recipe here.</p></body></html>`)

	r := ParseRecipe(doc)

	if r.Name != "" || len(r.Ingredients) != 0 || len(r.Instructions) != 0 {
		t.Errorf("expected empty recipe, got %+v", r)
	}
}
