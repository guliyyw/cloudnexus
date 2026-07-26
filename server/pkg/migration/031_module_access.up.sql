CREATE TABLE IF NOT EXISTS user_permissions (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    permission_id BIGINT REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by BIGINT,
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, permission_id)
);

INSERT INTO permissions (id, name, code, description, group_name) VALUES
    (310000000000000001, '文件模块', 'module:files', '访问文件工作台', '模块访问'),
    (310000000000000002, '分享模块', 'module:shares', '访问我的分享', '模块访问'),
    (310000000000000003, '回收站模块', 'module:trash', '访问回收站', '模块访问'),
    (310000000000000004, '文档模块', 'module:documents', '访问在线文档', '模块访问'),
    (310000000000000005, '聊天模块', 'module:chat', '访问即时通讯', '模块访问'),
    (310000000000000006, '好友模块', 'module:friends', '访问好友管理', '模块访问'),
    (310000000000000007, 'Docker 模块', 'module:docker', '访问 Docker 管理', '模块访问'),
    (310000000000000008, '摄像头模块', 'module:cameras', '访问摄像头中心', '模块访问'),
    (310000000000000009, '相册模块', 'module:album', '访问相册', '模块访问'),
    (310000000000000010, '音乐模块', 'module:music', '访问音乐中心', '模块访问'),
    (310000000000000011, '短剧模块', 'module:drama', '访问短剧工坊', '模块访问')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code IN ('super_admin', 'admin', 'user') AND p.code LIKE 'module:%'
ON CONFLICT DO NOTHING;
