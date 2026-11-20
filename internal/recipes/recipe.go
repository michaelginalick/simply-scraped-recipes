package recipes

import (
	"strings"

	"github.com/michaelginalick/recipe-app/internal/domain"
	"golang.org/x/net/html"
)

// ParseRecipe extracts the recipe name, ingredients, and instructions
// from WP Recipe Maker plugin markup.
func ParseRecipe(doc *html.Node) domain.Recipe {
	return extractRecipe(doc)
}

func extractRecipe(doc *html.Node) domain.Recipe {
	var recipe domain.Recipe

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case hasClass(n, "wprm-recipe-name"):
				recipe.Name = textContent(n)
				return
			case hasClass(n, "wprm-recipe-name wprm-block-text-none"):
				recipe.Name = textContent(n)
				return
			case hasClass(n, "wprm-recipe-ingredient"):
				if ing := ingredientText(n); ing.Name != "" {
					recipe.Ingredients = append(recipe.Ingredients, ing)
				}
				return
			case hasClass(n, "wprm-recipe-ingredients"):
				if ing := ingredientText(n); ing.Name != "" {
					recipe.Ingredients = append(recipe.Ingredients, ing)
				}
				return

			case hasClass(n, "wprm-recipe-instruction-text"):
				if t := textContent(n); t != "" {
					recipe.Instructions = append(recipe.Instructions, domain.Instruction{Step: t})
				}
				return
			case hasClass(n, "wprm-recipe-instruction-group"):
				if t := textContent(n); t != "" {
					recipe.Instructions = append(recipe.Instructions, domain.Instruction{Step: t})
				}
				return
			}

		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return recipe
}

// ingredientText builds an Ingredient from a wprm-recipe-ingredient <li> node.
// It uses amount and unit from the first (imperial) unit system, then the name and notes.
func ingredientText(n *html.Node) domain.Ingredient {
	var parts []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return
		}

		switch {
		case hasClass(n, "wprm-checkbox-container"),
			hasClass(n, "wprm-recipe-ingredient-unit-system-2"):
			return

		case hasClass(n, "wprm-recipe-ingredient-amount"),
			hasClass(n, "wprm-recipe-ingredient-unit"),
			hasClass(n, "wprm-recipe-ingredient-name"),
			hasClass(n, "wprm-recipe-ingredient-notes"):
			if t := textContent(n); t != "" {
				parts = append(parts, t)
			}
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return domain.Ingredient{Name: strings.Join(parts, " ")}
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}
