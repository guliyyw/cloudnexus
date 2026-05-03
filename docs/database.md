# CloudNexus 数据库设计文档

> 版本：v0.1.0 | 数据库：PostgreSQL 15 | ORM：GORM

## 1. 设计原则

- 使用 `BIGSERIAL` 作为主键，全局唯一自增
- 时间字段统一使用 `TIMESTAMPTZ`
- 软删除使用 `deleted_at` 字段 (GORM 软删除)
- 外键约束在应用层保证，数据库层保持灵活
- 集群模式考虑使用 UUID 主键或 Snowflake 替代 BIGSERIAL

---

## 2. 用户模块

### users — 用户表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | 用户 ID |
| username | VARCHAR(64) | UNIQUE, NOT NULL | 用户名 |
| email | VARCHAR(255) | UNIQUE, NOT NULL | 邮箱 |
| password | VARCHAR(255) | NOT NULL | bcrypt 哈希 |
| avatar | VARCHAR(512) | | 头像 URL |
| status | SMALLINT | DEFAULT 1 | 1=正常, 0=禁用 |
| last_login_at | TIMESTAMPTZ | | 最后登录时间 |
| created_at | TIMESTAMPTZ | NOT NULL | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新时间 |
| deleted_at | TIMESTAMPTZ | | 软删除时间 |

**索引：**
```sql
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

### refresh_tokens — 刷新令牌表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| user_id | BIGINT | NOT NULL | 关联 users.id |
| token | VARCHAR(512) | UNIQUE, NOT NULL | 令牌哈希 |
| expires_at | TIMESTAMPTZ | NOT NULL | 过期时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

---

## 3. 文件模块

### files — 文件/目录表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| user_id | BIGINT | NOT NULL | 所属用户 |
| name | VARCHAR(255) | NOT NULL | 文件名 |
| is_dir | BOOLEAN | DEFAULT false | 是否为目录 |
| parent_id | BIGINT | DEFAULT 0 | 父目录 ID，0=根 |
| size | BIGINT | DEFAULT 0 | 文件大小 (字节) |
| mime_type | VARCHAR(128) | | MIME 类型 |
| storage_key | VARCHAR(512) | | MinIO 对象键 |
| storage_sha256 | VARCHAR(64) | | 文件 SHA256 |
| is_shared | BOOLEAN | DEFAULT false | 是否已分享 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |
| deleted_at | TIMESTAMPTZ | | |

**索引：**
```sql
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_parent_id ON files(parent_id);
CREATE INDEX idx_files_user_parent ON files(user_id, parent_id);
CREATE INDEX idx_files_storage_key ON files(storage_key);
```

### file_shares — 文件分享表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| file_id | BIGINT | NOT NULL | 关联 files.id |
| owner_id | BIGINT | NOT NULL | 分享发起者 |
| share_code | VARCHAR(32) | UNIQUE, NOT NULL | 分享码 |
| password | VARCHAR(255) | | 访问密码 (bcrypt) |
| expires_at | TIMESTAMPTZ | | 过期时间 |
| download_limit | INT | | 下载次数限制 |
| download_count | INT | DEFAULT 0 | 已下载次数 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_file_shares_share_code ON file_shares(share_code);
```

---

## 4. 即时通讯模块

### conversations — 会话表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| type | VARCHAR(16) | NOT NULL | private / group |
| name | VARCHAR(128) | | 群聊名称，私聊可为空 |
| creator_id | BIGINT | NOT NULL | 创建者 |
| last_msg_seq | BIGINT | DEFAULT 0 | 最后消息序号 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |
| deleted_at | TIMESTAMPTZ | | |

**索引：**
```sql
CREATE INDEX idx_conversations_creator_id ON conversations(creator_id);
```

### conversation_members — 会话成员表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| conversation_id | BIGINT | NOT NULL | |
| user_id | BIGINT | NOT NULL | |
| role | VARCHAR(16) | DEFAULT 'member' | owner / admin / member |
| last_read_seq | BIGINT | DEFAULT 0 | 最后已读消息序号 |
| joined_at | TIMESTAMPTZ | NOT NULL | |

**唯一约束：** `(conversation_id, user_id)`

**索引：**
```sql
CREATE INDEX idx_conv_members_user_id ON conversation_members(user_id);
CREATE INDEX idx_conv_members_conv_id ON conversation_members(conversation_id);
```

### messages — 消息表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | 消息 ID (全局递增) |
| conversation_id | BIGINT | NOT NULL | |
| sender_id | BIGINT | NOT NULL | 发送者 |
| content | TEXT | NOT NULL | 消息内容 |
| msg_type | VARCHAR(16) | DEFAULT 'text' | text / image / file / system |
| seq | BIGINT | NOT NULL | 会话内消息序号 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**分区策略：** 按 `conversation_id` HASH 分区或按 `created_at` 时间范围分区 (数据量大时启用)

**索引：**
```sql
CREATE INDEX idx_messages_conv_seq ON messages(conversation_id, seq DESC);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
```

---

## 5. Docker 管理模块

### docker_nodes — Docker 节点表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| name | VARCHAR(64) | UNIQUE, NOT NULL | 节点名称 |
| host | VARCHAR(255) | NOT NULL | 主机 IP 或域名 |
| port | INT | DEFAULT 2376 | Docker TLS 端口 |
| tls_cert | TEXT | | TLS 客户端证书 |
| tls_key | TEXT | | TLS 客户端密钥 |
| ca_cert | TEXT | | TLS CA 证书 |
| status | VARCHAR(16) | DEFAULT 'offline' | online / offline / error |
| last_heartbeat | TIMESTAMPTZ | | 最后心跳时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_docker_nodes_status ON docker_nodes(status);
```

---

## 6. 系统配置模块

### system_configs — 系统配置表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| key | VARCHAR(128) | UNIQUE, NOT NULL | 配置键 |
| value | TEXT | | 配置值 |
| description | VARCHAR(255) | | 说明 |
| updated_at | TIMESTAMPTZ | NOT NULL | |

---

## 7. ER 关系图

```
users ──1:N──> refresh_tokens
users ──1:N──> files
users ──1:N──> file_shares
users ──< conversation_members >── conversations
conversations ──1:N──> messages
users ──1:N──> messages
docker_nodes (独立)
system_configs (独立)
```

---

## 8. 迁移计划

| 阶段 | 迁移内容 |
|------|----------|
| Phase 1 | users, refresh_tokens, files |
| Phase 2 | conversations, conversation_members, messages |
| Phase 3 | docker_nodes, file_shares |
| Phase 4 | 性能优化：分区、索引调优 |

---

## 9. 集群模式注意事项

- 主键 `BIGSERIAL` 在单机下高效，集群模式下可考虑：
  - Citus 分布式表使用 `BIGSERIAL` 配合分布键
  - 或使用 ULID/Snowflake 代替自增 ID
- `messages` 表建议按 `conversation_id` 做分布键
- Redis 维护每个会话的最后 seq，避免每次查询 MAX(seq)
