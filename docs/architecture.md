# CloudNexus 架构设计

> 版本：v1.2.0 | 更新：2026-05-15

## 1. 项目概述

CloudNexus 是一个自托管、数据私有的协作平台，目标覆盖：

- 云存储 (文件管理、分享、版本管理、回收站)
- 即时通讯 (私聊、群聊、好友系统、消息推送)
- Docker 管理 (容器生命周期、多主机)
- 视频监控 (摄像头管理、AI 识别、人脸考勤)
- 在线文档 (协作编辑、Markdown 增强)

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
                    /   |    |     \        \
                   /    |    |      \        \
     +-----------+ +-------+ +----------+ +----------+ +----------+
     |user-file-svc| |im-svc| |docker-svc| |camera-svc| |collab-svc|
     |  (:8081)   | |(:8082)| | (:8083)  | | (:8085)  | | (:8086)  |
     +-----------+ +-------+ +----------+ +----------+ +----------+
          |    |       |    |      |             |            |
          v    v       v    v      v             v            v
     +--------+ +-------+ +----+ +===========+ +----------+ +----------+
     |PostgreSQL| | MinIO | |Redis| |Docker   | | MediaMTX | |AI       |
     | (:5432) | |(:9000)| |:6379| |Daemon   | | (:8888)  | |Inference|
     +--------+ +-------+ +----+ |(socket) | +----------+ |(:8000)  |
                                 +===========+              +----------+

     所有服务通过 Docker 内部网络通信，仅 nginx 暴露端口
```

### 3.2 集群架构（共享基础设施）

**核心原则：基础设施集中化，应用服务分布式。** 所有应用节点共享同一套 PostgreSQL / Redis / MinIO，数据天然互通。

```
                      ┌──────────────────┐
                      │   负载均衡器      │  (可选，单机 nginx 已内置被动健康检查)
                      │  (Port 80/443)   │
                      └───┬──────────────┘
              ┌───────────┼───────────┐
              │           │           │
        ┌─────▼─────┐┌────▼──────┐┌───▼───────┐
        │ 应用节点1  ││ 应用节点2  ││ 应用节点3  │
        │ ┌───────┐ ││ ┌───────┐ ││ ┌───────┐ │
        │ │nginx   │ ││ │nginx   │ ││ │nginx   │ │
        │ │+5xGo服务│ ││ │+5xGo服务│ ││ │+5xGo服务│ │
        │ └───────┘ ││ └───────┘ ││ └───────┘ │
        └─────┬─────┘└────┬──────┘└───┬───────┘
              │           │           │
              └───────────┼───────────┘
                          │               ← 只需网络可达基础设施 IP:端口
         ┌────────────────┼────────────────┐
         │      基础设施服务器 (1台)          │
         │                                  │
         │  PostgreSQL :5432                │
         │  Redis      :6379                │
         │  MinIO      :9000                │
         └──────────────────────────────────┘
```

**应用节点**（每台服务器）：部署全部 5 个 Go 服务 + nginx，无状态，可任意横向扩展。

**基础设施服务器**（1 台）：集中部署 PostgreSQL + Redis + MinIO，所有应用节点共享。

**数据如何互通：**

| 场景 | 机制 |
|------|------|
| 用户认证 | JWT 使用同一 secret 签发，A 节点签的 token 在 B 节点可验证 |
| 文件管理 | 元数据写入同一个 PostgreSQL，文件对象写入同一个 MinIO bucket |
| 即时通讯 | 消息写入同一个 PostgreSQL，跨节点投递通过 Redis Pub/Sub (`im:broadcast`) |
| 在线文档 | 文档内容存储 MinIO，跨节点同步通过 Redis Pub/Sub (`collab:broadcast`) |
| 集群监控 | 所有节点向同一张 `docker_nodes` 表注册 + 心跳，HealthAggregator 统一探测 |
| Docker 管理 | 各节点管理本地 Docker Daemon，跨主机通过 EndpointManager + TLS 远程连接 |

**Snowflake 节点 ID：** 通过环境变量 `SNOWFLAKE_NODE_ID` 传入（user-file-svc=1, im-svc=2, docker-svc=3, camera-svc=5, collab-svc=6），多实例部署时每个实例使用不同的 worker ID。

**基础设施高可用（远期考虑）：** 当前推荐 1 台基础设施服务器 + 定时备份。多基础设施涉及 PostgreSQL 主从复制、Redis Sentinel/Cluster、MinIO 分布式模式，复杂度显著增加，建议在业务真正需要时再引入。

## 4. 数据流

### 4.1 文件上传

```
客户端 → nginx(:80) → user-file-svc → MinIO
                          ├── 写入 files 表 (PostgreSQL)
                          ├── 更新用户配额 (user_quota)
                          └── 返回文件 ID (Snowflake string)
