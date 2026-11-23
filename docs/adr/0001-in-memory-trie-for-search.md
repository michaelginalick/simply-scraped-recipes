# Replace SQLite FTS5 search with an in-memory trie

The recipe search endpoint (`GET /search`) previously delegated every query to SQLite's FTS5 virtual table. We replaced it with an in-memory prefix trie seeded at startup from the full recipe corpus (names and ingredient words). The trie uses Levenshtein DP traversal to tolerate typos (0 edits under 4 chars, 1 edit at 4–7, 2 edits at 8+), returns results ordered by edit distance, and never hits the database at query time.

## Considered Options

**SQLite FTS5** was the incumbent. It provides relevance-ranked full-text search across all recipe fields with no extra memory. The trade-off: every keystroke in the search box fires a synchronous DB read, and FTS5 does not support prefix-plus-fuzzy matching in a single pass.

**In-memory trie with Levenshtein DP traversal** trades a one-time memory cost (bounded by corpus size) for sub-millisecond query times and built-in typo tolerance without a DB round trip. The trie is seed-only: it is populated once at startup and not updated on write. Recipes added after startup are invisible to search until the next restart.

## Consequences

The search response shape changed from full recipe objects to `{id, name}` pairs. The existing click handler already issued a separate `GET /recipe/{id}` to load full detail, so no frontend change was required. The FTS5 virtual table, its query, and the `SearchRecipe` interface method were all removed.
