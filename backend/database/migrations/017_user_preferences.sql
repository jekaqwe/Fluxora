CREATE TABLE IF NOT EXISTS user_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{"theme":"light","notifications":{"operations":true,"goals":true,"family":true},"privacy":{"profile":true,"achievements":true,"activity":true}}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
