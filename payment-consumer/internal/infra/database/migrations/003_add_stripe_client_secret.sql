ALTER TABLE payment_attempts
    ADD COLUMN IF NOT EXISTS stripe_client_secret VARCHAR(255);