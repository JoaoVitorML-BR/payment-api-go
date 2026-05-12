CREATE TABLE IF NOT EXISTS payment_requests (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    merchant_reference VARCHAR(100),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency CHAR(3) NOT NULL,
    payment_method VARCHAR(20) NOT NULL DEFAULT 'credit'
    CHECK (payment_method IN ('credit', 'debit', 'pix', 'boleto')),
    installments INT CHECK (installments IS NULL OR installments BETWEEN 1 AND 12),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'processing', 'requires_action', 'succeeded', 'failed', 'canceled')),
    failure_code VARCHAR(50),
    failure_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    stripe_payment_intent_id VARCHAR(255) UNIQUE
);