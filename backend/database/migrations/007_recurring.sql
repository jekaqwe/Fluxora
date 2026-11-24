CREATE TABLE IF NOT EXISTS recurring_operations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    title VARCHAR(160) NOT NULL,
    operation_type VARCHAR(8) NOT NULL CHECK (operation_type IN ('income', 'expense')),
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    category VARCHAR(80) NOT NULL DEFAULT 'Без категории',
    frequency VARCHAR(10) NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly', 'yearly')),
    next_date DATE NOT NULL,
    end_date DATE,
    status VARCHAR(12) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT recurring_title_not_empty CHECK (length(trim(title)) > 0)
);

CREATE INDEX IF NOT EXISTS recurring_operations_due_idx
    ON recurring_operations (next_date, id)
    WHERE status = 'active';

CREATE TABLE IF NOT EXISTS recurring_occurrences (
    id BIGSERIAL PRIMARY KEY,
    recurring_id BIGINT NOT NULL REFERENCES recurring_operations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    occurrence_date DATE NOT NULL,
    status VARCHAR(12) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'skipped')),
    transaction_id BIGINT REFERENCES transactions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    CONSTRAINT recurring_occurrence_unique UNIQUE (recurring_id, occurrence_date)
);

CREATE INDEX IF NOT EXISTS recurring_occurrences_user_pending_idx
    ON recurring_occurrences (user_id, occurrence_date)
    WHERE status = 'pending';
