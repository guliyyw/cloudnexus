# CloudNexus API 接口文档

> 版本：v0.3.0 | 更新：2026-05-04

## 通用约定

### 请求格式

- Content-Type: `application/json`
- 认证方式：Bearer Token (JWT)，通过 `Authorization: Bearer <access_token>` 传递

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
    "id": 1,
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
    "expires_in": 28800
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
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "avatar": "https://static.example.com/avatars/alice.png",
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

上传文件 (需认证, multipart/form-data)。支持单文件和批量上传。

**表单字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File (可多次) | 上传的文件，批量上传时多次 append |
| parent_id | int | 父目录 ID，0 为根目录 |

**单文件响应 (201)：**
```json
{
  "code": 201,
  "message": "uploaded",
  "data": {
    "id": 100,
    "name": "report.pdf",
    "size": 204800,
    "mime_type": "application/pdf",
    "parent_id": 0,
    "created_at": "2026-05-03T10:00:00Z"
  }
}
```

**批量上传响应 (201)：**
```json
{
  "code": 201,
  "message": "uploaded",
  "data": {
    "files": [
      {
        "id": 100,
        "name": "report.pdf",
        "size": 204800,
        "mime_type": "application/pdf"
      }
    ],
    "errors": ["bigfile.mp4: 文件过大"],
    "total": 3,
    "ok": 2
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

#### POST /api/v1/file/mkdir

创建目录 (需认证)。

**请求体：**
```json
{
  "name": "documents",
  "parent_id": 0
}
```

#### GET /api/v1/file/search

搜索文件 (需认证)。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| q | string | 搜索关键词 |
| page | int | 页码 |
| page_size | int | 每页数量 |

### 1.4 WebDAV

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
      "id": 1,
      "type": "private",
      "name": "Bob",
      "last_message": {
        "content": "好的，明天见",
        "sender_id": 2,
        "sent_at": "2026-05-03T09:55:00Z"
      },
      "unread_count": 3,
      "created_at": "2026-05-03T08:00:00Z"
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
      "id": 500,
      "conversation_id": 1,
      "sender_id": 1,
      "content": "你好！",
      "type": "text",
      "created_at": "2026-05-03T09:00:00Z"
    }
  ]
}
```

#### DELETE /api/v1/im/conversations/:id

删除会话 (需认证)。当前用户视角的软删除，不影响其他成员。

**响应 (200)：**
```json
{
  "code": 200,
  "message": "deleted"
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
2. 节点 A 将消息发布到 Redis 频道 `im:conversation:{id}`
3. 所有节点订阅该频道
4. 节点 B 检查本地是否有接收方在线，有则推送

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
| v0.3.0 | 2026-05-04 | JWT TTL 延长至8小时、批量上传/预览/会话删除/好友系统前后端联调完成 |
| v0.2.0 | 2026-05-04 | 新增批量上传、文件预览、会话删除、好友系统接口 |
| v0.1.0 | 2026-05-03 | 初始版本，定义三个服务全部接口 |
