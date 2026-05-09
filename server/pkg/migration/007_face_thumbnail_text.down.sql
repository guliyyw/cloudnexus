-- 007_face_thumbnail_text.down.sql
ALTER TABLE face_profiles ALTER COLUMN thumbnail_url TYPE VARCHAR(512);
ALTER TABLE face_recognition_events ALTER COLUMN snapshot_url TYPE VARCHAR(512);
