DO $$
BEGIN
    BEGIN ALTER TABLE friends ADD COLUMN message VARCHAR(200) DEFAULT ''; EXCEPTION WHEN duplicate_column THEN NULL; END;
    BEGIN ALTER TABLE friends ADD COLUMN remark VARCHAR(50) DEFAULT ''; EXCEPTION WHEN duplicate_column THEN NULL; END;
    BEGIN ALTER TABLE friends ADD COLUMN expires_at TIMESTAMPTZ; EXCEPTION WHEN duplicate_column THEN NULL; END;
END $$;

CREATE TABLE IF NOT EXISTS blocklists (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    blocked_user_id BIGINT NOT NULL,
    reason VARCHAR(200) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, blocked_user_id)
);
