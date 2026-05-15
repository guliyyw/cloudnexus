# CloudNexus 数据库设计文档

> 版本：v1.2.0 | 数据库：PostgreSQL 15 | ORM：GORM

## 1. 设计原则

- 使用 Snowflake 算法生成全局唯一 uint64 主键（由 GORM `Before("gorm:create")` 回调自动分配）
- 所有 ID 字段在 JSON 中序列化为字符串（`json:",string"`），避免 JavaScript 精度丢失
- 各服务使用不同 Snowflake node ID (user-file-svc=1, im-svc=2, docker-svc=3, camera-svc=5, collab-svc=6)
- 时间字段统一使用 `TIMESTAMPTZ`
- 软删除使用 `deleted_at` 字段 (GORM 软删除)
- 外键约束在应用层保证，数据库层保持灵活

---

## 2. 用户模块

### users — 用户表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake 用户 ID |
| username | VARCHAR(64) | UNIQUE, NOT NULL | 用户名 |
| email | VARCHAR(255) | UNIQUE, NOT NULL | 邮箱 |
| password | VARCHAR(255) | NOT NULL | bcrypt 哈希 |
| avatar | VARCHAR(512) | | 头像 URL |
| nickname | VARCHAR(50) | | 昵称 |
| phone | VARCHAR(20) | | 手机号 |
| email_verified | BOOLEAN | DEFAULT false | 邮箱是否已验证 |
| phone_verified | BOOLEAN | DEFAULT false | 手机是否已验证 |
| locked_until | TIMESTAMPTZ | | 账号锁定截止时间 |
| login_fail_count | INT | DEFAULT 0 | 连续登录失败次数 |
| force_logout_after | TIMESTAMPTZ | | 强制登出截止时间（修改密码后设置） |
| delete_requested_at | TIMESTAMPTZ | | 账号注销申请时间 |
| is_admin | BOOLEAN | DEFAULT false | 是否为管理员 |
| status | SMALLINT | DEFAULT 1 | 1=正常, 0=禁用 |
| privacy | TEXT | DEFAULT '{"allow_search":true,"allow_add_friend":true,"show_online":true}' | 隐私设置 JSON |
| created_at | TIMESTAMPTZ | NOT NULL | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新时间 |
| deleted_at | TIMESTAMPTZ | | 软删除 |

**索引：**
```sql
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

### refresh_tokens — 刷新令牌表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL | 关联 users.id |
| token | VARCHAR(512) | UNIQUE, NOT NULL | 令牌 SHA256 哈希 |
| expires_at | TIMESTAMPTZ | NOT NULL | 过期时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

### user_sessions — 用户会话表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL, INDEX | 关联 users.id |
| jti | VARCHAR(36) | UNIQUE, NOT NULL | JWT Token ID (UUID) |
| user_agent | VARCHAR(500) | | 浏览器 User-Agent |
| ip_address | VARCHAR(45) | | 登录 IP 地址 |
| login_at | TIMESTAMPTZ | | 登录时间 |
| last_active_at | TIMESTAMPTZ | | 最后活跃时间 |
| expires_at | TIMESTAMPTZ | NOT NULL | 过期时间 |
| is_active | BOOLEAN | DEFAULT true | 是否活跃 |
| created_at | TIMESTAMPTZ | NOT NULL | |

### oauth_bindings — OAuth 绑定表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL, INDEX | 关联 users.id |
| provider | VARCHAR(20) | NOT NULL | 第三方平台 (github/google/wechat) |
| open_id | VARCHAR(255) | NOT NULL | 第三方用户标识 |
| access_token | TEXT | | 第三方访问令牌 (加密存储) |
| refresh_token | TEXT | | 第三方刷新令牌 (加密存储) |
| expires_at | TIMESTAMPTZ | | 令牌过期时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**唯一约束：** `(provider, open_id)`

---

## 3. 文件模块

### files — 文件/目录表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL | 所属用户 |
| name | VARCHAR(255) | NOT NULL | 文件名 |
| is_dir | BOOLEAN | DEFAULT false | 是否为目录 |
| parent_id | BIGINT | DEFAULT 0 | 父目录 ID，0=根 |
| size | BIGINT | DEFAULT 0 | 文件大小 (字节) |
| mime_type | VARCHAR(128) | | MIME 类型 |
| storage_key | VARCHAR(512) | | MinIO 对象键 |
| storage_sha256 | VARCHAR(64) | | 文件 SHA256 |
| is_shared | BOOLEAN | DEFAULT false | 是否已分享 |
| collab_type | VARCHAR(16) | DEFAULT '' | 协作文档类型 (空=非协作文档) |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |
| deleted_at | TIMESTAMPTZ | | 软删除（回收站） |

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
| id | BIGINT | PK | Snowflake ID |
| file_id | BIGINT | NOT NULL | 关联 files.id |
| owner_id | BIGINT | NOT NULL | 分享发起者 |
| share_code | VARCHAR(32) | UNIQUE, NOT NULL | 12位随机分享码 |
| password | VARCHAR(255) | | 访问密码 (bcrypt) |
| expires_at | TIMESTAMPTZ | | 过期时间 |
| download_limit | INT | DEFAULT 0 | 下载次数限制 (0=无限制) |
| download_count | INT | DEFAULT 0 | 已下载次数 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_file_shares_share_code ON file_shares(share_code);
```

