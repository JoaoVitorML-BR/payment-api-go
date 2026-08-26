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

## Mercado Pago Webhook Flow (CRITICAL - Security Boundary):

**Trust Model**: The webhook payload is NEVER trusted for financial state. It is treated as an untrusted trigger only.

1. HTTP handler receives `POST /webhooks/mercadopago` with `X-Signature` and `X-Request-Id` headers
2. Handler reads body (max 64 KiB), extracts only `data.id` from JSON payload
3. Handler calls `webhook.VerifySignature(xSignature, xRequestID, dataID, time.Now())` using the OFFICIAL Mercado Pago manifest: `id:<data.id>;request-id:<x-request-id>;ts:<ts>;`
4. On signature failure → 403 Forbidden (permanent, MP stops retrying)
5. On success → `service.ProcessMercadoPagoWebhook(ctx, dataID)`
6. Service queries local DB for `gateway_payment_id = dataID` → `PaymentGatewayValidationData` (uuid, expected_amount_cents, expected_currency, current_status)
7. If not found locally yet → `ErrWebhookRetryable` (5xx, MP retries)
8. Service calls `gatewayReader.GetPayment(dataID)` → Mercado Pago GET `/v1/payments/{id}`
9. Service validates **MANDATORY** fields from Mercado Pago response:
   - `external_reference` == local `payment_request.uuid`
   - `currency_id` == local expected currency (case-insensitive)
   - `transaction_amount` (converted to cents via round-half-up) == local expected amount_cents
10. If ANY mismatch → `ErrWebhookInvalidState` (422 Unprocessable Entity, MP stops retrying)
11. Service normalizes gateway status via `normalizeGatewayStatus()`:
    - `approved` → `succeeded`
    - `cancelled`/`canceled` → `canceled`
    - `rejected` → `failed`
    - `pending`/`in_process`/`in_mediation` → `pending`
12. Service enforces **STATE MACHINE** via `isAllowedStatusTransition(current, next)`:
    - `pending`/`processing`/`requires_*` → `pending`|`succeeded`|`failed`|`canceled`
    - `succeeded`|`failed`|`canceled` → NO transitions allowed (terminal states)
    - Same state → allowed (idempotent reprocessing)
13. If transition invalid → silently ignore (return nil, 200 OK to stop MP retries)
14. Service updates DB: `UPDATE payment_requests SET status=next_status, updated_at=NOW() WHERE gateway_payment_id=dataID`
15. HTTP 200 OK

Important behavior in the service:

- `idempotency_key` is required and must not be empty (ensures idempotent creates)
- `amount_cents` must be greater than zero
- `currency` is normalized to uppercase and must have exactly 3 characters
- `payment_method` is normalized to lowercase and must be one of: `pix` (only implemented)
- **Event publishing happens AFTER repository commit** for reliable delivery to RabbitMQ queue
- Idempotency is enforced at DB level: same idempotency_key returns existing payment
- **WEBHOOK SECURITY RULES (INVIOLÁVEIS)**:
  1. O cliente NUNCA define status financeiro. O webhook NUNCA é prova suficiente de pagamento.
  2. O payment ID do webhook DEVE ser consultado no Mercado Pago (GET /v1/payments/{id}).
  3. O pagamento consultado DEVE corresponder à cobrança local (external_reference == payment_request.uuid).
  4. O amount DEVE ser exatamente igual ao esperado (centavos, round-half-up).
  5. A currency DEVE ser exatamente igual à esperada (BRL).
  6. Um pagamento aprovado NUNCA pode ser sobrescrito por evento antigo (máquina de estados).
  7. Webhook duplicado NUNCA gera efeito financeiro duplicado.
  8. Nenhum segredo (access token, webhook secret) pode aparecer em logs.

What should not happen here:

- do not register routes here
- do not create the Gin router here
- do not load environment variables here
- do not make the handler know database details
- do not move generic infrastructure concerns into this package
- **do not trust webhook payload for status/amount/currency**

In short: `payment` is the feature boundary for payment requests. It owns the use case, the transport adapter, and the database adapter for that feature.