-- name: UpdatePaymentAttempt :exec
INSERT INTO payment_attempts (payment_request_uuid, stripe_payment_intent_id, stripe_client_secret, attempt_number, currency, status, error_code, error_message, response, processed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (payment_request_uuid, attempt_number) DO UPDATE SET
  stripe_payment_intent_id = EXCLUDED.stripe_payment_intent_id,
  stripe_client_secret = EXCLUDED.stripe_client_secret,
  currency = EXCLUDED.currency,
  status = EXCLUDED.status,
  error_code = EXCLUDED.error_code,
  error_message = EXCLUDED.error_message,
  response = EXCLUDED.response,
  processed_at = NOW();

-- name: UpdatePaymentRequestSuccess :exec
UPDATE payment_requests
SET stripe_payment_intent_id = $2,
    status = $3,
    updated_at = NOW()
WHERE uuid::text = $1;

-- name: UpdatePaymentRequestError :exec
UPDATE payment_requests
SET status = $2,
    failure_code = $3,
    failure_message = $4,
    updated_at = NOW()
WHERE uuid::text = $1;

-- name: GetLatestPaymentAttempt :one
SELECT payment_request_uuid, stripe_payment_intent_id, stripe_client_secret, attempt_number, currency, status, error_code, error_message, response, created_at, processed_at
FROM payment_attempts
WHERE payment_request_uuid = $1
ORDER BY attempt_number DESC
LIMIT 1;