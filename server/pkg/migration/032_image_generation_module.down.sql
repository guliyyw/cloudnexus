DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'module:image_generation');
DELETE FROM user_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'module:image_generation');
DELETE FROM permissions WHERE code = 'module:image_generation';
