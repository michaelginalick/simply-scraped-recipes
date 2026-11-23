# Glossary

## Crawl

A single execution of the recursive link-following process, starting from a root URL and visiting every unseen link scoped to the **rootHost**. A Crawl is the work performed by one **Job**. Crawls are not persisted — progress is in-memory only and lost if the process exits.

## Job

A user-submitted request to crawl a site. Tracks status (`pending → running → done | error`), the root URL, and timing. A new Job for a URL is rejected if a pending or running Job already exists for that URL; completed and errored Jobs are re-submittable.

## Recipe

A named culinary preparation with a list of **Ingredients** and **Instructions**, extracted from a single page during a Crawl. Stored with its `source_url` and `scraped_at` timestamp. Deduplicated on both `source_url` and `name` — whichever matches first wins; the first scraped version is kept.

## Ingredient

A single item in a Recipe's ingredient list, stored as a flat text string (amount + unit + name + notes joined).

## Instruction

A single step in a Recipe's instruction list, stored as plain text with an ordering position.

## rootHost

The hostname that scopes which links a Crawl will follow. Derived automatically from the submitted URL's host component. Links whose host does not match rootHost are discarded.

## WPRM

WP Recipe Maker — the WordPress plugin whose HTML markup the parser targets. The parser is intentionally scoped to WPRM only; additional parsers can be added per site as needed.

## seenLinks

The in-memory set of URLs already visited during a Crawl, used to prevent revisiting the same page. Not persisted — a re-submitted Job re-crawls from scratch.
