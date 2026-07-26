# CloudNexus 开发指南

> 版本：v1.1.0 | 更新：2026-05-15

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
│       │   ├── Layout.tsx           # 应用主布局 (侧边栏 + 顶栏)
│       │   ├── ErrorBoundary.tsx    # 错误边界
│       │   ├── PreviewModal.tsx     # 文件预览模态框
│       │   ├── ShareModal.tsx       # 分享创建模态框
│       │   ├── UploadModal.tsx      # 文件上传模态框
│       │   ├── FilePickerModal.tsx  # 云盘文件选择器
│       │   ├── DirectoryPickerModal.tsx # 目录选择器
│       │   ├── FileVersionPanel.tsx # 文件版本历史面板
│       │   ├── AccessControl.tsx    # 权限控制组件
│       │   ├── FaceOverlay.tsx      # 人脸边界框叠加层
│       │   ├── FaceRegisterModal.tsx # 人脸注册模态框
│       │   └── VideoAnalysisPanel.tsx # 视频分析控制面板
│       ├── pages/                   # 页面级组件 (20个)
│       │   ├── LoginPage.tsx        # 登录
│       │   ├── RegisterPage.tsx     # 注册
│       │   ├── ForgotPasswordPage.tsx # 忘记密码
│       │   ├── ResetPasswordPage.tsx  # 重置密码
│       │   ├── ForbiddenPage.tsx    # 403 无权限
│       │   ├── FileListPage.tsx     # 文件管理
│       │   ├── MySharesPage.tsx     # 我的分享
│       │   ├── ShareAccessPage.tsx  # 分享落地页 (公开)
│       │   ├── ChatPage.tsx         # 即时通讯
│       │   ├── FriendPage.tsx       # 好友管理
│       │   ├── DockerPage.tsx       # Docker 管理
│       │   ├── CameraListPage.tsx   # 摄像头列表
│       │   ├── CameraLiveView.tsx   # 实时监控画面
│       │   ├── FaceLibraryPage.tsx  # 人脸库管理
│       │   ├── FaceAttendancePage.tsx # 考勤记录
│       │   ├── DocumentListPage.tsx # 在线文档列表
│       │   ├── DocumentEditorPage.tsx # 文档编辑器
│       │   ├── RecycleBinPage.tsx   # 回收站
│       │   ├── UserSettingsPage.tsx # 用户设置
│       │   └── AdminPage.tsx        # 管理后台
│       ├── hooks/                   # 自定义 React Hooks
│       │   ├── useWebSocket.ts      # WebSocket 连接管理 (IM)
│       │   ├── useAccess.ts         # 权限与角色检查
│       │   └── useChunkUpload.ts    # 分块上传 (断点续传)
│       ├── services/                # 后端 API 调用封装
│       │   ├── api.ts               # axios 实例 + 拦截器
│       │   ├── file.ts              # 文件/分享/版本/分块/回收站/配额 API
│       │   ├── chat.ts              # IM/好友/导入导出 API
│       │   ├── docker.ts            # Docker 容器/镜像 API
│       │   ├── camera.ts            # 摄像头/人脸/考勤/检测 API
│       │   ├── admin.ts             # 管理后台/节点/告警/配额 API
│       │   ├── captcha.ts           # 验证码 API
│       │   ├── verify.ts            # 邮箱/手机验证 API
│       │   └── collab.ts            # 在线文档 API
│       ├── stores/                  # 状态管理 (Zustand)
│       │   ├── authStore.ts         # 用户认证状态
│       │   ├── fileStore.ts         # 文件列表状态
│       │   ├── chatStore.ts         # 聊天状态
│       │   ├── friendStore.ts       # 好友状态
│       │   ├── dockerStore.ts       # Docker 状态
│       │   └── quotaStore.ts        # 配额状态
│       └── utils/                   # 工具函数
│           ├── format.ts            # 文件大小格式化
│           ├── preview.ts           # 文件预览类型判断
│           ├── cameraDiscovery.ts   # 局域网摄像头发现
│           ├── faceDetection.ts     # 人脸检测 (face-api.js)
│           ├── faceTracker.ts       # 人脸跟踪 (IoU 匹配)
│           └── videoRecorder.ts     # Canvas 视频录制
│
├── server/                          # Go 后端
│   ├── cmd/                         # 服务入口 (每个子目录 = 一个服务)
│   │   ├── user-file-svc/           # 用户 & 文件服务 (8081)
│   │   ├── im-svc/                  # 即时通讯服务 (8082)
│   │   ├── docker-svc/              # Docker 管理服务 (8083)
│   │   ├── camera-svc/              # 摄像头 & AI 服务 (8085)
│   │   └── collab-svc/              # 在线文档协作服务 (8086)
│   ├── internal/                    # 服务私有逻辑 (Go 编译器强制隔离)
│   │   ├── userfile/                # handler → service → repository
│   │   │   ├── handler/             # user, file, share, trash, chunk, quota, system, captcha, verify, session, reset, delete, oauth, node, alert, search, role (17个)
│   │   │   ├── service/             # user, file, share, trash, chunk, quota, verify, session, reset, delete, oauth, role, cleanup_scheduler (13个)
│   │   │   └── repository/          # user, file, share, quota, chunk, role (6个)
│   │   ├── im/                      # handler → service → repository
│   │   │   ├── handler/             # im, friend
│   │   │   ├── service/             # hub (WebSocket), im, blocklist, presence
│   │   │   └── repository/          # im
│   │   ├── dockermgr/               # handler → service
│   │   │   ├── handler/             # docker
│   │   │   └── service/             # docker, endpoint
│   │   ├── camera/                  # handler → service → repository
│   │   │   ├── handler/             # camera, face
│   │   │   ├── service/             # camera, recognition, face, discovery
│   │   │   └── repository/          # camera
│   │   └── collab/                  # handler → service
│   │       ├── handler/             # collab
│   │       └── service/             # hub (Yjs WebSocket)
│   ├── pkg/                         # 跨服务共享包 (18个)
│   │   ├── auth/                    # JWT 令牌生成与校验
│   │   ├── middleware/              # Gin 中间件 (CORS, Logger, AuthRequired, AdminRequired, RequirePermission)
│   │   ├── database/                # PostgreSQL 连接 + Snowflake ID 回调
│   │   ├── cache/                   # Redis 客户端
│   │   ├── storage/                 # MinIO 对象存储客户端
│   │   ├── config/                  # YAML 配置加载
│   │   ├── model/                   # 共享数据模型 (18个模型文件)
│   │   ├── snowflake/               # Snowflake ID 生成
│   │   ├── response/                # HTTP 统一响应格式
│   │   ├── errors/                  # 错误码定义 (AppError + 哨兵错误)
│   │   ├── logger/                  # Zap 封装 (环形缓冲 + 按天分文件 + 30天清理)
│   │   ├── migration/               # 版本化 SQL 迁移 (go:embed + schema_migrations)
│   │   ├── crypto/                  # bcrypt 密码哈希
│   │   ├── captcha/                 # 图片验证码生成 + Redis 存储
│   │   ├── email/                   # SMTP 邮件发送
│   │   └── system/                  # 健康检查 + 节点注册 + 健康聚合 + 告警评估
│   ├── config/                      # 配置文件
│   │   ├── config.single.yaml       # 宿主机开发
│   │   ├── config.docker.yaml       # Docker 部署
│   │   └── config.cluster.yaml      # 集群部署
│   ├── bin/linux/                   # 预编译 Linux 二进制文件 (部署用)
│   ├── Dockerfile                   # 多阶段构建 (SERVICE build arg)
│   ├── .dockerignore
│   └── go.mod
│
├── deploy/                          # 部署配置
│   ├── docker-compose.single.yml    # 单机全栈 Docker Compose (12个服务)
│   ├── docker-compose.cluster.yml   # 集群应用服务
│   ├── deploy.sh                    # 远程自动部署脚本
│   ├── nginx/nginx.conf             # 反向代理 + 静态文件
│   ├── mediamtx/mediamtx.yml        # 流媒体服务器配置
│   ├── ai-inference/                # AI 推理服务
│   │   ├── Dockerfile               # Python YOLOv8 镜像
│   │   └── app.py                   # FastAPI 推理 API
│   └── k8s/                         # Kubernetes 资源 (预留)
│
├── docs/                            # 项目文档
│   ├── openapi.yaml                 # OpenAPI 3.0 接口规范
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

