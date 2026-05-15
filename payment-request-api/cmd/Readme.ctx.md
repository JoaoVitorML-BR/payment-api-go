# Main entrypoint context

The `main` package is the application entrypoint.

In this project, `cmd/main.go` is responsible for startup orchestration only:

- load the application configuration
- create the database pool through `config.LoadConfig()`
- pass the config to `bootstrap.NewRouter(cfg)`
- start the HTTP server with `server.Run(cfg, router)`
- close the database pool when the process exits

Why this file exists:

- it keeps process startup in one obvious place
- it separates lifecycle concerns from HTTP routing and business logic
- it prevents the router or bootstrap from owning the application entrypoint
- it makes the startup sequence easy to read from top to bottom

What should not happen here:

- do not add request validation here
- do not add route definitions here
- do not create repositories, services, or handlers here
- do not put business rules here

Relationship with bootstrap:

- `main` decides when the application starts
- `bootstrap` decides how the application is assembled
- `server` decides how routes are registered and the HTTP server runs

In short: `main` is only the launcher. The app composition lives in bootstrap, and the HTTP concerns live in server.
