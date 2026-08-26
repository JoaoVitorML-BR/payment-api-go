# Webhook context

The `infra/webhook` package implements the **official Mercado Pago webhook signature verification** according to the current Mercado Pago documentation.

## Why it exists

- Webhook signature verification is a security-critical operation that must match Mercado Pago's exact specification
- It isolates the cryptographic logic from HTTP handlers and business logic
- It provides a single, well-tested place for signature verification that can be reused

## Official Mercado Pago X-Signature Format

According to Mercado Pago documentation, the `X-Signature` header has this format:

```
ts=<timestamp>,v1=<hmac_sha256>
```

Where:
- `ts` = Unix timestamp (seconds) when Mercado Pago sent the notification
- `v1` = HMAC-SHA256 of the **manifest string** using the webhook secret as key

## Manifest Construction (CRITICAL - Exact Format)

The manifest string that is signed follows this exact format:

```
id:<data.id>;request-id:<x-request-id>;ts:<ts>;
```

Where:
- `data.id` = The payment/resource ID from the webhook payload (extracted by handler)
- `x-request-id` = Value of the `X-Request-Id` HTTP header
- `ts` = The timestamp from the `X-Signature` header (ts=...)

**Important**: The manifest MUST end with a semicolon (`;`). All components are lowercase. No spaces around colons or semicolons.

## Verification Algorithm

1. Parse `X-Signature` header to extract `ts` and `v1`
2. Construct manifest: `id:<dataID>;request-id:<xRequestID>;ts:<ts>;`
3. Compute HMAC-SHA256(manifest, webhookSecret)
4. Compare computed HMAC with `v1` using **constant-time comparison** (crypto/subtle.ConstantTimeCompare)
5. Optional: Check timestamp freshness (reject if |now - ts| > tolerance, e.g., 5 minutes) for replay protection

## Error Handling

- Missing/invalid `X-Signature` → return error (handler returns 403)
- Missing `X-Request-Id` → return error (handler returns 403)
- Invalid timestamp format → return error (handler returns 403)
- Signature mismatch → return error (handler returns 403)
- Timestamp too old/future → return error (handler returns 403, replay protection)

## Security Notes

- **NEVER log the webhook secret, X-Signature, or computed HMAC**
- The webhook secret must be stored in environment variable `MERCADO_PAGO_WEBHOOK_SECRET`
- Use constant-time comparison to prevent timing attacks
- The handler enforces max body size (64 KiB) before signature verification

## What should not happen here

- Do not make HTTP calls to Mercado Pago here
- Do not access database here
- Do not implement business logic here
- Do not parse the full webhook payload (only data.id is needed by caller)

In short: `infra/webhook` is a pure crypto utility. It takes (X-Signature, X-Request-Id, data.id, now) and returns valid/invalid. Nothing more.