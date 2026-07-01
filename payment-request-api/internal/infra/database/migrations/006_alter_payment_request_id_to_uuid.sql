-- Ensure extension for gen_random_uuid is available
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Add uuid column only if missing, and keep default generator
ALTER TABLE payment_requests
ADD COLUMN IF NOT EXISTS uuid UUID DEFAULT gen_random_uuid();

-- Backfill missing values only
UPDATE payment_requests
SET uuid = gen_random_uuid()
WHERE uuid IS NULL;