# 2. 启动 Go 服务（使用 localhost 配置）
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/user-file-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/im-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/docker-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/camera-svc &
CONFIG_PATH=config/config.single.yaml go run ./cmd/collab-svc &

# 3. 启动前端
cd client && npm run dev
# 访问 http://localhost:3000 (Vite 自带代理)
```

### 3.3 方式三：Vite 代理（纯前端开发）

不需要 nginx 时，Vite 自带代理转发 API 请求：

```bash
cd client && npm run dev
# Vite 自动代理：
#   /api/v1/im/*        → localhost:8082
#   /api/v1/docker/*    → localhost:8083
#   /api/v1/cameras/*   → localhost:8085
#   /api/v1/faces/*     → localhost:8085
#   /api/v1/detect-*    → localhost:8085
#   /api/v1/collab/*    → localhost:8086
#   /api/*              → localhost:8081
#   /ws/collab/*        → localhost:8086 (WebSocket)
#   /ws                 → localhost:8082 (WebSocket)
```

### 3.4 编译命令

```bash
# 编译所有服务
go build ./cmd/...

# 编译单个服务
go build -o bin/user-file-svc ./cmd/user-file-svc

# 交叉编译 Linux (部署用)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/linux/ ./cmd/...

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

### 4.4 主题系统