### file_versions — 文件版本表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| file_id | BIGINT | NOT NULL, INDEX | 所属文件 |
| version_num | INT | NOT NULL | 版本号（1, 2, 3...） |
| storage_key | VARCHAR(512) | NOT NULL | MinIO 对象 key |
| size | BIGINT | DEFAULT 0 | 文件大小 |
| sha256 | VARCHAR(64) | | 内容校验值 |
| message | VARCHAR(256) | | 版本说明 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_file_versions_file_id ON file_versions(file_id);
CREATE INDEX idx_file_versions_file_version ON file_versions(file_id, version_num DESC);
```

### chunk_uploads — 分块上传表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL, INDEX | 上传用户 |
| upload_id | VARCHAR(36) | UNIQUE, NOT NULL | 上传会话 ID (UUID) |
| file_name | VARCHAR(255) | NOT NULL | 原始文件名 |
| file_size | BIGINT | NOT NULL | 文件总大小 |
| chunk_size | INT | DEFAULT 10485760 | 分块大小 (默认 10MB) |
| mime_type | VARCHAR(128) | | MIME 类型 |
| parent_id | BIGINT | DEFAULT 0 | 目标目录 ID |
| total_chunks | INT | NOT NULL | 总块数 |
| completed | INTEGER[] | DEFAULT '{}' | 已完成块序号数组 |
| status | VARCHAR(20) | DEFAULT 'uploading' | uploading / completed / cancelled |
| version_message | VARCHAR(256) | | 版本说明 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

### quota_tiers — 配额等级表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| name | VARCHAR(64) | UNIQUE, NOT NULL | 等级名称 (free/premium) |
| storage_limit | BIGINT | NOT NULL | 存储上限 (字节) |
| description | VARCHAR(256) | | 等级描述 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

预置数据：free (1GB)、premium (10GB)

### user_quota — 用户配额表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| user_id | BIGINT | PK | 用户 ID |
| storage_used | BIGINT | NOT NULL, DEFAULT 0 | 已用存储 (字节) |
| storage_limit | BIGINT | | 自定义存储上限 (NULL=使用等级限额) |
| tier_id | BIGINT | | 关联 quota_tiers.id |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

实际表名为 `user_quota` (单数)，GORM 默认 NamingStrategy 自动转换。

### system_configs — 系统配置表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| key | VARCHAR(128) | PK | 配置键 |
| value | TEXT | NOT NULL | 配置值 (JSON/字符串) |
| updated_at | TIMESTAMPTZ | NOT NULL | |

预置配置：`upload.sequential_mode`、`upload.max_concurrent_chunks`

---

## 4. 即时通讯模块

### conversations — 会话表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
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
| id | BIGINT | PK | Snowflake ID |
| conversation_id | BIGINT | NOT NULL | |
| user_id | BIGINT | NOT NULL | |
| role | VARCHAR(16) | DEFAULT 'member' | owner / admin / member |
| last_read_seq | BIGINT | DEFAULT 0 | 最后已读消息序号 |
| joined_at | TIMESTAMPTZ | NOT NULL | |
| deleted_at | TIMESTAMPTZ | | 用户视角软删除 |

**唯一约束：** `(conversation_id, user_id)`

**索引：**
```sql
CREATE INDEX idx_conv_members_user_id ON conversation_members(user_id);
CREATE INDEX idx_conv_members_conv_id ON conversation_members(conversation_id);
```

### messages — 消息表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| conversation_id | BIGINT | NOT NULL | |
| sender_id | BIGINT | NOT NULL | 发送者 |
| content | TEXT | NOT NULL | 消息内容 |
| msg_type | VARCHAR(16) | DEFAULT 'text' | text / image / video / file / system |
| seq | BIGINT | NOT NULL | 会话内消息序号 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**分区策略：** 按 `conversation_id` HASH 分区或按 `created_at` 时间范围分区 (数据量大时启用)

**索引：**
```sql
CREATE INDEX idx_messages_conv_seq ON messages(conversation_id, seq DESC);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
```

### friends — 好友关系表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL | 请求发起者 |
| friend_id | BIGINT | NOT NULL | 被请求者 |
| status | VARCHAR(16) | DEFAULT 'pending' | pending / accepted / blocked |
| message | VARCHAR(200) | | 好友请求附言 |
| remark | VARCHAR(50) | | 好友备注名 |
| expires_at | TIMESTAMPTZ | | 请求过期时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

**唯一约束：** `(user_id, friend_id)`

**索引：**
```sql
CREATE UNIQUE INDEX idx_friend_pair ON friends(user_id, friend_id);
```

### blocklists — 用户拉黑表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL | 拉黑操作者 |
| blocked_user_id | BIGINT | NOT NULL | 被拉黑用户 |
| reason | VARCHAR(200) | | 拉黑原因 |
| created_at | TIMESTAMPTZ | NOT NULL | |

**唯一约束：** `(user_id, blocked_user_id)`

---

## 5. 验证与安全模块

### email_verifications — 邮箱验证码表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | INDEX | 关联用户 (注册时可为空) |
| email | VARCHAR(255) | NOT NULL | 验证邮箱 |
| code | VARCHAR(6) | NOT NULL | 6位验证码 |
| type | VARCHAR(20) | DEFAULT 'register' | register / login / bind / reset_password |
| expires_at | TIMESTAMPTZ | NOT NULL | 过期时间 |
| used | BOOLEAN | DEFAULT false | 是否已使用 |
| created_at | TIMESTAMPTZ | NOT NULL | |

### phone_verifications — 手机验证码表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | INDEX | 关联用户 |
| phone | VARCHAR(20) | NOT NULL | 验证手机号 |
| code | VARCHAR(6) | NOT NULL | 6位验证码 |
| type | VARCHAR(20) | DEFAULT 'register' | register / login / bind / reset_password |
| expires_at | TIMESTAMPTZ | NOT NULL | 过期时间 |
| used | BOOLEAN | DEFAULT false | 是否已使用 |
| created_at | TIMESTAMPTZ | NOT NULL | |

### password_reset_tokens — 密码重置令牌表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| user_id | BIGINT | NOT NULL, INDEX | 关联 users.id |
| token | VARCHAR(64) | UNIQUE, NOT NULL | 重置令牌 |
| expires_at | TIMESTAMPTZ | NOT NULL | 过期时间 |
| used | BOOLEAN | DEFAULT false | 是否已使用 |
| created_at | TIMESTAMPTZ | NOT NULL | |

---

## 6. RBAC 权限模块

### permissions — 权限表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| name | VARCHAR(100) | UNIQUE, NOT NULL | 权限名称（中文） |
| code | VARCHAR(100) | UNIQUE, NOT NULL | 权限编码 (如 `file:read`) |
| description | VARCHAR(300) | | 权限描述 |
| group_name | VARCHAR(50) | | 分组名称 |
| created_at | TIMESTAMPTZ | NOT NULL | |

预置权限包括：`file:read`, `file:write`, `file:delete`, `file:share`, `docker:read`, `docker:control`, `docker:admin`, `camera:read`, `camera:control`, `camera:admin`, `face:read`, `face:write`, `face:admin`, `attendance:read`, `attendance:admin`, `user:admin`, `system:admin`

### roles — 角色表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| name | VARCHAR(50) | UNIQUE, NOT NULL | 角色名称 |
| code | VARCHAR(50) | UNIQUE, NOT NULL | 角色编码 |
| description | VARCHAR(200) | | 角色描述 |
| is_system | BOOLEAN | DEFAULT false | 是否为系统角色（不可删除） |
| created_at | TIMESTAMPTZ | NOT NULL | |

预置角色：`super_admin`（全部权限）、`admin`（管理权限）、`user`（基本权限）

### role_permissions — 角色权限关联表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| role_id | BIGINT | PK | 关联 roles.id |
| permission_id | BIGINT | PK | 关联 permissions.id |

### user_roles — 用户角色关联表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| user_id | BIGINT | PK | 关联 users.id |
| role_id | BIGINT | PK | 关联 roles.id |
| granted_by | BIGINT | | 授权人 ID |
| granted_at | TIMESTAMPTZ | | 授权时间 |

---

## 7. Docker 管理模块

### docker_nodes — Docker 节点表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| name | VARCHAR(64) | UNIQUE, NOT NULL | 节点名称 (容器 hostname) |
| host | VARCHAR(255) | NOT NULL | 主机 IP 或域名 |
| port | INT | DEFAULT 2376 | Docker TLS 端口 |
| tls_cert | TEXT | | TLS 客户端证书 |
| tls_key | TEXT | | TLS 客户端密钥 |
| ca_cert | TEXT | | TLS CA 证书 |
| status | VARCHAR(16) | DEFAULT 'offline' | healthy / unresponsive / offline |
| node_type | VARCHAR(16) | DEFAULT 'service' | service / docker_endpoint / infrastructure |
| service | VARCHAR(32) | | 服务名 (如 user-file-svc, postgres) |
| first_seen_at | TIMESTAMPTZ | | 首次注册时间 |
| total_online_seconds | BIGINT | DEFAULT 0 | 累计在线时长 (秒) |
| offline_since | TIMESTAMPTZ | | 最近离线时间 |
| container_name | VARCHAR(128) | | 容器名称 |
| version | VARCHAR(32) | | 服务版本 |
| last_heartbeat | TIMESTAMPTZ | | 最后心跳时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_docker_nodes_status ON docker_nodes(status);
CREATE INDEX idx_docker_nodes_type ON docker_nodes(node_type);
```

