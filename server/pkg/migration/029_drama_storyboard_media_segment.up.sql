ALTER TABLE drama_storyboard_media ADD COLUMN IF NOT EXISTS segment_id BIGINT DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_segment_id ON drama_storyboard_media(segment_id);
