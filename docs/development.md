# CloudNexus 开发指南

> 版本：v1.0.0 | 更新：2026-05-06

## 1. 环境准备

### 1.1 必需工具

| 工具 | 版本要求 | 用途 |
|------|----------|------|
| Go | 1.25+ | 后端开发 |
| Node.js | 18+ | 前端开发 |
| Docker | 24+ | 运行所有服务（含后端） |
| Git | 2.x | 版本控制 |

### 1.2 推荐 IDE

- **GoLand** / **IntelliJ IDEA + Go 插件** — Go 开发
- **VS Code** — 前端开发

### 1.3 快速开始

```bash
# 克隆项目
git clone <repo-url> cloudnexus
cd cloudnexus

# 安装前端依赖
cd client && npm install

# 构建前端
npm run build

# 启动全栈 (Docker)
cd ../deploy
docker compose -f docker-compose.single.yml up --build -d

# 访问 http://localhost
```

---

## 2. 项目结构

```
cloudnexus/
├── client/                          # 前端 (React + TypeScript + Vite)
│   └── src/
│       ├── components/              # 可复用 UI 组件
│       ├── pages/                   # 页面级组件
│       ├── hooks/                   # 自定义 React Hooks
│       ├── services/                # 后端 API 调用封装
│       ├── stores/                  # 状态管理 (Zustand)
│       └── utils/                   # 工具函数
│
├── server/                          # Go 后端
│   ├── cmd/                         # 服务入口 (每个子目录 = 一个服务)
│   │   ├── user-file-svc/           # 用户 & 文件服务
│   │   ├── im-svc/                  # 即时通讯服务
│   │   └── docker-svc/              # Docker 管理服务
│   ├── internal/                    # 服务私有逻辑 (Go 编译器强制隔离)
│   │   ├── userfile/                # handler → service → repository
│   │   ├── im/
│   │   └── dockermgr/
│   ├── pkg/                         # 跨服务共享包
│   │   ├── auth/                    # JWT 令牌生成与校验
│   │   ├── middleware/              # Gin 中间件 (CORS, 日志, 认证)
│   │   ├── database/               # PostgreSQL 连接 + Snowflake ID 回调
│   │   ├── cache/                   # Redis 客户端
│   │   ├── storage/                 # MinIO 对象存储客户端
│   │   ├── config/                  # YAML 配置加载
│   │   ├── model/                   # 共享数据模型
│   │   ├── snowflake/               # Snowflake ID 生成
│   │   ├── response/               # HTTP 统一响应格式
│   │   ├── errors/                  # 错误码定义 (AppError + 哨兵错误)
│   │   ├── logger/                  # Zap 封装 (环形缓冲 + 按天分文件 + 30天清理)
│   │   ├── migration/               # 版本化 SQL 迁移 (go:embed + schema_migrations)
│   │   └── crypto/                  # bcrypt 密码哈希
│   ├── config/                      # 配置文件
│   │   ├── config.single.yaml       # 宿主机开发
│   │   ├── config.docker.yaml       # Docker 部署
│   │   └── config.cluster.yaml      # 集群部署
│   ├── Dockerfile                   # 多阶段构建 (SERVICE build arg)
│   ├── .dockerignore
│   └── go.mod
│
├── deploy/                          # 部署配置
│   ├── docker-compose.single.yml    # 单机全栈 Docker Compose
│   ├── docker-compose.cluster.yml   # 集群应用服务
│   ├── nginx/nginx.conf             # 反向代理 + 静态文件
│   └── k8s/                         # Kubernetes 资源 (预留)
│
├── docs/                            # 项目文档
│   ├── api.md                       # API 接口文档
│   ├── database.md                  # 数据库设计
│   ├── deployment.md                # 部署指南
│   ├── development.md               # 开发指南 (本文件)
│   ├── architecture.md              # 架构概览
│   ├── test-data.md                 # 测试数据参考
│   └── progress.md                  # 开发进度
│
└── scripts/                         # 工具脚本
```

---

## 3. 开发工作流

### 3.1 方式一：Docker 全栈开发（推荐）

所有服务运行在 Docker 中，代码变更后重建对应服务：

```bash
cd client && npm run build       # 前端变更后
cd deploy
docker compose -f docker-compose.single.yml up --build -d    # 全量重建

# 或只重建单个 Go 服务（更快）
docker compose -f docker-compose.single.yml up --build -d im-svc
```

### 3.2 方式二：宿主机开发（Go 开发迭代快）

