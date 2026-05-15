# Bootstrap context

Bootstrap is the composition root of the application.

In this project, `internal/bootstrap` exists to assemble the concrete dependencies that the HTTP API needs before the server starts:

- load the database-backed repository
- inject the repository into the payment service
- inject the service into the payment handler
- return a ready-to-use Gin router

Why it is necessary:

- it keeps dependency wiring out of `internal/server/router.go`
- it keeps `cmd/main.go` small and focused on startup only
- it makes the application easier to test because each layer can be created independently
- it gives the project one explicit place where dependency composition happens

Why the code lives here:

- `cmd/main.go` should only load config, call the bootstrap, and start the server
- `internal/server/router.go` should only register routes and receive already-built handlers
- `internal/bootstrap/bootstrap.go` is the place that knows how to connect config, repository, service, and handler

Current implementation details:

- `NewRouter(cfg *config.Config)` is the bootstrap entry point
- it creates `paymentRepository := handler.NewPaymentRepositoryDB(cfg.Pool)`
- it creates `paymentService := handler.NewPaymentService(paymentRepository)`
- it creates `paymentHandler := handler.NewPaymentHandler(paymentService)`
- it delegates route registration to `server.SetupRouter(paymentHandler)`

Important constraints for future AI or maintainers:

- do not add business rules here
- do not add request validation here
- do not define route paths here
- do not make the router build repositories or services again
- keep this package focused on wiring dependencies only

In short: bootstrap exists so the rest of the codebase can stay clean. It is the single place where the application is assembled before the HTTP server starts.