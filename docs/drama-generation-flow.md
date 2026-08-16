# 短剧生成流程与代码维护说明

## 目的

本文作为短剧生成链路的维护入口，记录真实执行路径、数据落点和已清理的无效入口。新增模型或工作流时，应先更新本文件，再修改实现。

## 真实执行路径

### 请求阶段

1. `DramaPage.tsx` 创建项目、解析剧本或创建生成任务。
2. `drama.ts` 调用 `/api/v1/drama` 接口。
3. `handler/drama.go` 校验用户、项目和请求参数。
4. `service/drama.go` 写入 `drama_tasks`，并将任务 ID 放入 Redis 队列。

### 执行阶段

1. `service/task_runner.go` 从 `drama:tasks:queue` 取任务。
2. `service/task_execution.go` 按任务类型分发：
   - `image_generation`
   - `image`
   - `asset_reference`
   - `video`
3. 图片和视频任务通过统一的 ComfyUI 初始化逻辑读取设置、检查连接和解析 payload。
4. 生成过程中的提示词、进度、结果和错误写回任务 payload 或任务状态。

### 结果阶段

1. ComfyUI 输出通过 `/history/{prompt_id}` 查询。
2. 服务从 ComfyUI 下载输出文件。
3. 文件保存到 MinIO，并创建云盘文件记录。
4. 图片或视频媒体记录写回分镜/片段。
5. 前端通过任务事件和项目详情刷新结果。

## 任务类型边界

| 类型 | 用途 | 当前状态 |
|---|---|---|
| `image_generation` | 独立图片生成 | 已实现 |
| `image` | 分镜图片 | 已实现 |
| `asset_reference` | 角色/场景参考图 | 已实现 |
| `video` | 分镜/片段视频 | 已实现，当前走 Wan2.2 I2V |
| `tts` | 语音生成 | 未实现，已从界面和执行分支移除 |

## 精简规则

- 不在前端展示尚未被执行器读取的配置。
- 不为未实现功能保留可创建的任务类型。
- ComfyUI 连接检查和任务 payload 解析只保留一个公共入口。
- 保留数据库字段时必须注明兼容用途，不能让它看起来像当前生效配置。
- 新增模型时优先新增工作流适配器，不在 Wan2.2 工作流中堆叠条件分支。

## 故障排查

1. 先检查 `/api/v1/drama/settings/comfyui/status`。
2. 再查看任务详情中的提示词日志和错误消息。
3. 确认首帧、参考图和云盘文件仍存在。
4. 最后检查 ComfyUI 节点、模型和输出记录。
