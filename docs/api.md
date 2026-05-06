# CloudNexus API 接口文档

> 版本：v1.0.0 | 更新：2026-05-06

## 通用约定

### 请求格式

- Content-Type: `application/json`
- 认证方式：Bearer Token (JWT)，通过 `Authorization: Bearer <access_token>` 传递
- 入口：所有 API 通过 nginx:80 统一访问，后端端口不对外暴露
- ID 类型：所有 ID 字段为 JSON 字符串（Snowflake uint64，避免 JS 精度丢失）

### 响应格式

```json
{
  "code": 200,
  "message": "ok",
  "data": {}
}
```

### 错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证或令牌过期 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 资源冲突 (重复) |
| 500 | 服务器内部错误 |

---

## 一、用户文件服务 (user-file-svc) — 端口 8081

### 1.1 用户认证

#### POST /api/v1/user/register

注册新用户。

**请求体：**
```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "securepass123"
}
```

**响应 (201)：**
```json
{
  "code": 201,
  "message": "registered",
  "data": {
    "id": "123456789012345678",
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2026-05-03T10:00:00Z"
  }
}
```

#### POST /api/v1/user/login

用户登录，返回 JWT 令牌对。

**请求体：**
```json
{
  "username": "alice",
  "password": "securepass123"
}
```

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "expires_in": 28800,
    "is_admin": false
  }
}
```

#### POST /api/v1/user/refresh

刷新访问令牌。

**请求体：**
```json
{
  "refresh_token": "eyJhbGci..."
}
```

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "expires_in": 28800
  }
}
```

### 1.2 用户信息

#### GET /api/v1/user/profile

获取当前用户信息 (需认证)。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": "2051225055077076992",
    "username": "alice",
    "email": "alice@example.com",
    "is_admin": false,
    "created_at": "2026-05-03T10:00:00Z"
  }
}
```

#### PUT /api/v1/user/profile

更新用户信息 (需认证)。

**请求体：**
```json
{
  "email": "newalice@example.com",
  "avatar": "https://static.example.com/avatars/alice2.png"
}
```

### 1.3 文件管理

#### POST /api/v1/file/upload

上传文件 (需认证, multipart/form-data)。支持单文件和批量上传。单文件和批量均返回 `BatchUploadResult` 格式。

**表单字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File (可多次) | 上传的文件，批量上传时多次 append |
| parent_id | string | 父目录 ID，"0" 为根目录 |

**响应 (201)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "files": [
      {
        "id": "2051903357815099392",
        "user_id": "2051901759848189952",
        "name": "report.pdf",
        "is_dir": false,
        "parent_id": "0",
        "size": 204800,
        "mime_type": "application/pdf",
        "storage_key": "...",
        "created_at": "2026-05-03T10:00:00Z"
      }
    ],
    "errors": [],
    "total": 1,
    "ok": 1
  }
}
```

#### GET /api/v1/file/list

列出文件 (需认证)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| parent_id | int | 父目录 ID，默认 0 (根目录) |
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 50 |

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": 100,
        "name": "report.pdf",
        "size": 204800,
        "type": "application/pdf",
        "is_dir": false,
        "parent_id": 0,
        "created_at": "2026-05-03T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 50
  }
}
```

#### GET /api/v1/file/download/:id

下载或预览文件 (需认证)。返回文件流。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| inline | bool | 设为 true 时浏览器内联预览（图片/视频/PDF），默认 false 触发下载 |

#### DELETE /api/v1/file/:id

删除文件 (需认证)。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "deleted"
}
```

#### POST /api/v1/file/batch-delete

批量删除文件 (需认证)。

