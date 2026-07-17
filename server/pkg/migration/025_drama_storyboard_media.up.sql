CREATE TABLE IF NOT EXISTS drama_storyboard_media (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    project_id BIGINT NOT NULL,
    storyboard_id BIGINT NOT NULL,
    owner_id BIGINT NOT NULL,
    kind VARCHAR(20) NOT NULL,
    file_id BIGINT NOT NULL DEFAULT 0,
    source VARCHAR(40) DEFAULT 'generated',
    prompt TEXT,
    sort_order INTEGER DEFAULT 0,
    selected BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_project_id ON drama_storyboard_media(project_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_storyboard_id ON drama_storyboard_media(storyboard_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_owner_id ON drama_storyboard_media(owner_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_kind ON drama_storyboard_media(kind);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_sort_order ON drama_storyboard_media(sort_order);
CREATE INDEX IF NOT EXISTS idx_drama_storyboard_media_selected ON drama_storyboard_media(selected);
