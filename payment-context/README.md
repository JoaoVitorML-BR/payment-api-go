# Payment Flow Context

## Mandatory maintenance rule

Every AI or developer who changes this payment flow MUST update this document before finishing. Do not delete this instruction or erase previous context. Append the new implementation status, business rules, tests, commit, and the exact next step so the next AI can resume safely.

## Current scope

This repository contains two Go services:

- `payment-request-api`: creates the local payment request, publishes the RabbitMQ event, receives Mercado Pago webhooks, and reconciles status.
- `payment-consumer`: consumes the event and creates the Mercado Pago payment.

## Business rules

1. `payment_requests.amount_cents` is the gross amount paid by the customer and must never be changed by a refund.
2. Refund amounts are additive operations. The sum of successful refunds must never exceed the gross approved amount.
3. The consulting system owns cancellation policy and sends an already-authorized refund amount plus a reason.
4. Current refund policy: before 24 hours, 100%; with 2 hours remaining, 75%; with 1 hour or less, 50%; consultant cancellation, 100%.
5. The payment service must not calculate consultation timing, attendance, or service fulfillment.
6. A refund must be idempotent and confirmed by the gateway before being reported as successful.
7. `succeeded`, `partially_refunded`, and `refunded` are distinct states.
8. Mercado Pago fees are not the platform commission. Both values must be stored separately.

## Split rules

Mercado Pago Split 1:1 requires a seller linked to the marketplace through OAuth. The payment must use that seller's access token and the marketplace fee/application fee. A single global platform token is not sufficient for a real seller split.

Persist at least the gross amount, Mercado Pago fee, marketplace fee, seller net amount, seller identifier, and calculation/version metadata. Never implement an arbitrary `50/50` rule in the payment service.

## Provider rules

- Never trust webhook financial fields without re-querying Mercado Pago.
- Never log access tokens, signatures, customer tax IDs, or full authorization headers.
- Map Mercado Pago `approved` to local `succeeded`.
- Gateway calls require idempotency keys.
- Provider time limits and balance requirements can make a refund fail; that is not a successful local refund.

## What must not change

- Amounts remain integer cents.
- The original gross amount is immutable.
- Webhook signature, external reference, amount, and currency validation remain mandatory.
- RabbitMQ retries remain available for transient failures.
- Refund success is never inferred from a local database update alone.

## Implemented

- Added financial columns and a `payment_refunds` operation table to both migration sets.
- Added partial/full refund states.
- Prevented fake refund success and mutation of the original amount.
- Prevented duplicate idempotent requests from publishing a second event.
- Normalized consumer `approved` to local `succeeded`.
- Removed sensitive webhook debug logs and rejected future timestamps.
- Added a testable Mercado Pago OAuth client that builds the seller authorization URL and exchanges an authorization code for seller credentials.
- Added OAuth tests for query parameters, form submission, incomplete responses, and secret non-disclosure.

## Still required

- Wire the OAuth client into an authenticated authorization/callback flow.
- Add encrypted seller token storage and token renewal; never log or publish tokens.
- `application_fee` in Mercado Pago payment creation.
- Real provider refund client and refund reconciliation webhook.
- Atomic refund reservation and cumulative amount validation.
- Integration tests with Mercado Pago test accounts for split, partial/full refund, insufficient balance, and duplicate notifications.

## Current continuation point

- Branch: `feature/mercado-pago-split-refunds`
- Last completed commit: `5dc7be9 feat: prepare payment flow for split refunds`
- Next step: implement seller OAuth authorization and encrypted token storage before sending `application_fee` to Mercado Pago.

The OAuth client exists in `payment-request-api/internal/infra/paymentmercadopago/oauth.go`, but it is intentionally not wired into a route or payment creation yet. Do not send `application_fee` with the global platform token; the seller OAuth token and secure storage must exist first.