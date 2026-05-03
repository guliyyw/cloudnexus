# CloudNexus

Self-hosted collaboration platform — cloud storage, instant messaging, Docker management.

## Quick Start (Single Node)

```bash
# Start dependencies
docker compose -f deploy/docker-compose.single.yml up -d

# Build and run services
cd server
go run ./cmd/user-file-svc &
go run ./cmd/im-svc &
go run ./cmd/docker-svc &

# Start frontend
cd client
npm install && npm run dev
```

## Architecture

```
client (React + Vite)  →  Nginx  →  user-file-svc (:8081)
                                   →  im-svc (:8082)
                                   →  docker-svc (:8083)
                                        ↓
                              PostgreSQL + Redis + MinIO
```

## Project Structure

```
cloudnexus/
├── client/          Frontend (React + TypeScript)
├── server/          Go backend services
│   ├── cmd/         Service entry points
│   ├── internal/    Per-service logic
│   ├── pkg/         Shared packages
│   └── config/      YAML configs
├── deploy/          Docker & K8s deployment
├── docs/            Documentation
└── scripts/         Dev scripts
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| user-file-svc | 8081 | User auth & file management |
| im-svc | 8082 | Instant messaging & WebSocket |
| docker-svc | 8083 | Docker container management |

## License

MIT
