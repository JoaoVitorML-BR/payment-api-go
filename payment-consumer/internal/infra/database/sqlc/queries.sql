-- name: UpdatePaymentAttempt :exec
UPDATE payment_attempts
SET stripe_payment_intent_id = $2,
    attempt_number = $3,
    currency = $4,
    status = $5,
    error_code = $6,
    error_message = $7,
    response = $8,
    processed_at = NOW()
WHERE payment_request_id = $1;