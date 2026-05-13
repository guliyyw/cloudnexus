-- 007_face_thumbnail_text.up.sql
-- 人脸缩略图和快照 URL 从 VARCHAR(512) 改为 TEXT，避免 base64 data URL 被截断
-- 注意：face_profiles 和 face_recognition_events 表由 GORM AutoMigrate 创建，
-- 全新部署时迁移先于 AutoMigrate 运行，因此需要 IF EXISTS 保护

ALTER TABLE IF EXISTS face_profiles ALTER COLUMN thumbnail_url TYPE TEXT;
ALTER TABLE IF EXISTS face_recognition_events ALTER COLUMN snapshot_url TYPE TEXT;
