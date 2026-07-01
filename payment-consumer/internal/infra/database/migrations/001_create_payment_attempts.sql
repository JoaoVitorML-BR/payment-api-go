CREATE TABLE IF NOT EXISTS payment_attempts (
    id SERIAL PRIMARY KEY,
    payment_request_uuid UUID NOT NULL,
    stripe_payment_intent_id VARCHAR(255),
    stripe_client_secret VARCHAR(255),
    attempt_number INTEGER NOT NULL DEFAULT 1,
    currency CHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_code VARCHAR(255),
    error_message TEXT,
    response JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_payment_attempts_request_uuid ON payment_attempts (payment_request_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_attempts_request_uuid_attempt_number ON payment_attempts (payment_request_uuid, attempt_number);
