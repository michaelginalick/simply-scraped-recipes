package search

import (
	"strings"
	"sync"
	"unicode"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

const (
	defaultSearchLimit    = 20
	defaultFuzzyThreshold = 3
)

type Option func(*Trie)

func WithSearchLimit(n int) Option {
	return func(t *Trie) { t.searchLimit = n }
}

func WithFuzzyThreshold(n int) Option {
	return func(t *Trie) { t.fuzzyThreshold = n }
}

type TrieRecipe struct {
	ID   string
	Name string
}

type node struct {
	children [26]*node
	recipes  []TrieRecipe
}

type Trie struct {
	root           *node
	vocabulary     map[string]struct{}
	mu             sync.RWMutex
	searchLimit    int
	fuzzyThreshold int
}

func New(opts ...Option) *Trie {
	t := &Trie{
		root:           &node{},
		vocabulary:     make(map[string]struct{}),
		searchLimit:    defaultSearchLimit,
		fuzzyThreshold: defaultFuzzyThreshold,
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

func NewDefault() *Trie {
	return New()
}

func normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func tokenize(s string) []string {
	return strings.Fields(normalize(s))
}

func insertIntoNode(root *node, vocab map[string]struct{}, word string, recipe TrieRecipe) {
	cur := root
	for _, ch := range word {
		idx := ch - 'a'
		if idx < 0 || idx > 25 {
			continue
		}
		if cur.children[idx] == nil {
			cur.children[idx] = &node{}
		}
		cur = cur.children[idx]
		cur.recipes = append(cur.recipes, recipe)
	}
	vocab[word] = struct{}{}
}

func (t *Trie) Insert(id, name string, ingredients []string) {
	r := TrieRecipe{ID: id, Name: name}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, w := range tokenize(name) {
		insertIntoNode(t.root, t.vocabulary, w, r)
	}
	for _, ing := range ingredients {
		for _, w := range tokenize(ing) {
			insertIntoNode(t.root, t.vocabulary, w, r)
		}
	}
}

func (t *Trie) Rebuild(recipes []domain.Recipe) {
	newRoot := &node{}
	newVocab := make(map[string]struct{})

	for _, r := range recipes {
		rec := TrieRecipe{ID: r.ID, Name: r.Name}
		for _, w := range tokenize(r.Name) {
			insertIntoNode(newRoot, newVocab, w, rec)
		}
		for _, ing := range r.Ingredients {
			for _, w := range tokenize(ing.Name) {
				insertIntoNode(newRoot, newVocab, w, rec)
			}
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.root = newRoot
	t.vocabulary = newVocab
}

func (t *Trie) prefixSearch(prefix string) []TrieRecipe {
	cur := t.root
	for _, ch := range prefix {
		idx := ch - 'a'
		if idx < 0 || idx > 25 {
			return nil
		}
		next := cur.children[idx]
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur.recipes
}

func (t *Trie) fuzzySearch(query string) []TrieRecipe {
	k := maxDist(query)
	var hits []TrieRecipe
	for word := range t.vocabulary {
		if distance(query, word) <= k {
			hits = append(hits, t.prefixSearch(word)...)
		}
	}
	return hits
}

func dedup(hits []TrieRecipe, limit int) []TrieRecipe {
	seen := make(map[string]struct{}, limit)
	out := make([]TrieRecipe, 0, limit)
	for _, r := range hits {
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}
		out = append(out, r)
		if len(out) == limit {
			break
		}
	}
	return out
}

func (t *Trie) Search(query string) []TrieRecipe {
	q := strings.TrimSpace(normalize(query))
	if q == "" {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	hits := dedup(t.prefixSearch(q), t.searchLimit)
	if len(hits) >= t.fuzzyThreshold {
		return hits
	}

	fuzzy := t.fuzzySearch(q)
	seen := make(map[string]struct{}, len(hits))
	for _, r := range hits {
		seen[r.ID] = struct{}{}
	}
	for _, r := range fuzzy {
		if _, ok := seen[r.ID]; !ok {
			hits = append(hits, r)
			seen[r.ID] = struct{}{}
		}
		if len(hits) >= t.searchLimit {
			break
		}
	}
	return hits
}
