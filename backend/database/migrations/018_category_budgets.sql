CREATE TABLE IF NOT EXISTS category_budgets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(80) NOT NULL,
    monthly_limit NUMERIC(20, 2) NOT NULL CHECK (monthly_limit > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT category_budgets_category_not_empty CHECK (length(trim(category)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS category_budgets_user_category_unique
    ON category_budgets (user_id, lower(category));
