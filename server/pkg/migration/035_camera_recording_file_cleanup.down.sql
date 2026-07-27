DROP TRIGGER IF EXISTS trg_cleanup_camera_recording_on_file_soft_delete ON files;
DROP TRIGGER IF EXISTS trg_cleanup_camera_recording_on_file_delete ON files;
DROP FUNCTION IF EXISTS cleanup_camera_recording_file_link();
