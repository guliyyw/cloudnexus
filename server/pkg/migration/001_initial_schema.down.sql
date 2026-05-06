-- 001: Rollback initial schema.
DROP TABLE IF EXISTS docker_nodes          CASCADE;
DROP TABLE IF EXISTS messages              CASCADE;
DROP TABLE IF EXISTS friends               CASCADE;
DROP TABLE IF EXISTS conversation_members  CASCADE;
DROP TABLE IF EXISTS conversations         CASCADE;
DROP TABLE IF EXISTS file_shares           CASCADE;
DROP TABLE IF EXISTS files                 CASCADE;
DROP TABLE IF EXISTS refresh_tokens        CASCADE;
DROP TABLE IF EXISTS users                 CASCADE;
