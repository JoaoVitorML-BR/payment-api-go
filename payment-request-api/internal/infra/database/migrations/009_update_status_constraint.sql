-- Update status constraint to include all Stripe payment intent statuses
ALTER TABLE payment_requests
    DROP CONSTRAINT payment_requests_status_check,
    ADD CONSTRAINT payment_requests_status_check 
    CHECK (status IN ('pending', 'processing', 'requires_action', 'requires_payment_method', 'requires_capture', 'succeeded', 'failed', 'canceled'));
