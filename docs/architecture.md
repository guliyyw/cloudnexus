# CloudNexus 架构设计

> 版本：v4.0 | 更新：2026-05-04

## 1. 项目概述

CloudNexus 是一个自托管、数据私有的协作平台，目标覆盖：

- 云存储 (文件管理、分享、WebDAV)
- 即时通讯 (私聊、群聊、消息推送)
- Docker 管理 (容器生命周期、多主机)

支持桌面端 (Windows/macOS/Linux)、Web 端、安卓移动端。

## 2. 核心设计原则

- **全容器化**：所有服务（含后端、前端、基础设施）均运行在 Docker 容器中
- **单端口暴露**：仅 nginx 对外暴露 80 端口，后端服务端口仅内部可访问
- **无状态服务**：所有应用服务不保存本地状态，共享数据库和缓存
- **单机起步，集群扩展**：初期 docker-compose 一键部署，后期平滑迁移到集群
- **数据私有**：用户完全掌控自己的数据，依赖全部开源可自建
- **API First**：前后端分离，所有功能通过 RESTful API 暴露

## 3. 系统架构

### 3.1 单机架构（全容器化）

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

### 3.2 集群架构

```
                      ┌──────────────┐
                      │  负载均衡器   │
                      │  (Port 80/443)│
                      └───┬──────────┘
              ┌───────────┼───────────┐
              │           │           │
        ┌─────▼─────┐┌────▼──────┐┌───▼───────┐
        │ 节点 1    ││ 节点 2    ││ 节点 3    │
        │ (nginx +  ││ (nginx +  ││ (nginx +  │
        │  Go svcs) ││  Go svcs) ││  Go svcs) │
        └─────┬─────┘└────┬──────┘└───┬───────┘
              │           │           │
              └───────────┼───────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
    ┌────▼─────┐  ┌──────▼──────┐  ┌──────▼──────┐
    │PostgreSQL│  │  Redis 集群  │  │ MinIO 集群   │
    │ (共享)   │  │(缓存/消息总线)│  │(对象存储)    │
    └──────────┘  └─────────────┘  └─────────────┘
```

## 4. 数据流

### 4.1 文件上传

```
客户端 → nginx(:80) → user-file-svc (Docker) → MinIO (Docker)
                          ├── 写入 files 表 (PostgreSQL)
                          └── 返回文件 ID (Snowflake string)
```

### 4.2 IM 消息流

```
发送方浏览器 → nginx(:80 /ws) → im-svc(Docker) ──┬──→ PostgreSQL (存储)
                                                  └──→ Redis Pub/Sub (广播)
                                                            │
接收方浏览器 ← nginx(:80 /ws) ← im-svc(Docker) ←──────────┘
```

### 4.3 Docker 操作

```
客户端 HTTP → nginx(:80) → docker-svc (Docker) → Docker Socket (/var/run/docker.sock)
```

## 5. 技术栈

| 层 | 技术 |
|----|------|
| 入口 | nginx:alpine (Docker) |
| 后端 | Go + Gin + GORM |
| 前端 | React 18 + TypeScript + Vite + Ant Design 5 + Zustand |
| 数据库 | PostgreSQL 15 |
| 缓存/消息 | Redis 7 |
| 对象存储 | MinIO (S3 兼容) |
| 容器化 | Docker Compose, 多阶段构建 |

## 6. 安全模型

- **认证**：JWT (access token 8h + refresh token 7d)
- **授权**：服务内权限检查，管理员 vs 普通用户
- **传输安全**：HTTPS + WSS (生产环境必须)
- **存储安全**：密码 bcrypt 哈希，文件可选加密
- **Docker 安全**：Socket 挂载到 docker-svc 容器，通过 API 权限控制
- **端口安全**：仅 nginx 80 端口对外，所有后端端口仅内网

## 7. 扩展点

| 扩展项 | 预留方式 |
|--------|----------|
| 多存储后端 | storage 包接口抽象，可切换 S3/MinIO/本地 |
| 第三方登录 | userfile 认证模块预留 OAuth handler |
| 消息端到端加密 | messages 表支持 msg_type=encrypted |
| 水平扩容 | 所有服务无状态，增加实例 + 负载均衡即可 |
| Docker 多主机 | node 参数 + TLS 客户端工厂模式 |
| K8s 部署 | deploy/k8s/ 目录预留 |

## 8. 相关文档

- [API 接口文档](api.md)
- [数据库设计](database.md)
- [部署指南](deployment.md)
- [开发指南](development.md)
