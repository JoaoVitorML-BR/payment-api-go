BEGIN;

ALTER TABLE payment_requests
    ADD COLUMN IF NOT EXISTS gateway VARCHAR(30) NOT NULL DEFAULT 'mercado_pago',
    ADD COLUMN IF NOT EXISTS gateway_payment_id VARCHAR(255);

UPDATE payment_requests
SET gateway_payment_id = stripe_payment_intent_id
WHERE gateway_payment_id IS NULL AND stripe_payment_intent_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_payment_requests_gateway_payment_id
    ON payment_requests(gateway_payment_id)
    WHERE gateway_payment_id IS NOT NULL;

COMMIT;
