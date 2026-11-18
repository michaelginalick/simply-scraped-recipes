package search

import (
	"context"
	"fmt"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

type backingReader interface {
	GetRecipe(ctx context.Context, id string) (domain.Recipe, error)
	GetJob(ctx context.Context, id string) (domain.Job, error)
	ListRecipes(ctx context.Context) ([]domain.Recipe, error)
}

type Reader struct {
	sql  backingReader
	trie *Trie
}

// func NewReader(sql backingReader, trie *Trie) *Reader {
// 	return &Reader{sql: sql, trie: trie}
// }

func NewReaderWithIndex(ctx context.Context, sql backingReader, opts ...Option) (*Reader, error) {
	trie := New(opts...)
	recipes, err := sql.ListRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("build search index: %w", err)
	}
	for _, r := range recipes {
		ingNames := make([]string, len(r.Ingredients))
		for i, ing := range r.Ingredients {
			ingNames[i] = ing.Name
		}
		trie.Insert(r.ID, r.Name, ingNames)
	}
	return &Reader{sql: sql, trie: trie}, nil
}

func (r *Reader) Trie() *Trie { return r.trie }

func (r *Reader) GetRecipe(ctx context.Context, id string) (domain.Recipe, error) {
	return r.sql.GetRecipe(ctx, id)
}

func (r *Reader) GetJob(ctx context.Context, id string) (domain.Job, error) {
	return r.sql.GetJob(ctx, id)
}

func (r *Reader) SearchRecipe(_ context.Context, q string) ([]domain.Recipe, error) {
	hits := r.trie.Search(q)
	recipes := make([]domain.Recipe, len(hits))
	for i, h := range hits {
		recipes[i] = domain.Recipe{ID: h.ID, Name: h.Name}
	}
	return recipes, nil
}

func (r *Reader) Rebuild(ctx context.Context) error {
	recipes, err := r.sql.ListRecipes(ctx)
	if err != nil {
		return fmt.Errorf("list recipes for rebuild: %w", err)
	}
	r.trie.Rebuild(recipes)
	return nil
}
