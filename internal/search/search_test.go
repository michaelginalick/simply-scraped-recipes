package search

import (
	"testing"
)

func TestTrie_ExactPrefix(t *testing.T) {
	tr := New()
	tr.Insert("1", "Chocolate Cake", []string{"chocolate", "flour", "butter"})
	tr.Insert("2", "Chicken Soup", []string{"chicken", "broth"})

	hits := tr.Search("choc")
	if len(hits) != 1 || hits[0].ID != "1" {
		t.Errorf("got %v, want recipe 1", hits)
	}
}

func TestTrie_WordBoundaryIngredient(t *testing.T) {
	tr := New()
	tr.Insert("1", "Pasta", []string{"butter", "salt"})
	tr.Insert("2", "Cake", []string{"flour", "sugar"})

	hits := tr.Search("butt")
	if len(hits) != 1 || hits[0].ID != "1" {
		t.Errorf("got %v, want recipe 1 (via ingredient)", hits)
	}
}

func TestTrie_Deduplication(t *testing.T) {
	tr := New()
	// "chocolate" in both name and ingredient — should appear once
	tr.Insert("1", "Chocolate Mousse", []string{"chocolate", "cream"})

	hits := tr.Search("choc")
	if len(hits) != 1 {
		t.Errorf("got %d results, want 1 (deduplicated)", len(hits))
	}
}

func TestTrie_CaseInsensitive(t *testing.T) {
	tr := New()
	tr.Insert("1", "Lentil Soup", []string{"lentils"})

	for _, q := range []string{"lentil", "Lentil", "LENTIL"} {
		hits := tr.Search(q)
		if len(hits) != 1 {
			t.Errorf("query %q: got %d results, want 1", q, len(hits))
		}
	}
}

func TestTrie_ResultLimit(t *testing.T) {
	tr := New(WithSearchLimit(20))
	for i := range 25 {
		tr.Insert(string(rune('a'+i)), "Apple Recipe", nil)
	}

	hits := tr.Search("apple")
	if len(hits) > 20 {
		t.Errorf("got %d results, want at most 20", len(hits))
	}
}

func TestTrie_EmptyQuery(t *testing.T) {
	tr := New()
	tr.Insert("1", "Pasta", nil)

	hits := tr.Search("")
	if len(hits) != 0 {
		t.Errorf("got %d results for empty query, want 0", len(hits))
	}
}

func TestTrie_FuzzyFallback(t *testing.T) {
	tr := New()
	tr.Insert("1", "Chocolate Cake", []string{"chocolate"})

	// "choclate" is 1 edit away from "chocolate" — should find it
	hits := tr.Search("choclate")
	if len(hits) == 0 {
		t.Error("fuzzy search for 'choclate' returned no results, want chocolate recipe")
	}
}

func TestTrie_FuzzyTwoEdits(t *testing.T) {
	tr := New()
	tr.Insert("1", "Tomato Sauce", []string{"tomato"})

	// "tmoato" is 2 edits from "tomato" — should match (len > 4, k=2)
	hits := tr.Search("tmoato")
	if len(hits) == 0 {
		t.Error("fuzzy search for 'tmoato' returned no results, want tomato recipe")
	}
}

func TestTrie_Rebuild(t *testing.T) {
	tr := New()
	tr.Insert("1", "Old Recipe", nil)

	hits := tr.Search("old")
	if len(hits) != 1 {
		t.Fatalf("before rebuild: got %d, want 1", len(hits))
	}

	tr.Rebuild(nil) // empty rebuild wipes the trie

	hits = tr.Search("old")
	if len(hits) != 0 {
		t.Errorf("after rebuild: got %d, want 0", len(hits))
	}
}

func TestTrie_CustomOptions(t *testing.T) {
	tr := New(WithSearchLimit(5), WithFuzzyThreshold(2))
	for i := range 10 {
		tr.Insert(string(rune('a'+i)), "Apple Recipe", nil)
	}

	hits := tr.Search("apple")
	if len(hits) > 5 {
		t.Errorf("got %d results, want at most 5 (custom limit)", len(hits))
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"chocolate", "choclate", 1},
		{"tomato", "tmoato", 2},
	}
	for _, c := range cases {
		got := distance(c.a, c.b)
		if got != c.want {
			t.Errorf("distance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestMaxDist(t *testing.T) {
	if maxDist("cat") != 1 {
		t.Error("len<=4 should give k=1")
	}
	if maxDist("chicken") != 2 {
		t.Error("len>4 should give k=2")
	}
}
