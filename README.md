# CloudNexus

自托管、数据私有的协作平台 — 云存储、即时通讯、Docker 管理。

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## 核心功能

| 模块 | 功能 | 服务 |
|------|------|------|
| 云存储 | 文件上传/下载、目录管理、分享链接、WebDAV | user-file-svc |
| 即时通讯 | 私聊、群聊、文件消息、在线状态、消息漫游 | im-svc |
| Docker 管理 | 容器启停、镜像管理、多主机管理、日志查看 | docker-svc |

## 架构

```
桌面端 / Web / 移动端
        │
   Nginx (负载均衡)
        │
  ┌─────┼─────┐
  │     │     │
 8081  8082  8083
 用户   即时  Docker
 文件   通讯  管理
  │     │     │
  └─────┼─────┘
        │
 PostgreSQL + Redis + MinIO
```

设计支持**单机快速部署**，预留**多服务器集群**扩展能力。详见 [架构设计文档](docs/architecture.md)。

## 快速开始

### 单机部署

```bash
# 1. 启动依赖服务
docker compose -f deploy/docker-compose.single.yml up -d

# 2. 启动后端
cd server
go run ./cmd/user-file-svc &
go run ./cmd/im-svc &
go run ./cmd/docker-svc &

# 3. 启动前端
cd ../client
npm install && npm run dev
```

浏览器打开 `http://localhost:3000`

### 健康检查验证

```bash
curl http://localhost:8081/healthz  # user-file-svc
curl http://localhost:8082/healthz  # im-svc
curl http://localhost:8083/healthz  # docker-svc
```

## 文档

| 文档 | 内容 |
|------|------|
| [API 接口文档](docs/api.md) | REST & WebSocket 接口规范 |
| [数据库设计](docs/database.md) | 表结构、索引、ER 关系 |
| [部署指南](docs/deployment.md) | 单机/集群部署、备份、监控 |
| [开发指南](docs/development.md) | 环境搭建、项目结构、代码规范 |
| [架构设计](docs/architecture.md) | 系统架构概览 |

## 项目结构

```
cloudnexus/
├── client/          React + Vite + TypeScript 前端
├── server/          Go 多服务后端
│   ├── cmd/         服务入口 (user-file-svc, im-svc, docker-svc)
│   ├── internal/    每服务独立逻辑 (handler → service → repository)
│   ├── pkg/         跨服务共享 (auth, middleware, database, cache...)
│   └── config/      YAML 配置 (单机/集群)
├── deploy/          Docker Compose、Nginx、K8s 配置
├── docs/            项目文档
└── scripts/         开发脚本
```

## 服务列表

| 服务 | 端口 | 技术栈 | 说明 |
|------|------|--------|------|
| user-file-svc | 8081 | Gin + GORM + MinIO SDK | 用户认证 & 文件管理 |
| im-svc | 8082 | Gin + WebSocket + Redis Pub/Sub | 即时通讯 |
| docker-svc | 8083 | Gin + Docker SDK | Docker 容器管理 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Go 1.22, Gin |
| 数据库 | PostgreSQL 15, GORM |
| 缓存/消息 | Redis 7 |
| 对象存储 | MinIO |
| 认证 | JWT (access + refresh) |
| 前端 | React 18, TypeScript, Vite |
| 桌面端 | Tauri (规划中) |
| 移动端 | Capacitor (规划中) |
| 部署 | Docker Compose, Nginx |
| 集群 | Docker Swarm / K8s (规划中) |

## License

MIT
