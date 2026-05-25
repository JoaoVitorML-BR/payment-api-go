-- 005_add_installments.sql
ALTER TABLE payment_requests
ADD COLUMN IF NOT EXISTS installments INT;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'payment_requests_installments_check'
	) THEN
		ALTER TABLE payment_requests
		ADD CONSTRAINT payment_requests_installments_check
		CHECK (installments IS NULL OR installments BETWEEN 1 AND 12);
	END IF;
END $$;