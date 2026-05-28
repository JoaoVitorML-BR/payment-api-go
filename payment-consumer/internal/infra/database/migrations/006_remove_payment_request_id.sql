-- Remove payment_request_id column - no longer needed since payment_request_uuid is the FK
-- This simplifies the schema and removes redundancy

-- Drop indices that used payment_request_id
DROP INDEX IF EXISTS payment_attempts_payment_request_id_idx;
DROP INDEX IF EXISTS payment_attempts_payment_request_id_attempt_number_idx;

-- Drop the old column
ALTER TABLE payment_attempts 
DROP COLUMN payment_request_id;