**请求体：**
```json
{
  "ids": ["100", "101", "102"]
}
```

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "deleted": 3,
    "errors": []
  }
}
```

#### POST /api/v1/file/batch-download

批量下载文件，打包为 ZIP (需认证)。

**请求体：**
```json
{
  "ids": ["100", "101"]
}
```

**响应 (200)：** 返回 `application/zip` 文件流，`Content-Disposition: attachment; filename="files-20260504.zip"`。

> 注意：下载多个大文件时服务端流式打包，不使用内存缓冲，避免 OOM。

#### POST /api/v1/file/mkdir

创建目录 (需认证)。

**请求体：**
```json
{
  "name": "documents",
  "parent_id": 0
}
```

#### POST /api/v1/file/move

移动文件到目标目录 (需认证)。

**请求体：**
```json
{
  "ids": ["100", "101"],
  "target_parent_id": "200"
}
```

#### POST /api/v1/file/copy

复制文件到目标目录 (需认证)。

**请求体：**
```json
{
  "ids": ["100"],
  "target_parent_id": "300"
}
```

目录递归复制时自动复制所有子文件和目录，MinIO 对象同步复制。

#### GET /api/v1/file/search

搜索文件 (需认证)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| q | string | 搜索关键词 |
| page | int | 页码 |
| page_size | int | 每页数量 |

### 1.4 文件分享

#### POST /api/v1/share

创建文件分享链接 (需认证)。

**请求体：**
```json
{
  "file_id": "100",
  "password": "optional-pass",
  "expires_in": 86400
}
```

**响应 (201)：**
```json
{
  "code": 201,
  "message": "ok",
  "data": {
    "id": "...",
    "file_id": "100",
    "share_code": "a1b2c3d4e5f6",
    "has_password": true,
    "expires_at": "2026-05-07T10:00:00Z"
  }
}
```

#### GET /api/v1/share/:code

公开访问分享链接 (无需认证)。返回文件信息和是否需要密码。

#### POST /api/v1/share/:code/verify

验证分享密码 (无需认证)。

**请求体：**
```json
{
  "password": "optional-pass"
}
```

#### GET /api/v1/share/:code/download

下载分享文件 (无需认证)。支持 `?inline=true` 内联预览。

#### GET /api/v1/shares

列出我的分享链接 (需认证)。

#### DELETE /api/v1/share/:code

取消分享 (需认证)。

### 1.5 WebDAV

WebDAV 端点挂载在 `/webdav/` 路径下，支持标准 WebDAV 协议方法：

- `PROPFIND` — 列出目录
- `GET` — 下载文件
- `PUT` — 上传文件
- `DELETE` — 删除文件
- `MKCOL` — 创建目录
- `MOVE` — 移动/重命名

认证通过 HTTP Basic Auth 使用用户名和密码。

---

## 二、即时通讯服务 (im-svc) — 端口 8082

### 2.1 REST 接口

#### GET /api/v1/im/conversations

获取会话列表 (需认证)。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": [
    {
      "id": "2051901589047746560",
      "type": "private",
      "name": "Bob",
      "creator_id": "2051225055077076992",
      "last_msg_seq": 42,
      "unread": 3,
      "created_at": "2026-05-03T08:00:00Z",
      "updated_at": "2026-05-03T09:55:00Z"
    }
  ]
}
```

#### POST /api/v1/im/conversations

创建会话 (需认证)。

**私聊请求体：**
```json
{
  "type": "private",
  "member_ids": [2]
}
```

**群聊请求体：**
```json
{
  "type": "group",
  "name": "项目组",
  "member_ids": [2, 3, 4]
}
```

#### GET /api/v1/im/conversations/:id/messages

获取会话历史消息 (需认证)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| before | int | 获取该消息 ID 之前的消息 (游标分页) |
| limit | int | 每页数量，默认 50，最大 100 |

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": [
    {
      "id": "2051901589047746561",
      "conversation_id": "2051901589047746560",
      "sender_id": "2051225055077076992",
      "content": "你好！",
      "msg_type": "text",
      "seq": 1,
      "created_at": "2026-05-03T09:00:00Z"
    }
  ]
}
```

msg_type 可选值: `text` / `image` / `video` / `file` / `system`

#### DELETE /api/v1/im/conversations/:id

删除会话 (需认证)。当前用户视角的软删除，不影响其他成员。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "deleted"
}
```

