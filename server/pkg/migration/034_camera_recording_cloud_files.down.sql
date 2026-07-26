DROP INDEX IF EXISTS idx_camera_recordings_file_id;

ALTER TABLE camera_recordings
    DROP COLUMN IF EXISTS file_id;
