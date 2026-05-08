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
    ├── /api/v1/cameras/* → camera-svc:8085 (摄像头管理)
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
| **mediamtx HLS** | **8888** | **是** | 摄像头 HLS 视频流直连（绕过 nginx cookie 检查） |
| user-file-svc | 8081 | 否 | 用户 & 文件 |
| im-svc | 8082 | 否 | 即时通讯 & WebSocket |
| docker-svc | 8083 | 否 | Docker 管理 |
| camera-svc | 8085 | 否 | 摄像头管理 & AI 识别 |
| PostgreSQL | 5432 | 否 | 数据库 |
| Redis | 6379 | 否 | 缓存 / 消息总线 |
| MinIO API | 9000 | 否 | 对象存储 S3 API |
| MinIO Console | 9001 | 否 | MinIO Web 管理 |

### 2.5 Nginx 路由规则

| 路径 | 目标 | 说明 |
|------|------|------|
| `/api/v1/im/*` | im-svc:8082 | IM REST API |
| `/api/v1/docker/*` | docker-svc:8083 | Docker 管理 API |
| `/api/v1/cameras/*` | camera-svc:8085 | 摄像头管理 API |
| `/api/v1/detect-image` | camera-svc:8085 | AI 图片检测 |
| `/api/*` | user-file-svc:8081 | 用户/文件 API (兜底) |
| `/ws` | im-svc:8082 | WebSocket (需 Upgrade 头) |
| `/cam_*/` | mediamtx:8888 | HLS 视频流 (nginx 反向代理) |
| `/` | 静态文件 | SPA (try_files 回退到 index.html) |

> **注意**：前端 HLS.js 播放器直连 `服务器IP:8888` 获取视频流，不经过 nginx，
> 避免 MediaMTX cookie 检查导致的重定向循环。8888 端口需对外开放。

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

## 3. 集群部署（共享基础设施）

### 3.1 架构说明

```
应用节点1 (192.168.1.10)   应用节点2 (192.168.1.11)
┌─────────────────────┐   ┌─────────────────────┐
│ nginx (80)          │   │ nginx (80)          │
│ user-file-svc (8081)│   │ user-file-svc (8081)│
│ im-svc (8082)       │   │ im-svc (8082)       │
│ docker-svc (8083)   │   │ docker-svc (8083)   │
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

**核心原则：** Go 服务无状态，所有数据存储在共享基础设施中。任一应用节点宕机不影响数据完整性，用户可访问任意节点看到相同数据。

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

### 3.4 第一步：部署基础设施服务器

选定一台服务器作为基础设施节点，在其上部署 PostgreSQL + Redis + MinIO。

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

# 3. 验证
docker compose -f deploy/docker-compose.infra.yml ps
```

> 生产环境请修改 `cloudnexus`、`minioadmin` 等默认密码。

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
  password: ""
  db: 0

minio:
  endpoint: "192.168.1.100:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  bucket: "cloudnexus"

log:
  level: "info"
  format: "json"

jwt:
  access_secret: "change-me-in-production"
  refresh_secret: "change-me-in-production-refresh"
  access_ttl_sec: 28800
  refresh_ttl_sec: 604800
```

> **注意**：所有应用节点的 JWT secret 必须一致，否则 A 节点签发的 token 在 B 节点无法验证。

### 3.6 第三步：部署应用节点

在每台应用服务器上执行：

```bash
# 1. 构建前端
cd client && npm install && npm run build

# 2. 启动应用服务
cd ../deploy
docker compose -f docker-compose.cluster.yml up --build -d

# 3. 验证
curl http://localhost/healthz
```

首节点启动后 `users` 表为空，`SeedDefaultAdmin()` 会自动创建默认管理员。后续节点启动时检测到已有用户，跳过种子。

### 3.7 第四步：注册节点互见（跨部署监控）

两套部署使用同一个 PostgreSQL，`NodeRegistrar` 自动将节点写入共享的 `docker_nodes` 表，`HealthAggregator` 自动探测所有节点。无需额外操作。

如需手动注册外部节点（如独立部署的基础设施实例）：

```bash
# 获取管理员 token
TOKEN=$(curl -s -X POST http://localhost/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"CloudNexus@admin"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

# 注册服务节点
curl -X POST http://localhost/api/v1/admin/nodes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"node2-user-file-svc","host":"192.168.1.11","port":8081,"node_type":"service","service":"user-file-svc"}'

# 注册基础设施节点
curl -X POST http://localhost/api/v1/admin/nodes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"infra-postgres","host":"192.168.1.100","port":5432,"node_type":"infrastructure","service":"postgres"}'
```

服务节点通过 HTTP `/healthz` 探测，基础设施节点通过 TCP 端口探测。

### 3.8 Snowflake 节点 ID 配置

**当前限制：** 每个服务的 Snowflake worker ID 写死在 `cmd/*-svc/main.go` 中（user-file-svc=1, im-svc=2, docker-svc=3）。同一服务多实例部署时会产生 ID 冲突。

**临时解决：** 在 Docker Compose 中为每个实例指定不同的 `SNOWFLAKE_NODE_ID` 环境变量，需修改 `cmd/*-svc/main.go` 读取该变量替代硬编码值。此改动列入 Phase 3 待办。

### 3.9 基础设施高可用（远期）

当前推荐 **1 台基础设施服务器 + 定时备份**。对于中小型部署，这已经足够：

```bash
# PostgreSQL 每日备份（在基础设施服务器上）
0 2 * * * docker exec deploy-postgres-1 pg_dump -U cloudnexus cloudnexus | gzip > /backup/db_$(date +\%Y\%m\%d).sql.gz

# MinIO 每日备份
0 3 * * * mc mirror minio/cloudnexus /backup/minio/cloudnexus
```

多基础设施高可用涉及：
- PostgreSQL：主从流复制或 Patroni + etcd 自动故障转移
- Redis：Sentinel 哨兵（3 台）或 Cluster（6 台）
- MinIO：分布式模式（4 节点起，纠删码）

运维复杂度显著增加，建议在业务规模真正需要时再引入。

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