#### GET /api/v1/im/conversations/:id/members

获取群聊成员列表 (需认证)。

#### POST /api/v1/im/conversations/:id/members

添加群成员 (需认证，仅群主)。

**请求体：**
```json
{
  "user_id": "123"
}
```

#### DELETE /api/v1/im/conversations/:id/members/:uid

移除群成员 (需认证，仅群主)。

#### POST /api/v1/im/conversations/:id/leave

退出群聊 (需认证)。

#### GET /api/v1/im/conversations/:id/export

导出会话全部消息为 JSON 文件 (需认证)。返回 JSON 文件下载，包含校验码。

**响应 (200)：** 返回 `application/json` 文件流，`Content-Disposition: attachment; filename=chat_{id}_{timestamp}.json`。

**JSON 结构：**
```json
{
  "version": "1.0",
  "conversation_id": "2051176674648662016",
  "conversation_type": "private",
  "conversation_name": "alice",
  "participants": ["alice", "testuser"],
  "exported_at": "2026-05-06T06:39:49Z",
  "exported_by": "testuser",
  "message_count": 2,
  "last_message_seq": 1,
  "checksum": "f6cb9772...",
  "messages": [
    {
      "id": "2051220917752700928",
      "conversation_id": "2051176674648662016",
      "sender_id": "2051175871523328000",
      "sender_name": "testuser",
      "content": "Hello!",
      "msg_type": "text",
      "seq": 1,
      "created_at": "2026-05-04T08:42:12.196Z"
    }
  ]
}
```

校验码 = SHA256(`conversation_id|message_count|last_seq`)

#### POST /api/v1/im/conversations/import

导入聊天记录 JSON 文件 (需认证, multipart/form-data)。校验码验证后，按 ID 去重批量写入。

**表单字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File | 导出的 JSON 文件 |

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "inserted": 5,
    "skipped": 2,
    "total": 7,
    "last_seq": 7
  }
}
```

若校验码不匹配返回 400 错误。

> 注意：路由 `/import` 在 `/:id/export` 之前注册，避免 "import" 被解析为 conversation ID。

#### POST /api/v1/im/link-preview

获取链接预览元数据 (需认证)。

**请求体：**
```json
{
  "url": "https://github.com"
}
```

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "url": "https://github.com",
    "title": "GitHub",
    "description": "Let's build from here",
    "image": "https://github.githubassets.com/...",
    "site_name": "GitHub"
  }
}
```

### 2.2 好友系统

#### POST /api/v1/im/friends/requests

发送好友请求 (需认证)。

**请求体：**
```json
{
  "friend_name": "bob"
}
```

**响应 (201)：**
```json
{
  "code": 201,
  "message": "ok",
  "data": {
    "id": 1,
    "user_id": 1,
    "friend_id": 2,
    "status": "pending",
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

#### GET /api/v1/im/friends/requests

列出待处理的好友请求 (需认证)。返回发给当前用户的 pending 请求。

#### PUT /api/v1/im/friends/requests/:id/accept

接受好友请求 (需认证)。接受后自动创建私聊会话。

#### PUT /api/v1/im/friends/requests/:id/reject

拒绝好友请求 (需认证)。

#### GET /api/v1/im/friends

列出已接受的好友 (需认证)。

#### DELETE /api/v1/im/friends/:friend_id

删除好友 (需认证)。

#### GET /api/v1/im/friends/search?q=

搜索用户 (需认证)。按用户名搜索。

### 2.3 WebSocket 协议

**连接地址：** `ws://host:8082/ws?token=<access_token>`

客户端通过 WebSocket 连接后，发送 JSON 格式的消息帧。

#### 客户端 → 服务端

**发送消息：**
```json
{
  "type": "message",
  "conversation_id": 1,
  "content": "你好！",
  "msg_type": "text"
}
```

**已读回执：**
```json
{
  "type": "read_receipt",
  "conversation_id": 1,
  "last_read_msg_id": 500
}
```

**心跳 (每 30 秒)：**
```json
{
  "type": "ping"
}
```

