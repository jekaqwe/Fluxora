CREATE TABLE IF NOT EXISTS family_goals (
    id BIGSERIAL PRIMARY KEY,
    family_id BIGINT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
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
    CONSTRAINT family_goals_title_not_empty CHECK (length(trim(title)) > 0)
);

CREATE INDEX IF NOT EXISTS family_goals_family_status_idx
    ON family_goals (family_id, status, due_date);

CREATE TABLE IF NOT EXISTS family_goal_entries (
    id BIGSERIAL PRIMARY KEY,
    goal_id BIGINT NOT NULL REFERENCES family_goals(id) ON DELETE CASCADE,
    family_id BIGINT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    author_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    comment VARCHAR(500) NOT NULL DEFAULT '',
    transaction_id BIGINT REFERENCES transactions(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT family_goal_entry_idempotency_unique UNIQUE (family_id, author_user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS family_goal_entries_goal_created_idx
    ON family_goal_entries (goal_id, created_at DESC, id DESC);
