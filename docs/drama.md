# AI 短剧工坊

> 本文描述当前实现，以代码为准。

AI 短剧工坊提供从剧本拆解到图像、视频和音频素材管理的完整流程。前端入口是 `/drama`，后端服务是 `drama-svc`，默认端口 `8087`。

## 模块边界

| 层 | 位置 | 职责 |
|---|---|---|
| 前端页面 | `client/src/pages/DramaPage.tsx` | 项目、分镜、片段、资产、任务和设置界面 |
| 前端 API | `client/src/services/drama.ts` | 短剧接口类型和请求封装 |
| 服务入口 | `server/cmd/drama-svc/main.go` | 初始化依赖、注册路由和启动任务执行器 |
| HTTP 层 | `server/internal/drama/handler/` | 参数校验、鉴权和响应封装 |
| 业务层 | `server/internal/drama/service/` | 剧本解析、提示词构建、生成任务和文件保存 |
| 数据层 | `server/internal/drama/repository/` | 项目、分镜、片段、资产、媒体和任务的数据库访问 |

## 生成流程

```text
创建项目
  → 导入或粘贴剧本
  → 解析为分镜
  → 导入片段和角色/场景资产
  → 上传或生成参考图
  → 创建图片/视频任务
  → Redis 队列异步执行
  → ComfyUI 生成
  → 结果保存到 MinIO 和云盘文件表
  → 回写分镜媒体、片段媒体和任务结果
```

### 1. 项目与剧本

创建项目时保存标题、描述和视觉风格。解析剧本后，服务生成分镜、项目前言和角色/场景资产。追加剧本会在已有分镜之后继续生成，不覆盖原有分镜。

### 2. 分镜、片段和资产

- 分镜保存剧情文本、场景锚点、对白和基础提示词。
- 片段保存时长、角色、动作、镜头、构图提示词、视频提示词和负面提示词。
- 资产分为 `character` 和 `scene`，可保存描述、参考提示词和参考图。
- 视频生成优先选择片段参考图，其次选择分镜媒体或资产参考图作为首帧。

### 3. 图片任务

图片任务支持三种场景：

- `image_generation`：独立图片生成页创建的任务。
- `image`：批量生成分镜图片。
- `asset_reference`：生成角色或场景资产参考图。

图片任务会读取短剧设置中的 SDXL checkpoint、尺寸、采样步数、CFG、采样器和负面提示词，并根据项目视觉风格选择合适的 checkpoint。

### 4. 视频任务

当前视频任务类型为 `video`，执行流程是：

1. 读取分镜或片段的首帧。
2. 构建包含场景、角色、动作、镜头和音频意图的提示词。
3. 上传首帧到 ComfyUI。
4. 优先使用本地 Wan2.2 I2V 工作流。
5. 轮询 ComfyUI 历史记录并下载生成的视频。
6. 保存视频文件，并写入分镜/片段媒体记录。

如果没有可用首帧，任务会直接失败，并提示先生成或选择图片。视频任务默认时长由任务执行逻辑控制，未实现独立的“视频默认参数 JSON”配置。

### 5. 任务执行

`server/internal/drama/service/task_runner.go` 负责：

- 从 Redis 队列取出任务。
- 恢复服务重启前的 `pending` 和 `running` 任务。
- 更新进度和状态。
- 支持取消、失败重试和任务事件订阅。

任务 payload 中保留来源、提示词日志和生成结果，便于定位生成失败。

## ComfyUI 检查

接口：`GET /api/v1/drama/settings/comfyui/status`

检查内容包括：

- ComfyUI 是否可连接。
- 可用 checkpoint。
- IP-Adapter 和 ReActor 节点。
- CLIP Vision、IP-Adapter、FaceID 和 Wan2.2 模型。

检查通过不代表每个工作流都可用；最终仍以任务提交时的节点校验和 ComfyUI 返回结果为准。

## API 摘要

| Endpoint | Method | 说明 |
|---|---|---|
| `/api/v1/drama/projects` | GET / POST | 项目列表 / 创建项目 |
| `/api/v1/drama/projects/:id` | GET / PUT / DELETE | 项目详情 / 更新 / 删除 |
| `/api/v1/drama/projects/:id/parse` | POST | 解析剧本 |
| `/api/v1/drama/projects/:id/append` | POST | 追加分镜 |
| `/api/v1/drama/projects/:id/tasks` | GET / POST | 任务列表 / 创建任务 |
| `/api/v1/drama/projects/:id/tasks/:taskId/cancel` | POST | 取消任务 |
| `/api/v1/drama/projects/:id/tasks/:taskId/retry` | POST | 重试任务 |
| `/api/v1/drama/settings` | GET / PUT | 获取 / 保存设置 |
| `/api/v1/drama/settings/comfyui/status` | GET | 检查 ComfyUI |

## 当前未实现内容

- TTS 任务执行器未接入，不在生成任务类型和设置页中展示。
- 视频设置 JSON 暂不参与视频工作流参数生成。
- 最终短剧拼接、配音合成和字幕烧录不属于当前 `drama-svc` 生成任务链路。

本地启动：

```bash
cd server
CONFIG_PATH=config/config.single.yaml go run ./cmd/drama-svc
```
