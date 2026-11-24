CREATE TABLE IF NOT EXISTS goals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(120) NOT NULL,
    description VARCHAR(1000) NOT NULL DEFAULT '',
    target_amount NUMERIC(20, 2) NOT NULL CHECK (target_amount > 0),
    starting_amount NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (starting_amount >= 0),
    due_date DATE,
    priority VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    status VARCHAR(12) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed')),
    image_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT goals_title_not_empty CHECK (length(trim(title)) > 0)
);

CREATE INDEX IF NOT EXISTS goals_user_status_idx
    ON goals (user_id, status, due_date);

CREATE TABLE IF NOT EXISTS goal_entries (
    id BIGSERIAL PRIMARY KEY,
    goal_id BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_type VARCHAR(12) NOT NULL CHECK (entry_type IN ('contribution', 'withdrawal')),
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    comment VARCHAR(500) NOT NULL DEFAULT '',
    transaction_id BIGINT REFERENCES transactions(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT goal_entries_user_idempotency_unique UNIQUE (user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS goal_entries_goal_created_idx
    ON goal_entries (goal_id, created_at DESC, id DESC);
