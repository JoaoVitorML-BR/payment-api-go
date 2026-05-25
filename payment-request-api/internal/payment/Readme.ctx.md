# Payment context

The `payment` package contains the payment use case, its HTTP handler, and its repository implementation.

This package currently includes three responsibilities that belong to the same feature boundary:

- `handler.go` exposes the HTTP controller for payment requests
- `service.go` contains the payment use case and validation rules
- `repository.go` persists payment requests to PostgreSQL through sqlc-generated code

Why it exists:

- it keeps all payment feature code together
- it gives the application a single domain boundary for payment requests
- it separates request handling, business rules, and persistence concerns while keeping them close to each other
- it makes the payment flow easier to test and reason about

## CreatePayment End-to-End Flow:

1. `internal/server/router.go` receives a ready handler from bootstrap
2. Handler binds incoming JSON to `CreatePaymentRequest`
3. Handler calls `PaymentService.CreatePayment(...)`
4. Service validates payment rules
5. Service calls `PaymentRepository.CreatePaymentRequest()` → persists to PostgreSQL
6. Service calls `publisher.Publish(event)` → publishes to RabbitMQ queue
7. Event remains in queue for consumer to process
8. HTTP response returns 201 Created with payment ID

Important behavior in the service:

- `idempotency_key` is required and must not be empty (ensures idempotent creates)
- `amount_cents` must be greater than zero
- `currency` is normalized to uppercase and must have exactly 3 characters
- `payment_method` is normalized to lowercase and must be one of: `credit`, `debit`, `pix`, `boleto`
- `credit` payments allow optional installments between 1-12
- **Event publishing happens AFTER repository commit** for reliable delivery to RabbitMQ queue
- Idempotency is enforced at DB level: same idempotency_key returns existing payment

What should not happen here:

- do not register routes here
- do not create the Gin router here
- do not load environment variables here
- do not make the handler know database details
- do not move generic infrastructure concerns into this package

In short: `payment` is the feature boundary for payment requests. It owns the use case, the transport adapter, and the database adapter for that feature.
