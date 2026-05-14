CREATE TABLE IF NOT EXISTS chunk_uploads (
    id            BIGINT PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    upload_id     VARCHAR(36) NOT NULL,
    file_name     VARCHAR(255) NOT NULL,
    file_size     BIGINT NOT NULL,
    chunk_size    INT NOT NULL DEFAULT 10485760,
    mime_type     VARCHAR(128) DEFAULT '',
    parent_id     BIGINT DEFAULT 0,
    total_chunks  INT NOT NULL,
    completed     INT[] DEFAULT '{}',
    status        VARCHAR(20) NOT NULL DEFAULT 'uploading',
    version_message VARCHAR(256) DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chunk_uploads_upload_id ON chunk_uploads(upload_id);
CREATE        INDEX IF NOT EXISTS idx_chunk_uploads_user_id   ON chunk_uploads(user_id);
CREATE        INDEX IF NOT EXISTS idx_chunk_uploads_status    ON chunk_uploads(status);
