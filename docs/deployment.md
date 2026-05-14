# CloudNexus 部署指南

> 版本：v1.1.0 | 更新：2026-05-15

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
    ├── /api/*              → user-file-svc:8081 (用户、文件)
    ├── /api/v1/im/*        → im-svc:8082 (即时通讯 REST)
    ├── /api/v1/docker/*    → docker-svc:8083 (Docker 管理)
    ├── /api/v1/cameras/*   → camera-svc:8085 (摄像头管理)
    ├── /api/v1/faces/*     → camera-svc:8085 (人脸识别)
    ├── /api/v1/detect-*    → camera-svc:8085 (AI 检测)
    ├── /api/v1/collab/*    → user-file-svc:8081 (协作文档 REST)
    ├── /ws                 → im-svc:8082 (IM WebSocket)
    ├── /ws/collab/*        → collab-svc:8086 (文档协作 WebSocket)
    └── /                   → client/dist 静态文件 (SPA)
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
| **mediamtx HLS** | **8888** | **是** | 摄像头 HLS 视频流直连（绕过 nginx cookie 检查） |
| user-file-svc | 8081 | 否 | 用户 & 文件 & RBAC |
| im-svc | 8082 | 否 | 即时通讯 & WebSocket |
| docker-svc | 8083 | 否 | Docker 管理 |
| camera-svc | 8085 | 否 | 摄像头管理 & AI 识别 |
| collab-svc | 8086 | 否 | 在线文档协作 |
| PostgreSQL | 5432 | 否 | 数据库 |
| Redis | 6379 | 否 | 缓存 / 消息总线 |
| MinIO API | 9000 | 否 | 对象存储 S3 API |
| MinIO Console | 9001 | 否 | MinIO Web 管理 |
| MediaMTX API | 8889 | 否 | 流媒体管理 API |
| AI Inference | 8000 | 否 | YOLOv8 推理服务 |

### 2.5 Nginx 路由规则

| 路径 | 目标 | 说明 |
|------|------|------|
| `/api/v1/im/*` | im-svc:8082 | IM REST API |
| `/api/v1/docker/*` | docker-svc:8083 | Docker 管理 API |
| `/api/v1/cameras/*` | camera-svc:8085 | 摄像头管理 API |
| `/api/v1/faces/*` | camera-svc:8085 | 人脸识别 API |
| `/api/v1/detect-image` | camera-svc:8085 | AI 图片检测 |
| `/api/v1/detect-video` | camera-svc:8085 | AI 视频分析 (超时 300s) |
| `/api/v1/collab/*` | user-file-svc:8081 | 协作文档 REST API |
| `/api/*` | user-file-svc:8081 | 用户/文件 API (兜底) |
| `/ws/collab/*` | collab-svc:8086 | 文档协作 WebSocket |
| `/ws` | im-svc:8082 | IM WebSocket (需 Upgrade 头) |
| `/cam_*/` | mediamtx:8888 | HLS 视频流 (nginx 反向代理) |
| `/healthz/*` | 各服务 | 健康检查 |
| `/` | 静态文件 | SPA (try_files 回退到 index.html) |

> **注意**：前端 HLS.js 播放器直连 `服务器IP:8888` 获取视频流，不经过 nginx，
> 避免 MediaMTX cookie 检查导致的重定向循环。8888 端口需对外开放。

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

**集群部署**使用 `server/config/config.cluster.yaml`（host 指向基础设施服务器 IP）。

**宿主机开发**使用 `config.single.yaml`（host 均为 `localhost`）。

### 2.7 默认管理员账号

系统首次启动时会自动创建默认管理员（仅当 `users` 表为空）：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `DEFAULT_ADMIN_USERNAME` | `admin` | 管理员用户名 |
| `DEFAULT_ADMIN_PASSWORD` | `CloudNexus@admin` | 管理员密码 |
| `DEFAULT_ADMIN_EMAIL` | `admin@cloudnexus.local` | 管理员邮箱 |

同时自动初始化 RBAC 角色权限（super_admin、admin、user）并为管理员分配 super_admin 角色。

### 2.8 Docker Compose 服务一览

| 服务 | 来源 | 说明 |
|------|------|------|
| postgres | `postgres:15-alpine` | 数据库 (含性能调优) |
| redis | `redis:7-alpine` | 缓存/消息 |
| minio | `minio/minio:latest` | 对象存储 |
| user-file-svc | 构建 `../server` (SERVICE=user-file-svc) | Go 服务 |
| im-svc | 构建 `../server` (SERVICE=im-svc) | Go 服务 |
| docker-svc | 构建 `../server` (SERVICE=docker-svc) | Go 服务，挂载 docker.sock |
| mediamtx | `bluenviron/mediamtx:latest` | RTSP→HLS 流媒体 |
| ai-inference | 构建 `./ai-inference` | YOLOv8 Python 推理 |
| camera-svc | 构建 `../server` (SERVICE=camera-svc) | Go 服务 |
| collab-svc | 构建 `../server` (SERVICE=collab-svc) | Go 服务 |
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

## 3. 集群部署（共享基础设施）

### 3.1 架构说明

```
应用节点1 (192.168.1.10)   应用节点2 (192.168.1.11)
┌─────────────────────┐   ┌─────────────────────┐
│ nginx (80)          │   │ nginx (80)          │
│ user-file-svc (8081)│   │ user-file-svc (8081)│
│ im-svc (8082)       │   │ im-svc (8082)       │
│ docker-svc (8083)   │   │ docker-svc (8083)   │
│ camera-svc (8085)   │   │ camera-svc (8085)   │
│ collab-svc (8086)   │   │ collab-svc (8086)   │
└──────────┬──────────┘   └──────────┬──────────┘
           │                         │
           └──────────┬──────────────┘
                      │
           ┌──────────▼──────────────────────┐
           │     基础设施服务器 (192.168.1.100) │
           │  PostgreSQL :5432                │
           │  Redis      :6379                │
           │  MinIO      :9000                │
           └─────────────────────────────────┘
```

**核心原则：** Go 服务无状态，所有数据存储在共享基础设施中。任一应用节点宕机不影响数据完整性。

### 3.2 前提条件

- 至少 2 台 Linux 服务器（1 台基础设施 + 1 台起应用节点）
- 网络互通（同一内网，防火墙放行所需端口）
- 各节点安装 Docker 24+ 和 Docker Compose v2

### 3.3 端口规划

**基础设施服务器：**

| 服务 | 端口 | 需对应用节点开放 |
|------|------|:---:|
| PostgreSQL | 5432 | 是 |
| Redis | 6379 | 是 |
| MinIO API | 9000 | 是 |
| MinIO Console | 9001 | 否（仅管理用） |

**应用节点服务器：**

| 服务 | 端口 | 对外 |
|------|------|:---:|
| nginx | 80 | 是（用户入口） |
| user-file-svc | 8081 | 否（内部） |
| im-svc | 8082 | 否（内部） |
| docker-svc | 8083 | 否（内部） |
| camera-svc | 8085 | 否（内部） |
| collab-svc | 8086 | 否（内部） |

### 3.4 第一步：部署基础设施服务器

```bash
# 1. 创建基础设施 Compose 文件 deploy/docker-compose.infra.yml
cat > deploy/docker-compose.infra.yml << 'EOF'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: cloudnexus
      POSTGRES_PASSWORD: cloudnexus
      POSTGRES_DB: cloudnexus
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redisdata:/data

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - miniodata:/data

volumes:
  pgdata:
  redisdata:
  miniodata:
EOF

# 2. 启动基础设施
docker compose -f deploy/docker-compose.infra.yml up -d
```

### 3.5 第二步：修改集群配置

编辑 `server/config/config.cluster.yaml`，将基础设施地址改为实际 IP：

```yaml
server:
  port: 8081
  host: "0.0.0.0"

database:
  dsn: "host=192.168.1.100 user=cloudnexus password=cloudnexus dbname=cloudnexus port=5432 sslmode=disable"

redis:
  addr: "192.168.1.100:6379"

minio:
  endpoint: "192.168.1.100:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  bucket: "cloudnexus"

jwt:
  access_secret: "change-me-in-production"
  refresh_secret: "change-me-in-production-refresh"
  access_ttl_sec: 28800
  refresh_ttl_sec: 604800
```

> **注意**：所有应用节点的 JWT secret 必须一致。

### 3.6 第三步：部署应用节点

在每台应用服务器上执行：

```bash
# 1. 构建前端
cd client && npm install && npm run build

# 2. 启动应用服务
cd ../deploy
docker compose -f docker-compose.cluster.yml up --build -d
```

### 3.7 Snowflake 节点 ID 配置

**集群模式下必须为不同实例分配不同的 Snowflake 节点 ID**，通过环境变量配置：

| 服务 | 环境变量 | 默认值 |
|------|----------|--------|
| user-file-svc | `USER_FILE_NODE_ID` | 1 |
| im-svc | `IM_NODE_ID` | 2 |
| docker-svc | `DOCKER_NODE_ID` | 3 |
| camera-svc | `CAMERA_NODE_ID` | 5 |
| collab-svc | `COLLAB_NODE_ID` | 6 |

同一服务的不同实例必须使用不同 ID（如第一个 user-file-svc 实例用 1，第二个用 11）。

### 3.8 注意事项

- 集群 Compose 不包含 mediamtx / ai-inference / camera-svc / collab-svc，如需摄像头和文档协作功能，需单独部署
- 节点自动注册到共享 `docker_nodes` 表，HealthAggregator 自动探测
- 设置 `NODE_HOST=${SERVER_HOST}` 确保节点注册正确的主机地址

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
curl http://localhost/healthz                   # 聚合
curl http://localhost/healthz/user-file-svc     # 用户文件服务
curl http://localhost/healthz/im-svc            # IM 服务
curl http://localhost/healthz/docker-svc        # Docker 服务
curl http://localhost/healthz/camera-svc        # 摄像头服务
```

所有 Go 服务均暴露 `/healthz`，docker compose 中配置了健康检查依赖。

---

## 6. 安全清单

- [ ] 修改所有默认密码 (数据库、Redis、MinIO)
- [ ] 生成强随机 JWT Secret
- [ ] 启用 HTTPS (Let's Encrypt / 自签证书)
- [ ] 配置防火墙 (仅开放 80/443/8888)
- [ ] 数据库连接使用 SSL
- [ ] Docker 端口不暴露到公网
- [ ] 定期更新依赖和镜像
- [ ] 配置 AI 推理服务访问令牌 (`AI_INFERENCE_TOKEN`)
