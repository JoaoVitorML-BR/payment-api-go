-- Add payment_request_uuid to payment_attempts and backfill from payment_requests.uuid
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

ALTER TABLE payment_attempts ADD COLUMN IF NOT EXISTS payment_request_uuid UUID;

DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM information_schema.columns
		WHERE table_name = 'payment_attempts'
		  AND column_name = 'payment_request_id'
	) THEN
		UPDATE payment_attempts
		SET payment_request_uuid = pr.uuid
		FROM payment_requests pr
		WHERE pr.uuid::text = payment_attempts.payment_request_id;
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_payment_attempts_request_uuid ON payment_attempts (payment_request_uuid);
