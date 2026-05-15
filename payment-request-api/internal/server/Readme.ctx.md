# Server context

The `server` package is responsible for HTTP server setup and route registration.

In this project, `internal/server` has two clear jobs:

- `router.go` builds the Gin engine and registers HTTP routes
- `http.go` runs the server using the configured port

Why it exists:

- it keeps HTTP concerns separate from startup composition and business logic
- it avoids putting routing code into `main`
- it keeps the router focused on endpoint registration only
- it gives the application one place for HTTP server behavior

Current responsibilities:

- `SetupRouter(paymentHandler)` creates the Gin router
- it registers `/health`
- it registers `POST /payment`
- `Run(cfg, router)` starts the HTTP server on `cfg.Port`

How it relates to other packages:

- `main` loads config and starts the process
- `bootstrap` assembles repositories, services, and handlers
- `server` receives already-built handlers and exposes them through HTTP

What should not happen here:

- do not create repositories here
- do not create services here
- do not load environment variables here
- do not embed payment business rules here
- do not move persistence logic into the router

In short: `server` is the HTTP delivery layer. It exposes the application over the network, but it should not assemble the domain itself.
