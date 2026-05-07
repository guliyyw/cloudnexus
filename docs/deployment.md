# CloudNexus 部署指南

> 版本：v1.0.0 | 更新：2026-05-06

## 1. 部署模式概述

| 模式 | 适用场景 | 复杂度 | 可用性 |
|------|----------|--------|--------|
| 单机 | 个人使用、小团队 (< 50 人) | 低 | 单点 |
| 集群 | 中大型团队、生产环境 | 中 | 高可用 |

---

## 2. 单机部署 (Docker Compose)

### 2.1 环境要求

- Linux / macOS / Windows (Docker Desktop)
- Docker 24+
- Docker Compose v2
- 至少 4 GB 可用内存
- 至少 20 GB 可用磁盘

### 2.2 架构说明

所有服务（Go 后端 + 前端 + 基础设施）都运行在 Docker 中，仅 nginx 暴露 80 端口：

```
http://localhost:80 (唯一入口)
    │
  nginx (静态文件 + 反向代理)
    │
    ├── /api/*          → user-file-svc:8081 (用户、文件)
    ├── /api/v1/im/*    → im-svc:8082 (即时通讯 REST)
    ├── /api/v1/docker/* → docker-svc:8083 (Docker 管理)
    ├── /ws             → im-svc:8082 (WebSocket)
    └── /               → client/dist 静态文件 (SPA)
```

### 2.3 快速启动

```bash
# 1. 构建前端 (生成 client/dist/)
cd client
npm install
npm run build

# 2. 启动全栈 (包含构建 Go 服务镜像)
cd ../deploy
docker compose -f docker-compose.single.yml up --build -d

# 3. 验证
curl http://localhost/healthz
# 浏览器打开 http://localhost
```

首次启动会构建 Go 服务 Docker 镜像（约 2-3 分钟），后续启动只需 `docker compose up -d`。

### 2.4 端口规划

仅 nginx 对外暴露 80 端口，其他端口均为 Docker 内部通信或开发调试用：

| 服务 | 端口 | 对外 | 说明 |
|------|------|------|------|
| **nginx** | **80** | **是** | 唯一对外端口，统一入口 |
| user-file-svc | 8081 | 否 | 用户 & 文件 |
| im-svc | 8082 | 否 | 即时通讯 & WebSocket |
| docker-svc | 8083 | 否 | Docker 管理 |
| PostgreSQL | 5432 | 否 | 数据库 |
| Redis | 6379 | 否 | 缓存 / 消息总线 |
| MinIO API | 9000 | 否 | 对象存储 S3 API |
| MinIO Console | 9001 | 否 | MinIO Web 管理 |

### 2.5 Nginx 路由规则

| 路径 | 目标 | 说明 |
|------|------|------|
| `/api/v1/im/*` | im-svc:8082 | IM REST API |
| `/api/v1/docker/*` | docker-svc:8083 | Docker 管理 API |
| `/api/*` | user-file-svc:8081 | 用户/文件 API (兜底) |
| `/ws` | im-svc:8082 | WebSocket (需 Upgrade 头) |
| `/` | 静态文件 | SPA (try_files 回退到 index.html) |

配置文件：`deploy/nginx/nginx.conf`

### 2.6 配置文件

**Docker 部署**使用 `server/config/config.docker.yaml`：

```yaml
database:
  dsn: "host=postgres user=cloudnexus password=cloudnexus dbname=cloudnexus port=5432 sslmode=disable"

redis:
  addr: "redis:6379"

minio:
  endpoint: "minio:9000"

jwt:
  access_secret: "change-me-in-production"
  refresh_secret: "change-me-in-production-refresh"
```

所有 host 使用 Docker 服务名（`postgres`、`redis`、`minio`），通过 Docker 内部 DNS 解析。

**宿主机开发**使用 `config.single.yaml`（host 均为 `localhost`）。

### 2.7 默认管理员账号

