-- 007_face_thumbnail_text.up.sql
-- 人脸缩略图和快照 URL 从 VARCHAR(512) 改为 TEXT，避免 base64 data URL 被截断

ALTER TABLE face_profiles ALTER COLUMN thumbnail_url TYPE TEXT;
ALTER TABLE face_recognition_events ALTER COLUMN snapshot_url TYPE TEXT;
