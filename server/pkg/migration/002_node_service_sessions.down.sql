DROP TABLE IF EXISTS node_online_sessions;

ALTER TABLE docker_nodes DROP COLUMN IF EXISTS container_name;
ALTER TABLE docker_nodes DROP COLUMN IF EXISTS version;
ALTER TABLE docker_nodes DROP COLUMN IF EXISTS service;
ALTER TABLE docker_nodes DROP COLUMN IF EXISTS first_seen_at;
ALTER TABLE docker_nodes DROP COLUMN IF EXISTS total_online_seconds;
ALTER TABLE docker_nodes DROP COLUMN IF EXISTS offline_since;