```

支持普通上传、批量上传、分块上传（>10MB 自动分块，支持断点续传）、覆盖上传（自动保存旧版本）。

### 4.2 IM 消息流

```
发送方浏览器 → nginx(:80 /ws) → im-svc ──┬──→ PostgreSQL (存储)
                                          └──→ Hub 本地推送 → 接收方 (同节点)
                                          └──→ Redis Pub/Sub (im:broadcast) → 接收方 (跨节点)
```

**消息类型：** text / image / video / file / system
- image/video：上传到云盘 → 发送 JSON `{file_id, url, ...}` → 内联渲染
- file：从云盘文件选择器选取 → 发送文件卡片
- text：支持 URL 自动检测 → 后端抓取 OG 元数据 → 链接卡片展示

### 4.3 Docker 操作

```
客户端 HTTP → nginx(:80) → docker-svc → Docker Socket (/var/run/docker.sock)
                                         ├── 标签归属 (cloudnexus.creator)
                                         └── 远程端点 (TLS + EndpointManager)
```

### 4.4 摄像头视频流

```
客户端 HLS.js → nginx(:80) → MediaMTX (:8888) → RTSP 摄像头 (局域网)
客户端 HTTP  → nginx(:80) → camera-svc (:8085) ──→ MediaMTX API (动态注册)
                                    ├── PostgreSQL (摄像头/事件)
                                    ├── ai-inference (:8000) → YOLO 目标检测
                                    └── MinIO (截图)
```

### 4.5 人脸识别与考勤

```
浏览器 (face-api.js) → 摄像头视频帧 → 提取 128-d 人脸嵌入向量
    → camera-svc POST /faces/match → Go 余弦相似度匹配 face_profiles 表
    → 匹配成功 → 记录 FaceAttendanceSession (5分钟间隔内合并)
    → 每日汇总 (签到/签退时间)
```

### 4.6 在线文档协作

```
客户端 TipTap/Yjs → nginx(:80 /ws/collab/:id) → collab-svc
    → DocHub 管理 Yjs 文档感知
    → MinIO (文档持久化存储)
    → Redis Pub/Sub (collab:broadcast) → 跨节点实时同步
```

### 4.7 聊天记录备份与恢复

```
导出: 客户端 → im-svc → PostgreSQL (JOIN messages + users) → JSON 文件 (含 SHA256 校验码)
                       → 自动上传到云盘 聊天记录/{私聊|群聊}/
