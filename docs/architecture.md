# CloudNexus 架构设计

> 版本：v1.0.0 | 更新：2026-05-06

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
                                                  └──→ Hub 本地推送 → 接收方 (同节点)
                                                  └──→ Redis Pub/Sub (im:broadcast) → 接收方 (跨节点)
```

**消息类型：** text / image / video / file / system
- image/video：上传到云盘 → 发送 JSON `{file_id, url, ...}` → 内联渲染
- file：从云盘文件选择器选取 → 发送文件卡片
- text：支持 URL 自动检测 → 后端抓取 OG 元数据 → 链接卡片展示

### 4.3 Docker 操作

```
客户端 HTTP → nginx(:80) → docker-svc (Docker) → Docker Socket (/var/run/docker.sock)
```

## 5. 技术栈

| 层 | 技术 |
|----|------|
| 入口 | nginx:alpine (Docker) |
| 后端 | Go + Gin + GORM |
| 前端 | React 18 + TypeScript + Vite + Ant Design 6 + Zustand 5 |
| 数据库 | PostgreSQL 15 |
| 缓存/消息 | Redis 7 |
| 对象存储 | MinIO (S3 兼容) |
| 日志 | zap (结构化日志) + 环形缓冲 + 按天分文件 + 30天自动清理 |
| ID 生成 | Snowflake 算法 (uint64 → JSON string) |
| 数据库迁移 | 版本化 SQL 迁移 (go:embed) + schema_migrations 追踪 |
| 容器化 | Docker Compose, 多阶段构建 |

## 6. 日志与监控

### 6.1 日志系统

所有 Go 服务采用结构化日志 (zap)，统一格式：

```json
{
  "level": "info",
  "ts": "2026-05-04T17:00:00.000Z",
  "caller": "service/hub.go:42",
  "msg": "user connected",
  "request_id": "uuid-xxx",
  "user_id": "2051225055077076992",
  "service": "im-svc"
}
```

**设计要点：**
- `request_id` 从请求生成 (UUID)，贯穿全链路追踪
- 日志级别通过配置控制（debug/info/warn/error），生产默认 info
- 三路输出：stdout (Docker 收集) + 2048 条环形缓冲 (实时查询) + 按天分文件 (10MB 拆分, 30天自动清理)
- 管理后台可实时查看/过滤/下载日志

### 6.2 健康检查与监控

**已有端点：**
- `GET /healthz` — 存活检查（各服务已实现）
- `GET /api/v1/admin/stats` — 管理后台系统状态 API（用户数、文件数、在线用户等）
- `GET /system/log/services` — 可查询的日志服务列表
- `POST /system/log/query` — 环形缓冲实时日志查询
- `POST /system/log/read` — 文件日志读取
- `GET /system/log/files` — 日志文件列表
- `GET /system/log/download` — 日志文件下载

### 6.3 请求追踪

```
客户端 → nginx (添加 X-Request-Id)
           → Go 服务 (提取 request_id, 注入 context)
              → GORM (SQL 日志带 request_id)
              → Redis (操作日志带 request_id)
              → MinIO (操作日志带 request_id)
```

## 7. 安全模型

- **认证**：JWT (access token 8h + refresh token 7d)
- **授权**：JWT 内嵌 `is_admin` 字段，`AdminRequired` 中间件校验管理员权限
- **传输安全**：HTTPS + WSS (生产环境必须)
- **存储安全**：密码 bcrypt 哈希，文件可选加密
- **Docker 安全**：Socket 挂载到 docker-svc 容器，通过 API 权限控制
- **端口安全**：仅 nginx 80 端口对外，所有后端端口仅内网

## 8. 扩展点

| 扩展项 | 预留方式 |
|--------|----------|
| 多存储后端 | storage 包接口抽象，可切换 S3/MinIO/本地 |
| 第三方登录 | userfile 认证模块预留 OAuth handler |
| 消息端到端加密 | messages 表支持 msg_type=encrypted |
| 水平扩容 | 所有服务无状态，增加实例 + 负载均衡即可 |
| Docker 多主机 | node 参数 + TLS 客户端工厂模式 |
| K8s 部署 | deploy/k8s/ 目录预留 |
| 管理后台 | ✅ 已实现：用户管理 + 系统状态 + 日志查看 |
| 日志系统 | ✅ 已实现：zap 三路输出 + 管理后台实时查询 |
| 跨节点 IM | ✅ 已实现：Redis Pub/Sub (im:broadcast) 跨节点消息中继 |
| 监控告警 | /metrics 端点预留 Prometheus 格式，可集成 Grafana 告警 |

## 9. 相关文档

- [API 接口文档](api.md)
- [数据库设计](database.md)
- [部署指南](deployment.md)
- [开发指南](development.md)