Go 服务在宿主机运行，基础设施 + nginx 在 Docker 中：

```bash
# 1. 启动基础设施
cd deploy
docker compose -f docker-compose.single.yml up -d postgres redis minio

# 2. 修改 nginx.conf 将 proxy_pass 指向 host.docker.internal
# （或使用 Vite 自带的代理，见 3.3）

# 3. 启动 Go 服务（使用 localhost 配置）
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/user-file-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/im-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/docker-svc &

# 4. 启动前端
cd client && npm run dev
# 访问 http://localhost:3000 (Vite 自带代理)
```

### 3.3 方式三：Vite 代理（纯前端开发）

不需要 nginx 时，Vite 自带代理转发 API 请求：

```bash
cd client && npm run dev
# Vite 自动代理：
#   /api/v1/im/*    → localhost:8082
#   /api/v1/docker/* → localhost:8083
#   /api/*          → localhost:8081
#   /ws             → localhost:8082
```

### 3.4 编译命令

```bash
# 编译所有服务
go build ./cmd/...

# 编译单个服务
go build -o bin/user-file-svc ./cmd/user-file-svc

# Docker 构建单个服务
docker build --build-arg SERVICE=user-file-svc -t user-file-svc .

# 静态分析
go vet ./...

# 格式化
go fmt ./...
```

---

## 4. 代码规范

### 4.1 Go 代码

- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 包名小写单数：`handler` 不是 `handlers`
- 错误处理：使用 `pkg/errors` 中的 `AppError`
- 日志：使用 `pkg/logger`（zap 封装，环形缓冲 + 按天分文件 + 30天清理）
- **ID 类型**：所有 uint64 ID 字段 JSON tag 加 `,string`，避免 JavaScript 精度丢失

### 4.2 分层架构

每个服务内部采用三层结构：

```
handler (HTTP 层)  →  解析请求、校验参数、调用 service
    ↓
service (业务层)   →  核心业务逻辑、事务编排
    ↓
repository (数据层) → 数据库操作、缓存访问
```

规则：
- handler 不直接访问数据库
- repository 不做业务判断
- service 是唯一有业务逻辑的地方

### 4.3 TypeScript 代码

- 严格模式 (`tsconfig.json` 中 `strict: true`)
- 使用函数组件 + Hooks，不写 class 组件
- API 调用统一封装在 `services/` 中
- 状态管理使用 Zustand
- **ID 类型**：所有 ID 为 `string` 类型（对应后端 Snowflake uint64）

### 4.4 文件命名

- Go: `snake_case.go`
- TypeScript: `PascalCase.tsx` (组件), `camelCase.ts` (工具)
- 配置文件: `kebab-case.yaml`

---

## 5. 配置说明

两个配置文件对应不同部署模式：

| 文件 | host 值 | 用途 |
|------|---------|------|
| `config.single.yaml` | `localhost` | 宿主机开发 |
| `config.docker.yaml` | `postgres`, `redis`, `minio` | Docker 部署 |

通过 `CONFIG_PATH` 环境变量指定，每个 `main.go` 已支持。

---

## 6. 添加新功能

以"在用户服务中新增获取用户列表接口"为例：

1. **定义模型** — 在 `pkg/model/` 中已有 `User` 结构体
2. **编写 repository** — `internal/userfile/repository/user.go` 中添加 `ListUsers()`
3. **编写 service** — `internal/userfile/service/user.go` 中添加 `GetUserList()`
4. **编写 handler** — `internal/userfile/handler/user.go` 中添加 `HandleListUsers`
5. **注册路由** — 在 `cmd/user-file-svc/main.go` 中添加路由
6. **更新文档** — 在 `docs/api.md` 中记录新接口

---

## 7. 调试

### 7.1 查看 Docker 日志

```bash
docker compose -f deploy/docker-compose.single.yml logs -f user-file-svc
docker compose -f deploy/docker-compose.single.yml logs -f im-svc
```

### 7.2 数据库调试

```bash
docker exec -it deploy-postgres-1 psql -U cloudnexus
# 查看表结构
\d users
# 查看索引
\di
```

---

## 8. 常见问题

**Q: `go mod tidy` 下载依赖慢？**
```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

**Q: Docker 镜像构建慢？**
- 首次构建需要下载 Go 依赖，后续利用 Docker 层缓存
- 可在 Dockerfile 中设置 `ENV GOPROXY=https://goproxy.cn,direct`

**Q: 端口被占用？**
```bash
# Windows
netstat -ano | findstr :80
# Linux/Mac
lsof -i :80
```
