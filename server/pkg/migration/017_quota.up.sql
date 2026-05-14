CREATE TABLE IF NOT EXISTS quota_tiers (
    id            BIGINT PRIMARY KEY,
    name          VARCHAR(64) NOT NULL,
    storage_limit  BIGINT NOT NULL,
    description   VARCHAR(256) DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_quota_tiers_name ON quota_tiers(name);

CREATE TABLE IF NOT EXISTS user_quota (
    user_id       BIGINT PRIMARY KEY,
    storage_used  BIGINT NOT NULL DEFAULT 0,
    storage_limit BIGINT,
    tier_id       BIGINT DEFAULT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO quota_tiers (id, name, storage_limit, description)
VALUES (1, 'free', 1073741824, '免费版 1GB存储')
ON CONFLICT DO NOTHING;

INSERT INTO quota_tiers (id, name, storage_limit, description)
VALUES (2, 'premium', 10737418240, '高级版 10GB存储')
ON CONFLICT DO NOTHING;
