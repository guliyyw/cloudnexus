# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture

CloudNexus is a self-hosted collaboration platform with three Go microservices and a React frontend.

```
client/                    React 18 + TypeScript + Vite + Ant Design 6 + Zustand 5
server/
  cmd/
    user-file-svc/         用户认证 + 文件管理 (port 8081)
    im-svc/                即时通讯 + WebSocket (port 8082)
    docker-svc/            Docker 容器管理 (port 8083)
  internal/
    userfile/              handler → service → repository 三层
    im/                    handler → service → repository 三层 (含 WebSocket Hub)
    dockermgr/             handler → service → repository 三层
  pkg/                     跨服务共享库
    auth/                  JWT 生成与解析 (HS256)
    middleware/             AuthRequired (JWT), Logger, CORS
    model/                 GORM 模型: User, File, Conversation, Message, Friend...
    config/                YAML 配置加载
    database/              PostgreSQL 连接 (GORM)
    cache/                 Redis 连接
    storage/               MinIO 客户端
    crypto/                bcrypt 密码哈希
    errors/                AppError (Code + Message + Err) 与标准哨兵错误
    response/              APIResponse{Code, Message, Data} 统一 JSON 响应
  config/
    config.single.yaml     单机配置 (DSN, Redis, MinIO, JWT)
    config.cluster.yaml    集群配置
deploy/                    Docker Compose (PostgreSQL + Redis + MinIO)
docs/                      api.md, database.md, deployment.md, development.md, progress.md, test-data.md
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

All three Go services are built via multi-stage Docker builds from `server/Dockerfile` with a `SERVICE` build arg (e.g., `SERVICE=user-file-svc`). The Docker Compose file builds all three in parallel.

### Individual services (host development)

```bash
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/user-file-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/im-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/docker-svc &
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
- `/api/*` → `localhost:8081` (catch-all, must be last)
- `/ws` → `http://localhost:8082` (WebSocket; Vite target must be http://, not ws://)

New IM or Docker endpoints automatically route correctly. New user-file endpoints on `/api/v1/...` also work.

### Auth flow
- Auth middleware extracts Bearer token from `Authorization` header, or from `?token=` query param (for WebSocket connections)
- Sets `user_id` (uint64) and `username` in Gin context via `c.Set(...)`
- JWT access token TTL: 8 hours (28800s). Refresh token TTL: 7 days.
- Refresh tokens are SHA256-hashed before storing in `refresh_tokens` table

### Error handling
Handlers call `handleError(c, err)` which type-asserts to `*apperrors.AppError` to get the HTTP status code and message. Service methods return `apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)`.

### File upload & preview
Multipart form with `file` field (repeated for batch). Stored in MinIO, metadata in `files` table. Download supports `?inline=true` for browser preview (images, video, audio, PDF) vs `attachment` for download. Preview URLs must include `?token=` query param since browser `<img>`/`<video>`/`<iframe>` tags don't send Authorization headers.

### WebSocket (im-svc)
Hub pattern: `internal/im/service/hub.go` manages `map[uint64]*Client` (one connection per user). Message types: `message`, `ping`/`pong`, `read_receipt`, `presence`, `ack`, `error`. Token passed as `?token=` query param on WebSocket upgrade.

### Conversation membership
Private conversations have two `ConversationMember` rows. Per-user soft-delete via `deleted_at` on the member row. Private conversation names are derived dynamically from the other member's username (via JOIN on users table).

### ID generation
All model IDs are generated via Snowflake algorithm (`pkg/snowflake/`). A GORM `Before("gorm:create")` callback in `pkg/database/postgres.go` auto-generates IDs for any model with an `ID` field whose value is zero. Each service gets a unique node ID: user-file-svc=1, im-svc=2. Snowflake must be initialized before database connection.

**JSON serialization**: All `uint64` ID fields use `json:"id,string"` tags to serialize as JSON strings, preventing JavaScript `number` precision loss (JS `number` max safe integer is 2^53-1, while Snowflake uint64 can exceed this). All frontend TypeScript ID types are `string`.

### Nginx proxy
Nginx runs in Docker (`deploy/docker-compose.single.yml`) as the single entry point on **port 80** — the only port exposed to the host:
- `/api/v1/im/*` → im-svc:8082 (Docker service name, not host IP)
- `/api/v1/docker/*` → docker-svc:8083
- `/api/*` → user-file-svc:8081
- `/ws` → im-svc:8082 (WebSocket upgrade)
- `/` → Serves `client/dist/` static files with SPA `try_files` fallback

The frontend connects to `window.location.host` for both API calls and WebSocket. Vite proxy config is retained as an alternative for devs who prefer not to run nginx.

### Docker multi-stage build
`server/Dockerfile` uses a `SERVICE` build arg to select which `cmd/` binary to build:
- Builder stage: `golang:1.25-alpine`, downloads deps, compiles with `CGO_ENABLED=0 GOOS=linux -ldflags="-s -w"`
- Runtime stage: `alpine:3.21` with `ca-certificates`, `tzdata`, `curl` (for healthcheck)
- docker-svc compiled as `GOOS=linux` defaults to `unix:///var/run/docker.sock` (no code change needed)

### Config files
- `config.single.yaml` — host development (all hostnames = `localhost`)
- `config.docker.yaml` — Docker deployment (hostnames = `postgres`, `redis`, `minio`)
- Set via `CONFIG_PATH` env var (each `main.go` reads it)

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
Public routes (no auth, no layout chrome): `/login`, `/register`, `/s/:code`
Protected routes (wrapped in `ProtectedRoute > AppLayout`): `/files`, `/shares`, `/chat`, `/friends`, `/docker`, `/admin`
Catch-all redirects to `/files`

## Testing

No test files exist yet. Infrastructure tests require running Docker services. API can be tested manually via curl — see `docs/test-data.md` for test account credentials and command examples.

## Common Gotchas

- **Working directory**: Go commands must run from `server/` directory (where `go.mod` is)
- **Windows Docker**: docker-svc in Docker uses `/var/run/docker.sock` (mounted from host). For host development on Windows, set `DOCKER_HOST=tcp://localhost:2375` and enable Docker Desktop TCP exposure.
- **Docker build context**: The build context is `../server` (relative to `deploy/`). All Go source changes invalidate the layer cache, so incremental builds recompile.
- **GORM AutoMigrate**: Each service's `main.go` calls AutoMigrate for its models. Adding a field to a model struct will auto-add the column on restart.
- **Config changes**: Services must be restarted to pick up config changes (e.g., JWT TTL)
- **docker-svc**: Does NOT connect to PostgreSQL or initialize Snowflake (no database models). DockerNode model is defined for future cluster features.
- **ID types**: All IDs are `string` on the frontend and `uint64` with `json:",string"` on the backend. Never use `number` for IDs in TypeScript code.
