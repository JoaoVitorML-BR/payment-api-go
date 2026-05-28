-- Add Foreign Key Constraint and make payment_request_uuid NOT NULL
-- This ensures referential integrity between payment_attempts and payment_requests

-- Make payment_request_uuid NOT NULL (safe because 004_add_payment_request_uuid.sql backfilled all rows)
ALTER TABLE payment_attempts 
ALTER COLUMN payment_request_uuid SET NOT NULL;

-- Foreign Key will be added after payment-request-api setup completes
-- This prevents circular migration dependencies in the initial setup

-- Add UNIQUE constraint to ensure one primary attempt per request
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_attempts_request_uuid_unique 
  ON payment_attempts(payment_request_uuid);