系统首次启动时会自动创建默认管理员（仅当 `users` 表为空）：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `DEFAULT_ADMIN_USERNAME` | `admin` | 管理员用户名 |
| `DEFAULT_ADMIN_PASSWORD` | `CloudNexus@admin` | 管理员密码 |
| `DEFAULT_ADMIN_EMAIL` | `admin@cloudnexus.local` | 管理员邮箱 |

在 `docker-compose.single.yml` 中修改 `user-file-svc` 的 environment 即可自定义。
生产环境请务必修改默认密码。

### 2.8 Docker Compose 服务一览

| 服务 | 来源 | 说明 |
|------|------|------|
| postgres | `postgres:15-alpine` | 数据库 |
| redis | `redis:7-alpine` | 缓存/消息 |
| minio | `minio/minio:latest` | 对象存储 |
| user-file-svc | 构建 `../server` (SERVICE=user-file-svc) | Go 服务 |
| im-svc | 构建 `../server` (SERVICE=im-svc) | Go 服务 |
| docker-svc | 构建 `../server` (SERVICE=docker-svc) | Go 服务，挂载 docker.sock |
| nginx | `nginx:alpine` | 入口 + 静态文件 |

### 2.9 常用命令

```bash
# 查看日志
docker compose -f deploy/docker-compose.single.yml logs -f

# 仅重启某个服务
docker compose -f deploy/docker-compose.single.yml restart im-svc

# 代码更新后重建
docker compose -f deploy/docker-compose.single.yml up --build -d

# 停止
docker compose -f deploy/docker-compose.single.yml down

# 完全清理（含数据库数据）
docker compose -f deploy/docker-compose.single.yml down -v
```

---

## 3. 集群部署

### 3.1 架构说明

```
负载均衡 (Nginx/HAProxy/Traefik)
    ├── 节点1: nginx + user-file-svc + im-svc + docker-svc
    ├── 节点2: nginx + user-file-svc + im-svc + docker-svc
    └── 节点3: nginx + user-file-svc + im-svc + docker-svc
                  ↓
    PostgreSQL 集群 (Patroni/Citus)
    Redis 集群 (Sentinel/Cluster)
    MinIO 分布式 (4 节点纠删码)
```

### 3.2 前提条件

- 3 台以上 Linux 服务器
- 网络互通 (同一内网或 VPC)
- PostgreSQL、Redis、MinIO 独立部署
- 各节点安装 Docker

### 3.3 部署应用服务

```bash
# 使用集群配置
cd server
CONFIG_PATH=config/config.cluster.yaml go run ./cmd/user-file-svc &
CONFIG_PATH=config/config.cluster.yaml go run ./cmd/im-svc &
CONFIG_PATH=config/config.cluster.yaml go run ./cmd/docker-svc &
```

或在 Docker Swarm 中：

```bash
docker stack deploy -c deploy/docker-compose.cluster.yml cloudnexus
```

---

## 4. 数据备份

### PostgreSQL

```bash
# 全量备份
pg_dump -h localhost -U cloudnexus cloudnexus > backup_$(date +%Y%m%d).sql

# 定时备份 (crontab)
0 2 * * * pg_dump -h localhost -U cloudnexus cloudnexus | gzip > /backup/db_$(date +\%Y\%m\%d).sql.gz
```

### MinIO

```bash
mc mirror minio/cloudnexus /backup/minio/cloudnexus
```

### Redis

```bash
cp /data/dump.rdb /backup/redis/dump_$(date +%Y%m%d).rdb
```

---

## 5. 监控与健康检查

### 健康检查端点

```bash
curl http://localhost/healthz
```

所有 Go 服务均暴露 `/healthz`，docker compose 中配置了 `depends_on` + `condition: service_healthy`。

---

## 6. 安全清单

- [ ] 修改所有默认密码 (数据库、Redis、MinIO)
- [ ] 生成强随机 JWT Secret
- [ ] 启用 HTTPS (Let's Encrypt / 自签证书)
- [ ] 配置防火墙 (仅开放 80/443)
- [ ] 数据库连接使用 SSL
- [ ] Docker 端口不暴露到公网
- [ ] 定期更新依赖和镜像
