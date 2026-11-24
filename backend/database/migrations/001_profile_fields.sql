ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_key TEXT,
    ADD COLUMN IF NOT EXISTS avatar_content_type VARCHAR(100),
    ADD COLUMN IF NOT EXISTS first_name_changed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_name_changed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS username_changed_at TIMESTAMPTZ;