### node_online_sessions — 节点在线会话表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| node_name | VARCHAR(64) | NOT NULL, INDEX | 节点名称 |
| start_time | TIMESTAMPTZ | NOT NULL | 上线时间 |
| end_time | TIMESTAMPTZ | | 下线时间 |
| duration | BIGINT | DEFAULT 0 | 在线时长 (秒) |
| container_name | VARCHAR(128) | | 容器名称 |
| version | VARCHAR(32) | | 服务版本 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

---

## 8. 告警模块

### alert_rules — 告警规则表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| name | VARCHAR(128) | UNIQUE, NOT NULL | 规则名称 |
| description | VARCHAR(512) | | 规则描述 |
| enabled | BOOLEAN | DEFAULT true | 是否启用 |
| node_name | VARCHAR(64) | DEFAULT '*' | 适用节点 (*=全部) |
| trigger_type | VARCHAR(32) | DEFAULT 'status_change' | 触发类型 |
| condition | TEXT | | 触发条件 JSON |
| webhook_url | VARCHAR(512) | NOT NULL | Webhook 回调地址 |
| cooldown_seconds | INT | DEFAULT 300 | 冷却时间 (秒) |
| created_by | BIGINT | DEFAULT 0 | 创建者 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

### alert_history — 告警历史表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| rule_id | BIGINT | NOT NULL, INDEX | 关联 alert_rules.id |
| rule_name | VARCHAR(128) | NOT NULL | 规则名称快照 |
| node_name | VARCHAR(64) | NOT NULL, INDEX | 告警节点 |
| alert_type | VARCHAR(32) | NOT NULL | 告警类型 |
| status | VARCHAR(16) | DEFAULT 'firing' | firing / resolved |
| message | TEXT | | 告警消息 |
| fired_at | TIMESTAMPTZ | NOT NULL, INDEX | 触发时间 |
| resolved_at | TIMESTAMPTZ | | 解决时间 |
| webhook_url | VARCHAR(512) | | Webhook 地址快照 |
| response_code | INT | DEFAULT 0 | Webhook 响应状态码 |
| error_message | TEXT | | Webhook 错误信息 |

