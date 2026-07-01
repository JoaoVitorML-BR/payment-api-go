-- Migrate from BIGSERIAL id to UUID as primary key
-- This simplifies the schema and aligns with microservices pattern

BEGIN;

-- Step 1: Add UUID as UNIQUE constraint (if not already exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'payment_requests' 
        AND constraint_name = 'uk_payment_requests_uuid'
    ) THEN
        ALTER TABLE payment_requests 
        ADD CONSTRAINT uk_payment_requests_uuid UNIQUE (uuid);
    END IF;
END $$;

-- Step 2: Make uuid the primary key by dropping old PK and adding new one
ALTER TABLE payment_requests 
DROP CONSTRAINT IF EXISTS payment_requests_pkey CASCADE;

ALTER TABLE payment_requests 
ADD PRIMARY KEY (uuid);

-- Step 3: Drop the old id column (it's no longer needed)
ALTER TABLE payment_requests 
DROP COLUMN IF EXISTS id;

COMMIT;

-- Note: payment_attempts FK management is handled by payment-consumer service
