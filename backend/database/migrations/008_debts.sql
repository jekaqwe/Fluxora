CREATE TABLE IF NOT EXISTS debts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    direction VARCHAR(12) NOT NULL CHECK (direction IN ('i_owe', 'owed_to_me')),
    initial_amount NUMERIC(20, 2) NOT NULL CHECK (initial_amount > 0),
    due_date DATE,
    comment VARCHAR(1000) NOT NULL DEFAULT '',
    status VARCHAR(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT debts_name_not_empty CHECK (length(trim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS debts_user_status_due_idx
    ON debts (user_id, status, due_date);

CREATE TABLE IF NOT EXISTS debt_payments (
    id BIGSERIAL PRIMARY KEY,
    debt_id BIGINT NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    transaction_id BIGINT REFERENCES transactions(id) ON DELETE SET NULL,
    payment_date DATE NOT NULL,
    comment VARCHAR(500) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT debt_payments_user_idempotency_unique UNIQUE (user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS debt_payments_debt_date_idx
    ON debt_payments (debt_id, payment_date DESC, id DESC);
