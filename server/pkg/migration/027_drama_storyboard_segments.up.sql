CREATE TABLE IF NOT EXISTS drama_storyboard_segments (
  id BIGINT PRIMARY KEY,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ,
  project_id BIGINT NOT NULL,
  storyboard_id BIGINT NOT NULL,
  owner_id BIGINT NOT NULL,
  seq INTEGER NOT NULL,
  title VARCHAR(200),
  duration_sec INTEGER DEFAULT 3,
  purpose TEXT,
  characters TEXT,
  scene VARCHAR(200),
  dialogue TEXT,
  action TEXT,
  shot TEXT,
  reference_prompt TEXT,
  video_prompt TEXT,
  negative_prompt TEXT,
  reference_file_id BIGINT DEFAULT 0,
  video_file_id BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_drama_storyboard_segments_project_id ON drama_storyboard_segments(project_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_segments_storyboard_id ON drama_storyboard_segments(storyboard_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_segments_owner_id ON drama_storyboard_segments(owner_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_segments_seq ON drama_storyboard_segments(seq);
