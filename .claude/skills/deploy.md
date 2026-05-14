---
name: deploy
description: 将代码变更部署到远程服务器 (121.43.145.157)
argument-hint: "all | user-file-svc | frontend | 或留空自动检测"
---

# Deploy Skill — CloudNexus 部署

将代码变更编译并部署到生产服务器 (121.43.145.157)。

## 部署流程

### 1. 确定部署范围

- 如果用户指定了服务名 (`user-file-svc`, `im-svc`, `docker-svc`, `camera-svc`)，只部署该服务
- 如果用户说 `frontend` / `前端`，只部署前端
- 如果用户说 `all` / `全部`，全量部署
- 如果未指定，运行 `git diff --name-only HEAD~1 HEAD` 检测变更，根据变更路径决定：
  - `server/cmd/user-file-svc/`, `server/internal/userfile/`, `server/pkg/` → user-file-svc
  - `server/cmd/im-svc/`, `server/internal/im/` → im-svc
  - `server/cmd/docker-svc/`, `server/internal/dockermgr/` → docker-svc
  - `server/cmd/camera-svc/`, `server/internal/camera/` → camera-svc
  - `client/` → 前端
  - `deploy/nginx/` → nginx 配置

### 2. 构建

**Go 服务:**
```bash
cd D:/code/cloudnexus/server && \
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -ldflags="-s -w" -o /tmp/<service-name> ./cmd/<service-name>
```

**前端:**
```bash
cd D:/code/cloudnexus/client && npm run build
cd D:/code/cloudnexus/client/dist && tar czf /tmp/dist_new.tar.gz .
```

### 3. 上传到服务器

使用 mcp__sshmcp__upload_file:
- server_id: `cloudnexus-prod`
- Go 二进制: local_path `/tmp/<service-name>`, remote_path `/home/user/cloudnexus/deploy/service-bins/<service-name>`
- 前端: local_path `/tmp/dist_new.tar.gz`, remote_path `/tmp/dist_new.tar.gz`

### 4. 重启服务

**Go 服务 (推荐方式 — 快速替换二进制):**
```bash
# 在服务器上执行:
cd /home/user/cloudnexus/deploy
docker compose -f docker-compose.single.yml stop <service-name>
docker compose -f docker-compose.single.yml create <service-name>
docker cp /home/user/cloudnexus/deploy/service-bins/<service-name> $(docker compose -f docker-compose.single.yml ps -q <service-name>):/app/service
docker start $(docker compose -f docker-compose.single.yml ps -a -q <service-name> | head -1)
```

**前端:**
```bash
# 在服务器上执行:
tar xzf /tmp/dist_new.tar.gz -C /home/user/cloudnexus/client/dist_new/
cd /home/user/cloudnexus/client
rm -rf dist_old 2>/dev/null
mv dist dist_old 2>/dev/null
mv dist_new dist
cd /home/user/cloudnexus/deploy
docker compose -f docker-compose.single.yml restart nginx
```

**nginx 配置:**
```bash
cd /home/user/cloudnexus/deploy
docker compose -f docker-compose.single.yml restart nginx
```

### 5. 验证

部署后运行健康检查:
```bash
curl -s http://localhost/healthz/<service-name>
```

如果 nginx 无法解析容器名 (新容器)，需要先启动服务再启动 nginx:
```bash
docker start $(docker compose -f docker-compose.single.yml ps -a -q <service-name> | head -1)
sleep 2
docker compose -f docker-compose.single.yml restart nginx
```

### 6. 汇报结果

简洁汇报: 部署了什么、版本/commit、健康状态。
