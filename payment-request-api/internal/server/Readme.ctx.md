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
- it registers `GET /health` for liveness checks
- it registers `POST /payment` which routes to `paymentHandler.CreatePaymentRequestHandler()`
- it registers `GET /payment/client-secret/:id` for frontend polling
- it registers `POST /webhooks/mercadopago` for Mercado Pago notifications
- `Run(cfg, router)` starts the HTTP server on `cfg.Port`

## CreatePayment HTTP Endpoint
- **Route**: `POST /payment`
- **Payload**: JSON `CreatePaymentRequest` with idempotency_key, amount_cents, currency, payment_method
- **Success (201)**: Returns payment ID, method, status, timestamps
- **Validation Error (400)**: Returns validation error message
- **Processing Error (500)**: Returns error message
- **Flow**: HTTP handler → service (validates + persists + publishes) → response

## GetPaymentClientSecret Endpoint (Frontend Polling)
- **Route**: `GET /payment/client-secret/:id`
- **Response**: Returns payment attempt data (Pix QR Code, Pix QR Code Base64, expiration, or Stripe client_secret)
- **Success (200)**: Payment attempt found
- **Not Found (404)**: Payment attempt not yet created by worker (frontend should retry)
- **Error (500)**: Internal error

## Mercado Pago Webhook Endpoint
- **Route**: `POST /webhooks/mercadopago`
- **Headers**: `X-Signature`, `X-Request-Id`
- **Security Rules (NON-NEGOTIABLE)**:
  1. The webhook payload is NEVER trusted for financial state. Only the "data.id" is extracted and used to re-query Mercado Pago.
  2. The X-Signature header is verified using the official Mercado Pago manifest (id:<data.id>;request-id:<x-request-id>;ts:<ts>;).
  3. The service layer validates external_reference, amount and currency against the local database before any status change.
- **Response Contract**:
  - 200/201: notification processed (or intentionally ignored) — stop retries.
  - 4xx (except 401/403): permanent rejection — stop retries.
  - 5xx / 408: temporary failure — Mercado Pago will retry with backoff.
- **Flow**: HTTP handler → extract data.id → verify X-Signature → service.ProcessMercadoPagoWebhook(data.id) → consult Mercado Pago → validate external_reference/amount/currency → apply state machine → update DB → 200 OK

How it relates to other packages:

- `main` loads config and starts the process
- `bootstrap` wires repositories, publishers (RabbitMQ), services, handlers, and gatewayReader together
- `server` receives already-built handlers and exposes operations through HTTP
- Request flows: HTTP POST /payment → handler (JSON bind) → service (validate + repo.create + publisher.publish) → HTTP 201

What should not happen here:

- do not create repositories here
- do not create services here
- do not load environment variables here
- do not embed payment business rules here
- do not move persistence logic into the router

In short: `server` is the HTTP delivery layer. It exposes the application over the network, but it should not assemble the domain itself.