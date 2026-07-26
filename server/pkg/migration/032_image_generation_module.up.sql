INSERT INTO permissions (id, name, code, description, group_name) VALUES
    (320000000000000001, '图片生成模块', 'module:image_generation', '使用提示词和参考图片生成图片', '模块访问')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code IN ('super_admin', 'admin', 'user') AND p.code = 'module:image_generation'
ON CONFLICT DO NOTHING;
