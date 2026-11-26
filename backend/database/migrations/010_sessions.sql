CREATE INDEX IF NOT EXISTS sessions_user_created_idx
    ON sessions (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx
    ON sessions (expires_at);
