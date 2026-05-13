# CloudNexus v0.1.0-dev 全面代码审查报告

**日期**: 2026-05-13  
**审查范围**: 4 个 Go 微服务 + React 前端 + 数据层 + 部署配置 + 安全  
**审查方法**: 4 个并行 Agent 分别审查后端/前端/数据/安全

---

## 严重 (Critical) — 共 9 项

| # | 来源 | 问题 | 位置 |
|---|------|------|------|
| C1 | 安全 | 硬编码默认管理员凭证 `admin / CloudNexus@admin` 首次部署时自动播种 | `deploy/docker-compose.single.yml:57-59` |
| C2 | 安全 | 所有配置硬编码默认密码 — DB/MiniO/JWT secret 全为默认值 | `config/*.yaml` |
| C3 | 前端 | `useAccess` 权限缓存永不过期 — `useMemo([], [])` JWT 刷新后权限不更新 | `hooks/useAccess.ts:32` |
| C4 | 前端 | `collab.ts` listDocuments 返回值层级错误 — 返回 `data` 而非 `data.data` | `services/collab.ts:22-23` |
| C5 | 前端 | `authStore.login` 不调 `fetchProfile` — 登录后 `user` 为 null | `stores/authStore.ts:25-31` |
| C6 | 前端 | `videoRecorder.destroy` 内存泄漏 — Object URL 从不释放 | `utils/videoRecorder.ts:90` |
| C7 | 后端 | 注册接口验证码被完全忽略 — 接受字段但从不校验 | `handler/user.go:38-56` |
| C8 | 后端 | docker-svc nil DB → panic — DB 连接失败后 `EndpointManager.db.Where()` 必崩 | `internal/dockermgr/service/endpoint.go:119` |
| C9 | 数据 | 迁移 007 全新部署必失败 — 引用 `face_profiles` 表但该表仅在 AutoMigrate 阶段创建 | `pkg/migration/007_face_thumbnail_text.up.sql` |

---

## 高 (High) — 共 16 项

| # | 来源 | 问题 | 位置 |
|---|------|------|------|
| H1 | 安全 | CORS `Access-Control-Allow-Origin: *` 且允许 `Authorization` 头 | `pkg/middleware/middleware.go:63` |
| H2 | 安全 | Nginx 纯 HTTP 无 TLS | `deploy/nginx/nginx.conf:44` |
| H3 | 安全 | JWT 中间件不检查吊销黑名单，令牌签发后无法失效 | `pkg/middleware/middleware.go:87` |
| H4 | 安全 | 令牌可通过 URL query param 传递（日志/Referer 泄露风险） | `pkg/middleware/middleware.go:81` |
| H5 | 安全 | AI 推理 `/detect` 端点完全无认证 | `deploy/ai-inference/app.py:52-131` |
| H6 | 安全 | Docker socket 挂载到容器（攻破 docker-svc = 宿主机 root） | `deploy/docker-compose.single.yml:115` |
| H7 | 后端 | `HandleConfirmDelete` 已实现但未注册路由 | `internal/userfile/handler/delete.go:37-44` |
| H8 | 后端 | 协作编辑 `collab/` 完整实现但未在任何 `main.go` 引入 | `internal/collab/` |
| H9 | 后端 | camera-svc 文件读取用 `f.Read(buf)` 不保证读满 | `internal/camera/handler/camera.go:256-257` |
| H10 | 后端 | IM WebSocket `CheckOrigin` 返回 `true` 允许任意来源 | `internal/im/handler/im.go:25-27` |
| H11 | 前端 | `UploadModal` 用不存在的 `mask` prop，应为 `maskClosable` | `components/UploadModal.tsx:72` |
| H12 | 前端 | 整个路由树零 Error Boundary | `App.tsx` |
| H13 | 前端 | 多处 `user!.id` 非空断言 — user 为 null 时运行时崩溃 | `pages/ChatPage.tsx:675`, `pages/FriendPage.tsx:83` |
| H14 | 数据 | 迁移 002 down.sql 缺 `container_name`/`version` 列的 DROP | `pkg/migration/002_node_service_sessions.down.sql` |
| H15 | 数据 | `FaceRecognitionEvent.FaceID` 是 `*uint64` 但 `json:",string"` 对指针无效 | `pkg/model/face.go:18` |
| H16 | 数据 | 多个服务 AutoMigrate 同一张表并发启动可能冲突 | `docker_nodes`, `node_online_sessions`, `alert_rules` 等 |

---

## 中 (Medium) — 共 17 项

