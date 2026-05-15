DROP TRIGGER IF EXISTS trg_payment_requests_set_updated_at ON payment_requests;

DROP FUNCTION IF EXISTS set_updated_at_column();

CREATE OR REPLACE FUNCTION set_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW IS DISTINCT FROM OLD THEN
        NEW.updated_at = CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_payment_requests_set_updated_at
BEFORE UPDATE ON payment_requests
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_column();