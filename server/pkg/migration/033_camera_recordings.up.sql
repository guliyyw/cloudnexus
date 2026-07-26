CREATE TABLE IF NOT EXISTS camera_recordings (
    id BIGINT PRIMARY KEY,
    camera_id BIGINT NOT NULL,
    owner_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(1024) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ready',
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_camera_recordings_camera_started
    ON camera_recordings (camera_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_camera_recordings_owner_started
    ON camera_recordings (owner_id, started_at DESC);
