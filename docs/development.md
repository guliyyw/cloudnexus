# CloudNexus 开发指南

> 版本：v0.1.0 | 更新：2026-05-03

## 1. 环境准备

### 1.1 必需工具

| 工具 | 版本要求 | 用途 |
|------|----------|------|
| Go | 1.22+ | 后端开发 |
| Node.js | 18+ | 前端开发 |
| Docker | 24+ | 运行依赖服务 |
| Git | 2.x | 版本控制 |

### 1.2 推荐 IDE

- **GoLand** / **IntelliJ IDEA + Go 插件** — Go 开发
- **VS Code** — 前端开发

### 1.3 快速开始

```bash
# 克隆项目
git clone <repo-url> cloudnexus
cd cloudnexus

# 启动依赖服务
docker compose -f deploy/docker-compose.single.yml up -d

# 安装前端依赖
cd client && npm install && cd ..

# 运行脚本一键初始化 (Mac/Linux)
bash scripts/setup_dev.sh
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
│       ├── stores/                  # 状态管理
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
│   │   ├── database/               # PostgreSQL 连接
│   │   ├── cache/                   # Redis 客户端
│   │   ├── storage/                 # MinIO 对象存储客户端
│   │   ├── config/                  # YAML 配置加载
│   │   ├── model/                   # 共享数据模型
│   │   ├── response/               # HTTP 统一响应格式
│   │   └── errors/                  # 错误码定义
│   ├── config/                      # 配置文件
│   │   ├── config.single.yaml       # 单机部署
│   │   └── config.cluster.yaml      # 集群部署
│   └── go.mod
│
├── deploy/                          # 部署配置
│   ├── docker-compose.single.yml    # 单机基础设施
│   ├── docker-compose.cluster.yml   # 集群应用服务
│   ├── nginx/nginx.conf             # 反向代理配置
│   └── k8s/                         # Kubernetes 资源 (后期)
│
├── docs/                            # 项目文档
│   ├── api.md                       # API 接口文档
│   ├── database.md                  # 数据库设计
│   ├── deployment.md                # 部署指南
│   ├── development.md               # 开发指南 (本文件)
│   └── architecture.md              # 架构概览
│
└── scripts/                         # 工具脚本
    ├── setup_dev.sh
    └── migrate.sh
```

---

## 3. 开发工作流

### 3.1 启动后端服务

每个服务独立运行，开发时可逐个启动：

```bash
cd server

# 终端 1 — 用户文件服务
go run ./cmd/user-file-svc

# 终端 2 — IM 服务
go run ./cmd/im-svc

# 终端 3 — Docker 管理服务
go run ./cmd/docker-svc
```

验证：
```bash
curl http://localhost:8081/healthz  # → {"code":200,"message":"user-file-svc healthy"}
curl http://localhost:8082/healthz  # → {"code":200,"message":"im-svc healthy"}
curl http://localhost:8083/healthz  # → {"code":200,"message":"docker-svc healthy"}
```

### 3.2 启动前端

```bash
cd client
npm run dev
# 浏览器打开 http://localhost:3000
```

Vite 配置了代理：`/api/*` → `localhost:8081`，`/ws` → `localhost:8082`

### 3.3 编译命令

```bash
# 编译所有服务
go build ./cmd/...

# 编译单个服务
go build -o bin/user-file-svc ./cmd/user-file-svc

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
- 接口定义在使用方，而非实现方
- 错误处理：不忽略 error，使用 `pkg/errors` 中的 `AppError`
- 日志：使用标准库 `log`，后期可切换到 `slog` 或 `zap`

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
- 状态管理使用 React Context 或 Zustand

### 4.4 文件命名

- Go: `snake_case.go`
- TypeScript: `PascalCase.tsx` (组件), `camelCase.ts` (工具)
- 配置文件: `kebab-case.yaml`

---

## 5. 测试

### 5.1 Go 测试

```bash
# 运行所有测试
go test ./...

# 带覆盖率
go test -cover ./...

# 详细输出
go test -v ./...
```

### 5.2 测试规范

- 单元测试文件与源文件同目录：`xxx_test.go`
- 使用 [testify](https://github.com/stretchr/testify) 断言库
- 数据库测试使用测试专用 PostgreSQL 实例
- 集成测试放在 `internal/*/` 下的 `_test.go` 文件中

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

### 7.1 Go 调试

使用 delve:
```bash
dlv debug ./cmd/user-file-svc
```

或 IDE 内置调试器 (GoLand / VS Code Go 插件)。

### 7.2 前端调试

- Chrome DevTools
- React DevTools 浏览器扩展

### 7.3 数据库调试

```bash
# 连接本地 PostgreSQL
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

**Q: Docker 容器启动失败？**
```bash
docker compose -f deploy/docker-compose.single.yml logs postgres
docker compose -f deploy/docker-compose.single.yml logs redis
```

**Q: 端口被占用？**
```bash
# Windows
netstat -ano | findstr :8081

# Linux/Mac
lsof -i :8081
```
