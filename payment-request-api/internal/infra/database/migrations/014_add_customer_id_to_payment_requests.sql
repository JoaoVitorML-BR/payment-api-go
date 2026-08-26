ALTER TABLE payment_requests
ADD COLUMN IF NOT EXISTS customer_uuid UUID NULL
    REFERENCES customers(uuid);

CREATE INDEX IF NOT EXISTS idx_payment_requests_customer_uuid
    ON payment_requests (customer_uuid);