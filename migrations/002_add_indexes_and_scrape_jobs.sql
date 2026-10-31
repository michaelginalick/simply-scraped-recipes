CREATE INDEX IF NOT EXISTS idx_ingredients_recipe_id ON ingredients(recipe_id);
CREATE INDEX IF NOT EXISTS idx_instructions_recipe_id_step_num ON instructions(recipe_id, step_num);

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
