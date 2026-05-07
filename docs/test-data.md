# CloudNexus 测试数据参考

## 测试账号

所有 ID 由 Snowflake 算法自动生成，测试时以实际返回值为准。

### 默认管理员

首次启动 user-file-svc 时，若 `users` 表为空，会自动创建默认管理员账号：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `DEFAULT_ADMIN_USERNAME` | `admin` | 管理员用户名 |
| `DEFAULT_ADMIN_PASSWORD` | `CloudNexus@admin` | 管理员密码 |
| `DEFAULT_ADMIN_EMAIL` | `admin@cloudnexus.local` | 管理员邮箱 |

> 这些值可通过 Docker Compose 的 `environment` 或 K8s env vars 修改。
> 仅当数据库中无任何用户时才会触发种子创建，不会覆盖已有数据。

### 手动测试账号

| 用户名 | 密码 | 备注 |
|--------|------|------|
| admin | CloudNexus@admin | 默认管理员 (自动种子) |
| testuser | 123456 | 默认测试账号 |
| alice | alice123 | |
| bob | bob123 | |

## 启动全栈

```bash
# 1. 构建前端
cd client && npm run build

# 2. 启动所有服务 (Go + 前端 + 基础设施)
cd ../deploy
docker compose -f docker-compose.single.yml up --build -d

# 3. 访问
# http://localhost (唯一入口，所有流量经 nginx)
```

## 服务一览

所有服务在 Docker 中运行，仅 nginx 80 端口对外：

| 服务 | 端口 | 对外 | 说明 |
|------|------|------|------|
| nginx | 80 | 是 | 统一入口 + 静态文件 |
| user-file-svc | 8081 | 否 | 用户 & 文件 |
| im-svc | 8082 | 否 | 即时通讯 & WebSocket |
| docker-svc | 8083 | 否 | Docker 管理 |
| PostgreSQL | 5432 | 否 | 数据库 (cloudnexus/cloudnexus) |
| Redis | 6379 | 否 | 缓存 (无密码) |
| MinIO | 9000/9001 | 否 | 对象存储 (minioadmin/minioadmin) |

## 快速测试命令

所有命令通过 Nginx (端口 80) 访问。ID 使用占位符 `{id}`，请替换为实际 Snowflake 生成的字符串 ID。

### 用户认证

```bash
# 注册
curl -X POST http://localhost/api/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@test.com","password":"alice123"}'

# 登录（获取 token）
curl -X POST http://localhost/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"123456"}'

# 获取个人信息
curl http://localhost/api/v1/user/profile \
  -H "Authorization: Bearer {access_token}"
```

### 文件管理

```bash
# 上传文件
curl -X POST http://localhost/api/v1/file/upload \
  -H "Authorization: Bearer {token}" \
  -F "file=@test.txt" -F "parent_id=0"

# 文件列表
curl "http://localhost/api/v1/file/list?parent_id=0&page=1&page_size=20" \
  -H "Authorization: Bearer {token}"

# 下载文件
curl -o output http://localhost/api/v1/file/download/{id} \
  -H "Authorization: Bearer {token}"

# 在线预览 (图片/视频/音频/PDF)
curl "http://localhost/api/v1/file/download/{id}?inline=true&token={token}" -o output

# 搜索文件
curl "http://localhost/api/v1/file/search?q=test&page=1&page_size=20" \
  -H "Authorization: Bearer {token}"

# 创建目录
curl -X POST http://localhost/api/v1/file/mkdir \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"name":"docs","parent_id":0}'

# 删除文件
curl -X DELETE http://localhost/api/v1/file/{id} \
  -H "Authorization: Bearer {token}"
```

### 即时通讯

```bash
# 会话列表
curl http://localhost/api/v1/im/conversations \
  -H "Authorization: Bearer {token}"

# 创建私聊会话
curl -X POST http://localhost/api/v1/im/conversations \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"type":"private","member_ids":[{friend_id}]}'

# 获取消息历史
curl "http://localhost/api/v1/im/conversations/{id}/messages?limit=50" \
  -H "Authorization: Bearer {token}"

# WebSocket 连接
# wscat -c "ws://localhost/ws?token={access_token}"
```

### 好友管理

```bash
# 发送好友请求
curl -X POST http://localhost/api/v1/im/friends/requests \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"friend_name":"bob"}'

# 好友列表
curl http://localhost/api/v1/im/friends \
  -H "Authorization: Bearer {token}"

# 待处理请求
curl http://localhost/api/v1/im/friends/requests \
  -H "Authorization: Bearer {token}"

# 接受请求
curl -X PUT http://localhost/api/v1/im/friends/requests/{id}/accept \
  -H "Authorization: Bearer {token}"

# 删除好友
curl -X DELETE http://localhost/api/v1/im/friends/{friend_id} \
  -H "Authorization: Bearer {token}"
```

### 链接预览

```bash
# 获取链接 OG 元数据
curl -X POST http://localhost/api/v1/im/link-preview \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://github.com"}'
```

### 聊天记录备份/恢复

```bash
# 导出会话消息 (JSON 文件下载)
curl -o chat_export.json "http://localhost/api/v1/im/conversations/{id}/export" \
  -H "Authorization: Bearer {token}"

# 导入聊天记录 (multipart/form-data)
curl -X POST http://localhost/api/v1/im/conversations/import \
  -H "Authorization: Bearer {token}" \
  -F "file=@chat_export.json"

# 校验码测试：修改 JSON 文件中的 checksum 后导入应失败
```

前端操作：
1. 进入聊天页面 → 选择一个会话 → 点击导出按钮 (↓) → 自动下载 JSON 并保存到云盘 `聊天记录/{私聊|群聊}/`
2. 点击导入按钮 (↑) → 选择 JSON 文件 → 显示导入结果 (新增/跳过/总计)

### 图片消息

在聊天页面中：
1. 点击输入框旁的图片按钮 → 选择图片/视频文件 → 自动上传并发送
2. 或直接粘贴剪贴板中的图片到输入框 → 自动上传并发送
3. 图片消息内联渲染 (最大 320x320)，点击可全屏查看
4. 视频消息支持播放控件

### Docker 管理

```bash
# 容器列表
curl http://localhost/api/v1/docker/containers \
  -H "Authorization: Bearer {token}"

# 启停容器
curl -X POST http://localhost/api/v1/docker/containers/{id}/start \
  -H "Authorization: Bearer {token}"

# 容器日志
curl "http://localhost/api/v1/docker/containers/{id}/logs?tail=100" \
  -H "Authorization: Bearer {token}"
```
