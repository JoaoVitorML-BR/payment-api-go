-- name: CreatePaymentRequest :one
INSERT INTO payment_requests (
    idempotency_key,
    merchant_reference,
    amount_cents,
    currency,
    status,
    failure_code,
    failure_message,
    stripe_payment_intent_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, status, created_at, updated_at;
