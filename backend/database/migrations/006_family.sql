CREATE TABLE IF NOT EXISTS families (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    owner_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    initial_balance NUMERIC(20, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT families_name_not_empty CHECK (length(trim(name)) > 0)
);

CREATE TABLE IF NOT EXISTS family_members (
    id BIGSERIAL PRIMARY KEY,
    family_id BIGINT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(10) NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, user_id)
);

CREATE INDEX IF NOT EXISTS family_members_family_idx
    ON family_members (family_id, joined_at, id);

CREATE TABLE IF NOT EXISTS family_invitations (
    id BIGSERIAL PRIMARY KEY,
    family_id BIGINT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    invited_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS family_invitations_family_idx
    ON family_invitations (family_id, created_at DESC);

CREATE TABLE IF NOT EXISTS family_transactions (
    id BIGSERIAL PRIMARY KEY,
    family_id BIGINT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    author_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    operation_type VARCHAR(8) NOT NULL CHECK (operation_type IN ('income', 'expense')),
    title VARCHAR(160) NOT NULL,
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    category VARCHAR(80) NOT NULL DEFAULT 'Без категории',
    operation_date DATE NOT NULL,
    comment VARCHAR(1000) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT family_transactions_user_idempotency_unique UNIQUE (author_user_id, idempotency_key),
    CONSTRAINT family_transactions_title_not_empty CHECK (length(trim(title)) > 0)
);

CREATE INDEX IF NOT EXISTS family_transactions_family_date_idx
    ON family_transactions (family_id, operation_date DESC, id DESC);