| # | 来源 | 问题 | 位置 |
|---|------|------|------|
| M1 | 安全 | 全栈无速率限制（Nginx 无 `limit_req_zone`，应用层也无） | 全局 |
| M2 | 安全 | 缺少安全 HTTP 头 (`X-Content-Type-Options`, `X-Frame-Options`, `CSP`) | `pkg/middleware/middleware.go:62-71` |
| M3 | 安全 | 所有容器以 root 运行，Dockerfile 无 `USER` 指令 | `server/Dockerfile` |
| M4 | 安全 | 密码重置 token 通过 URL query string 发送（泄露到服务器日志） | `internal/userfile/service/reset.go:49` |
| M5 | 后端 | `SeedDefaultAdmin` 静默吞所有错误（无日志、无告警） | `internal/userfile/service/user.go:250-286` |
| M6 | 后端 | `SeedRBAC` 未检查权限批量创建是否成功即继续角色分配 | `internal/userfile/service/role.go:160` |
| M7 | 后端 | 4 个服务全部硬编码端口而非用 config 值 | 4 个 `cmd/*-svc/main.go` |
| M8 | 后端 | `SystemHandler`/`NodeHandler`/`AlertHandler` 直接注入 `*gorm.DB` 跳过 service 层 | `internal/userfile/handler/system.go:27` 等 |
| M9 | 后端 | `SearchHandler` 直接注入 repository 跳过 service 层 | `internal/userfile/handler/search.go:14` |
| M10 | 后端 | docker-svc 未传 `Service`/`LogDir` 给 logger，日志功能不完整 | `cmd/docker-svc/main.go:35` |
| M11 | 前端 | `ForgotPasswordPage` 用裸 axios + catch 块始终显示"发送成功" | `pages/ForgotPasswordPage.tsx:16-20` |
| M12 | 前端 | `fileStore` 多数写操作无错误处理 (remove/move/copy/mkdir) | `stores/fileStore.ts` |
| M13 | 前端 | `FaceLibraryPage` 硬编码 `/api/v1` URL 前缀 | `pages/FaceLibraryPage.tsx:55` |
| M14 | 前端 | 多个 `useEffect` 缺依赖项 | `FileListPage/ChatPage/DockerPage` |
| M15 | 数据 | 迁移 006 索引创建缺 `IF NOT EXISTS`，与其他迁移不一致 | `pkg/migration/006_file_versions.up.sql` |
| M16 | 数据 | 软删除实现不一致：`User`(json:"-") vs `File`(默认无索引) vs `ConversationMember`(json可见+索引) | `pkg/model/user.go` 等 |
| M17 | 数据 | `PasswordResetToken.UserID` 缺索引 | `pkg/model/password_reset.go:7` |

---

## 低 (Low) — 共 13 项

| # | 来源 | 问题 |
|---|------|------|
| L1 | 安全 | Redis 密码为空 |
| L2 | 安全 | `config.cluster.yaml` 与 `docker.yaml` 服务定义漂移 |
| L3 | 后端 | Snowflake 节点 ID 跳过 4 |
| L4 | 后端 | 指标端点重复注册 (`/api/v1/metrics` 和 `/metrics`) |
| L5 | 后端 | docker-svc 无 `X-Registry-Auth` 支持，无法拉私有镜像 |
| L6 | 后端 | 手机验证 handler 返回 501 但 service 实际已成功 |
| L7 | 后端 | docker-svc `snowflake.Init` 被调用但从不用 Snowflake ID |
| L8 | 前端 | `UploadModal` 文件上传前即显示 `status: 'done'` |
| L9 | 前端 | `ChatPage` read_receipt 存在空的 `else if` 块 |
| L10 | 前端 | `ShareAccessPage` 使用废弃的 `Spin.tip` prop |
| L11 | 前端 | `DockerPage` 用 `as any` 绕过 TypeScript |
| L12 | 前端 | `CameraLiveView` 不必要的动态 import |
| L13 | 数据 | BaseModel 嵌入不一致：有的内嵌，有的手动复制字段 |

---

## 统计

| 类别 | 严重 | 高 | 中 | 低 | 合计 |
|------|------|------|------|------|------|
| 安全 & 配置 | 2 | 6 | 4 | 2 | 14 |
| 后端 Go 服务 | 2 | 4 | 6 | 5 | 17 |
| 前端 React | 4 | 3 | 5 | 5 | 17 |
| 数据模型 & 迁移 | 1 | 3 | 3 | 1 | 8 |
| **合计** | **9** | **16** | **18** | **13** | **56** |

---

## 建议修复顺序

- **P0 (立即)**: C7 验证码校验, C8 nil DB 防护, C9 迁移 007, C4 collab.ts 返回值, C5 login后fetchProfile
- **P1 (本周)**: C1/C2 凭证外部化, H1 CORS限制, H3 JWT吊销, H7 注册删除路由, H8 接入协作文档, H10 CheckOrigin, H11 maskClosable, H15 FaceID 指针json tag
- **P2 (下迭代)**: H2 TLS, M1 速率限制, M2 安全头, M3 非root用户, M5/M6 种子数据错误处理, M7 端口可配置, M8/M9 架构统一
- **P3 (技术债)**: 所有 L 级 + M10-M17
