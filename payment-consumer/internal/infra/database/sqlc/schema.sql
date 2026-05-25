CREATE TABLE IF NOT EXISTS payment_attempts (
    id SERIAL PRIMARY KEY,
    payment_request_id VARCHAR(255) NOT NULL,
    stripe_payment_intent_id VARCHAR(255),
    attempt_number INTEGER NOT NULL DEFAULT 1,
    currency CHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_code VARCHAR(255),
    error_message TEXT,
    response JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX ON payment_attempts (payment_request_id);
CREATE UNIQUE INDEX ON payment_attempts (payment_request_id, attempt_number);