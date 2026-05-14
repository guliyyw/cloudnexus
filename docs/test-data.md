# CloudNexus 测试数据参考

> 更新：2026-05-15

## 测试账号

所有 ID 由 Snowflake 算法自动生成，测试时以实际返回值为准。

### 默认管理员

首次启动 user-file-svc 时，若 `users` 表为空，会自动创建默认管理员账号并初始化 RBAC 角色权限：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `DEFAULT_ADMIN_USERNAME` | `admin` | 管理员用户名 |
| `DEFAULT_ADMIN_PASSWORD` | `CloudNexus@admin` | 管理员密码 |
| `DEFAULT_ADMIN_EMAIL` | `admin@cloudnexus.local` | 管理员邮箱 |

> 这些值可通过 Docker Compose 的 `environment` 修改。仅当数据库中无任何用户时才会触发种子创建。

### 手动测试账号

| 用户名 | 密码 | 备注 |
|--------|------|------|
| admin | CloudNexus@admin | 默认管理员 (super_admin 角色) |
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
| mediamtx HLS | 8888 | 是 | HLS 视频流 (直连，绕过 cookie) |
| user-file-svc | 8081 | 否 | 用户 & 文件 & RBAC |
| im-svc | 8082 | 否 | 即时通讯 & WebSocket |
| docker-svc | 8083 | 否 | Docker 管理 |
| camera-svc | 8085 | 否 | 摄像头 & AI 识别 |
| collab-svc | 8086 | 否 | 在线文档协作 |
| PostgreSQL | 5432 | 否 | 数据库 (cloudnexus/cloudnexus) |
| Redis | 6379 | 否 | 缓存 (无密码) |
| MinIO | 9000/9001 | 否 | 对象存储 (minioadmin/minioadmin) |
| MediaMTX API | 8889 | 否 | 流媒体管理 |
| AI Inference | 8000 | 否 | YOLOv8 推理 |

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

# 刷新令牌
curl -X POST http://localhost/api/v1/user/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"{refresh_token}"}'

# 修改密码
curl -X PUT http://localhost/api/v1/user/password \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"123456","new_password":"newpass"}'

# 忘记密码 (发送重置邮件)
curl -X POST http://localhost/api/v1/user/password/forgot \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@test.com"}'

# 重置密码 (使用邮件中的 token)
curl -X POST http://localhost/api/v1/user/password/reset \
  -H "Content-Type: application/json" \
  -d '{"token":"{reset_token}","new_password":"newpass"}'

# 用户搜索
curl "http://localhost/api/v1/user/search?q=ali" \
  -H "Authorization: Bearer {token}"

# 隐私设置
curl -X PUT http://localhost/api/v1/user/privacy \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"allow_search":true,"allow_add_friend":true,"show_online":true}'
```

### 邮箱/手机验证

```bash
# 发送邮箱验证码
curl -X POST http://localhost/api/v1/user/email/send-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","type":"register"}'

# 验证邮箱
curl -X POST http://localhost/api/v1/user/email/verify \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}'
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

# 删除文件 (移入回收站)
curl -X DELETE http://localhost/api/v1/file/{id} \
  -H "Authorization: Bearer {token}"

# 批量删除
curl -X POST http://localhost/api/v1/file/batch-delete \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"ids":["id1","id2"]}'

# 批量下载 (ZIP)
curl -X POST http://localhost/api/v1/file/batch-download \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"ids":["id1","id2"]}' -o batch.zip

# 移动文件
curl -X POST http://localhost/api/v1/file/move \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"ids":["id1"],"target_parent_id":"dir_id"}'

# 复制文件
curl -X POST http://localhost/api/v1/file/copy \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"ids":["id1"],"target_parent_id":"dir_id"}'
```

### 分块上传

```bash
# 初始化分块上传
curl -X POST http://localhost/api/v1/file/chunk/init \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"large.zip","file_size":52428800,"chunk_size":10485760,"parent_id":"0"}'

# 上传分块
curl -X POST http://localhost/api/v1/file/chunk/upload \
  -H "Authorization: Bearer {token}" \
  -F "upload_id={upload_id}" -F "chunk_index=0" -F "chunk=@chunk_0"

# 查询进度
curl "http://localhost/api/v1/file/chunk/status/{upload_id}" \
  -H "Authorization: Bearer {token}"

# 完成上传
curl -X POST http://localhost/api/v1/file/chunk/complete \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"upload_id":"{upload_id}"}'

# 取消上传
curl -X DELETE http://localhost/api/v1/file/chunk/cancel/{upload_id} \
  -H "Authorization: Bearer {token}"

# 未完成上传列表
curl http://localhost/api/v1/file/chunk/incomplete \
  -H "Authorization: Bearer {token}"
```

### 回收站

```bash
# 回收站列表
curl http://localhost/api/v1/file/trash \
  -H "Authorization: Bearer {token}"

# 恢复文件
curl -X POST http://localhost/api/v1/file/trash/{id}/restore \
  -H "Authorization: Bearer {token}"