---

## 9. 摄像头与 AI 识别模块

### cameras — 摄像头表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| owner_id | BIGINT | NOT NULL, INDEX | 所属用户 |
| name | VARCHAR(128) | NOT NULL | 摄像头名称 |
| stream_url | VARCHAR(512) | NOT NULL | RTSP/RTMP 地址 |
| protocol | VARCHAR(16) | DEFAULT 'rtsp' | rtsp / rtmp / onvif |
| status | VARCHAR(16) | DEFAULT 'offline' | online / offline |
| last_seen_at | TIMESTAMPTZ | | 最近在线时间 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |
| deleted_at | TIMESTAMPTZ | INDEX | 软删除 |

### recognition_events — AI 识别事件表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| camera_id | BIGINT | NOT NULL, INDEX | 所属摄像头 |
| event_type | VARCHAR(32) | NOT NULL | 如 person / car |
| confidence | FLOAT | NOT NULL | 置信度 0-1 |
| snapshot_url | VARCHAR(512) | | MinIO 截图 URL |
| metadata | TEXT | | JSON 格式检测框数据 (x1,y1,x2,y2) |
| created_at | TIMESTAMPTZ | NOT NULL, INDEX | |

**索引：**
```sql
CREATE INDEX idx_recognition_events_camera_id ON recognition_events(camera_id);
CREATE INDEX idx_recognition_events_created_at ON recognition_events(created_at);
```

