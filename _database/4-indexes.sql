-- Indexes for common lookup and history queries.
-- Unique constraints already index account_number, iban, transaction code,
-- customer identification_number, email, and phone.

CREATE INDEX IF NOT EXISTS idx_accounts_customer_id
    ON accounts (customer_id);

CREATE INDEX IF NOT EXISTS idx_transactions_source_id
    ON transactions (source_id);

CREATE INDEX IF NOT EXISTS idx_transactions_destination_id
    ON transactions (destination_id);

-- Transaction history is listed newest first.
CREATE INDEX IF NOT EXISTS idx_transactions_created_at
    ON transactions (created_at DESC, transaction_id DESC);
