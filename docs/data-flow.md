# Data Flow — Recipe API

Dependency always flows **inward** toward `domain`. Outer layers know about inner layers; inner layers know nothing about outer layers.

```
┌─────────────────────────────────────────────────────────┐
│  HTTP (outer)                                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │  handler / dto (edge)                             │  │
│  │  ┌─────────────────────────────────────────────┐  │  │
│  │  │  sqlstore (adapter)                         │  │  │
│  │  │  ┌───────────────────────────────────────┐  │  │  │
│  │  │  │  domain (core)                        │  │  │  │
│  │  │  │  Recipe, Ingredient, Instruction      │  │  │  │
│  │  │  │  ErrNotFound, ErrAlreadyExists        │  │  │  │
│  │  │  └───────────────────────────────────────┘  │  │  │
│  │  └─────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## Read — GET /recipe/{id}

```
HTTP Request
    │
    ▼
middleware.Chain          (Recovery → Logging → JSONContentType)
    │
    ▼
handler.GetHandler
    │  1. r.PathValue("id") — extract path param
    │  2. strconv.ParseInt — validate id is numeric
    │
    ▼
handler.Reader.GetRecipe(ctx, id)   ← interface defined in handler
    │                                  implemented by sqlstore.sqlReader
    ▼
sqlstore.sqlReader.GetRecipe
    │  1. executes GetRecipe SQL via read-only *sql.DB (10 conns)
    │  2. scans name, ingredients (GROUP_CONCAT char(31)), instructions (GROUP_CONCAT char(30))
    │  3. sql.NullString guards against NULL when no ingredients/instructions exist
    │  4. splits concatenated strings → []string
    │  5. dbRecipe.toDomain() → domain.Recipe
    │     returns domain.ErrNotFound if sql.ErrNoRows
    │
    ▼
domain.Recipe             (core type — no external dependencies)
    │
    ▼
handler.GetHandler (continued)
    │  dto.NewRecipeResponse(id, domain.Recipe) → dto.RecipeResponse
    │  json.NewEncoder(w).Encode(envelope[dto.RecipeResponse]{Data: rr})
    │
    ▼
HTTP Response  {"data": {"id":"1","name":"...","ingredients":[...],"instructions":[...]}}
```

---

## Read — GET /search?q=

```
HTTP Request
    │
    ▼
middleware.Chain          (Recovery → Logging → JSONContentType)
    │
    ▼
handler.SearchHandler
    │  1. r.URL.Query().Get("q")
    │  2. sanitizeQuery — strips FTS5 special chars ["*^():+\-;]
    │     returns 400 if query is empty after sanitization
    │
    ▼
handler.Reader.SearchRecipe(ctx, q)
    │
    ▼
sqlstore.sqlReader.SearchRecipe
    │  1. strings.ToLower(q) — case-insensitive match
    │  2. executes SearchRecipe SQL against recipe_search FTS5 table
    │     FTS5 JOIN recipes → returns id, name, ingredients, instructions
    │  3. scans each row → dbRecipe.toDomain() → domain.Recipe
    │
    ▼
[]domain.Recipe
    │
    ▼
handler.SearchHandler (continued)
    │  dto.NewRecipesResponse([]domain.Recipe) → []dto.RecipeResponse
    │  json.Encode(envelope[[]dto.RecipeResponse]{Data: resp})
    │
    ▼
HTTP Response  {"data": [{"id":"1","name":"..."},...]}
```

---

## Write — POST /recipe

```
HTTP Request  {"name":"...","ingredients":[...],"instructions":[...]}
    │
    ▼
middleware.Chain          (Recovery → Logging)
    │
    ▼
handler.CreateHandler
    │  1. http.MaxBytesReader — cap body at 1MB
    │  2. json.Decode → dto.CreateRecipeRequest
    │  3. req.Validate() — returns 400 if name is empty
    │
    ▼
handler.Writer.CreateRecipe(ctx, dto.CreateRecipeRequest)
    │
    ▼
sqlstore.sqlWriter.CreateRecipe
    │  1. db.BeginTx — single write connection (1 conn, WAL mode)
    │  2. INSERT OR IGNORE INTO recipes (name)
    │     RowsAffected == 0 → domain.ErrAlreadyExists → 409
    │  3. LastInsertId → recipe id
    │  4. INSERT INTO ingredients for each ingredient
    │  5. INSERT INTO instructions for each instruction (with step_num)
    │  6. tx.Commit
    │     tx.Rollback deferred — runs only if Commit not reached
    │
    ▼
HTTP Response  201 Created
```

---

## Dependency map

```
main.go
  ├── imports handler      (wires Reader/Writer interfaces to sqlstore impls)
  ├── imports sqlstore     (NewSQLReader, NewSQLWriter, SchemaExists)
  ├── imports middleware   (Chain, Recovery, Logging, JSONContentType)
  └── imports migrations   (schema SQL applied at startup)

handler
  ├── imports domain       (Recipe, ErrNotFound, ErrAlreadyExists)
  └── imports dto          (CreateRecipeRequest, RecipeResponse)

sqlstore
  ├── imports domain       (Recipe, Ingredient, Instruction, ErrNotFound, ErrAlreadyExists)
  └── imports dto          (CreateRecipeRequest — reads Name, Ingredients, Instructions directly)

dto
  └── imports domain       (Recipe — used in NewRecipeResponse)

domain
  └── imports nothing      (stdlib errors only)
```

---

## Key invariants

| Invariant | Where enforced |
|-----------|---------------|
| Numeric recipe ID | `handler.GetHandler` — `strconv.ParseInt` |
| Non-empty name | `dto.CreateRecipeRequest.Validate()` |
| FTS5 injection prevention | `handler.sanitizeQuery` — strips `["*^():+\-;]` |
| Body size cap | `http.MaxBytesReader` (1MB) in `CreateHandler` |
| Duplicate recipe | `INSERT OR IGNORE` + `RowsAffected` check in `sqlwriter` |
| Write serialisation | `maxWriteConnections = 1` — single SQLite writer |
| NULL scan safety | `sql.NullString` for `GROUP_CONCAT` columns |
| Panic safety | `middleware.Recovery` — re-panics `http.ErrAbortHandler`, logs and 500s everything else |