---

## 10. 人脸识别与考勤模块

### face_profiles — 人脸库表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| owner_id | BIGINT | NOT NULL, INDEX | 所属用户 |
| name | VARCHAR(64) | NOT NULL | 人员姓名 |
| embedding | TEXT | NOT NULL | 128维人脸嵌入向量 (JSON float64 数组) |
| thumbnail_url | TEXT | | MinIO 缩略图对象 key |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |
| deleted_at | TIMESTAMPTZ | INDEX | 软删除 |

### face_recognition_events — 人脸识别事件表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| camera_id | BIGINT | NOT NULL, INDEX | 所属摄像头 |
| face_id | BIGINT | INDEX | 匹配的人脸 ID |
| face_name | VARCHAR(64) | | 匹配的人脸名称 |
| confidence | FLOAT | | 匹配置信度 0-1 |
| snapshot_url | TEXT | | 截图 URL |
| bbox_json | TEXT | | 人脸边界框 JSON |
| created_at | TIMESTAMPTZ | NOT NULL, INDEX | |

### face_attendance_sessions — 考勤签到表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| face_id | BIGINT | NOT NULL, INDEX | 关联 face_profiles.id |
| face_name | VARCHAR(64) | NOT NULL | 人员姓名快照 |
| camera_id | BIGINT | NOT NULL, INDEX | 签到摄像头 |
| start_time | TIMESTAMPTZ | NOT NULL, INDEX | 签到时间 |
| end_time | TIMESTAMPTZ | NOT NULL | 签退时间 |
| date | VARCHAR(10) | NOT NULL, INDEX | 日期 (YYYY-MM-DD) |

**考勤逻辑：** 同人在同摄像头 5 分钟内的重复签到会合并（延长 end_time）。每日汇总：`start_time` 最小值=签到时间，`end_time` 最大值=签退时间。

---

## 11. 相册模块 (v0.2.0 新增)

### albums — 相册表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| owner_id | BIGINT | NOT NULL, INDEX | 所属用户 |
| name | VARCHAR(200) | NOT NULL | 相册名称 |
| description | TEXT | | 相册描述 |
| cover_file_id | BIGINT | | 封面文件 ID |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

**索引：**
```sql
CREATE INDEX idx_albums_owner_id ON albums(owner_id);
```
**设计要点：** 相册仅存储文件引用（file_id），不复制文件数据。file_count 为实时聚合字段（GORM `-` tag）。

### album_files — 相册文件关联表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| album_id | BIGINT | PK (联合) | 关联 albums.id |
| file_id | BIGINT | PK (联合) | 关联 files.id |
| added_at | TIMESTAMPTZ | NOT NULL | 添加时间 |

**联合主键：** `(album_id, file_id)`

### exif_metadata — EXIF 元数据表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| file_id | BIGINT | PK | 关联 files.id |
| make | VARCHAR(100) | | 相机制造商 |
| model | VARCHAR(100) | | 相机型号 |
| datetime_taken | TIMESTAMPTZ | | 拍摄时间 |
| latitude | DOUBLE PRECISION | | GPS 纬度 |
| longitude | DOUBLE PRECISION | | GPS 经度 |
| iso | INTEGER | | ISO 感光度 |
| f_number | DOUBLE PRECISION | | 光圈值 |
| exposure_time | VARCHAR(20) | | 曝光时间 |
| focal_length | DOUBLE PRECISION | | 焦距 (mm) |

**设计要点：** 文件上传时异步 goroutine 使用 `github.com/rwcarlsen/goexif/exif` 解析 image/jpeg、image/tiff EXIF 数据。时间线视图按 `datetime_taken` 月份分组。

---

## 12. 音乐模块 (v0.2.0 新增)

