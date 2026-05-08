CREATE TABLE IF NOT EXISTS file_versions (
    id          BIGINT PRIMARY KEY,
    file_id     BIGINT NOT NULL,
    version_num INT NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    size        BIGINT DEFAULT 0,
    sha256      VARCHAR(64) DEFAULT '',
    message     VARCHAR(256) DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_file_versions_file_id ON file_versions(file_id);
CREATE INDEX idx_file_versions_file_version ON file_versions(file_id, version_num DESC);
