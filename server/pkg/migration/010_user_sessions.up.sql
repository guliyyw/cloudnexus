DO $$
BEGIN
    BEGIN ALTER TABLE users ADD COLUMN force_logout_after TIMESTAMPTZ; EXCEPTION WHEN duplicate_column THEN NULL; END;
END $$;

CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    jti VARCHAR(36) UNIQUE NOT NULL,
    user_agent VARCHAR(500) DEFAULT '',
    ip_address VARCHAR(45) DEFAULT '',
    login_at TIMESTAMPTZ DEFAULT NOW(),
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_jti ON user_sessions(jti);
