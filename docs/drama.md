# AI 短剧工坊

> 更新日期：2026-07-18

AI 短剧工坊是 CloudNexus 的剧本拆解与媒体生成工作台，对应前端 `/drama` 页面和后端 `drama-svc` 服务。它把项目、分镜、片段、角色/场景资产、生成任务和 ComfyUI 连接配置放在同一个工作流里，目标是让短剧从文本脚本进入可追踪、可复现的图像/视频生成流程。

## 服务边界

- 前端入口：`client/src/pages/DramaPage.tsx`
- 前端 API：`client/src/services/drama.ts`
- 后端入口：`server/cmd/drama-svc/main.go`
- 后端业务：`server/internal/drama/`
- 数据模型：`DramaProject`、`DramaStoryboard`、`DramaStoryboardSegment`、`DramaStoryboardMedia`、`DramaAsset`、`DramaTask`、`DramaSetting`
- 默认端口：`8087`
- API 前缀：`/api/v1/drama`

`drama-svc` 依赖 PostgreSQL 保存项目与任务数据，依赖 MinIO 保存生成文件和参考图，依赖 Redis 维护任务队列与任务事件发布。

## 核心能力

- 项目管理：创建、编辑、删除、导入、导出短剧项目。
- 剧本拆解：把脚本文本解析为分镜，并支持继续追加分镜。
- 片段管理：每个分镜可导入片段 JSON，片段可维护动作、构图、图片提示词、视频提示词和负面提示词。
- 资产管理：支持角色与场景资产，资产可保存描述、参考提示词、声音名称和参考图片。
- 音频管理：支持单个分镜上传音频，也支持批量导入音频并按文件名匹配分镜。
- 生成任务：支持资产参考图、分镜图片和分镜视频生成任务，任务可取消、重试、查看进度和任务详情。
- 任务详情：前端可查看任务来源、进度、结果、生成提示词日志和原始 payload，便于排查失败任务。
- ComfyUI 检测：设置页可检测 ComfyUI 连通性、checkpoint、IP-Adapter、ReActor 以及关键本地模型是否就绪。

## ComfyUI 依赖

图片一致性工作流依赖：

- `CLIP-ViT-H-14-laion2B-s32B-b79K.safetensors`
- `ip-adapter-plus_sdxl_vit-h.safetensors`
- 可选：`ip-adapter-plus-face_sdxl_vit-h.safetensors`

Wan2.2 本地视频工作流依赖：

- `wan2.2_i2v_high_noise_14B_fp8_scaled.safetensors`
- `wan2.2_i2v_low_noise_14B_fp8_scaled.safetensors`
- `umt5_xxl_fp8_e4m3fn_scaled.safetensors`
- `wan_2.1_vae.safetensors`

`GET /api/v1/drama/settings/comfyui/status` 会返回 `models` 与 `missing` 字段，前端会把上述模型展示为检查清单。

## 任务执行

`task_runner.go` 是短剧工坊的异步任务执行器。它启动后从 Redis 队列 `drama:tasks:queue` 中取出任务，更新任务状态，并通过 Redis Pub/Sub 向前端发布任务进度。服务重启后会恢复 `pending` 或 `running` 状态的任务，避免任务长期卡在中间态。

任务执行过程中会把生成结果写回任务 payload，并尽量保留：

- 来源信息：项目、分镜、片段或资产。
- 生成结果：生成文件 ID、URL、标题和类型。
- 提示词日志：每个生成目标使用的最终提示词。
- 错误信息：ComfyUI 节点、模型缺失、输出文件缺失等失败原因。

## API 摘要

| Endpoint | Method | 说明 |
|---|---|---|
| `/api/v1/drama/projects` | GET / POST | 项目列表 / 创建项目 |
| `/api/v1/drama/projects/import` | POST | 导入项目 |
| `/api/v1/drama/projects/:id` | GET / PUT / DELETE | 项目详情 / 更新 / 删除 |
| `/api/v1/drama/projects/:id/parse` | POST | 解析剧本 |
| `/api/v1/drama/projects/:id/append` | POST | 追加分镜 |
| `/api/v1/drama/projects/:id/export` | GET | 导出项目 |
| `/api/v1/drama/projects/:id/tasks` | GET / POST | 任务列表 / 创建生成任务 |
| `/api/v1/drama/projects/:id/tasks/:taskId/cancel` | POST | 取消任务 |
| `/api/v1/drama/projects/:id/tasks/:taskId/retry` | POST | 重试任务 |
| `/api/v1/drama/projects/:id/storyboards/:storyboardId` | PUT | 更新分镜 |
| `/api/v1/drama/projects/:id/storyboards/:storyboardId/media/:mediaId/select` | PUT | 选择分镜媒体 |
| `/api/v1/drama/projects/:id/storyboards/:storyboardId/media/:mediaId` | DELETE | 删除分镜媒体 |
| `/api/v1/drama/projects/:id/storyboards/:storyboardId/segments/import` | POST | 导入分镜片段 |
| `/api/v1/drama/projects/:id/storyboards/:storyboardId/audio` | POST | 上传分镜音频 |
| `/api/v1/drama/projects/:id/audio/import` | POST | 批量导入音频 |
| `/api/v1/drama/projects/:id/assets/import` | POST | 导入角色/场景资产 |
| `/api/v1/drama/projects/:id/assets/:assetId` | PUT | 更新资产 |
| `/api/v1/drama/projects/:id/assets/:assetId/reference` | POST | 上传资产参考图 |
| `/api/v1/drama/settings` | GET / PUT | 获取 / 保存短剧生成设置 |
| `/api/v1/drama/settings/comfyui/status` | GET | 检测 ComfyUI 状态 |

## 当前注意事项

- 视频与图片生成依赖外部 ComfyUI 工作流和本地模型文件，接口可用不代表模型一定就绪。
- 分镜视频生成会优先使用片段参考图；没有参考图时，需要先生成或选择可用图片作为首帧。
- 多人物场景建议在片段提示词中明确人数、站位、镜头距离、动作优先级和负面约束。
- 如果生成任务失败，优先查看任务详情中的提示词日志、原始 payload 和 ComfyUI 模型检查清单。
