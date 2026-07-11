# Before run

> Copy .env.example to .env in the repo root (single source for backend, frontend, and docker compose)

# Run

To run:

> make run

To run tests:

> make test

Before commit (make sure `golangci-lint` is installed):

> make lint

# Schema

The internal store is SQLite. The schema is managed by GORM `AutoMigrate`, which
runs at application startup (see `runAutoMigrate` in `cmd/main.go`) — there is no
separate migration step.