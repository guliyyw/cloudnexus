# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture

CloudNexus is a self-hosted collaboration platform with five Go microservices, AI inference, media streaming, and a React frontend.

```
client/                    React 18 + TypeScript + Vite + Ant Design 6 + Zustand 5
server/
  cmd/
    user-file-svc/         用户认证 + 文件管理 + RBAC + 管理后台 (port 8081)
    im-svc/                即时通讯 + WebSocket (port 8082)
    docker-svc/            Docker 容器管理 (port 8083)
    camera-svc/            摄像头管理 + AI 识别 + 人脸考勤 (port 8085)
    collab-svc/            在线文档协作 + Yjs WebSocket (port 8086)
  internal/
    userfile/              handler → service → repository 三层 (17 handler, 13 service, 6 repository)
    im/                    handler → service → repository 三层 (含 WebSocket Hub + 好友 + 拉黑)
    dockermgr/             handler → service 两层 (无 repository，直接调用 Docker API)
    camera/                handler → service → repository 三层 (含人脸/考勤/发现)
    collab/                handler → service 两层 (Yjs DocHub + Redis Pub/Sub)
  pkg/                     跨服务共享库 (18个包)
    auth/                  JWT 生成与解析 (HS256)
    middleware/             AuthRequired, AdminRequired, RequirePermission (RBAC), Logger, CORS
    model/                 GORM 模型 (18个模型文件: User, File, Conversation, Message, Friend, Camera, FaceProfile, AlertRule, Permission, Role...)
    config/                YAML 配置加载 + 环境变量覆盖
    database/              PostgreSQL 连接 (GORM) + Snowflake ID 自动生成回调
    cache/                 Redis 连接
    storage/               MinIO 客户端
    crypto/                bcrypt 密码哈希
    errors/                AppError (Code + Message + Err) 与标准哨兵错误
    response/              APIResponse{Code, Message, Data} 统一 JSON 响应
    logger/                 Zap 封装 (环形缓冲 + 按天分文件 + 30天清理)
    migration/              版本化 SQL 迁移 (18个迁移 + schema_migrations 追踪表 + go:embed)
    snowflake/              Twitter Snowflake ID 生成器
    system/                健康检查构建器 + 节点注册与心跳 + 健康聚合器 + 告警评估器
    captcha/               图片验证码生成 + Redis 存储
    email/                 SMTP 邮件发送
  config/
    config.single.yaml     本机开发配置 (hostnames = localhost)
    config.docker.yaml     Docker 部署配置 (hostnames = 容器名)
    config.cluster.yaml    集群部署配置 (hostnames = 基础设施 IP)
deploy/
  ai-inference/            YOLOv8 Python 推理服务 (FastAPI)
  mediamtx/                MediaMTX RTSP→HLS 流媒体配置
  nginx/                   Nginx 反向代理 + 静态文件
  docker-compose.single.yml  单机部署 (12 服务: 5 Go + nginx + AI + MediaMTX + PG + Redis + MinIO)
  docker-compose.cluster.yml 集群部署模板 (5 Go 服务 + nginx，共享基础设施)
  deploy.sh                远程自动部署脚本
  k8s/                     Kubernetes 资源 (预留)
docs/                      openapi.yaml, architecture.md, database.md, deployment.md, development.md, progress.md, test-data.md
```

## Build & Run

### Full-stack in Docker (recommended)

```bash
# Build frontend
cd client && npm install && npm run build

# Start everything (infra + Go services + nginx + frontend)
cd ../deploy
docker compose -f docker-compose.single.yml up --build -d

# Access at http://localhost (only port 80 exposed)
```

All five Go services are built via multi-stage Docker builds from `server/Dockerfile` with a `SERVICE` build arg (e.g., `SERVICE=camera-svc`). The Docker Compose file builds all five in parallel.

### Individual services (host development)

```bash
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/user-file-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/im-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/docker-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/camera-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/collab-svc &
```

### Infrastructure only

```bash
cd deploy
docker compose up -d              # PostgreSQL:5432, Redis:6379, MinIO:9000/9001
```

### Frontend dev

