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
    stripe_payment_intent_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
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
SELECT stripe_client_secret
FROM payment_attempts
WHERE payment_request_uuid::text = $1
ORDER BY attempt_number DESC
LIMIT 1
;