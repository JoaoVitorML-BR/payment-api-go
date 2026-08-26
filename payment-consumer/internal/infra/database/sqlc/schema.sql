CREATE TABLE IF NOT EXISTS payment_requests (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    stripe_payment_intent_id VARCHAR(255) UNIQUE,
    gateway VARCHAR(30) NOT NULL DEFAULT 'mercado_pago',
    gateway_payment_id VARCHAR(255) UNIQUE
);

CREATE TABLE IF NOT EXISTS payment_attempts (
    id SERIAL PRIMARY KEY,
    payment_request_uuid UUID NOT NULL,
    stripe_payment_intent_id VARCHAR(255),
    stripe_client_secret VARCHAR(255),
    gateway VARCHAR(30) NOT NULL DEFAULT 'mercado_pago',
    gateway_payment_id VARCHAR(255),
    pix_qr_code TEXT,
    pix_qr_code_base64 TEXT,
    pix_expiration_at TIMESTAMPTZ,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    currency CHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_code VARCHAR(255),
    error_message TEXT,
    response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMPTZ,
    CONSTRAINT fk_payment_attempts_payment_requests 
      FOREIGN KEY (payment_request_uuid) 
      REFERENCES payment_requests(uuid) 
      ON DELETE RESTRICT
      ON UPDATE CASCADE,
    UNIQUE (payment_request_uuid, attempt_number)
);