### public_tracks — 公共音乐库表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| title | VARCHAR(300) | NOT NULL | 歌曲标题 |
| artist | VARCHAR(200) | | 艺术家 |
| album | VARCHAR(200) | | 专辑名 |
| duration | INT | DEFAULT 0 | 时长 (秒) |
| track_num | INT | | 音轨号 |
| storage_key | VARCHAR(500) | NOT NULL | MinIO 存储 key |
| mime_type | VARCHAR(50) | | MIME 类型 |
| file_size | BIGINT | | 文件大小 |
| uploaded_by | BIGINT | | 上传者 |
| created_at | TIMESTAMPTZ | NOT NULL | |

### playlists — 播放列表表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| owner_id | BIGINT | NOT NULL, INDEX | 所属用户 |
| name | VARCHAR(200) | NOT NULL | 播放列表名称 |
| is_public | BOOLEAN | DEFAULT false | 是否公开 |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

**设计要点：** track_count 为实时聚合字段（GORM `-` tag）。

### playlist_tracks — 播放列表歌曲表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| playlist_id | BIGINT | PK (联合) | 关联 playlists.id |
| track_id | BIGINT | PK (联合) | 歌曲 ID |
| source | VARCHAR(10) | NOT NULL | 来源: public / cloud |
| sort_order | INT | DEFAULT 0 | 排序序号 |

**联合主键：** `(playlist_id, track_id)`
**设计要点：** `source` 区分公共音乐库 (`public`) 和用户云盘音频 (`cloud`)。拖拽排序更新 `sort_order`。

---

## 13. 系统监控历史表 (v0.2.0 新增)

### dashboard_health_history — 健康状态快照表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| timestamp | TIMESTAMPTZ | NOT NULL, INDEX | 快照时间 |
| status_data | TEXT | NOT NULL | 健康状态 JSON (各模块绿/黄/红) |
| type | VARCHAR(20) | NOT NULL, INDEX | 快照类型: health / resources |

**设计要点：** HealthAggregator 探测循环每 5 分钟写入一次，24 小时保留期。复用环形缓冲思路控制内存。

### resource_metrics_history — 资源指标历史表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK | Snowflake ID |
| timestamp | TIMESTAMPTZ | NOT NULL, INDEX | 采样时间 |
| service | VARCHAR(30) | NOT NULL, INDEX | 服务名称 |
| cpu_percent | DOUBLE PRECISION | | CPU 使用率 % |
| memory_used | BIGINT | | 内存使用 (bytes) |
| memory_total | BIGINT | | 内存总量 (bytes) |
| disk_used | BIGINT | | 磁盘使用 (bytes) |
| disk_total | BIGINT | | 磁盘总量 (bytes) |

**索引：**
```sql
CREATE INDEX idx_resource_metrics_svc_time ON resource_metrics_history(service, timestamp);
```
**设计要点：** 每分钟采集一次，复用 Docker 容器 CPU/内存数据。服务状态页的 recharts 折线图数据源。

---

## 14. ER 关系图

```
users ──1:N──> refresh_tokens
users ──1:N──> user_sessions
users ──1:N──> oauth_bindings
users ──1:N──> password_reset_tokens
users ──1:N──> files
users ──1:N──> file_shares
users ──1:N──> cameras
users ──1:N──> face_profiles
users ──1:N──> albums
users ──1:N──> playlists
albums ──< album_files >── files
files  ──1:1──> exif_metadata
files  ──1:N──> file_versions
cameras ──1:N──> recognition_events
cameras ──1:N──> face_recognition_events
face_profiles ──1:N──> face_attendance_sessions
playlists ──< playlist_tracks >── public_tracks
users ──< conversation_members >── conversations
conversations ──1:N──> messages
users ──1:N──> messages
users ──< friends >── users (好友双向关系)
users ──< user_roles >── roles
roles ──< role_permissions >── permissions
users ──1:N──> alert_rules
alert_rules ──1:N──> alert_history
users ──1:1──> user_quota
user_quota ──N:1──> quota_tiers
users ──1:N──> chunk_uploads
users ──1:N──> email_verifications
users ──1:N──> phone_verifications
users ──< blocklists >── users (拉黑关系)
dashboard_health_history
resource_metrics_history
docker_nodes
node_online_sessions
system_configs
schema_migrations (版本化 SQL 迁移追踪)
```

---

## 15. 集群模式注意事项

- 主键使用 Snowflake uint64，天然支持分布式全局唯一，无需额外配置
- 集群模式下各服务使用不同的 Snowflake node ID (user-file-svc=1, im-svc=2, docker-svc=3, camera-svc=5, collab-svc=6)
- 同一服务的多实例必须分配不同的 Snowflake node ID
- 所有服务运行在 Docker 容器中，通过 Docker 内部网络通信
