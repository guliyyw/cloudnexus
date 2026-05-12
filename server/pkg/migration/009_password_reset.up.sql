CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- For login failure lockout (A6)
DO $$
BEGIN
    BEGIN ALTER TABLE users ADD COLUMN locked_until TIMESTAMPTZ; EXCEPTION WHEN duplicate_column THEN NULL; END;
    BEGIN ALTER TABLE users ADD COLUMN login_fail_count INT DEFAULT 0; EXCEPTION WHEN duplicate_column THEN NULL; END;
END $$;
