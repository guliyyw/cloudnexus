# CloudNexus 架构设计

> 版本：v3.0 | 更新：2026-05-03

## 1. 项目概述

CloudNexus 是一个自托管、数据私有的协作平台，目标覆盖：

- 云存储 (文件管理、分享、WebDAV)
- 即时通讯 (私聊、群聊、消息推送)
- Docker 管理 (容器生命周期、多主机)

支持桌面端 (Windows/macOS/Linux)、Web 端、安卓移动端。

## 2. 核心设计原则

- **无状态服务**：所有应用服务不保存本地状态，共享数据库和缓存
- **单机起步，集群扩展**：初期 docker-compose 一键部署，后期平滑迁移到集群
- **数据私有**：用户完全掌控自己的数据，依赖全部开源可自建
- **API First**：前后端分离，所有功能通过 RESTful API 暴露

## 3. 系统架构

### 3.1 单机架构

```
客户端 → Nginx → user-file-svc / im-svc / docker-svc → PostgreSQL + Redis + MinIO
```

所有服务与依赖放在一台服务器上。

### 3.2 集群架构

```
                      ┌──────────────┐
                      │  负载均衡器   │
                      └───┬──────────┘
              ┌───────────┼───────────┐
              │           │           │
        ┌─────▼─────┐┌────▼──────┐┌───▼───────┐
        │ 节点 1    ││ 节点 2    ││ 节点 3    │
        │ userfile  ││ userfile  ││ userfile  │
        │ im        ││ im        ││ im        │
        │ docker    ││ docker    ││ docker    │
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
客户端 → Nginx → user-file-svc → MinIO (存储文件)
                  ├── 写入 files 表 (元数据)
                  └── 返回文件 ID
```

### 4.2 IM 消息流

```
发送方 WS → im-svc(节点1) ──┬──→ PostgreSQL (存储)
                           └──→ Redis Pub/Sub (广播)
                                     │
接收方 WS ← im-svc(节点2) ←──────────┘ (其他节点订阅)
```

### 4.3 Docker 操作

```
客户端 HTTP → docker-svc → Docker Socket (本地) 或 Docker TLS (远程)
```

## 5. 安全模型

- **认证**：JWT (access token 短期 + refresh token 长期)
- **授权**：服务内权限检查，管理员 vs 普通用户
- **传输安全**：HTTPS + WSS (生产环境必须)
- **存储安全**：密码 bcrypt 哈希，文件可选加密
- **Docker 安全**：TLS 双向认证，非必要不暴露 2376 端口

## 6. 扩展点

| 扩展项 | 预留方式 |
|--------|----------|
| 多存储后端 | storage 包接口抽象，可切换 S3/MinIO/本地 |
| 第三方登录 | userfile 认证模块预留 OAuth handler |
| 消息端到端加密 | messages 表支持 msg_type=encrypted |
| 水平扩容 | 所有服务无状态，增加实例 + 负载均衡即可 |
| Docker 多主机 | node 参数 + TLS 客户端工厂模式 |
| K8s 部署 | deploy/k8s/ 目录预留 |

## 7. 技术选型依据

| 选择 | 原因 |
|------|------|
| Go | 编译为单一二进制，低内存，高并发 (goroutine) |
| Gin | 轻量 HTTP 框架，性能好，生态成熟 |
| GORM | Go 最流行的 ORM，支持 PostGIS、软删除 |
| PostgreSQL | ACID 事务，全文搜索，Citus 扩展支持集群 |
| Redis | 高性能缓存 + Pub/Sub 消息总线路由 |
| MinIO | S3 兼容，单机和分布式统一 API |
| React + Vite | 组件化开发，热更新快 |
| JWT | 无状态认证，天然支持多节点 |

## 8. 相关文档

- [API 接口文档](api.md)
- [数据库设计](database.md)
- [部署指南](deployment.md)
- [开发指南](development.md)