#### 服务端 → 客户端

**收到新消息：**
```json
{
  "type": "message",
  "id": 501,
  "conversation_id": 1,
  "sender_id": 2,
  "content": "你好，Alice！",
  "msg_type": "text",
  "created_at": "2026-05-03T09:01:00Z"
}
```

**消息已送达确认：**
```json
{
  "type": "ack",
  "msg_id": 500,
  "status": "delivered"
}
```

**在线状态变更：**
```json
{
  "type": "presence",
  "user_id": 2,
  "status": "online"
}
```

**心跳响应：**
```json
{
  "type": "pong"
}
```

### 2.4 跨节点消息同步

当用户连接在不同节点时，消息通过 Redis Pub/Sub 跨节点路由：

1. 节点 A 收到发送方消息，写入数据库
2. 本地 Hub 检查接收方是否在线，若在线则直接推送
3. 若接收方不在本地，将消息发布到 Redis 频道 `im:broadcast`
4. 所有节点订阅该频道，收到消息后推送给本地已连接的接收方

---

## 三、Docker 管理服务 (docker-svc) — 端口 8083

### 3.1 容器管理

#### GET /api/v1/docker/containers

列出容器 (需认证)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| all | bool | 是否包含已停止容器，默认 false |
| node | string | 目标节点名称 (集群模式)，默认本地 |

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": [
    {
      "id": "abc123",
      "name": "nginx",
      "image": "nginx:alpine",
      "status": "running",
      "ports": ["80:80", "443:443"],
      "created": "2026-05-03T08:00:00Z"
    }
  ]
}
```

#### POST /api/v1/docker/containers

创建并启动容器 (需认证)。

**请求体：**
```json
{
  "image": "nginx:alpine",
  "name": "my-nginx",
  "ports": {"80/tcp": 8080},
  "env": ["ENV=production"],
  "volumes": {"host/path": "container/path"},
  "node": "local"
}
```

#### POST /api/v1/docker/containers/:id/stop

停止容器。

#### POST /api/v1/docker/containers/:id/start

启动已停止的容器。

#### POST /api/v1/docker/containers/:id/restart

重启容器。

#### DELETE /api/v1/docker/containers/:id

删除容器。查询参数 `force=true` 可强制删除运行中的容器。

#### GET /api/v1/docker/containers/:id/logs

获取容器日志。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| tail | int | 返回最后 N 行，默认 100 |
| since | string | 起始时间 |

#### GET /api/v1/docker/containers/:id/stats

获取容器资源使用统计 (需认证)。返回 CPU 使用率 (通过两次采样计算 delta)、内存使用/限制。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "cpu_percent": 2.5,
    "memory_usage": 10485760,
    "memory_limit": 67108864,
    "memory_percent": 15.6,
    "pids": 12
  }
}
```

### 3.2 镜像管理

#### GET /api/v1/docker/images

列出本地镜像。

#### DELETE /api/v1/docker/images/:id

删除镜像。

#### POST /api/v1/docker/images/pull

拉取镜像。

**请求体：**
```json
{
  "image": "nginx:alpine"
}
```

### 3.3 网络管理

#### GET /api/v1/docker/networks

列出网络。

#### POST /api/v1/docker/networks

创建网络。

**请求体：**
```json
{
  "name": "cloudnexus-net",
  "driver": "bridge"
}
```

### 3.4 数据卷管理

#### GET /api/v1/docker/volumes

列出数据卷。

#### POST /api/v1/docker/volumes

创建数据卷。

**请求体：**
```json
{
  "name": "cloudnexus-data"
}
```

### 3.5 集群节点

#### GET /api/v1/cluster/nodes

