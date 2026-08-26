# Payment Mercado Pago Gateway Reader context

The `infra/paymentmercadopago` package provides a **read-only HTTP client** for querying the authoritative state of a payment directly from the Mercado Pago API.

## Why it exists

- The webhook handler receives an untrusted notification; the ONLY way to know the real payment state is to query Mercado Pago directly
- This package implements the "trust boundary": webhook triggers → re-query Mercado Pago → validate against local DB
- It isolates the Mercado Pago HTTP client and response parsing from business logic
- It uses the same access token as the consumer but for READ operations only

## Responsibilities

- `GetPayment(ctx, gatewayPaymentID)` → calls Mercado Pago `GET /v1/payments/{id}`
- Validates payment ID format (numeric string)
- Parses response: `id`, `status`, `external_reference`, `transaction_amount`, `currency_id`
- Converts `transaction_amount` (float64) to integer cents using **round-half-up** (math.Round) to avoid float truncation errors
- Returns `*payment.GatewayPaymentDetails` (defined in the payment package) to satisfy `payment.GatewayPaymentReader` interface
- Handles HTTP errors: 404 → not found, 5xx → retryable, other → error

## Mercado Pago API Contract (GET /v1/payments/{id})

Relevant fields from the response:
```json
{
  "id": 123456789,
  "status": "approved",           // "approved", "pending", "rejected", "cancelled", "in_process", "in_mediation"
  "status_detail": "accredited",
  "external_reference": "uuid-from-payment-request",
  "transaction_amount": 150.00,   // decimal, MUST be converted to cents via round-half-up
  "currency_id": "BRL"
}
```

## Amount Conversion: Float → Integer Cents

**CRITICAL**: Mercado Pago returns `transaction_amount` as a decimal (float64 in JSON). Direct `int64(amount * 100)` truncates (e.g., 149.99999999997 → 14999 instead of 15000).

This package uses `math.Round(v * 100)` (round-half-up) to guarantee correct cent conversion:
- 149.99999999997 → 15000 ✓
- 150.00000000001 → 15000 ✓
- 149.49 → 14949 ✓

## Security

- **NEVER log the access token** (`Authorization: Bearer <token>`)
- The access token is passed at construction time from environment variable `MERCADO_PAGO_ACCESS_TOKEN`
- HTTP timeout: 10 seconds (configurable via constant)
- Base URL: `https://api.mercadopago.com` (production) - can be overridden for sandbox/testing

## What should not happen here

- Do not create payments (POST /v1/payments) - that's the consumer's job
- Do not process refunds
- Do not implement business logic (state machine, validation against local DB) - that's the service layer
- Do not handle webhook signatures - that's `infra/webhook`

In short: `infra/paymentmercadopago` is a thin, read-only HTTP client that fetches the authoritative payment state from Mercado Pago and returns it in a normalized, type-safe structure for the service layer to validate.