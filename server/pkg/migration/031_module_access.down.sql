DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code LIKE 'module:%');
DROP TABLE IF EXISTS user_permissions;
DELETE FROM permissions WHERE code LIKE 'module:%';