项目支持**暗色主题** (Active Theory 风格) 和**亮色主题** (暖橙风格)，通过 TopNav 按钮一键切换。

- **暗色**: 深蓝黑背景 `#04050a`、青蓝主色 `#81ecfe`、玻璃态卡片、四层 CSS 背景（光晕/噪点/暗角/旋转光斑）
- **亮色**: 暖白背景 `#fafaf8`、暖橙主色 `#e8964a`、无背景装饰层

**核心架构**: `tokens.ts` 中定义双套 Token，`colors`/`radius`/`shadow`/`chart` 导出为可变对象。切换主题时 `applyTheme()` 通过 `Object.assign` 原地更新属性值，29 个组件文件无需修改即可响应主题变更。

**状态管理**: `stores/themeStore.ts` (Zustand)，持久化到 `localStorage` (key: `cloudnexus-theme`)。初始化时从 localStorage 读取并设置 `<html data-theme="dark|light">`。

**CSS 选择器**: 所有暗色背景层包裹在 `[data-theme="dark"]` 中，亮色模式用 `content: none`/`display: none` 隐藏。滚动条样式分别定义。`index.html` 中有防闪烁内联脚本，在首帧渲染前设置 `data-theme`。

**Ant Design**: `App.tsx` ConfigProvider 根据 `isDark` 动态使用 `theme.darkAlgorithm` / `theme.defaultAlgorithm`，token 和 components 中的硬编码值按主题分支。

**颜色 Token** 统一定义在 `theme/tokens.ts`：

```typescript
import { colors, radius, shadow, spacing, motion, chart } from '../theme/tokens'
// colors.primary, colors.text, colors.textSecondary, colors.bgCard ...
// chart.gridStroke, chart.tickFill, chart.tooltip ...
```

- 组件内联样式硬编码颜色时，优先引用 `tokens.ts` 中的值
- Ant Design 组件通过 `App.tsx` 的 `ConfigProvider theme` 自动适配，80% 组件无需手动设色
- Recharts 图表使用 `chart` Token 对象统一网格线、刻度、提示框样式
- 暗色全局效果（光晕/噪点/暗角/旋转光斑）在 `index.css` 中通过 CSS 伪元素实现

### 4.5 RBAC 权限控制

后端通过 `middleware.RequirePermission(permCode)` 中间件控制 API 访问，前端通过 `useAccess` Hook 控制 UI 元素可见性：

```go
// 后端：路由注册时指定所需权限
api.POST("/file/upload", fileHandler.HandleUpload, middleware.RequirePermission("file:write"))
```

```typescript
// 前端：条件渲染受控 UI
const { hasPermission } = useAccess();
{hasPermission('file:write') && <UploadButton />}
```

预置角色：`super_admin`（全部权限）、`admin`（管理权限）、`user`（基本权限）。

### 4.6 文件命名

- Go: `snake_case.go`
- TypeScript: `PascalCase.tsx` (组件), `camelCase.ts` (工具)
- 配置文件: `kebab-case.yaml`

---

## 5. 配置说明

三个配置文件对应不同部署模式：

| 文件 | host 值 | 用途 |
|------|---------|------|
| `config.single.yaml` | `localhost` | 宿主机开发 |
| `config.docker.yaml` | `postgres`, `redis`, `minio` | Docker 部署 |
| `config.cluster.yaml` | 基础设施服务器 IP | 集群部署 |

通过 `CONFIG_PATH` 环境变量指定，每个 `main.go` 已支持。

---

## 6. 添加新功能

以"在用户服务中新增获取用户列表接口"为例：

1. **定义模型** — 在 `pkg/model/` 中添加/复用模型结构体
2. **编写 repository** — `internal/userfile/repository/` 中添加数据访问方法
3. **编写 service** — `internal/userfile/service/` 中添加业务逻辑
4. **编写 handler** — `internal/userfile/handler/` 中添加 HTTP 处理函数
5. **注册路由** — 在 `cmd/user-file-svc/main.go` 中添加路由（含权限中间件）
6. **更新文档** — 在 `docs/openapi.yaml` 中记录新接口

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

---

## 9. 短剧工坊开发补充

`drama-svc` 是 AI 短剧工坊的独立 Go 服务，默认监听 `8087`，前端入口为 `/drama`，API 前缀为 `/api/v1/drama`。本模块遵循 `handler -> service -> repository` 三层结构，代码位于：