查看集群节点状态 (需管理员权限)。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": [
    {
      "name": "node1",
      "host": "192.168.1.10",
      "status": "healthy",
      "containers": 5
    },
    {
      "name": "node2",
      "host": "192.168.1.11",
      "status": "healthy",
      "containers": 3
    }
  ]
}
```

#### POST /api/v1/cluster/join

新节点加入集群 (需管理令牌)。

**请求体：**
```json
{
  "name": "node3",
  "host": "192.168.1.12",
  "tls_cert": "-----BEGIN CERTIFICATE-----..."
}
```

---

## 四、管理后台 (需管理员权限)

> 所有管理后台接口需要管理员角色，由 `AdminRequired` 中间件校验。
> 管理员通过 `users` 表中 `role` 字段标识（`user` / `admin`）。

### 4.1 系统状态

#### GET /api/v1/admin/stats

获取系统运行状态概览 (需管理员)。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "services": {
      "user-file-svc": "healthy",
      "im-svc": "healthy",
      "docker-svc": "healthy"
    },
    "online_users": 5,
    "total_users": 32,
    "total_files": 128,
    "total_conversations": 15,
    "db_connections": 8,
    "redis_latency_ms": 2,
    "uptime_seconds": 86400
  }
}
```

#### GET /api/v1/admin/stats/history

获取历史统计指标 (需管理员)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| from | string | 起始时间 (ISO 8601) |
| to | string | 结束时间 (ISO 8601) |
| interval | string | 聚合间隔，默认 1h |

### 4.2 服务日志

日志系统提供环形缓冲 (实时) + 文件日志 (持久化) 双通道查询。

#### GET /system/log/services

获取可查询的日志服务列表 (需管理员)。

#### POST /system/log/query

查询环形缓冲区日志 (需管理员, 实时, 最多 2048 条)。

**请求体：**
```json
{
  "service": "im-svc",
  "level": "error",
  "request_id": "uuid-xxx",
  "user_id": "2051225055077076992"
}
```

#### POST /system/log/read

读取文件日志 (需管理员)。

**请求体：**
```json
{
  "date": "2026-05-06",
  "service": "user-file-svc",
  "filename": "user-file-svc.log"
}
```

#### GET /system/log/files

列出日志文件 (需管理员)。查询参数: `?service=user-file-svc`

#### GET /system/log/download

下载日志文件 (需管理员)。查询参数: `?service=user-file-svc&date=2026-05-06&filename=user-file-svc.log`

### 4.3 用户管理

#### GET /api/v1/admin/users

列出所有用户 (需管理员)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 20 |

**响应 (200)：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "2051225055077076992",
        "username": "testuser1",
        "email": "test1@test.com",
        "is_admin": false,
        "status": 1,
        "created_at": "2026-05-04T09:12:00Z"
      }
    ],
    "total": 32
  }
}
```

#### PUT /api/v1/admin/users/:id/admin

切换用户管理员状态 (需管理员)。

#### PUT /api/v1/admin/users/:id/status

启用/禁用用户 (需管理员)。

---

## 健康检查

所有服务提供统一健康检查端点：

```
GET /healthz → 200 OK
```

集群负载均衡器可使用此端点做故障节点自动摘除。

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| v1.0.0 | 2026-05-06 | 聊天记录备份/恢复 (导出 JSON + SHA256 校验码 + 导入去重) |
| v1.0.0 | 2026-05-06 | Phase 2.5 完成：图片/视频消息、链接预览卡片、跨节点 IM Redis Pub/Sub、数据库迁移系统 |
| v0.9.0 | 2026-05-06 | Docker 镜像管理/容器监控 API、用户设置 API、文件移动/复制 API、分享链接 API |
| v0.8.0 | 2026-05-05 | 文件移动/复制、分享管理、暖色主题 |
| v0.7.0 | 2026-05-04 | 新增文件批量删除/下载 API、管理后台 API（系统状态/日志/用户管理） |
| v0.6.0 | 2026-05-04 | 全容器化部署 (Go 服务 + 前端 Docker 化)，仅 nginx 80 端口对外 |
| v0.5.0 | 2026-05-04 | ID 类型安全 (Go json:,string + TS number→string)、Nginx Docker 统一入口 |
| v0.1.0 | 2026-05-03 | 初始版本，定义三个服务全部接口 |
