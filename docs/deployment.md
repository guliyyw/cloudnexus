# CloudNexus 部署指南

> 版本：v0.1.0 | 更新：2026-05-03

## 1. 部署模式概述

| 模式 | 适用场景 | 复杂度 | 可用性 |
|------|----------|--------|--------|
| 单机 | 个人使用、小团队 (< 50 人) | 低 | 单点 |
| 集群 | 中大型团队、生产环境 | 中 | 高可用 |

---

## 2. 单机部署 (Docker Compose)

### 2.1 环境要求

- Linux / macOS / Windows (WSL2)
- Docker 24+
- Docker Compose v2
- 至少 4 GB 可用内存
- 至少 20 GB 可用磁盘

### 2.2 快速启动

```bash
# 1. 克隆项目
git clone <repo-url> cloudnexus
cd cloudnexus

# 2. 启动基础设施 (PostgreSQL + Redis + MinIO)
docker compose -f deploy/docker-compose.single.yml up -d

# 3. 等待依赖就绪
docker compose -f deploy/docker-compose.single.yml ps

# 4. 构建并启动后端服务
cd server
go build -o bin/user-file-svc ./cmd/user-file-svc
go build -o bin/im-svc ./cmd/im-svc
go build -o bin/docker-svc ./cmd/docker-svc

# 5. 分别启动 (开发阶段)
./bin/user-file-svc &
./bin/im-svc &
./bin/docker-svc &

# 6. 启动前端
cd ../client
npm install && npm run build
# 将 dist/ 目录部署到 Nginx 或使用 npm run preview 预览
```

### 2.3 端口规划

| 服务 | 端口 | 说明 |
|------|------|------|
| Nginx | 80, 443 | 反向代理入口 |
| user-file-svc | 8081 | 用户 & 文件 |
| im-svc | 8082 | 即时通讯 & WebSocket |
| docker-svc | 8083 | Docker 管理 |
| PostgreSQL | 5432 | 数据库 |
| Redis | 6379 | 缓存 / 消息总线 |
| MinIO API | 9000 | 对象存储 |
| MinIO Console | 9001 | MinIO Web 管理 |

### 2.4 配置文件

服务配置通过 YAML 文件加载，默认路径：
- 开发：`server/config/config.single.yaml`
- 生产：通过环境变量 `CONFIG_PATH` 指定

关键配置项：

```yaml
server:
  port: 8081          # 服务监听端口
  host: "0.0.0.0"    # 监听地址

database:
  dsn: "host=localhost user=cloudnexus password=cloudnexus dbname=cloudnexus port=5432 sslmode=disable"

redis:
  addr: "localhost:6379"

minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  bucket: "cloudnexus"

jwt:
  access_secret: "<random-secret>"
  refresh_secret: "<random-secret>"
  access_ttl_sec: 900
  refresh_ttl_sec: 604800
```

### 2.5 环境变量覆盖

所有配置项可通过环境变量覆盖（优先级高于 YAML）：

```bash
export DB_DSN="host=prod-db.internal user=cloudnexus..."
export REDIS_ADDR="redis.internal:6379"
export JWT_ACCESS_SECRET="production-secret"
```

---

## 3. 集群部署

### 3.1 架构说明

```
负载均衡 (Nginx/HAProxy/Traefik)
    ├── 节点1: user-file-svc + im-svc + docker-svc
    ├── 节点2: user-file-svc + im-svc + docker-svc
    └── 节点3: user-file-svc + im-svc + docker-svc
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

### 3.3 基础设施部署

#### PostgreSQL 高可用 (Patroni 方案)

```bash
# 每个 PostgreSQL 节点
docker run -d \
  --name patroni \
  --network host \
  -e PATRONI_NAME=pg-node1 \
  -e PATRONI_SCOPE=cloudnexus \
  -e PATRONI_RESTAPI_LISTEN=0.0.0.0:8008 \
  patroni:latest
```

#### Redis Sentinel

```bash
# 一主两从 + 三个 Sentinel
docker compose -f deploy/docker-compose.redis-cluster.yml up -d
```

#### MinIO 分布式

```bash
# 4 节点，每节点 1 块盘，纠删码模式
docker run -d \
  --name minio1 \
  -v /data/minio:/data \
  minio/minio server \
  http://minio{1...4}/data
```

### 3.4 部署应用服务

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

### 3.5 节点加入

新节点加入集群：

```bash
# 1. 在新节点上部署依赖和代码
# 2. 配置指向共享 PostgreSQL、Redis、MinIO
# 3. 启动服务
CONFIG_PATH=config/config.cluster.yaml go run ./cmd/user-file-svc &

# 4. 通过 API 注册节点 (可选，用于 docker-svc 发现)
curl -X POST http://<master>/api/v1/cluster/join \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"node4","host":"192.168.1.14"}'
```

---

## 4. Nginx 反向代理

```nginx
upstream userfile {
    server user-file-svc-1:8081;
    server user-file-svc-2:8081;
}

server {
    listen 80;

    location /api/v1/ {
        proxy_pass http://userfile;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws {
        proxy_pass http://im;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
    }
}
```

参考文件：`deploy/nginx/nginx.conf`

---

## 5. 数据备份

### PostgreSQL

```bash
# 全量备份
pg_dump -h localhost -U cloudnexus cloudnexus > backup_$(date +%Y%m%d).sql

# 定时备份 (crontab)
0 2 * * * pg_dump -h localhost -U cloudnexus cloudnexus | gzip > /backup/db_$(date +\%Y\%m\%d).sql.gz
```

### MinIO

```bash
# 使用 mc 客户端镜像同步
mc mirror minio/cloudnexus /backup/minio/cloudnexus
```

### Redis

```bash
# Redis 自动生成 dump.rdb，定期备份即可
cp /data/dump.rdb /backup/redis/dump_$(date +%Y%m%d).rdb
```

---

## 6. 监控与健康检查

### 健康检查端点

```bash
# 检查所有服务
curl http://localhost:8081/healthz
curl http://localhost:8082/healthz
curl http://localhost:8083/healthz
```

### 推荐监控栈

- **Prometheus** — 指标采集
- **Grafana** — 仪表盘
- **Node Exporter** — 主机指标
- **cAdvisor** — 容器指标

---

## 7. 安全清单

- [ ] 修改所有默认密码 (数据库、Redis、MinIO)
- [ ] 生成强随机 JWT Secret
- [ ] 启用 HTTPS (Let's Encrypt / 自签证书)
- [ ] 配置防火墙 (仅开放 80/443)
- [ ] 数据库连接使用 SSL
- [ ] Docker 端口不暴露到公网
- [ ] 定期更新依赖和系统补丁

---

## 8. 故障恢复

| 故障 | 恢复方式 |
|------|----------|
| PostgreSQL 宕机 | Patroni 自动故障转移 |
| Redis 宕机 | Sentinel 自动提升从节点 |
| MinIO 节点故障 | 纠删码自动恢复，替换坏盘即可 |
| 应用节点故障 | 负载均衡器自动摘除，重启服务即可 |
| 全部节点宕机 | 按顺序启动：基础设施 → 后端 → 前端 |
