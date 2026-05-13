# Infra context

The `infra` area contains infrastructure concerns that support the application but are not part of HTTP routing or business rules.

In this project, `internal/infra` currently focuses on database support:

- SQL schema definitions used by sqlc
- SQL queries used to generate typed database code
- generated bridge code used by the payment repository
- migration files that create and evolve the PostgreSQL schema

Why it exists:

- it keeps persistence details out of the payment service and handler layers
- it centralizes schema, query, and migration work in one place
- it separates generated database code from domain logic
- it gives the repository a stable package to depend on

What lives here now:

- `internal/infra/database/sqlc/schema.sql`
- `internal/infra/database/sqlc/queries.sql`
- `internal/infra/database/sqlc/sqlc.yaml`
- `internal/infra/database/bridge/`
- `internal/infra/database/migrations/`

How this is used:

- `sqlc` reads the schema and queries to generate typed Go code
- the generated bridge package is consumed by `internal/payment/repository.go`
- migrations define the actual database tables and constraints

What should not happen here:

- do not put HTTP handlers here
- do not put validation or business rules here
- do not make the infra package depend on the payment service
- do not mix generated code with ad hoc application logic unless it is clearly part of the database boundary

In short: `infra` is the persistence and platform support layer. It exists so the rest of the application can stay focused on domain behavior.
