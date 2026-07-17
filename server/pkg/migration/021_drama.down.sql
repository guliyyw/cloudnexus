DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code LIKE 'drama:%');

DELETE FROM permissions WHERE code LIKE 'drama:%';

DROP TABLE IF EXISTS drama_settings;
DROP TABLE IF EXISTS drama_tasks;
DROP TABLE IF EXISTS drama_assets;
DROP TABLE IF EXISTS drama_storyboards;
DROP TABLE IF EXISTS drama_projects;
