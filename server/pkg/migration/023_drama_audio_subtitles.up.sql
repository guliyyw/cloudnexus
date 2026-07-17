ALTER TABLE drama_storyboards ADD COLUMN IF NOT EXISTS audio_duration_ms INTEGER DEFAULT 0;
ALTER TABLE drama_storyboards ADD COLUMN IF NOT EXISTS subtitle_ass TEXT;