- `server/cmd/drama-svc/`：服务启动、依赖初始化、路由注册、节点注册、任务执行器启动。
- `server/internal/drama/handler/`：项目、分镜、资产、任务、设置等 HTTP 入口。
- `server/internal/drama/service/`：剧本解析、ComfyUI 调用、生成任务执行、任务队列。
- `server/internal/drama/repository/`：短剧项目、分镜、媒体、资产、任务和设置的数据访问。
- `client/src/pages/DramaPage.tsx`：短剧工坊主页面。
- `client/src/services/drama.ts`：短剧 API 类型与请求封装。

本地开发时，后端可按单服务方式启动：

```bash
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/drama-svc
```

短剧生成依赖 ComfyUI。开发调试时优先通过 `/api/v1/drama/settings/comfyui/status` 或前端“ComfyUI 检测”确认连通性、checkpoint、IP-Adapter、ReActor 以及 Wan2.2/IP-Adapter 相关本地模型是否就绪。

任务执行由 `server/internal/drama/service/task_runner.go` 管理：任务进入 Redis 队列 `drama:tasks:queue` 后异步执行，执行进度写回 `drama_tasks`，并通过事件发布给前端。排查失败任务时，应优先查看任务详情里的生成提示词、原始 payload、错误消息和 ComfyUI 模型检查清单。
---

## 10. 多格式在线文档开发说明

### 10.1 前端依赖

文档工作台新增以下依赖：

| 包 | 用途 |
|----|------|
| `marked` | Markdown 预览与粘贴转换 |
| `mammoth` | `.docx` 导入为 HTML |
| `docx-preview` | 按 Word 原始页面尺寸、分页、页眉页脚和表格样式预览 `.docx` |
| `xlsx` | Excel/CSV 导入导出、Sheet/公式/合并区域处理 |

安装或更新依赖：

```bash
cd client
npm install
```

### 10.2 编辑器入口

核心入口为 `client/src/pages/DocumentEditorPage.tsx`，同一页面根据文件元信息和扩展名分流：

- `.clouddoc`：沿用 TipTap + Yjs + WebSocket 实时协作。
- `.md` / `.markdown`：源码、预览、双栏三模式，保存走 `PUT /api/v1/file/:id/text`。
- `.doc` / `.docx`：下载原文件后用 mammoth 导入为 HTML，进入富文本编辑，支持导出 `.docx` 和服务端转 PDF。
- `.xls` / `.xlsx` / `.csv`：用 xlsx 解析 workbook，支持多个 sheet、单元格编辑、筛选、按列排序、公式输入，导出 `.xlsx`。
- `.pdf`：使用现有文件下载接口以内联方式预览，不编辑 PDF 正文。

文件列表入口在 `client/src/pages/FileListPage.tsx` 中判断这些扩展名并跳转到 `/files/:id/edit`。

### 10.3 后端接口

`user-file-svc` 新增：

| 位置 | 说明 |
|------|------|
| `FileHandler.HandleSaveText` | 接收文本内容并保存为新对象，同时保留旧版本 |
| `FileService.SaveTextContent` | 写入 MinIO、更新文件大小和配额、记录版本 |
| `FileHandler.HandleConvertWordToPDF` | 返回 Word 转换后的 PDF |
| `FileService.ConvertWordToPDF` | 临时下载 Word 文件，调用 LibreOffice/soffice 转 PDF，响应后清理临时目录 |

路由注册在 `server/cmd/user-file-svc/main.go`：

```go
file.PUT("/:id/text", middleware.RequirePermission("file:write"), fileH.HandleSaveText)
file.PUT("/:id/word", middleware.RequirePermission("file:write"), fileH.HandleSaveWord)
file.POST("/:id/convert/docx", middleware.RequirePermission("file:read"), fileH.HandleExportWord)
file.GET("/:id/convert/pdf", middleware.RequirePermission("file:read"), fileH.HandleConvertWordToPDF)
```

### 10.4 本地验证

```bash
cd client
npm run build

cd ../deploy
docker compose -f docker-compose.single.yml build user-file-svc
docker compose -f docker-compose.single.yml up -d user-file-svc
docker compose -f docker-compose.single.yml restart nginx
```

如果在宿主机直接运行 `go test`，需要本机 PATH 中存在 `go` 和 `gofmt`。Docker 构建 `user-file-svc` 会在镜像内完成 Go 编译校验。


### ???? Office ??

???? `createOfficeDoc(title, parentId, kind)` ?? `POST /api/v1/file/office`?`kind=word` ?? `.docx`?`kind=excel` ?? `.xlsx`???? `FileService.CreateOfficeDoc` ????? OOXML ????? MinIO??????????????????
