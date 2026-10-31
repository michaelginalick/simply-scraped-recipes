PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS recipes (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    source_url TEXT
);

CREATE TABLE IF NOT EXISTS ingredients (
    id        INTEGER PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name      TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS instructions (
    id        INTEGER PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_num  INTEGER NOT NULL,
    step      TEXT    NOT NULL
);

-- FTS5 virtual table: tokenize with porter stemmer so "baking" matches "bake"
CREATE VIRTUAL TABLE IF NOT EXISTS recipe_search USING fts5(
    name,
    ingredients,
    tokenize = 'porter ascii'
);

-- Keep recipe_search in sync with inserts/deletes
CREATE TRIGGER IF NOT EXISTS recipes_ai
AFTER INSERT ON recipes BEGIN
    INSERT INTO recipe_search(rowid, name, ingredients)
    SELECT NEW.id, NEW.name,
           COALESCE((SELECT GROUP_CONCAT(name, ' ') FROM ingredients WHERE recipe_id = NEW.id), '');
END;

CREATE TRIGGER IF NOT EXISTS recipes_ad
AFTER DELETE ON recipes BEGIN
    DELETE FROM recipe_search WHERE rowid = OLD.id;
END;

CREATE INDEX IF NOT EXISTS idx_ingredients_recipe_id ON ingredients(recipe_id);
CREATE INDEX IF NOT EXISTS idx_instructions_recipe_id_step_num ON instructions(recipe_id, step_num);

-- Refresh the search index after an ingredient is inserted
CREATE TRIGGER IF NOT EXISTS ingredients_ai
AFTER INSERT ON ingredients BEGIN
    DELETE FROM recipe_search WHERE rowid = NEW.recipe_id;

    INSERT INTO recipe_search(rowid, name, ingredients)
    SELECT r.id, r.name,
           GROUP_CONCAT(i.name, ' ')
    FROM recipes r
    JOIN ingredients i ON i.recipe_id = r.id
    WHERE r.id = NEW.recipe_id
    GROUP BY r.id;
END;

CREATE TABLE IF NOT EXISTS scrape_jobs (
    id           INTEGER PRIMARY KEY,
    url          TEXT NOT NULL UNIQUE,
    root_host    TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at   DATETIME,
    completed_at DATETIME,
    error_msg    TEXT
);

CREATE INDEX IF NOT EXISTS idx_scrape_jobs_status_created_at ON scrape_jobs(status, created_at);
