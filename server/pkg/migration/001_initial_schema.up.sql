-- 001: Initial schema for CloudNexus v0.9.x
-- Applies the full database schema matching all GORM models.

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password      VARCHAR(255) NOT NULL,
    avatar        VARCHAR(512) DEFAULT '',
    status        SMALLINT     DEFAULT 1,
    is_admin      BOOLEAN      DEFAULT FALSE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email    ON users(email);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id            BIGINT PRIMARY KEY,
    user_id       BIGINT       NOT NULL,
    token         VARCHAR(512) NOT NULL,
    expires_at    TIMESTAMPTZ  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_token  ON refresh_tokens(token);
CREATE        INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS files (
    id             BIGINT PRIMARY KEY,
    user_id        BIGINT       NOT NULL,
    name           VARCHAR(255) NOT NULL,
    is_dir         BOOLEAN      DEFAULT FALSE,
    parent_id      BIGINT       DEFAULT 0,
    size           BIGINT       DEFAULT 0,
    mime_type      VARCHAR(128) DEFAULT '',
    storage_key    VARCHAR(512) DEFAULT '',
    storage_sha256 VARCHAR(64)  DEFAULT '',
    is_shared      BOOLEAN      DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_files_user_id   ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_parent_id ON files(parent_id);

CREATE TABLE IF NOT EXISTS file_shares (
    id             BIGINT PRIMARY KEY,
    file_id        BIGINT       NOT NULL,
    owner_id       BIGINT       NOT NULL,
    share_code     VARCHAR(32)  NOT NULL,
    password       VARCHAR(255) DEFAULT '',
    expires_at     TIMESTAMPTZ,
    download_limit INT          DEFAULT 0,
    download_count INT          DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_file_shares_share_code ON file_shares(share_code);

CREATE TABLE IF NOT EXISTS conversations (
    id             BIGINT PRIMARY KEY,
    type           VARCHAR(16)  NOT NULL,
    name           VARCHAR(128) DEFAULT '',
    creator_id     BIGINT       NOT NULL,
    last_msg_seq   BIGINT       DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_members (
    id              BIGINT PRIMARY KEY,
    conversation_id BIGINT       NOT NULL,
    user_id         BIGINT       NOT NULL,
    role            VARCHAR(16)  DEFAULT 'member',
    last_read_seq   BIGINT       DEFAULT 0,
    joined_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conv_user       ON conversation_members(conversation_id, user_id);
CREATE        INDEX IF NOT EXISTS idx_convm_deleted_at ON conversation_members(deleted_at);

CREATE TABLE IF NOT EXISTS friends (
    id            BIGINT PRIMARY KEY,
    user_id       BIGINT       NOT NULL,
    friend_id     BIGINT       NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_friend_pair ON friends(user_id, friend_id);

CREATE TABLE IF NOT EXISTS messages (
    id              BIGINT PRIMARY KEY,
    conversation_id BIGINT  NOT NULL,
    sender_id       BIGINT  NOT NULL,
    content         TEXT    NOT NULL,
    msg_type        VARCHAR(16) DEFAULT 'text',
    seq             BIGINT  NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_conv_seq  ON messages(conversation_id, seq);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

CREATE TABLE IF NOT EXISTS docker_nodes (
    id             BIGINT PRIMARY KEY,
    name           VARCHAR(64)  NOT NULL,
    host           VARCHAR(255) NOT NULL,
    port           INT          DEFAULT 2376,
    tls_cert       TEXT         DEFAULT '',
    tls_key        TEXT         DEFAULT '',
    ca_cert        TEXT         DEFAULT '',
    status         VARCHAR(16)  DEFAULT 'offline',
    last_heartbeat TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_docker_nodes_name ON docker_nodes(name);