# 永久删除
curl -X DELETE http://localhost/api/v1/file/trash/{id} \
  -H "Authorization: Bearer {token}"

# 清空回收站
curl -X DELETE http://localhost/api/v1/file/trash \
  -H "Authorization: Bearer {token}"
```

### 文件分享

```bash
# 创建分享链接
curl -X POST http://localhost/api/v1/file/{id}/share \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"password":"1234","expires_in_hours":24,"download_limit":10}'

# 查看文件的所有分享
curl http://localhost/api/v1/file/{id}/shares \
  -H "Authorization: Bearer {token}"

# 我的分享列表
curl http://localhost/api/v1/shares/my \
  -H "Authorization: Bearer {token}"

# 删除分享
curl -X DELETE http://localhost/api/v1/shares/{share_id} \
  -H "Authorization: Bearer {token}"

# 公开访问分享 (无需认证)
curl http://localhost/api/v1/share/{share_code}

# 验证分享密码
curl -X POST http://localhost/api/v1/share/{share_code}/verify \
  -H "Content-Type: application/json" \
  -d '{"password":"1234"}'

# 下载分享文件
curl -o output "http://localhost/api/v1/share/{share_code}/download?token={verify_token}"
```

### 文件版本

```bash
# 查看文件版本历史
curl http://localhost/api/v1/file/{id}/versions \
  -H "Authorization: Bearer {token}"

# 恢复某个版本
curl -X POST http://localhost/api/v1/file/{id}/versions/{version_id}/restore \
  -H "Authorization: Bearer {token}"

# 下载某个版本
curl -o old_version "http://localhost/api/v1/file/{id}/versions/{version_id}/download?token={token}"
```

### 存储配额

```bash
# 查看配额
curl http://localhost/api/v1/user/quota \
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
  -d '{"friend_name":"bob","message":"hello"}'

# 好友列表
curl http://localhost/api/v1/im/friends \
  -H "Authorization: Bearer {token}"

# 待处理请求
curl http://localhost/api/v1/im/friends/requests \
  -H "Authorization: Bearer {token}"

# 接受请求
curl -X PUT http://localhost/api/v1/im/friends/requests/{id}/accept \
  -H "Authorization: Bearer {token}"

# 设置备注
curl -X PUT http://localhost/api/v1/im/friends/{friend_id}/remark \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"remark":"小明"}'

# 拉黑用户
curl -X POST http://localhost/api/v1/im/friends/{friend_id}/block \
  -H "Authorization: Bearer {token}"

# 取消拉黑
curl -X DELETE http://localhost/api/v1/im/friends/{friend_id}/block \
  -H "Authorization: Bearer {token}"

# 拉黑列表
curl http://localhost/api/v1/im/blocklist \
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
```

### Docker 管理

```bash
# 端点列表
curl http://localhost/api/v1/docker/endpoints \
  -H "Authorization: Bearer {token}"

# 容器列表
curl "http://localhost/api/v1/docker/containers?endpoint=local" \
  -H "Authorization: Bearer {token}"

# 启停容器
curl -X POST http://localhost/api/v1/docker/containers/{id}/start \
  -H "Authorization: Bearer {token}"

# 容器日志
curl "http://localhost/api/v1/docker/containers/{id}/logs?tail=100" \
  -H "Authorization: Bearer {token}"

# 容器资源监控
curl "http://localhost/api/v1/docker/containers/{id}/stats" \
  -H "Authorization: Bearer {token}"

# 镜像列表
curl http://localhost/api/v1/docker/images \
  -H "Authorization: Bearer {token}"
```

### 摄像头管理

```bash
# 摄像头列表
curl http://localhost/api/v1/cameras \
  -H "Authorization: Bearer {token}"

# 添加摄像头
curl -X POST http://localhost/api/v1/cameras \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"name":"门口","stream_url":"rtsp://192.168.1.50:554/stream","protocol":"rtsp"}'

# 启动视频流
curl -X POST http://localhost/api/v1/cameras/{id}/stream/start \
  -H "Authorization: Bearer {token}"

# 停止视频流
curl -X POST http://localhost/api/v1/cameras/{id}/stream/stop \
  -H "Authorization: Bearer {token}"

# 启动 AI 识别
curl -X POST http://localhost/api/v1/cameras/{id}/recognition/start \
  -H "Authorization: Bearer {token}"

# 停止 AI 识别
curl -X POST http://localhost/api/v1/cameras/{id}/recognition/stop \
  -H "Authorization: Bearer {token}"

# 检测事件列表
curl http://localhost/api/v1/cameras/{id}/events \
  -H "Authorization: Bearer {token}"

# 图片检测
curl -X POST http://localhost/api/v1/detect-image \
  -H "Authorization: Bearer {token}" \
  -F "image=@photo.jpg"

# 视频文件分析
curl -X POST "http://localhost/api/v1/detect-video?interval=2.0" \
  -H "Authorization: Bearer {token}" \
  -F "video=@test.mp4"

