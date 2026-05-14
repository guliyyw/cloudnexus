CREATE TABLE IF NOT EXISTS system_configs (
    key         VARCHAR(128) PRIMARY KEY,
    value       TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 默认上传配置
INSERT INTO system_configs (key, value) VALUES
    ('upload.sequential_mode', 'false'),
    ('upload.max_concurrent_chunks', '3')
ON CONFLICT (key) DO NOTHING;
