ALTER TABLE camera_recordings
    ADD COLUMN IF NOT EXISTS file_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_camera_recordings_file_id
    ON camera_recordings (file_id)
    WHERE file_id <> 0;
