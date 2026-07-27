CREATE OR REPLACE FUNCTION cleanup_camera_recording_file_link()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM camera_recordings WHERE file_id = OLD.id;
        RETURN OLD;
    END IF;
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        DELETE FROM camera_recordings WHERE file_id = OLD.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_cleanup_camera_recording_on_file_delete ON files;
CREATE TRIGGER trg_cleanup_camera_recording_on_file_delete
AFTER DELETE ON files
FOR EACH ROW
EXECUTE FUNCTION cleanup_camera_recording_file_link();

DROP TRIGGER IF EXISTS trg_cleanup_camera_recording_on_file_soft_delete ON files;
CREATE TRIGGER trg_cleanup_camera_recording_on_file_soft_delete
AFTER UPDATE OF deleted_at ON files
FOR EACH ROW
EXECUTE FUNCTION cleanup_camera_recording_file_link();

DELETE FROM camera_recordings
WHERE file_id <> 0
  AND NOT EXISTS (
      SELECT 1
      FROM files
      WHERE files.id = camera_recordings.file_id
        AND files.deleted_at IS NULL
  );
