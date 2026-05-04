# CloudNexus 测试数据参考

## 测试账号

所有 ID 由 Snowflake 算法自动生成，测试时以实际返回值为准。

| 用户名 | 密码 | 备注 |
|--------|------|------|
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