导入: 客户端选择 JSON → im-svc → 校验码验证 → 按 ID 去重 → 批量写入 messages 表
```

校验码 = SHA256(`conversation_id|message_count|last_seq`)，确保导出文件完整性。

## 5. 技术栈

| 层 | 技术 |
|----|------|
| 入口 | nginx:alpine (Docker) |
| 后端 | Go 1.25 + Gin + GORM |
| 前端 | React 18 + TypeScript + Vite + Ant Design 6 + Zustand 5 |
| 流媒体 | MediaMTX (RTSP→HLS/WebRTC) |
| AI | YOLOv8 (ultralytics) GPU/CPU 自适应 |
| 协作 | Yjs CRDT + TipTap 富文本 + Redis Pub/Sub |
| 人脸 | face-api.js (浏览器端) + Go 余弦相似度 (后端) |
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
  "ts": "2026-05-15T17:00:00.000Z",
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

**当前端点：**

| 端点 | user-file-svc | im-svc | docker-svc | camera-svc | collab-svc |
|------|:---:|:---:|:---:|:---:|:---:|
| `GET /healthz` | 详细（DB/MinIO/内存/goroutine/uptime） | 详细（DB/Redis/内存/goroutine/uptime） | 详细（Docker/内存/goroutine/uptime） | 详细（DB/内存/goroutine/uptime） | 详细（DB/Redis/内存/goroutine/uptime） |
| `GET /metrics` | 进程级（堆内存/GC/goroutine） | ❌ | ❌ | ❌ | ❌ |
| `GET /metrics/resources` | 主机级（CPU%/内存/磁盘/网络） | ❌ | ❌ | ❌ | ❌ |
| `GET /metrics/history` | 300 点环形缓冲/10s 间隔 | ❌ | ❌ | ❌ | ❌ |
| `GET /api/v1/admin/stats` | 业务统计（用户数/文件数/在线用户等） | ❌ | ❌ | ❌ | ❌ |
| `GET /system/log/*` | 日志服务/查询/读取/下载 | ❌ | ❌ | ❌ | ❌ |

**Nginx 健康检查路由：**
- `GET /healthz` → user-file-svc（聚合入口）
- `GET /healthz/user-file-svc` → user-file-svc:8081/healthz
- `GET /healthz/im-svc` → im-svc:8082/healthz
- `GET /healthz/docker-svc` → docker-svc:8083/healthz
- `GET /healthz/camera-svc` → camera-svc:8085/healthz

### 6.3 集群监控设计（Phase 3）

多节点部署时，需要统一监控所有节点和 Docker 主机的健康状态。

**节点注册与心跳：**
- 各节点启动时向 `docker_nodes` 表注册（节点名、主机地址、端口）
- 每 10 秒更新 `last_heartbeat` 时间戳
- 聚合器每 15 秒扫描所有节点，渐进式状态：healthy → unresponsive (2次失败/~30s) → offline (5次失败/~75s)
- `DockerNode` 模型支持 NodeType、Service、ContainerName、Version 等字段

**docker-svc 多主机探测：**
- `EndpointManager` 管理多台 Docker 主机的客户端连接
- 支持 TLS（CA 证书 + 客户端证书）安全连接
- 每 30s ping 所有端点并更新节点状态

**告警系统：**
- 节点离线/恢复通知
- 资源超阈值检测（可配置规则）
- Webhook 回调对接外部系统（企业微信/钉钉/邮件）
- 冷却时间防止告警风暴

**前端展示：**
- admin 面板"集群节点"页面，列表展示所有节点
- 每个节点显示：名称、主机、状态（healthy/unresponsive/offline）、最后心跳时间
- 在线时长追踪（NodeOnlineSession）
- 告警规则管理 + 告警历史查看

## 7. 安全模型

- **认证**：JWT (access token 8h + refresh token 7d)，支持会话管理（JTI 追踪、强制登出）
- **授权**：RBAC 角色权限系统 — Permission → Role → UserRole 三层模型
  - 预置角色：super_admin（全部权限）、admin（管理权限）、user（基本权限）
  - 权限粒度按资源+操作划分（如 `file:read`、`file:write`、`docker:admin`）
  - `hasPermission('*')` 为超级管理员提供通配符匹配
- **传输安全**：HTTPS + WSS (生产环境必须)
- **存储安全**：密码 bcrypt 哈希，文件可选加密，TLS 证书存储加密
- **Docker 安全**：Socket 挂载到 docker-svc 容器，容器标签归属 + API 权限控制
- **端口安全**：仅 nginx 80 端口对外，所有后端端口仅内网
- **账号安全**：验证码、邮箱/手机验证、登录失败锁定、账号注销（软删除 + 冷静期）
- **隐私控制**：用户隐私设置（允许搜索、允许添加好友、显示在线状态）
- **速率限制**：Nginx 层 login/register/forgot-password API 限流

## 8. 扩展点

| 扩展项 | 预留方式 |
|--------|----------|
| 多存储后端 | storage 包接口抽象，可切换 S3/MinIO/本地 |
| 第三方登录 | userfile 认证模块预留 OAuth handler + oauth_bindings 表 |
| 消息端到端加密 | messages 表支持 msg_type=encrypted |
| 水平扩容 | 所有服务无状态，增加实例 + 负载均衡即可 |
| Docker 多主机 | EndpointManager + TLS 客户端工厂模式 |
| K8s 部署 | deploy/k8s/ 目录预留 |
| 管理后台 | ✅ 已实现：用户管理 + 角色权限 + 系统状态 + 日志查看 + 集群节点 + 告警规则 + 配额管理 |
| 日志系统 | ✅ 已实现：zap 三路输出 + 管理后台实时查询 |
| 跨节点 IM | ✅ 已实现：Redis Pub/Sub (im:broadcast) 跨节点消息中继 |
| 聊天记录备份 | ✅ 已实现：导出为 JSON (SHA256 校验码) + 自动存云盘 + 导入去重 |
| 视频监控 | ✅ 已实现：camera-svc + MediaMTX + YOLO AI 识别 |
| 人脸识别 | ✅ 已实现：face-api.js 前端提取 + Go 余弦相似度匹配 + 考勤签到 |
| 在线文档 | ✅ 已实现：collab-svc + Yjs CRDT + TipTap + Redis 跨节点同步 |
| 集群监控 | ✅ 已实现：节点注册/心跳/健康聚合/docker_nodes 表/Docker 多主机探测/告警预留 |
| 文件版本 | ✅ 已实现：覆盖上传自动保存旧版本、版本历史/恢复/下载 |
| 分块上传 | ✅ 已实现：>10MB 自动分块、断点续传、顺序/并发模式可配 |
| 回收站 | ✅ 已实现：软删除 → 回收站 → 恢复/永久删除/清空 |
| 配额管理 | ✅ 已实现：配额等级 (quota_tiers) + 用户配额 (user_quota) + 前端进度条 |
| 监控告警 | /metrics 端点预留 Prometheus 格式，AlertRule + Webhook 告警规则引擎 |

## 9. 相关文档

- [API 接口文档](openapi.yaml)
- [数据库设计](database.md)
- [部署指南](deployment.md)
- [开发指南](development.md)
