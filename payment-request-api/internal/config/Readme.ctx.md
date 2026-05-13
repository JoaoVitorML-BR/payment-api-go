# Config context

The `config` package is responsible for loading runtime configuration and preparing shared infrastructure needed by the application at startup.

In this project, `internal/config/config.go` currently does two things:

- loads environment variables from `.env` when present
- builds and validates the PostgreSQL connection pool used by the rest of the application

What this package contains:

- the `Config` struct with startup settings
- `LoadConfig()` as the entry point for application configuration
- default values for missing environment variables
- creation of `pgxpool.Pool`

Why it exists:

- it keeps environment parsing in one place
- it avoids duplicating defaults across the codebase
- it ensures the DB pool is created once and shared by the rest of the app
- it gives `main` and `bootstrap` a single configuration object to work with

What should not happen here:

- do not register HTTP routes here
- do not add business rules here
- do not create handlers or services here
- do not hide application behavior in multiple config files

Important behavior in this project:

- `LoadConfig()` loads `.env` if available
- `PORT` defaults to `8080`
- DB settings default to local PostgreSQL values when missing
- the database connection is created and pinged before startup continues
- the pool is returned so `main` can close it when the process exits

In short: `config` is the startup input layer. It prepares the values and shared resources that the bootstrap and server layers need.
