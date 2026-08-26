-- name: CreatePaymentRequest :one
INSERT INTO payment_requests (
    idempotency_key,
    merchant_reference,
    amount_cents,
    currency,
    payment_method,
    status,
    failure_code,
    failure_message,
    stripe_payment_intent_id,
    gateway,
    gateway_payment_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING uuid::text AS uuid, payment_method, status, created_at, updated_at;

-- name: GetPaymentRequestByIdempotencyKey :one
SELECT uuid::text AS uuid, payment_method, status, created_at, updated_at
FROM payment_requests
WHERE idempotency_key = $1
;

-- name: GetPaymentRequestStripePaymentIntentID :one
SELECT stripe_payment_intent_id
FROM payment_requests
WHERE uuid::text = $1
;

-- name: GetPaymentClientSecret :one
SELECT status, stripe_client_secret, gateway, gateway_payment_id, pix_qr_code, pix_qr_code_base64, pix_expiration_at
FROM payment_attempts
WHERE payment_request_uuid::text = $1
ORDER BY attempt_number DESC
LIMIT 1
;

-- name: UpdatePaymentStatus :exec
UPDATE payment_requests
SET status = @status, amount_cents = @amount_cents
WHERE uuid = @uuid::uuid;
