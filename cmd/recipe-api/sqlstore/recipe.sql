-- name: GetRecipe
SELECT
    r.name,
    (SELECT GROUP_CONCAT(i.name, char(31))
     FROM ingredients i
     WHERE i.recipe_id = r.id) AS ingredients,
    (SELECT GROUP_CONCAT(step, char(30))
     FROM (SELECT step FROM instructions WHERE recipe_id = r.id ORDER BY step_num)) AS instructions
FROM recipes r
WHERE r.id = @id

-- name: CreateRecipe
INSERT OR IGNORE INTO recipes (name) VALUES (@name)

-- name: CreateIngredient
INSERT INTO ingredients (recipe_id, name) VALUES (@recipe_id, @name)

-- name: CreateInstruction
INSERT INTO instructions (recipe_id, step_num, step) VALUES (@recipe_id, @step_num, @step)

-- name: SearchRecipe
SELECT
    r.id,
    r.name,
    (SELECT GROUP_CONCAT(i.name, char(31))
     FROM ingredients i
     WHERE i.recipe_id = r.id) AS ingredients,
    (SELECT GROUP_CONCAT(step, char(30))
     FROM (SELECT step FROM instructions WHERE recipe_id = r.id ORDER BY step_num)) AS instructions
FROM recipe_search
JOIN recipes r ON r.id = recipe_search.rowid
WHERE recipe_search MATCH @query
ORDER BY rank


-- name: SchemaExists
SELECT EXISTS (
    SELECT name from sqlite_schema WHERE type='table' AND name='recipes'
) AS table_exists;