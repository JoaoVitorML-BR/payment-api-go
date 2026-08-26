BEGIN;

ALTER TABLE payment_attempts
    ADD COLUMN IF NOT EXISTS gateway VARCHAR(30) NOT NULL DEFAULT 'mercado_pago',
    ADD COLUMN IF NOT EXISTS gateway_payment_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS pix_qr_code TEXT,
    ADD COLUMN IF NOT EXISTS pix_qr_code_base64 TEXT,
    ADD COLUMN IF NOT EXISTS pix_expiration_at TIMESTAMPTZ;

UPDATE payment_attempts
SET gateway_payment_id = stripe_payment_intent_id
WHERE gateway_payment_id IS NULL AND stripe_payment_intent_id IS NOT NULL;

COMMIT;
