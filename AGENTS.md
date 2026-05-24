# Project Guidelines

## Architecture

Go web application for submitting and reviewing wiki edit patches (subject/episode/person/character). Uses Chi v5 router, Uber Fx for dependency injection, and PostgreSQL via pgx/v5.

Entry point: `main.go` → Fx app with `handler` struct holding all dependencies.

Key flows:
- Users submit patches → stored in PostgreSQL
- Reviewers accept/reject patches → wiki updated via external API
- Canal (Debezium CDC) auto-marks patches outdated when wiki changes
- Notifications sent via Kafka (protobuf-encoded messages on `notify.v1` topic)

## Code Style

- Go 1.26, standard library style
- Templates: [templ](https://templ.guide/) (`.templ` files in `templates/`)
- SQL: sqlc for type-safe query generation from `db/query.sql`
- Error handling: `handleError` helper returns appropriate HTTP responses
- HTTP handlers are methods on the `handler` struct (Chi router)

## Build and Test

Uses [Taskfile](https://taskfile.dev/):

- `task build` — build binary to `dist/app.exe`
- `task dev` — build and run with dev tag
- `task gen` — run sqlc code generation
- `task gen:template` — run templ code generation
- `task format` — gofmt + templ fmt
- `task migrate` — run database migrations

## Conventions

- Session/CSRF middleware via gorilla packages
- Config loaded from environment variables (see `config.go`)
- Database transactions use a `tx` helper pattern
- Kafka messages use manual protowire encoding (no generated Go protobuf code)
- External wiki API calls go through `api.go` client methods

## Patch Field Change Detection

Each patch table (subject_patch, character_patch, person_patch, episode_patch) uses **NULL vs non-NULL** to distinguish which fields the user explicitly modified:

- **NULL** — user did not submit/change this field
- **non-NULL** — user provided a new value

The `original_*` columns always store the baseline wiki value at submission time (used for conflict detection during approval). They are unconditionally populated regardless of which fields changed.

Example for `subject_patch`:

| Column | NULL? | Meaning |
|---|---|---|
| `name` | non-NULL | user changed the name |
| `name` | NULL | user did not change the name |
| `meta_tags` | non-NULL | user changed meta_tags |
| `meta_tags` | NULL | user did not change meta_tags |
| `original_meta_tags` | always non-NULL | baseline for conflict detection |

When building patch creation/update handlers, only set the field column when the user's value actually differs from the original. Never unconditionally assign a field column — doing so would incorrectly mark an unchanged field as "user modified" and produce spurious diffs.
