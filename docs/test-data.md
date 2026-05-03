# CloudNexus 测试数据参考

## 测试账号

| 用户名 | 密码 | 用户 ID | 备注 |
|--------|------|---------|------|
| testuser | 123456 | 1 | 默认测试账号 |
| alice | alice123 | 4 | |
| bob | bob123 | 5 | |

## 基础设施服务

| 服务 | 地址 | 端口 | 账号/密码 |
|------|------|------|-----------|
| PostgreSQL | localhost | 5432 | cloudnexus / cloudnexus |
| Redis | localhost | 6379 | (无密码) |
| MinIO API | localhost | 9000 | minioadmin / minioadmin |
| MinIO Console | localhost | 9001 | minioadmin / minioadmin |

启动基础设施：

```bash
cd deploy
docker compose up -d
```

## 后端服务端口

| 服务 | 端口 | 健康检查 |
|------|------|----------|
| user-file-svc | 8081 | `GET /healthz` |
| im-svc | 8082 | `GET /healthz` |
| docker-svc | 8083 | `GET /healthz` |

## 前端

| 服务 | 端口 | 地址 |
|------|------|------|
| Vite Dev Server | 3000 | http://localhost:3000 |

## 快速测试命令

### 用户认证

```bash
# 注册
curl -X POST http://localhost:8081/api/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@test.com","password":"alice123"}'

# 登录（获取 token）
curl -X POST http://localhost:8081/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"123456"}'

# 获取个人信息
curl http://localhost:8081/api/v1/user/profile \
  -H "Authorization: Bearer <access_token>"
```

### 文件管理

```bash
# 上传文件
curl -X POST http://localhost:8081/api/v1/file/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@test.txt" -F "parent_id=0"

# 文件列表
curl "http://localhost:8081/api/v1/file/list?parent_id=0&page=1&page_size=20" \
  -H "Authorization: Bearer <token>"

# 下载文件
curl -O http://localhost:8081/api/v1/file/download/1 \
  -H "Authorization: Bearer <token>"

# 搜索文件
curl "http://localhost:8081/api/v1/file/search?q=test&page=1&page_size=20" \
  -H "Authorization: Bearer <token>"

# 创建目录
curl -X POST http://localhost:8081/api/v1/file/mkdir \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"docs","parent_id":0}'

# 删除文件
curl -X DELETE http://localhost:8081/api/v1/file/1 \
  -H "Authorization: Bearer <token>"
```

### 即时通讯

```bash
# 会话列表
curl http://localhost:8082/api/v1/im/conversations \
  -H "Authorization: Bearer <token>"

# 创建私聊会话
curl -X POST http://localhost:8082/api/v1/im/conversations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"type":"private","member_ids":[5]}'

# 获取消息历史
curl "http://localhost:8082/api/v1/im/conversations/1/messages?limit=50" \
  -H "Authorization: Bearer <token>"

# WebSocket 连接
# wscat -c "ws://localhost:8082/ws?token=<access_token>"
```

### Docker 管理

```bash
# 容器列表
curl http://localhost:8083/api/v1/docker/containers \
  -H "Authorization: Bearer <token>"

# 启停容器
curl -X POST http://localhost:8083/api/v1/docker/containers/<id>/start \
  -H "Authorization: Bearer <token>"

# 容器日志
curl "http://localhost:8083/api/v1/docker/containers/<id>/logs?tail=100" \
  -H "Authorization: Bearer <token>"
```
