CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS customers (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL,
    email        VARCHAR(255) NOT NULL,
    phone        VARCHAR(20)  NOT NULL,
    tax_id       VARCHAR(20)  NOT NULL,
    address      VARCHAR(255) NOT NULL,
    city         VARCHAR(100) NOT NULL,
    state        CHAR(2)      NOT NULL,
    postal_code  VARCHAR(10)  NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS trg_customers_set_updated_at ON customers;

CREATE TRIGGER trg_customers_set_updated_at
BEFORE UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_column();