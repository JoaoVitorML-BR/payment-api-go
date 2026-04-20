CREATE INDEX IF NOT EXISTS idx_payment_requests_created_at
    ON payment_requests (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_requests_status
    ON payment_requests (status);