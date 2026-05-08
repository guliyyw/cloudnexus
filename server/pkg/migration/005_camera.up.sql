CREATE TABLE IF NOT EXISTS cameras (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    owner_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    stream_url VARCHAR(512) NOT NULL,
    protocol VARCHAR(16) NOT NULL DEFAULT 'rtsp',
    status VARCHAR(16) NOT NULL DEFAULT 'offline',
    last_seen_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_cameras_owner ON cameras(owner_id);

CREATE TABLE IF NOT EXISTS recognition_events (
    id BIGINT PRIMARY KEY,
    camera_id BIGINT NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    snapshot_url VARCHAR(512) NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recognition_events_camera ON recognition_events(camera_id);
CREATE INDEX IF NOT EXISTS idx_recognition_events_created ON recognition_events(created_at);
