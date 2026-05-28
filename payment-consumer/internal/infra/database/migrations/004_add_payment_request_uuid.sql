-- Add payment_request_uuid to payment_attempts and backfill from payment_requests.uuid
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

ALTER TABLE payment_attempts ADD COLUMN IF NOT EXISTS payment_request_uuid UUID;

UPDATE payment_attempts
SET payment_request_uuid = pr.uuid
FROM payment_requests pr
WHERE pr.uuid::text = payment_attempts.payment_request_id;

CREATE INDEX IF NOT EXISTS idx_payment_attempts_request_uuid ON payment_attempts (payment_request_uuid);
