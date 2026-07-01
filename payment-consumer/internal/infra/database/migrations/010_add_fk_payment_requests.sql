-- Add Foreign Key Constraint after payment-request-api schema is ready
-- This ensures payment_requests table has UUID primary key before we add the FK

BEGIN;

-- Verify payment_requests table exists and has uuid column
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'payment_requests'
    ) THEN
        RAISE EXCEPTION 'payment_requests table does not exist yet. payment-request-api migrations must complete first.';
    END IF;
END $$;

-- Add Foreign Key Constraint
DO $$
BEGIN
        IF NOT EXISTS (
                SELECT 1
                FROM information_schema.table_constraints
                WHERE table_name = 'payment_attempts'
                    AND constraint_name = 'fk_payment_attempts_payment_requests'
        ) THEN
                ALTER TABLE payment_attempts
                ADD CONSTRAINT fk_payment_attempts_payment_requests
                    FOREIGN KEY (payment_request_uuid)
                    REFERENCES payment_requests(uuid)
                    ON DELETE RESTRICT
                    ON UPDATE CASCADE;
        END IF;
END $$;

COMMIT;
