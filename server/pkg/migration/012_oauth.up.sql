CREATE TABLE IF NOT EXISTS oauth_bindings (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    provider VARCHAR(20) NOT NULL,
    open_id VARCHAR(255) NOT NULL,
    access_token TEXT DEFAULT '',
    refresh_token TEXT DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, open_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_bindings_user_id ON oauth_bindings(user_id);
