-- 002: Add service tracking and online session history for docker_nodes

ALTER TABLE docker_nodes ADD COLUMN IF NOT EXISTS service VARCHAR(32) DEFAULT '';
ALTER TABLE docker_nodes ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMPTZ;
ALTER TABLE docker_nodes ADD COLUMN IF NOT EXISTS total_online_seconds BIGINT DEFAULT 0;
ALTER TABLE docker_nodes ADD COLUMN IF NOT EXISTS offline_since TIMESTAMPTZ;
ALTER TABLE docker_nodes ADD COLUMN IF NOT EXISTS container_name VARCHAR(128) DEFAULT '';
ALTER TABLE docker_nodes ADD COLUMN IF NOT EXISTS version VARCHAR(32) DEFAULT '';

CREATE TABLE IF NOT EXISTS node_online_sessions (
    id             BIGINT PRIMARY KEY,
    node_name      VARCHAR(64)  NOT NULL,
    start_time     TIMESTAMPTZ  NOT NULL,
    end_time       TIMESTAMPTZ,
    duration       BIGINT       DEFAULT 0,
    container_name VARCHAR(128) DEFAULT '',
    version        VARCHAR(32)  DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_node_sessions_name ON node_online_sessions(node_name);
