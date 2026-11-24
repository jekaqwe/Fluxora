CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    transfer_account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    title VARCHAR(160) NOT NULL,
    operation_type VARCHAR(12) NOT NULL CHECK (operation_type IN ('income', 'expense', 'transfer')),
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    category VARCHAR(80) NOT NULL DEFAULT 'Без категории',
    operation_date DATE NOT NULL,
    comment VARCHAR(1000) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_transfer_shape CHECK (
        (operation_type = 'transfer' AND transfer_account_id IS NOT NULL AND transfer_account_id <> account_id)
        OR
        (operation_type IN ('income', 'expense') AND transfer_account_id IS NULL)
    ),
    CONSTRAINT transactions_title_not_empty CHECK (length(trim(title)) > 0),
    CONSTRAINT transactions_category_not_empty CHECK (length(trim(category)) > 0),
    CONSTRAINT transactions_transfer_accounts_distinct CHECK (transfer_account_id IS NULL OR transfer_account_id <> account_id),
    CONSTRAINT transactions_user_idempotency_unique UNIQUE (user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS transactions_user_date_idx
    ON transactions (user_id, operation_date DESC, id DESC);

CREATE INDEX IF NOT EXISTS transactions_account_date_idx
    ON transactions (account_id, operation_date DESC, id DESC);

CREATE INDEX IF NOT EXISTS transactions_transfer_account_idx
    ON transactions (transfer_account_id, operation_date DESC)
    WHERE transfer_account_id IS NOT NULL;
