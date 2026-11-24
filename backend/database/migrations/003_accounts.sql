CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    account_type VARCHAR(20) NOT NULL CHECK (account_type IN ('cash', 'bank', 'card', 'savings', 'other')),
    initial_balance NUMERIC(20, 2) NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'RUB' CHECK (currency = 'RUB'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT accounts_name_not_empty CHECK (length(trim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS accounts_user_id_created_at_idx
    ON accounts (user_id, created_at, id);
