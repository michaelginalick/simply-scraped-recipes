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
