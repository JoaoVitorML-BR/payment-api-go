-- Add payment_method column to payment_requests table
ALTER TABLE payment_requests
ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20) NOT NULL DEFAULT 'credit'
CHECK (payment_method IN ('credit', 'debit', 'pix', 'boleto'));
