CREATE TABLE IF NOT EXISTS drama_projects (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    owner_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    description VARCHAR(1000),
    preface TEXT,
    raw_script TEXT,
    settings JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_drama_projects_owner_id ON drama_projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_drama_projects_updated_at ON drama_projects(updated_at);

CREATE TABLE IF NOT EXISTS drama_storyboards (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    project_id BIGINT NOT NULL,
    owner_id BIGINT NOT NULL,
    seq INTEGER NOT NULL,
    title VARCHAR(200),
    content TEXT,
    original TEXT,
    prompt TEXT,
    dialogue TEXT,
    scene_anchor TEXT,
    plot TEXT,
    modified BOOLEAN DEFAULT FALSE,
    image_file_id BIGINT DEFAULT 0,
    audio_file_id BIGINT DEFAULT 0,
    video_file_id BIGINT DEFAULT 0,
    UNIQUE(project_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_drama_storyboards_project_id ON drama_storyboards(project_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboards_owner_id ON drama_storyboards(owner_id);
CREATE INDEX IF NOT EXISTS idx_drama_storyboards_modified ON drama_storyboards(modified);

CREATE TABLE IF NOT EXISTS drama_assets (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    project_id BIGINT NOT NULL,
    owner_id BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL,
    name VARCHAR(120) NOT NULL,
    description TEXT,
    reference_file_id BIGINT DEFAULT 0,
    UNIQUE(project_id, type, name)
);

CREATE INDEX IF NOT EXISTS idx_drama_assets_project_id ON drama_assets(project_id);
CREATE INDEX IF NOT EXISTS idx_drama_assets_owner_id ON drama_assets(owner_id);
CREATE INDEX IF NOT EXISTS idx_drama_assets_type ON drama_assets(type);

CREATE TABLE IF NOT EXISTS drama_tasks (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    project_id BIGINT NOT NULL,
    owner_id BIGINT NOT NULL,
    type VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    progress INTEGER DEFAULT 0,
    message VARCHAR(1000),
    storyboard_id BIGINT DEFAULT 0,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_drama_tasks_project_id ON drama_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_drama_tasks_owner_id ON drama_tasks(owner_id);
CREATE INDEX IF NOT EXISTS idx_drama_tasks_type ON drama_tasks(type);
CREATE INDEX IF NOT EXISTS idx_drama_tasks_status ON drama_tasks(status);

CREATE TABLE IF NOT EXISTS drama_settings (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    owner_id BIGINT NOT NULL UNIQUE,
    comfy_ui_url VARCHAR(500),
    tts_engine VARCHAR(50) DEFAULT 'edge-tts',
    tts_config JSONB DEFAULT '{}'::jsonb,
    video_settings JSONB DEFAULT '{}'::jsonb,
    storage_root VARCHAR(200) DEFAULT '短剧工坊'
);

INSERT INTO permissions (id, name, code, description, group_name, created_at)
SELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT + v.n, v.name, v.code, v.description, '短剧', NOW()
FROM (VALUES
    (1, '短剧查看', 'drama:read', '查看短剧项目、分镜、资产和任务'),
    (2, '短剧编辑', 'drama:write', '创建和编辑短剧项目、分镜与资产'),
    (3, '短剧生成', 'drama:generate', '执行 TTS、图片和视频生成任务'),
    (4, '短剧管理', 'drama:admin', '管理短剧工坊全局与个人设置')
) AS v(n, name, code, description)
WHERE NOT EXISTS (SELECT 1 FROM permissions p WHERE p.code = v.code);

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('drama:read', 'drama:write', 'drama:generate', 'drama:admin')
WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('drama:read', 'drama:write', 'drama:generate', 'drama:admin')
WHERE r.code = 'admin'
ON CONFLICT DO NOTHING;
