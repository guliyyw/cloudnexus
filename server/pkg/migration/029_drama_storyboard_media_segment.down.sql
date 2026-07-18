DROP INDEX IF EXISTS idx_drama_storyboard_media_segment_id;
ALTER TABLE drama_storyboard_media DROP COLUMN IF EXISTS segment_id;
