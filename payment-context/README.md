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
- Added an opt-in OAuth start/callback flow at `/oauth/mercadopago/start` and `/oauth/mercadopago/callback`.
- Added AES-256-GCM encrypted file storage for seller credentials with restrictive file permissions.
- Added one-time, ten-minute OAuth state validation to prevent callback replay and state reuse.
- Propagated `seller_id` and `marketplace_fee_cents` from the payment request through RabbitMQ to the consumer.
- Added seller-token decryption in the consumer and sent `application_fee` only with the matching seller token.
- Shared the encrypted token volume between Docker services.
- Added real Mercado Pago full/partial refund execution through the SDK.
- Added idempotent local refund reservation with cumulative amount protection.
- Added `succeeded`, `partially_refunded`, and `refunded` state updates after gateway approval.
- Added unit coverage for the successful refund path.
- Added refund recovery that lists local `processing` operations and matches already-approved gateway refunds before any new attempt.
- Added row locking during refund reservation so concurrent refunds cannot exceed the original amount.
- Added unit coverage for the recovery path.
- Added `MANUAL_TESTING.md` with the complete manual test sequence and acceptance criteria.
- Started the Orders API migration in `payment-consumer`: Pix creation and lookup now use `order.Client`, `processing_mode=automatic`, `transactions.payments`, and `marketplace_fee`.
- Added flexible JSON unmarshaling for `user_id` in `OAuthToken` to support both numeric and string IDs from Mercado Pago's OAuth API.
- Updated `.env.example` files across `payment-request-api` and `payment-consumer` to include all required Mercado Pago environment variables (`MERCADO_PAGO_WEBHOOK_SECRET`, `MERCADO_PAGO_ACCESS_TOKEN`, `MERCADO_PAGO_WEBHOOK_URL`, `PIX_EXPIRATION_TIME`) and credential mappings.

## Still required

- Add application authentication/authorization around the OAuth start route for production use.
- Add token renewal; never log or publish tokens.
- Token renewal and application authentication around the OAuth start route for production use.
- Real provider refund client and refund reconciliation webhook.
- Atomic refund reservation and cumulative amount validation.
- Recovery/reconciliation for refunds left in `processing` after a process or network failure.
- Production integration tests for refund recovery, ambiguous equal-amount refunds, and provider failures.
- Integration tests with Mercado Pago test accounts for split, partial/full refund, insufficient balance, and duplicate notifications.

## Current continuation point

- Branch: `feature/mercado-pago-split-refunds`
- Last completed commit: `e801d1b feat: reconcile pending Mercado Pago refunds`.
- Documentation commit: `336785d docs: add manual payment testing guide`.
- Next step: add application authentication/authorization around the OAuth start route.

Payment creation now sends `application_fee` only when `seller_id` and the matching encrypted seller token are available. A global token is never used for a seller split.

Orders API credential rule: test Public Key and test Access Token are separate from `MERCADO_PAGO_OAUTH_CLIENT_ID` and `MERCADO_PAGO_OAUTH_CLIENT_SECRET`. The OAuth pair belongs to the Marketplace/Split application, not to the marketplace User ID. If the panel does not show a Client Secret, the application was created as a regular Orders integration instead of through the Split 1:1 Marketplace/OAuth flow.

## Manual OAuth test

1. Set all optional OAuth variables from `payment-request-api/.env.example` in the local `.env`.
2. Generate a 32-byte base64 key with PowerShell: `([Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)))`.
3. Register the exact redirect URI in Mercado Pago: `http://localhost:8080/oauth/mercadopago/callback`.
4. Start `payment-request-api` and open `http://localhost:8080/oauth/mercadopago/start`.
5. Authorize the seller. The callback returns only `seller_id` and `status`; it must never return tokens.
6. Confirm that the configured token file exists and contains encrypted data.

After seller linking, create a payment with `seller_id` and `marketplace_fee_cents` to test Split using Mercado Pago test accounts. The token file must be mounted at the same path in both Docker services.

The current implementation supports one encrypted seller-token file for the manual test environment. A production marketplace must replace this with durable per-seller storage and token renewal before supporting multiple concurrent sellers.

## Manual refund test

Send `POST /payment/refund` with `payment_id`, `amount_cents`, `idempotency_key`, and a business `reason`. The consulting API owns the policy percentage; this payment API receives the already-authorized amount. A duplicate `idempotency_key` does not call the gateway again after local success. Mercado Pago may still require balance and may reject refunds after its provider limits.

Refund recovery matches an approved Mercado Pago refund by payment and amount. If multiple approved refunds have the same amount, the provider result is ambiguous and must be reviewed rather than automatically attributed.