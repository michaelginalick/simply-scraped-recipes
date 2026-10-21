# Recipe App

A recipe scraper and search API. Scrape recipes from *almost any recipe site, store them in SQLite, and search them via a web UI.
  ** Recipe sites that use recipe cards (visual layouts) combined with JSON-LD schema markup.

Live at [simplyscrapedrecipes.com](https://simplyscrapedrecipes.com).

## Notes
 This is a mostly vibe coded application for me to learn how to better use AI. However, I was able to exercise some ideas and implement concepts I've found in books and on the internet. The resources are listed below

- The overall API design was inspired but this post:
  - https://golangbot.com/secure-go-code/
- Separating the SQL from the application code was taken from the excellent book:
  - https://pragprog.com/titles/mtgo/effective-go-recipes/
- The web scraping algorithm was taken from another excellent book:
  - https://www.thriftbooks.com/w/the-go-programming-language_brian-w-kernighan_alan-a-a-donovan/13867697/item/23690184/
- The middleware design was taken from this post:
  - https://www.alexedwards.net/blog/making-and-using-middleware
- The Hexagonal Architecture approach came from this Udemy course:
  - https://www.udemy.com/course/rest-based-microservices-api-development-in-go-lang/


## Architecture

```
cmd/
  scraper/       — CLI that fetches and parses recipe pages
  recipe-api/
    handler/     — HTTP handlers (parse input, call domain, emit output)
    sqlstore/    — SQLite implementation of domain interfaces
    middleware/  — logging, recovery, JSON content-type
    static/      — HTML/CSS/JS frontend
pkg/
  recipes/       — domain types, interfaces, errors
  links/         — HTTP link fetching
  application/   — scraper orchestration
migrations/      — schema SQL
terraform/       — Lightsail infrastructure
```

## Prerequisites

- Go 1.22+
- SQLite (`sqlite3` CLI for DB management)
- Docker (optional)

## Local development

```bash
# Run the API (defaults to ./recipes.db)
make run-api

# Scrape a recipe into the local DB
make run-scraper SOURCE=https://example.com/some-recipe

# Scrape with debug logging
make run-scraper SOURCE=https://example.com/some-recipe DEBUG=true
```

The API serves the frontend at `http://localhost:8080`.

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/search?q=<query>` | Full-text search recipes |
| `GET` | `/recipe/{id}` | Get a recipe by ID |
| `POST` | `/recipe` | Create a recipe |

All JSON responses are wrapped in `{"data": ...}`.


## Docker

```bash
make docker-build
make docker-up       # runs the API
make docker-scraper  # runs the scraper
make docker-down
```
