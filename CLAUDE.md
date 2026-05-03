# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture

CloudNexus is a self-hosted collaboration platform with three Go microservices and a React frontend.

```
client/                    React 18 + TypeScript + Vite + Ant Design 5 + Zustand 5
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

### Infrastructure (first time)
```bash
cd deploy
docker compose up -d              # PostgreSQL:5432, Redis:6379, MinIO:9000/9001
```

### Backend (run from server/ directory)
```bash
cd server
go build ./cmd/user-file-svc/     # build one
go build ./cmd/...                # build all
go run ./cmd/user-file-svc &
go run ./cmd/im-svc &
go run ./cmd/docker-svc &
```

All services read `config/config.single.yaml` by default, overridable via `CONFIG_PATH` env var.

### Frontend
```bash
cd client
npm install
npm run dev                       # Vite dev server on :3000
npx tsc --noEmit                  # type-check only
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
All model IDs are generated via Snowflake algorithm (`pkg/snowflake/`). A GORM `BeforeCreate` callback in `pkg/database/postgres.go` auto-generates IDs for any model with an `ID` field whose value is zero. Each service gets a unique node ID: user-file-svc=1, im-svc=2. Snowflake must be initialized before database connection.

### Friend system
Bidirectional `Friend` model with `uniqueIndex:idx_friend_pair` on `(user_id, friend_id)`. Status: `pending` → `accepted`. Auto-creates private conversation on accept. If both users send requests simultaneously, the second auto-accepts. Friend queries return `FriendInfo` (embeds Friend + `friend_username` from JOIN on users table) for display without extra queries.

## Testing

No test files exist yet. Infrastructure tests require running Docker services. API can be tested manually via curl — see `docs/test-data.md` for test account credentials and command examples.

## Common Gotchas

- **Working directory**: Go commands must run from `server/` directory (where `go.mod` is)
- **Windows Docker**: The Docker service connects via `tcp://localhost:2375` on Windows (not Unix socket). Docker Desktop must expose this port.
- **GORM AutoMigrate**: Each service's `main.go` calls AutoMigrate for its models. Adding a field to a model struct will auto-add the column on restart.
- **Config changes**: Services must be restarted to pick up config changes (e.g., JWT TTL)