# 局域网摄像头发现
curl -X POST http://localhost/api/v1/cameras/discover \
  -H "Authorization: Bearer {token}"
```

### 人脸识别

```bash
# 人脸库列表
curl http://localhost/api/v1/faces \
  -H "Authorization: Bearer {token}"

# 注册人脸
curl -X POST http://localhost/api/v1/faces \
  -H "Authorization: Bearer {token}" \
  -F "name=张三" -F "thumbnail=@face.jpg" -F "embedding=[0.12,-0.34,...]"

# 人脸匹配
curl -X POST http://localhost/api/v1/faces/match \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"embedding":[0.12,-0.34,...],"camera_id":"{camera_id}"}'

# 获取人脸缩略图
curl "http://localhost/api/v1/faces/{id}/thumbnail?token={token}" -o thumb.jpg

# 删除人脸
curl -X DELETE http://localhost/api/v1/faces/{id} \
  -H "Authorization: Bearer {token}"

# 人脸识别事件列表
curl http://localhost/api/v1/cameras/{id}/faces \
  -H "Authorization: Bearer {token}"

# 清空识别事件
curl -X DELETE http://localhost/api/v1/cameras/{id}/faces \
  -H "Authorization: Bearer {token}"
```

### 考勤查询

```bash
# 每日考勤汇总
curl "http://localhost/api/v1/faces/attendance/daily?date=2026-05-15" \
  -H "Authorization: Bearer {token}"

# 某人考勤记录
curl "http://localhost/api/v1/faces/attendance?face_id={face_id}&date=2026-05-15" \
  -H "Authorization: Bearer {token}"

# 人员签到状态 (所有已注册人员)
curl "http://localhost/api/v1/faces/attendance/status?date=2026-05-15" \
  -H "Authorization: Bearer {token}"

# 删除考勤记录
curl -X DELETE "http://localhost/api/v1/faces/attendance/{id}" \
  -H "Authorization: Bearer {token}"
```

### 在线文档

```bash
# 文档列表
curl http://localhost/api/v1/collab \
  -H "Authorization: Bearer {token}"

# 创建文档
curl -X POST http://localhost/api/v1/collab \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"name":"会议记录","content":""}'

# 协作编辑 (WebSocket)
# 前端使用: ws://localhost/ws/collab/{doc_id}?token={token}
```

### 管理后台 (admin 权限)

```bash
# 用户列表
curl http://localhost/api/v1/admin/users \
  -H "Authorization: Bearer {admin_token}"

# 系统资源指标
curl http://localhost/api/v1/admin/metrics/resources \
  -H "Authorization: Bearer {admin_token}"

# 节点列表
curl "http://localhost/api/v1/admin/nodes?service=&host=&type=&status=" \
  -H "Authorization: Bearer {admin_token}"

# 节点在线会话
curl http://localhost/api/v1/admin/nodes/{name}/sessions \
  -H "Authorization: Bearer {admin_token}"

# 告警规则列表
curl http://localhost/api/v1/admin/alerts/rules \
  -H "Authorization: Bearer {admin_token}"

# 创建告警规则
curl -X POST http://localhost/api/v1/admin/alerts/rules \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{"name":"节点离线告警","enabled":true,"trigger_type":"status_change","webhook_url":"https://hooks.example.com/alert","cooldown_seconds":300}'

# 告警历史
curl http://localhost/api/v1/admin/alerts/history \
  -H "Authorization: Bearer {admin_token}"

# 角色列表
curl http://localhost/api/v1/admin/roles \
  -H "Authorization: Bearer {admin_token}"

# 权限列表
curl http://localhost/api/v1/admin/roles/permissions \
  -H "Authorization: Bearer {admin_token}"

# 配额等级列表
curl http://localhost/api/v1/admin/quota/tiers \
  -H "Authorization: Bearer {admin_token}"

# 设置用户配额
curl -X PUT http://localhost/api/v1/admin/users/{id}/quota \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{"storage_limit":1073741824,"tier_id":"{tier_id}"}'

# 系统配置
curl http://localhost/api/v1/admin/config \
  -H "Authorization: Bearer {admin_token}"
```

### 前端浏览器测试要点

1. **文件管理**：上传、下载、预览、搜索、创建目录、批量操作、拖拽移动、回收站
2. **分享**：创建分享（密码/有效期/次数限制）、公开访问 /s/:code、密码验证、预览/下载
3. **聊天**：WebSocket 连接、文字/图片/视频/文件消息、群聊、好友、拉黑、在线状态
4. **Docker**：容器列表、启停、日志、镜像管理、端点选择
5. **摄像头**：列表 CRUD、实时 HLS 播放、AI 检测控制
6. **人脸**：人脸注册、实时识别、考勤签到/汇总
7. **在线文档**：文档 CRUD、TipTap 协作编辑、Markdown 增强
8. **管理后台**：用户管理、角色权限、系统状态、日志查看、集群节点、告警规则、配额等级
