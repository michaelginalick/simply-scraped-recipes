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
  api/           — API server entry point
    static/      — HTML/CSS/JS frontend
  scraper/       — CLI scraper entry point
internal/
  domain/        — core types and errors
  dto/           — request/response types
  handler/       — HTTP handlers
  middleware/    — logging, recovery, JSON content-type
  service/       — job and probe services
  sqlstore/      — SQLite persistence
  application/   — scraper orchestration
  links/         — HTTP link fetching and parsing
  recipes/       — recipe extraction from HTML
  store/         — scraper write store
migrations/      — schema SQL
terraform/       — Lightsail infrastructure
```

## Prerequisites

- Go 1.26+
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
| `GET` | `/recipe/{id}` | Get a recipe by ID (JSON) |
| `GET` | `/r/{id}` | Shareable recipe page |
| `POST` | `/recipe` | Create a recipe |
| `POST` | `/jobs` | Submit a URL to scrape |
| `GET` | `/jobs/{id}` | Get scrape job status |
| `POST` | `/probe` | Test whether a URL contains a recipe |

All JSON responses are wrapped in `{"data": ...}`.


## Docker

```bash
make docker-build
make docker-up       # runs the API
make docker-scraper  # runs the scraper
make docker-down
```
