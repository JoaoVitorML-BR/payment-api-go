BEGIN;

ALTER TABLE payment_requests
    ADD COLUMN IF NOT EXISTS seller_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS marketplace_fee_cents BIGINT,
    ADD COLUMN IF NOT EXISTS gateway_fee_cents BIGINT,
    ADD COLUMN IF NOT EXISTS seller_amount_cents BIGINT;

ALTER TABLE payment_requests
    DROP CONSTRAINT IF EXISTS payment_requests_status_check;

ALTER TABLE payment_requests
    ADD CONSTRAINT payment_requests_status_check
    CHECK (status IN (
        'pending', 'processing', 'requires_action', 'requires_payment_method',
        'requires_capture', 'succeeded', 'failed', 'canceled',
        'partially_refunded', 'refunded'
    ));

CREATE TABLE IF NOT EXISTS payment_refunds (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_request_uuid UUID NOT NULL REFERENCES payment_requests(uuid) ON DELETE RESTRICT,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    reason VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'requested'
        CHECK (status IN ('requested', 'processing', 'succeeded', 'failed')),
    gateway_refund_id VARCHAR(255),
    error_code VARCHAR(100),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (payment_request_uuid, idempotency_key)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_payment_refunds_gateway_refund_id
    ON payment_refunds(gateway_refund_id)
    WHERE gateway_refund_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payment_refunds_payment_request_uuid
    ON payment_refunds(payment_request_uuid);

COMMIT;