```bash
cd client
npm install
npm run dev                       # Vite dev server on :3000
npx tsc --noEmit                  # type-check only
```

### Build all Go binaries

```bash
cd server
go build ./cmd/...                # build all
```

## Key Patterns

### Three-layer architecture
Every service follows the same pattern: `handler (HTTP) → service (business logic) → repository (database)`. Models are in `pkg/model/` and shared across services. Each service's `cmd/*-svc/main.go` wires dependencies and registers routes.

### Vite proxy routing
The frontend dev server proxies /api requests to the correct backend service based on path prefix:
- `/api/v1/im/*` → `localhost:8082`
- `/api/v1/docker/*` → `localhost:8083`
- `/api/v1/cameras` → `localhost:8085`
- `/api/v1/faces` → `localhost:8085`
- `/api/v1/detect-image` → `localhost:8085`
- `/api/v1/detect-video` → `localhost:8085`
- `/api/v1/collab` → `localhost:8086`
- `/api/*` → `localhost:8081` (catch-all, must be last)
- `/ws/collab` → `http://localhost:8086` (WebSocket for collab-svc; Vite target must be http://, not ws://)
- `/ws` → `http://localhost:8082` (WebSocket for im-svc; Vite target must be http://, not ws://)

### Frontend API client
`client/src/services/api.ts` creates a shared axios instance (`/api/v1` base, 30s timeout). Two interceptors:
- **Request**: attaches `Bearer {access_token}` from localStorage to every request
- **Response**: on 401, automatically tries `POST /api/v1/user/refresh` with the stored refresh token. On success, updates both tokens in localStorage and retries the original request once. On failure, clears tokens and redirects to `/login`.

Services like `file.ts` and `chat.ts` import this shared instance. Public endpoints (share access) use a raw `axios` call without the interceptor, since they don't require auth.

### Auth flow
- Auth middleware extracts Bearer token from `Authorization` header, or from `?token=` query param (for WebSocket connections)
- Sets `user_id` (uint64) and `username` in Gin context via `c.Set(...)`
- JWT access token TTL: 8 hours (28800s). Refresh token TTL: 7 days.
- Refresh tokens are SHA256-hashed before storing in `refresh_tokens` table

### Database migrations
`pkg/migration/` provides versioned SQL migrations with tracking via `schema_migrations` table. Run before GORM AutoMigrate in each service's startup: `migration.Up(db)`. New migration: add `NNN_name.up.sql` and `NNN_name.down.sql` to `pkg/migration/`. The `go:embed` directive bundles SQL files into the binary. Migrations already applied are skipped on subsequent runs.

### Error handling
Handlers call `handleError(c, err)` which type-asserts to `*apperrors.AppError` to get the HTTP status code and message. Service methods return `apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)`.

### Logging
Custom `pkg/logger/` wraps `go.uber.org/zap` with three sinks:
- **Stdout**: console or JSON format (configurable per service)
- **Ring buffer**: 2048-entry in-memory circular buffer, queried via admin API with filters (`level`, `request_id`, `user_id`). Powers the live log viewer in AdminPage.
- **Daily file**: writes to `{LogDir}/{YYYY-MM-DD}/{service}.log`, 10 MB max per file (auto-split with numeric suffix `.log.1`, `.log.2`...), auto-deletes directories older than 30 days.

`logger.FromContext(c *gin.Context)` returns a request-scoped logger with `request_id` and `user_id` fields when available. Admin endpoints: `GET /system/log/services`, `POST /system/log/query` (ring buffer), `POST /system/log/read` (file), `GET /system/log/files`, `GET /system/log/download`.

### File upload & preview
Multipart form with `file` field (repeated for batch). Stored in MinIO, metadata in `files` table. Download supports `?inline=true` for browser preview (images, video, audio, PDF) vs `attachment` for download. Preview URLs must include `?token=` query param since browser `<img>`/`<video>`/`<iframe>` tags don't send Authorization headers.

### WebSocket (im-svc)
Hub pattern: `internal/im/service/hub.go` manages `map[uint64]*Client` (one connection per user). Message types: `message`, `ping`/`pong`, `read_receipt`, `presence`, `ack`, `error`. Token passed as `?token=` query param on WebSocket upgrade.

**Client hook** (`client/src/hooks/useWebSocket.ts`): uses `handlerRef` pattern — the handler is stored in a `useRef` updated on every render, so the WebSocket `onmessage` callback always sees the latest closures (e.g., current `currentConvId`). Without this, `useEffect([], [])` would capture stale state from the initial mount.

### Cross-Node IM Relay (Redis Pub/Sub)
When `hub.SendToUser()` cannot find a local WebSocket connection, it publishes to Redis channel `im:broadcast`. Every im-svc node subscribes to this channel and forwards received messages to its local clients. This enables multi-node deployment behind a load balancer. Graceful degradation: if Redis is unreachable, the service starts normally (cross-node relay skipped) and local messaging still works.

### Conversation membership
Private conversations have two `ConversationMember` rows. Per-user soft-delete via `deleted_at` on the member row. Private conversation names are derived dynamically from the other member's username (via JOIN on users table).

### Chat messages — rich content
Message `msg_type` values: `text`, `image`, `video`, `file`, `system`.

- **Image/video**: uploaded via button (image/*, video/*) or paste (clipboard). Content is JSON `{file_id, file_name, file_size, mime_type, url, download_url}`. Rendered inline (max 320x320), click to full-screen.
- **File**: selected via `FilePickerModal` (browse cloud drive). Content is JSON `{file_id, file_name, file_size, mime_type}`. Rendered as a card with preview (if image/video/PDF) and download buttons.
- **Link preview**: on text messages, URL detection triggers `POST /api/v1/im/link-preview` to fetch OG metadata (title, description, image, site_name). Rendered as a link card below the text bubble. Fetched once per message (tracked via `fetchedMsgIds` ref to avoid duplicate requests).
- **System**: centered gray text for join/leave notifications.

### Chat backup & restore
- **Export**: `GET /api/v1/im/conversations/:id/export` returns JSON with `ChatExport` model (`pkg/model/im_export.go`). Checksum = `SHA256(conversation_id|message_count|last_seq)`. Frontend downloads JSON + auto-uploads to cloud drive under `聊天记录/{私聊|群聊}/` (auto-creates directories via `ensureChatBackupDir`).
- **Import**: `POST /api/v1/im/conversations/import` (multipart file upload). Validates checksum, deduplicates by message ID, batch inserts. Returns `ImportSummary{inserted, skipped, total}`. Route `/import` registered BEFORE `/:id/export` to avoid "import" being parsed as an ID.
- **Import dedup**: `FindExistingMessageIDs` returns `map[uint64]bool` of existing IDs; `BatchCreateMessages` skips duplicates via GORM `Create` (no `OnConflict` needed since new messages have unique IDs).

### Conversation sidebar — last message + real-time unread
- `GET /api/v1/im/conversations` returns `last_message` and `last_msg_type` (subquery on messages by `MAX(seq)` per conversation). Displayed in sidebar list item description.
- On WebSocket message: `updateLastMessage` always called (updates sidebar preview for the conversation). If message is for a non-current conversation, `incrementUnread` is also called (real-time badge increment). Both methods update conversations array in Zustand store.

### ID generation
All model IDs are generated via Snowflake algorithm (`pkg/snowflake/`). A GORM `Before("gorm:create")` callback in `pkg/database/postgres.go` auto-generates IDs for any model with an `ID` field whose value is zero. Each service gets a unique node ID: user-file-svc=1, im-svc=2, docker-svc=3, camera-svc=5, collab-svc=6. Snowflake must be initialized before database connection.

**JSON serialization**: All `uint64` ID fields use `json:"id,string"` tags to serialize as JSON strings, preventing JavaScript `number` precision loss (JS `number` max safe integer is 2^53-1, while Snowflake uint64 can exceed this). All frontend TypeScript ID types are `string`.

### Nginx proxy
Nginx runs in Docker (`deploy/docker-compose.single.yml`) as the single entry point on **port 80** — the only port exposed to the host:
- `/api/v1/im/*` → im-svc:8082 (Docker service name, not host IP)
- `/api/v1/docker/*` → docker-svc:8083
- `/api/v1/cameras` → camera-svc:8085
- `/api/v1/faces` → camera-svc:8085
- `/api/v1/detect-image` → camera-svc:8085
- `/api/v1/detect-video` → camera-svc:8085
- `/api/v1/collab/*` → user-file-svc:8081 (REST API handled by user-file-svc)
- `/api/*` → user-file-svc:8081 (catch-all, must be after more specific routes)
- `/ws/collab/*` → collab-svc:8086 (Yjs WebSocket for collaborative editing)
- `/ws` → im-svc:8082 (IM WebSocket upgrade)
- `/cam_*/` → mediamtx:8888 (HLS video stream proxy with cookie workaround)
- `/healthz` → user-file-svc:8081 (aggregated); `/healthz/{user-file-svc,im-svc,docker-svc,camera-svc}` for per-service probing
- `/` → Serves `client/dist/` static files with SPA `try_files` fallback
- `//api/*` → 308 redirect to `/api/*` (handles double-slash from misconfigured clients like Apifox)

Upstream blocks: `user_file_backend`, `im_backend` (ip_hash sticky), `docker_backend`, `camera_backend`, `mediamtx_hls`, `collab_backend`.

Config is at `deploy/nginx/nginx.conf` and is volume-mounted (restart, not rebuild, to apply changes).
The frontend connects to `window.location.host` for both API calls and WebSocket. Vite proxy config is retained as an alternative for devs who prefer not to run nginx.

**Important**: `nginx:alpine` image ships with `/etc/nginx/conf.d/default.conf` containing a default server block that conflicts with our config and rejects POST requests (returns 405). Docker Compose files use a `command` override to remove it before starting: `rm -f /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'`. If switching to a custom nginx image, this may not be needed.

### Docker multi-stage build
`server/Dockerfile` uses a `SERVICE` build arg to select which `cmd/` binary to build:
- Builder stage: `golang:1.25-alpine`, downloads deps, compiles with `CGO_ENABLED=0 GOOS=linux -ldflags="-s -w"`
- Runtime stage: `alpine:3.21` with `ca-certificates`, `tzdata`, `curl` (for healthcheck)
- docker-svc compiled as `GOOS=linux` defaults to `unix:///var/run/docker.sock` (no code change needed)

### Config files
- `config.single.yaml` — host development (all hostnames = `localhost`)
- `config.docker.yaml` — Docker deployment (hostnames = container names: `postgres`, `redis`, `minio`)
- Set via `CONFIG_PATH` env var (each `main.go` reads it). Docker Compose mounts `config.docker.yaml` into each service container at `/app/config/` and sets `CONFIG_PATH=/app/config/config.docker.yaml`.

### Remote server deploy workflow
The production server is accessed via SSH MCP tools (server ID: `cloudnexus-server`). Source tree is at `/home/user/cloudnexus/`. Deploy steps:
1. Upload changed server files to the matching remote path
2. `cd /home/user/cloudnexus/deploy && docker compose -f docker-compose.single.yml build <service>`
3. `docker compose -f docker-compose.single.yml up -d <service>`
4. For frontend: upload `client/dist` as `dist_new`, then `mv dist dist_old && mv dist_new dist` on the server
5. `docker compose restart nginx` after swapping dist (bind mount follows inode, directory rename needs re-mount)

### Friend system
Bidirectional `Friend` model with `uniqueIndex:idx_friend_pair` on `(user_id, friend_id)`. Status: `pending` → `accepted`. Auto-creates private conversation on accept. If both users send requests simultaneously, the second auto-accepts. Friend queries return `FriendInfo` (embeds Friend + `friend_username` from JOIN on users table) for display without extra queries.

### File sharing system
Shares are created per-file with an optional password (bcrypt hashed) and expiry. The share code is a random 12-char hex string. Two frontend views:
- **MySharesPage** (`/shares`) — authenticated user's table of all shares they created
- **ShareAccessPage** (`/s/:code`) — **public** landing page (no login required): shows file info, prompts for password if `has_password`, verifies via `POST /api/v1/share/:code/verify`, then offers preview (inline) and download
- Share download supports `?inline=true` for browser preview (same pattern as file download)
- `getShareUrl(code)` returns `/s/{code}` (frontend landing page), not the raw API endpoint
- Public share endpoints (no auth): `GET /share/:code`, `POST /share/:code/verify`, `GET /share/:code/download`

### File move/copy
- Backend: `POST /file/move`, `POST /file/copy` — validates ownership, prevents ancestor cycles, checks name conflicts
- Copy recursively duplicates directory trees, including MinIO object copy for each file
- Frontend: drag-and-drop onto directory rows triggers move (via `application/cloudnexus-move` data transfer)
- Batch operations: select multiple files → "移动到..." / "复制到..." opens `DirectoryPickerModal` for target selection

### Theme & styling
- **No CSS files** (except `client/src/index.css` for body/scrollbar). All component styling via inline `style` props.
- Custom `ConfigProvider theme` in `App.tsx`: warm amber primary `#e8964a`, light warm background `#fafaf8`, rounded 10-12px
- Sidebar: light mode (white bg, warm highlight `#fef3e7`), fixed position, logo in primary color
- Shared utilities: `client/src/utils/preview.ts` (`isPreviewable()`), `client/src/utils/format.ts` (`formatFileSize()`)
- Keep terminal/log viewer areas dark (`#1e1e1e`) regardless of theme

### Frontend routing
Public routes (no auth, no layout chrome): `/login`, `/register`, `/forgot-password`, `/reset-password`, `/forbidden`, `/s/:code`

Protected routes (wrapped in `AuthGuard > AppLayout`):
`/` → redirect `/dashboard`, `/dashboard`, `/files`, `/files/:id/edit`, `/shares`, `/chat`, `/friends`, `/docker`,
`/cameras` (统一页面：摄像头/人脸库/考勤 三Tab), `/cameras/:id`, `/documents`, `/documents/:id`,
`/album`, `/music`, `/trash`, `/settings`

Admin routes (wrapped in `AuthGuard > AdminGuard > AppLayout`): `/admin`, `/status`
Catch-all redirects to `/dashboard`

TopNav is minimal and context-aware: desktop shows only Logo (→/dashboard) + current section label + user avatar. No feature navigation buttons are shown — all feature discovery happens via dashboard cards. Each feature page manages its own internal navigation (tabs, toolbar buttons). Mobile uses a Drawer with full nav for quick access.

### Docker permission model
docker-svc uses container labels for ownership (no PostgreSQL needed):
- **Create**: adds labels `cloudnexus.creator=<user_id>` and `cloudnexus.creator_name=<username>` to every container
- **List**: non-admin → filters by `?filters={"label":["cloudnexus.creator=<userID>"]}` (server-side Docker API filter). Admin → returns all containers unfiltered.
- **Actions** (start/stop/restart/remove/logs/stats): call `checkOwnership(id, userID, isAdmin)` which inspects the container's labels. Admin bypasses, others must match `cloudnexus.creator` label.
- Handler helpers: `getUserID(c)`, `isAdmin(c)`, `getUsername(c)` extract values from Gin context

### Health check endpoints
All five services return a uniform detailed `/healthz` JSON:
```json
{"status":"ok","service":"im-svc","uptime":"2m6s","go_version":"go1.25.0","goroutines":10,"memory_mb":3,"components":{"database":"ok","redis":"ok"}}
```
Shared builder: `pkg/system/health.go` → `HealthzHandler(serviceName, ...ComponentCheck)` runs checks in parallel.

Per-service checks:
- **user-file-svc**: database (PostgreSQL ping), minio (ListBuckets)
- **im-svc**: database (PostgreSQL ping), redis (Ping)
- **docker-svc**: docker (GET /_ping on Docker Engine API)
- **camera-svc**: database (PostgreSQL ping)
- **collab-svc**: database (PostgreSQL ping), redis (Ping)

Nginx routes: `/healthz` → user-file-svc (aggregated); `/healthz/{user-file-svc,im-svc,docker-svc,camera-svc}` for per-service probing.

### Node registration & heartbeat
`pkg/system/nodereg.go` — `NodeRegistrar` manages lifecycle in `docker_nodes` table:
- `NewNodeRegistrar(db, name, host, serviceName, port)` — node name defaults to container hostname (`os.Hostname()`), overridable via `NODE_NAME` env var. Host defaults to `detectHostIP()` (first private IPv4), or `localhost` as last resort; overridable via `NODE_HOST`.
- `Start()`: upserts node by logical identity `(host, port, service)` — same logical service after rebuild merges into the existing record (updates name/container_name). Only heartbeat fields updated; status is managed by HealthAggregator. Old sessions are closed and node_name updated to preserve history.
- `Stop()`: marks node `status=offline`, closes stop channel.
- Wired in all five services and infrastructure nodes (postgres/redis/minio) via `HealthAggregator.RegisterInfra()`.
- Database model: `pkg/model/docker.go` — `DockerNode` with `NodeType` (service/infrastructure), `Service`, `ContainerName`, `Version`, `TotalOnlineSeconds`, `OfflineSince`.

### Health aggregator
`pkg/system/aggregator.go` — `HealthAggregator` runs in user-file-svc, probes every 15s:
- **Service nodes**: HTTP GET `/healthz`. Falls back from `host` to node `name` (Docker service name).
- **Infrastructure nodes**: TCP dial (postgres:5432, redis:6379) or HTTP endpoint (minio:9000/minio/health/live).
- **Progressive status**: 1 failure → stays healthy, 2 failures (~30s) → unresponsive, 5 failures (~75s) → offline. Counter resets on any success.
- **Session tracking**: `NodeOnlineSession` records each online period (start/end/duration) with container_name and version.

### Infrastructure nodes
Registered via `HealthAggregator.RegisterInfra()` in user-file-svc:
- `postgres` — TCP probe on `cfg.DBHost():5432`
- `redis` — TCP probe on `cfg.RedisHost():6379`
- `minio` — HTTP probe on `http://<MinIOHost>:9000/minio/health/live`
- Host values come from config DSN/addr/endpoint fields, not hardcoded `localhost`
- Additional infrastructure instances can be added via `POST /admin/nodes` with `node_type=infrastructure`

### Cluster nodes admin API
`internal/userfile/handler/node.go` — admin endpoints:
- `GET /admin/nodes` — list all, optional filters: `?service=` `?host=` `?type=` `?status=` (comma-separated)
- `GET /admin/nodes/:name/sessions` — online session history for a node
- `POST /admin/nodes` / `DELETE /admin/nodes/:name` — manual node management
- `addNodeRequest` supports `node_type` (docker_endpoint/service/infrastructure), `service`, and TLS fields (`tls_cert`, `tls_key`, `ca_cert`)

### Docker multi-endpoint + TLS
`internal/dockermgr/service/endpoint.go` — `EndpointManager` manages multiple Docker daemon connections:
- **Local endpoint**: always available from `DOCKER_HOST` env or platform default (unix socket on Linux, tcp://localhost:2375 on Windows)
- **Remote endpoints**: queried from `docker_nodes` where `node_type=docker_endpoint` and `service=docker`, lazy-cached
- **TLS**: `buildTLSClient(node)` builds a TLS-configured HTTP client when `tls_cert`/`tls_key` fields are non-empty. Uses `crypto/tls` with system CA pool + optional custom CA, client certificate, min TLS 1.2
- **Health**: `PingAll()` called every 30s from docker-svc goroutine, pings all endpoints via Docker `/_ping` and updates node status in DB
- All `DockerService` methods take `endpoint string` as first parameter; handler extracts via `getEndpoint(c)` = `c.DefaultQuery("endpoint", "local")`
- Routes: `GET /api/v1/docker/endpoints` (list), `GET /api/v1/docker/ping?endpoint=` (ping specific)
- Frontend `DockerPage` has `Select` host selector showing endpoint name, host:port, and status tag

### Camera service & AI pipeline

camera-svc (port 8085) manages camera CRUD, streams, AI recognition, face library, and attendance. It depends on two external services:

- **MediaMTX** (`mediamtx:8888`): RTSP→HLS proxy. camera-svc controls streams via MediaMTX's HTTP API (`:8889`). HLS segments served directly from MediaMTX on port 8888.
- **AI Inference** (`ai-inference:8000`): Python YOLOv8 service. camera-svc sends frames for object detection; the service returns bounding boxes + classes.

**Stream lifecycle**: `POST /cameras/:id/stream/start` → camera-svc calls MediaMTX API to start proxying the camera's RTSP URL → returns `hls_url` to frontend. `POST /cameras/:id/stream/stop` tears down the proxy.

**Object detection**: `POST /cameras/:id/recognition/start` → camera-svc spawns a goroutine that reads frames via ffmpeg, sends to AI inference, and stores `RecognitionEvent` rows (class, confidence, bbox, snapshot_url).

**Video file analysis**: `POST /api/v1/detect-video` → camera-svc sends the file to AI inference's `/detect-video` endpoint which uses cv2 to extract frames at configurable intervals, runs YOLO on each, and returns timestamped detection results.

### Collaborative document editing (collab-svc)

collab-svc (port 8086) provides real-time collaborative document editing via Yjs CRDT:

- **WebSocket endpoint**: `GET /ws/collab/:id` — Yjs sync protocol, auth via `?token=` query param
- **DocHub**: similar pattern to IM Hub, manages `map[string]*DocRoom` (one per document), each room maintains a Yjs document and broadcasts changes to all connected clients
- **Storage**: documents persisted to MinIO (keyed by file ID), loaded on first connection, periodically saved
- **Cross-node sync**: Redis Pub/Sub channel `collab:broadcast` — when a node receives a Yjs update, it publishes to Redis; other nodes forward the update to their local clients
- **REST API**: collab CRUD endpoints (`/api/v1/collab`) handled by user-file-svc (file management), collab-svc only handles WebSocket synchronization
- **Frontend**: TipTap editor with Yjs collaboration extension + code highlighting (lowlight) + task lists + BubbleMenu toolbar + Markdown paste conversion

### RBAC permission system

Role-Based Access Control implemented in user-file-svc:

- **Three-layer model**: `Permission` (code like `file:read`) → `Role` (name/code) → `UserRole` (user-role mapping)
- **Middleware**: `RequirePermission(permCode string)` checks if the authenticated user has the required permission through any of their assigned roles. Admin check: `hasPermission("*")` grants all.
- **Pre-seeded roles**: `super_admin` (all permissions), `admin` (most management permissions), `user` (basic permissions)
- **Frontend**: `useAccess()` hook provides `hasPermission(code)`, `hasRole(code)`, `isAdmin` for conditional UI rendering. `AccessControl` component wraps children with permission checks.
- **Admin routes**: protected by `AdminRequired()` middleware (checks `is_admin` field OR `super_admin`/`admin` role)
- **Permission groups**: file (read/write/delete/share), docker (read/control/admin), camera (read/control/admin), face (read/write/admin), attendance (read/admin), user (admin), system (admin)

### Face recognition & attendance

Face recognition is a **browser + backend** split:

- **Browser (face-api.js)**: loads TinyFaceDetector + FaceLandmark68Net + FaceRecognitionNet from `/models/`. Every 2-3s, captures a frame from `<video>`, detects faces, extracts 128-dim embeddings, sends to backend.
- **Backend (Go cosine similarity)**: `POST /faces/match` compares the embedding against all profiles in the user's face library using pure Go math (`dot(a,b) / (norm(a)*norm(b))`). Best match ≥ 0.6 threshold returned with name and confidence.

**Face profiles** (`face_profiles` table): name + JSON embedding + MinIO thumbnail. CRUD at `/api/v1/faces`.

**Attendance** (`face_attendance_sessions` table): when a face is matched with `camera_id` provided, `RecordAttendance` upserts a session. Sessions within 5-minute gap are merged (extend `end_time`). Daily summary at `GET /faces/attendance/daily` aggregates per face: min start_time = check_in, max end_time = check_out. Personnel status at `GET /faces/attendance/status` cross-references all face profiles with the day's sessions to show signed_in/not.

**Important**: Delete operations on attendance must verify face profile ownership via `FindFaceProfileByID()` before deleting, as the face profile carries the `owner_id`.

### Nginx load balancing
`deploy/nginx/nginx.conf` — upstream blocks for multi-server deployment:
- `user_file_backend`: user-file-svc instances (default: single server, add more for multi-server)
- `im_backend`: `ip_hash` sticky session for WebSocket
- `docker_backend`: docker-svc instances
- All servers use `max_fails=3 fail_timeout=30s` for passive health checks
- Single-server deployment: each upstream has one server, behavior unchanged
- Multi-server: add additional `server` lines to each upstream block
- `docker-compose.cluster.yml`: per-server deployment template with `SERVER_HOST` env var for node registration

## Testing

No test files exist yet. Infrastructure tests require running Docker services. API can be tested manually via curl — see `docs/test-data.md` for test account credentials and command examples.

## Common Gotchas

- **Working directory**: Go commands must run from `server/` directory (where `go.mod` is)
- **Windows Docker**: docker-svc in Docker uses `/var/run/docker.sock` (mounted from host). For host development on Windows, set `DOCKER_HOST=tcp://localhost:2375` and enable Docker Desktop TCP exposure.
- **Docker build context**: The build context is `../server` (relative to `deploy/`). All Go source changes invalidate the layer cache, so incremental builds recompile.
- **GORM AutoMigrate**: Each service's `main.go` calls AutoMigrate for its models. Adding a field to a model struct will auto-add the column on restart.
- **Config changes**: Services must be restarted to pick up config changes (e.g., JWT TTL)
- **docker-svc**: Now connects to PostgreSQL and initializes Snowflake (node_type field needs Snowflake IDs). Uses Docker container labels for ownership tracking.
- **ID types**: All IDs are `string` on the frontend and `uint64` with `json:",string"` on the backend. Never use `number` for IDs in TypeScript code.
- **WebSocket stale closures**: `useWebSocket` uses `handlerRef` pattern. Always use the ref to access current React state inside WebSocket callbacks — never close over state directly in `useEffect([], [])`.
- **Nginx config is volume-mounted**: Changes to `deploy/nginx/nginx.conf` only need `docker compose restart nginx`, not a full rebuild.
- **Node registration**: All five services register themselves as nodes via `NodeRegistrar`. Node host is auto-detected via `detectHostIP()` (first private IPv4) unless overridden by `NODE_HOST` env var. Heartbeat upsert does NOT overwrite `status` — only `host`, `port`, `last_heartbeat`. Infrastructure nodes (PostgreSQL, Redis, MinIO) are registered by the `HealthAggregator` in user-file-svc. The aggregator performs TCP/HTTP health probes for all nodes with progressive status: healthy → unresponsive (2 failures/~30s) → offline (5 failures/~75s).
- **camera-svc depends on MediaMTX + AI inference**: ensure both are running before testing camera features. MediaMTX API at `:8889`, HLS at `:8888`. AI inference at `:8000`.
- **collab-svc depends on MinIO + Redis**: document persistence uses MinIO, cross-node sync uses Redis Pub/Sub. The service does NOT call AutoMigrate — it reads the `files` table read-only for document metadata.
- **Face recognition requires face-api.js models**: model weights must be in `client/public/models/`. Without them, face detection silently produces no results.
- **MinIO thumbnail storage**: `FaceProfile.thumbnail_url` stores the object key (e.g., `faces/123.jpg`), not a full URL. Access via `GET /api/v1/faces/:id/thumbnail?token=`. Old profiles from before MinIO deployment may have `data:image/...` URLs that return 404 from the thumbnail API.
- **RBAC seeding**: `SeedRBAC()` runs on first startup (if no permissions exist), creates default permissions/roles and assigns `super_admin` role to the default admin user. `AssignDefaultRoleToAllUsers()` assigns `user` role to any users without roles.
- **Chunk upload**: files > 10MB are automatically split into chunks. System config `upload.sequential_mode` and `upload.max_concurrent_chunks` control upload behavior. Incomplete uploads can be resumed via `initChunkUpload` which returns completed chunk indices.
- **Trash/recycle bin**: files are soft-deleted (set `deleted_at`), moved to trash. `CleanupScheduler` runs every hour to permanently delete files older than 30 days in trash.
