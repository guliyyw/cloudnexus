# CloudNexus

自托管、数据私有的协作平台 — 云存储、即时通讯、Docker 管理。

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql)](https://www.postgresql.org)
[![Version](https://img.shields.io/badge/version-0.1.0--dev-orange)](.)

> 当前版本 v0.1.0-dev — Phase 3 集群化开发中。Phase 1–4 全部完成并测试通过后发布 v0.1.0。

## 核心功能

| 模块 | 功能 | 服务 |
|------|------|------|
| 云存储 | 文件上传/下载/预览、目录管理、批量操作、分享链接、文件移动/复制 | user-file-svc |
| 即时通讯 | 私聊、群聊、图片/视频/文件消息、好友系统、在线状态、聊天记录备份/恢复 | im-svc |
| Docker 管理 | 容器启停/日志/监控、镜像管理、权限分级 | docker-svc |
| 管理后台 | 用户管理、系统状态、日志查看、资源监控 | user-file-svc |

> WebDAV、文件版本管理、回收站、消息搜索、FCM 推送、Docker 网络/数据卷/Compose/Web TTY、CI/CD 等 P2 功能列入 [v0.2.0 规划](docs/progress.md)。

## 架构

```
                    宿主机 Port 80 (唯一对外端口)
                            |
                      +-----------+
                      |   nginx   |  (静态文件 + 反向代理)
                      +-----------+
                       /    |     \
                      /     |      \
          +-----------+  +-------+  +------------+
          |user-file-svc| |im-svc |  | docker-svc |
          |   (:8081)  |  |(:8082)|  |  (:8083)   |
          +-----------+  +-------+  +------------+
               |    |        |    |        |
               v    v        v    v        v
          +--------+  +-------+  +----+  +===============+
          |PostgreSQL| | MinIO |  |Redis|  | Docker Daemon |
          |  (:5432)|  |(:9000)|  |:6379|  |(socket mount) |
          +--------+  +-------+  +----+  +===============+

          所有服务通过 Docker 内部网络通信，仅 nginx 暴露端口
```

设计支持**单机快速部署**，预留**多服务器集群**扩展能力。详见 [架构设计文档](docs/architecture.md)。

## 快速开始

### 全栈 Docker 部署（推荐）

```bash
# 1. 构建前端
cd client && npm install && npm run build

# 2. 启动全部服务
cd ../deploy
docker compose -f docker-compose.single.yml up --build -d

# 3. 浏览器打开
# http://localhost  (仅 80 端口对外)
```

### 本地开发

```bash
# 仅启动基础设施
cd deploy && docker compose up -d

# 启动后端服务 (三个终端)
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/user-file-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/im-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/docker-svc &

# 启动前端
cd client && npm install && npm run dev
# 浏览器打开 http://localhost:3000
```

### 健康检查验证

```bash
# 通过 nginx 统一入口
curl http://localhost/healthz/user-file-svc | jq
curl http://localhost/healthz/im-svc       | jq
curl http://localhost/healthz/docker-svc   | jq

# 或直连服务端口 (开发模式)
curl http://localhost:8081/healthz  # DB + MinIO 状态
curl http://localhost:8082/healthz  # DB + Redis 状态
curl http://localhost:8083/healthz  # Docker 状态
```

## 文档

| 文档 | 内容 |
|------|------|
| [API 接口文档](docs/openapi.yaml) | OpenAPI 3.0 规范 |
| [数据库设计](docs/database.md) | 表结构、索引、ER 关系 |
| [部署指南](docs/deployment.md) | 单机/集群部署、备份、监控 |
| [开发指南](docs/development.md) | 环境搭建、项目结构、代码规范 |
| [架构设计](docs/architecture.md) | 系统架构概览 |
| [开发进度](docs/progress.md) | 需求清单、开发阶段、测试计划 |
| [测试数据](docs/test-data.md) | 测试账号、基础设施连接信息 |

## 项目结构

```
cloudnexus/
├── client/          React + Vite + TypeScript 前端
├── server/          Go 多服务后端
│   ├── cmd/         服务入口 (user-file-svc, im-svc, docker-svc)
│   ├── internal/    每服务独立逻辑 (handler → service → repository)
│   ├── pkg/         跨服务共享 (auth, middleware, database, cache, system...)
│   └── config/      YAML 配置 (单机/集群)
├── deploy/          Docker Compose、Nginx 配置
├── docs/            项目文档
└── scripts/         开发脚本
```

## 服务列表

| 服务 | 端口 | 技术栈 | 说明 |
|------|------|--------|------|
| user-file-svc | 8081 | Gin + GORM + MinIO SDK | 用户认证、文件管理、管理后台 |
| im-svc | 8082 | Gin + WebSocket + Redis Pub/Sub | 即时通讯、跨节点消息同步 |
| docker-svc | 8083 | Gin + Docker Engine API | Docker 容器管理、权限分级 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Go 1.22, Gin, GORM |
| 数据库 | PostgreSQL 15 |
| 缓存/消息 | Redis 7 (Pub/Sub 跨节点中继) |
| 对象存储 | MinIO (S3 兼容) |
| 认证 | JWT (access 8h + refresh 7d) |
| 日志 | zap 结构化日志 (环形缓冲 + 按天分文件 + 30天清理) |
| ID 生成 | Snowflake 算法 |
| 前端 | React 18, TypeScript, Vite, Ant Design 6, Zustand 5 |
| 部署 | Docker Compose, Nginx, 多阶段构建 |

## 开发阶段

| 阶段 | 内容 | 状态 |
|------|------|:----:|
| Phase 1 | 单机 MVP (用户/文件/IM/Docker) | ✅ |
| Phase 2 | 功能完善 (批量/日志/群聊/分享/管理后台) | ✅ |
| Phase 2.5 | P1 补齐 (镜像管理/容器监控/IM文件消息/响应式/跨节点同步) | ✅ |
| Phase 3 | 集群化 (节点注册/健康聚合/Docker多主机/集群监控) | 🔄 |
| Phase 4 | 高级功能 (OAuth/E2EE/K8s/性能调优) | ⬜ |

详见 [开发进度](docs/progress.md)。

## License

MIT
