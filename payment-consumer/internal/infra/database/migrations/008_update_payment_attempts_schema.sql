-- Update payment_attempts schema after payment_requests UUID migration
-- This migration handles payment_attempts schema updates

BEGIN;

-- Drop old indices if they exist
DROP INDEX IF EXISTS idx_payment_attempts_request_uuid_unique;

-- Ensure payment_request_uuid has proper indexing
CREATE UNIQUE INDEX IF NOT EXISTS uk_payment_attempts_request_uuid_attempt_number
  ON payment_attempts(payment_request_uuid, attempt_number);

COMMIT;
