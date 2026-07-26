import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Col,
  Collapse,
  Empty,
  Form,
  Image,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Progress,
  Row,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
  Upload,
  message,
} from 'antd'
import {
  CopyOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  ImportOutlined,
  LeftOutlined,
  PlusOutlined,
  RedoOutlined,
  RightOutlined,
  SaveOutlined,
  SettingOutlined,
  StopOutlined,
  ThunderboltOutlined,
  UploadOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons'
import type { UploadProps } from 'antd'
import { useAccess } from '../hooks/useAccess'
import {
  ComfyUIStatus,
  DramaAsset,
  DramaDetail,
  DramaProject,
  DramaSetting,
  DramaStoryboardMedia,
  DramaStoryboardSegment,
  DramaTask,
  appendDramaStoryboards,
  batchImportDramaAudio,
  cancelDramaTask,
  createDramaProject,
  createDramaTask,
  deleteDramaProject,
  deleteDramaStoryboardMedia,
  exportDramaProject,
  getComfyUIStatus,
  getDramaProject,
  getDramaSetting,
  importDramaAssets,
  importDramaStoryboardSegments,
  importDramaProject,
  listDramaProjects,
  listDramaTasks,
  parseDramaScript,
  retryDramaTask,
  saveDramaSetting,
  selectDramaStoryboardMedia,
  updateDramaAsset,
  updateDramaProject,
  updateDramaStoryboard,
  uploadDramaAssetReference,
  uploadStoryboardAudio,
} from '../services/drama'
import { getPreviewUrl } from '../services/file'

const { Text, Title } = Typography
const { TextArea } = Input

const defaultSuffix = '皮肤毛孔清晰，光影颗粒，新国风写实风格，电影级构图，细节丰富，高清质感'

export default function DramaPage() {
  const { hasPermission, loading: accessLoading } = useAccess()
  const [projects, setProjects] = useState<DramaProject[]>([])
  const [detail, setDetail] = useState<DramaDetail | null>(null)
  const [keyword, setKeyword] = useState('')
  const [sort, setSort] = useState('updated_desc')
  const [loading, setLoading] = useState(false)
  const [script, setScript] = useState('')
  const [currentIndex, setCurrentIndex] = useState(0)
  const [projectModalOpen, setProjectModalOpen] = useState(false)
  const [suffixModalOpen, setSuffixModalOpen] = useState(false)
  const [assetImportOpen, setAssetImportOpen] = useState(false)
  const [segmentImportOpen, setSegmentImportOpen] = useState(false)
  const [taskDetail, setTaskDetail] = useState<DramaTask | null>(null)
  const [suffix, setSuffix] = useState(defaultSuffix)
  const [setting, setSetting] = useState<DramaSetting | null>(null)
  const [comfyStatus, setComfyStatus] = useState<ComfyUIStatus | null>(null)
  const [comfyChecking, setComfyChecking] = useState(false)
  const [aiAssetText, setAiAssetText] = useState('')
  const [aiSegmentText, setAiSegmentText] = useState('')
  const [activeTab, setActiveTab] = useState('script')
  const [imageCount, setImageCount] = useState(3)
  const [projectForm] = Form.useForm()
  const [settingForm] = Form.useForm()

  const canRead = hasPermission('drama:read')
  const canWrite = hasPermission('drama:write')
  const canGenerate = hasPermission('drama:generate')
  const canAdmin = hasPermission('drama:admin')
  const current = detail?.storyboards[currentIndex]
  const currentMedia = useMemo(
    () => detail?.media?.filter((item) => item.storyboard_id === current?.id).sort((a, b) => a.sort_order - b.sort_order) || [],
    [detail?.media, current?.id],
  )
  const currentStoryboardMedia = useMemo(
    () => currentMedia.filter((item) => !item.segment_id || item.segment_id === '0'),
    [currentMedia],
  )
  const currentSegments = useMemo(
    () => detail?.segments?.filter((item) => item.storyboard_id === current?.id).sort((a, b) => a.seq - b.seq) || [],
    [detail?.segments, current?.id],
  )
  const modifiedCount = useMemo(() => detail?.storyboards.filter((item) => item.modified).length || 0, [detail])

  useEffect(() => {
    if (!accessLoading && canRead) {
      loadProjects()
      loadSetting()
    }
  }, [accessLoading, canRead, keyword, sort])

  useEffect(() => {
    const projectID = detail?.project.id
    const token = localStorage.getItem('access_token')
    if (!projectID || !token) return
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const socket = new WebSocket(`${protocol}//${window.location.host}/ws/drama/tasks?token=${encodeURIComponent(token)}`)
    socket.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data) as { type: string; task: DramaTask }
        if (update.type !== 'task_update' || update.task.project_id !== projectID) return
        setDetail((previous) => previous ? {
          ...previous,
          tasks: [update.task, ...previous.tasks.filter((item) => item.id !== update.task.id)],
        } : previous)
        if (['done', 'failed', 'canceled'].includes(update.task.status)) {
          refreshGeneratedResults(projectID)
        }
      } catch {
        // Ignore malformed progress events and keep the page usable.
      }
    }
    return () => socket.close()
  }, [detail?.project.id])

  useEffect(() => {
    if (activeTab !== 'tasks' || !detail) return
    const projectID = detail.project.id
    const timer = window.setInterval(async () => {
      try {
        const tasks = await listDramaTasks(projectID)
        setDetail((previous) => previous?.project.id === projectID ? { ...previous, tasks } : previous)
      } catch {
        // WebSocket remains the primary progress channel.
      }
    }, 3000)
    return () => window.clearInterval(timer)
  }, [activeTab, detail?.project.id])

  const loadProjects = async () => {
    setLoading(true)
    try {
      const data = await listDramaProjects({ keyword, sort })
      setProjects(data.items || [])
      if (!detail && data.items?.length) {
        await openProject(data.items[0].id)
      }
    } catch {
      message.error('读取项目失败')
    } finally {
      setLoading(false)
    }
  }

  const loadSetting = async () => {
    try {
      const data = await getDramaSetting()
      setSetting(data)
      settingForm.setFieldsValue({ ...data, ...parseImageSettings(data.image_settings) })
    } catch {
      // Keep the workbench usable if settings are unavailable.
    }
  }

  const refreshGeneratedResults = async (projectID: string) => {
    try {
      const remote = await getDramaProject(projectID)
      setDetail((previous) => {
        if (!previous || previous.project.id !== projectID) return previous
        return {
          ...previous,
          tasks: remote.tasks,
          media: remote.media || [],
          segments: remote.segments || [],
          assets: previous.assets.map((asset) => {
            const next = remote.assets.find((item) => item.id === asset.id)
            return next ? { ...asset, reference_file_id: next.reference_file_id } : asset
          }),
          storyboards: previous.storyboards.map((storyboard) => {
            const next = remote.storyboards.find((item) => item.id === storyboard.id)
            return next ? { ...storyboard, image_file_id: next.image_file_id, video_file_id: next.video_file_id } : storyboard
          }),
        }
      })
    } catch {
      // The next task refresh will retry.
    }
  }

  const openProject = async (id: string) => {
    setLoading(true)
    try {
      const data = await getDramaProject(id)
      setDetail({ ...data, media: data.media || [], segments: data.segments || [] })
      setScript(data.project.raw_script || '')
      setCurrentIndex(0)
    } catch {
      message.error('打开项目失败')
    } finally {
      setLoading(false)
    }
  }

  const refreshDetail = async () => {
    if (!detail) return
    const data = await getDramaProject(detail.project.id)
    setDetail({ ...data, media: data.media || [], segments: data.segments || [] })
  }

  const handleCreate = async () => {
    const values = await projectForm.validateFields()
    const project = await createDramaProject(values)
    setProjectModalOpen(false)
    projectForm.resetFields()
    await loadProjects()
    await openProject(project.id)
  }

  const handleDeleteProject = async (id: string) => {
    await deleteDramaProject(id)
    if (detail?.project.id === id) setDetail(null)
    await loadProjects()
    message.success('项目已删除，云盘文件已移入回收站')
  }

  const handleParse = async () => {
    if (!detail || !script.trim()) return
    const data = await parseDramaScript(detail.project.id, script)
    setDetail({ ...data, media: data.media || [], segments: data.segments || [] })
    setCurrentIndex(0)
    message.success('剧本已拆分为分镜')
  }

  const handleSaveStoryboard = async () => {
    if (!detail || !current) return
    const storyboard = await updateDramaStoryboard(detail.project.id, current.id, { content: current.content, prompt: current.prompt })
    setDetail({ ...detail, storyboards: detail.storyboards.map((item) => (item.id === storyboard.id ? storyboard : item)) })
    message.success('分镜已保存')
  }

  const handleUploadCurrentAudio = async (file: File) => {
    if (!detail || !current) return Upload.LIST_IGNORE
    const storyboard = await uploadStoryboardAudio(detail.project.id, current.id, file)
    setDetail({ ...detail, storyboards: detail.storyboards.map((item) => (item.id === storyboard.id ? storyboard : item)) })
    message.success('音频已导入，并生成字幕')
    return Upload.LIST_IGNORE
  }

  const updateCurrentContent = (content: string) => {
    if (!detail || !current) return
    setDetail({
      ...detail,
      storyboards: detail.storyboards.map((item, index) => (
        index === currentIndex ? { ...item, content, modified: content !== item.original } : item
      )),
    })
  }

  const handleAppend = async () => {
    if (!detail) return
    const storyboards = await appendDramaStoryboards(detail.project.id, suffix)
    setDetail({ ...detail, storyboards })
    setSuffixModalOpen(false)
    message.success('已追加到全部分镜')
  }

  const copyText = async (text: string) => {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  const copyAll = async () => {
    if (!detail) return
    await copyText([detail.project.preface, ...detail.storyboards.map((item) => item.content)].filter(Boolean).join('\n\n'))
  }

  const handleExport = async () => {
    if (!detail) return
    const blob = await exportDramaProject(detail.project.id)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${detail.project.title}-短剧项目.json`
    a.click()
    URL.revokeObjectURL(url)
    message.success('已导出，并保存到云盘 exports 目录')
  }

  const handleCreateTask = async (type: string, payload?: Record<string, unknown>, storyboardIDs?: string[]) => {
    if (!detail) return
    const targetStoryboardIDs = storyboardIDs || detail.storyboards.map((item) => item.id)
    const taskPayload = payload || {
      source: 'storyboards',
      source_label: `全部分镜（${detail.storyboards.length} 镜）`,
      storyboard_count: detail.storyboards.length,
      storyboard_titles: detail.storyboards.map((item) => item.title),
      image_count: type === 'image' ? imageCount : undefined,
    }
    const task = await createDramaTask(detail.project.id, {
      type,
      storyboard_ids: targetStoryboardIDs,
      payload: JSON.stringify(taskPayload),
    })
    setDetail({ ...detail, tasks: [task, ...detail.tasks] })
    message.success('任务已创建')
  }

  const handleSelectMedia = async (mediaId: string) => {
    if (!detail || !current) return
    const storyboard = await selectDramaStoryboardMedia(detail.project.id, current.id, mediaId)
    setDetail({
      ...detail,
      storyboards: detail.storyboards.map((item) => (item.id === storyboard.id ? storyboard : item)),
      media: (detail.media || []).map((item) => (
        item.storyboard_id === current.id && item.kind === 'image'
          ? { ...item, selected: item.id === mediaId }
          : item
      )),
    })
    message.success('已设为当前分镜图片')
  }

  const handleDeleteMedia = async (mediaId: string) => {
    if (!detail || !current) return
    const result = await deleteDramaStoryboardMedia(detail.project.id, current.id, mediaId)
    const otherMedia = (detail.media || []).filter((item) => item.storyboard_id !== current.id)
    setDetail({
      ...detail,
      storyboards: detail.storyboards.map((item) => (item.id === result.storyboard.id ? result.storyboard : item)),
      media: [...otherMedia, ...(result.media || [])],
    })
    message.success('图片已删除，文件已移入回收站')
  }

  const handleGenerateSegmentImage = async (segment: DramaStoryboardSegment) => {
    if (!detail || !current) return
    await handleCreateTask('image', {
      source: 'storyboard_segment',
      source_label: `分镜 ${current.seq} 片段 ${segment.seq}：${segment.title || '未命名片段'}`,
      storyboard_count: 1,
      image_count: 1,
      segment_ids: [segment.id],
      force_generate: true,
    }, [current.id])
  }

  const handleGenerateSegmentVideo = async (segment: DramaStoryboardSegment) => {
    if (!detail || !current) return
    await handleCreateTask('video', {
      source: 'storyboard_segment',
      source_label: `分镜 ${current.seq} 片段 ${segment.seq}：${segment.title || '未命名片段'}`,
      storyboard_count: 1,
      segment_ids: [segment.id],
      force_generate: true,
    }, [current.id])
  }

  const handleBatchAudioImport = async (files: File[]) => {
    if (!detail || !files.length) return
    const results = await batchImportDramaAudio(detail.project.id, files)
    const matched = results.filter((item) => item.matched)
    await refreshDetail()
    message.success(`已匹配 ${matched.length}/${results.length} 个音频文件`)
  }

  const handleSaveSetting = async () => {
    const values = await settingForm.validateFields()
    const imageSettings = {
      checkpoint: values.checkpoint || '',
      width: values.width || 768,
      height: values.height || 1024,
      steps: values.steps || 24,
      cfg: values.cfg || 7,
      sampler: values.sampler || 'euler',
      scheduler: values.scheduler || 'normal',
      negative_prompt: values.negative_prompt || '',
    }
    const next = await saveDramaSetting({
      comfyui_url: values.comfyui_url,
      image_settings: JSON.stringify(imageSettings),
      tts_engine: values.tts_engine,
      tts_config: values.tts_config,
      video_settings: values.video_settings,
      storage_root: values.storage_root,
    })
    setSetting(next)
    message.success('设置已保存')
  }

  const handleCheckComfyUI = async () => {
    setComfyChecking(true)
    try {
      const status = await getComfyUIStatus(settingForm.getFieldValue('comfyui_url'))
      setComfyStatus(status)
      if (status.connected) message.success('ComfyUI 连接成功')
      else message.error('ComfyUI 无法连接')
    } finally {
      setComfyChecking(false)
    }
  }

  const replaceTask = (task: DramaTask) => {
    setDetail((previous) => previous ? {
      ...previous,
      tasks: [task, ...previous.tasks.filter((item) => item.id !== task.id)],
    } : previous)
  }

  const handleCancelTask = async (task: DramaTask) => {
    if (!detail) return
    replaceTask(await cancelDramaTask(detail.project.id, task.id))
  }

  const handleRetryTask = async (task: DramaTask) => {
    if (!detail) return
    replaceTask(await retryDramaTask(detail.project.id, task.id))
  }

  const buildAssetPrompt = () => {
    if (!detail) return ''
    return `你是短剧视觉资产分析师，同时熟悉 ComfyUI / Stable Diffusion XL 提示词写法。请根据下面的短剧前言和分镜内容，提取角色与场景资产，输出严格 JSON，不要输出解释文字。
输出格式：
{
  "characters": [
    {
      "name": "角色名",
      "age": "年龄或年龄段",
      "appearance": "脸型、发型、五官、体态等稳定外貌",
      "clothing": "常见服装、颜色、配饰",
      "personality": "气质与表演状态",
      "voice_suggestion": "适合的中文音色建议",
      "voice_name": "可选：zh-CN-XiaoxiaoNeural 或 zh-CN-YunxiNeural 等",
      "reference_prompt": "可直接发给 ComfyUI/SDXL 的角色参考图提示词"
    }
  ],
  "scenes": [
    {
      "name": "场景名",
      "environment": "空间结构、陈设、时代、地域",
      "lighting": "光线、天气、时间",
      "style": "视觉风格",
      "reference_prompt": "可直接发给 ComfyUI/SDXL 的场景参考图提示词"
    }
  ]
}
要求：
1. 角色名称保持短且稳定，避免别名重复。
2. 外貌、服装、场景细节要适合做连续分镜一致性参考。
3. reference_prompt 必须是“适合 SDXL 的短提示词”，不要写成长篇档案，不要包含解释、字段名或换行。
4. reference_prompt 使用中英混合：开头先给英文强约束，再接中文关键细节。角色提示词格式参考：
   modern realistic cinematic photo of a [age] [Chinese man/woman], [face/hair/body], [exact clothing], [accessories], clear face, full body or medium full shot, natural skin texture, warm indoor lighting, 影视感, 写实摄影, 一致性保持
5. 现代都市/家庭/职场短剧角色必须保留现代现实服装。禁止把西装、衬衫、连衣裙、家居服改写成铠甲、长袍、斗篷、奇幻服装、游戏角色、概念设定图。
6. 场景 reference_prompt 要明确空间类型、时代、布置、光线和镜头，如：realistic cinematic environment photo, modern apartment living room, warm indoor light, clear spatial layout, no people, 影视感。
7. 如果原文没有奇幻、古装、科幻设定，不要生成 fantasy、medieval、armor、knight、sci-fi 等词。
8. 只输出 JSON。

前言：
${detail.project.preface || '无'}

分镜：
${detail.storyboards.map((item) => item.content).join('\n\n')}`
  }

  const buildSegmentPrompt = () => {
    if (!detail || !current) return ''
    const assets = detail.assets.map((asset) => ({
      type: asset.type,
      name: asset.name,
      description: asset.description,
      reference_prompt: asset.reference_prompt,
      has_reference_image: asset.reference_file_id !== '0',
    }))
    return `你是短剧分镜导演，同时熟悉 ComfyUI / SDXL / IPAdapter 多参考图工作流。请根据“当前分镜文本”在原有剧情、人物、场景、动作、台词基础上进一步完善提示词，生成可直接用于片段图片和片段视频的提示词，输出严格 JSON，不要输出解释文字。

重要前提：
- 当前分镜文本是唯一剧情依据，必须从这段分镜文本中提取人物、地点、动作、台词、情绪和镜头信息。
- 你的任务不是续写剧情，而是在不改变原分镜含义的前提下补全可见画面细节：人物数量、站位、姿态、手部动作、视线、道具、空间锚点、景别、光线、反向约束。
- “角色与场景资产”是已经确定的全局视觉资产，后续片段必须复用它们，不能重新设计人物或场景。
- has_reference_image=true 表示系统已有该资产参考图，生成片段图片时会用 IPAdapter 输入这张图；你的 reference_prompt 只需要描述本片段构图、动作、情绪、镜头和光线。
- 已有参考图只用于保持角色身份、服装和场景布局，不能照抄参考图的单人肖像构图；当前分镜的 composition_prompt 优先级最高。
- reference_prompt 中必须使用资产 name 原文，例如“丈夫”“妻子”“家中餐厅”，不要改名、不要写成陌生男人/陌生女人、不要替换服装和场景。
- 如果片段发生在某个资产场景中，scene 必须填写资产库里的场景 name；如果没有完全匹配，选择最接近的资产场景并在 action 中说明局部位置。

输出格式：
{
  "storyboard_seq": ${current.seq},
  "storyboard_title": "${current.title}",
  "segments": [
    {
      "seq": 1,
      "title": "片段标题",
      "duration_sec": 3,
      "purpose": "这个片段承担的剧情作用",
      "characters": ["角色名"],
      "scene": "场景名",
      "asset_names": ["本片段必须使用的角色资产名和场景资产名"],
      "scene_asset": "使用的场景资产名",
      "dialogue": "本片段台词，没有则为空字符串",
      "action": "人物动作与情绪变化",
      "shot": "景别、镜头角度、构图",
      "composition_prompt": "用于控制片段参考图人物位置、姿态、视线、手部动作、前后景关系的构图提示词",
      "reference_prompt": "用于先生成片段参考图的 ComfyUI/SDXL 正向提示词",
      "video_prompt": "用于图生视频的运动提示词",
      "negative_prompt": "用于排除错误内容的负面提示词"
    }
  ]
}

拆分要求：
1. 先阅读当前分镜文本，再按镜头/动作/台词变化拆成片段；每个片段建议 2-5 秒，只表达一个明确动作或情绪变化。
2. characters 只能填写资产库中已有的角色 name；scene 和 scene_asset 只能填写资产库中已有的场景 name。不要创造新角色、新地点、新服装。
3. asset_names 必须列出本片段用到的全部角色资产和场景资产，例如 ["丈夫","妻子","父亲","母亲","家中餐厅"]。
4. composition_prompt 是最重要的构图控制字段，必须用英文短语 + 中文补充，明确：
   - 先写镜头类型和人数；两人镜头必须以 two-shot, two characters visible in same frame, medium wide shot 开头；三人以上必须以 group shot, all listed characters visible in same frame 开头
   - 人物数量和每个人在画面中的位置：left/right/center/foreground/background/opposite side
   - 坐姿/站姿/身体朝向/头部朝向
   - 手部动作和道具位置，如 chopsticks above husband's bowl
   - 视线关系，如 husband looks down at bowl, wife looks at husband
   - 镜头距离和角度，如 medium wide shot, eye-level camera, rectangular dining table centered
   - 前后景关系，如 parents slightly blurred in background
5. composition_prompt 不要写情绪概念，必须写可见的画面事实。错误示例：warm family atmosphere。正确示例：wife on right side of table, husband on left, wife right hand holding chopsticks above husband's bowl, parents seated opposite in background, medium wide shot。
6. reference_prompt 用于静态片段参考图，必须描述“最终画面状态”，不要写镜头运动，不要写连续动作。它应当像导演给摄影师的画面说明，而不是角色设定表。
7. reference_prompt 必须在原分镜基础上完善，而不是替换原分镜；必须把 composition_prompt 放在开头并以这类结构输出：
   realistic cinematic still frame, use IPAdapter references for [资产名列表], keep exact identity, clothing and scene layout from references, [场景资产名], [人物站位和动作], [景别/构图], [光线], clear faces, natural skin texture, no redesign, 影视感, 写实摄影
8. reference_prompt 不要重复大段外貌档案；有参考图的角色只写资产名 + 必要动作/表情/站位。错误示例：32-year-old Chinese man in dark suit standing in modern hallway。正确示例：丈夫坐在家中餐厅餐桌右侧，保持资产参考图中的脸、发型和深色西装。
9. 如果 characters 有 2 个或更多角色，reference_prompt 必须明确 all listed characters visible in same frame, no solo portrait, no close-up portrait，并逐个写出每个角色的位置、朝向、手部动作和视线；不要只描述其中一个角色。
10. 如果分镜要求“家中餐厅”，reference_prompt 必须明确餐桌、餐椅、菜肴、暖黄餐厅灯光，不能生成走廊、办公室、卧室、酒店大堂等其它空间。
11. 如果一个动作包含前后变化，请拆成两个片段，不要让单张参考图同时表达“先低头再抬头”。每个片段只保留一个瞬间动作。
12. video_prompt 用于图生视频，必须描述“运动变化”，例如 slow push-in, 妻子夹菜放入丈夫碗中, 丈夫低头再抬眼, subtle breathing。
13. negative_prompt 必须根据本片段补强，至少包含：wrong identity, different face, different clothing, different room, wrong pose, wrong hand position, wrong gaze direction, single person, solo portrait, close-up portrait, cropped second person, missing character, missing husband, missing wife, mirror frame, decorative frame, oversized foreground lamp, foreground obstruction, hallway, office, bedroom, hotel lobby, extra people, armor, fantasy costume, medieval, knight, cape, cloak, game character, concept art, anime, illustration, text, watermark, logo, UI, close-up, extreme close-up。
14. 现代都市/家庭/职场短剧必须保持现实服装与现实空间，禁止把西装、衬衫、家居服改写成铠甲、斗篷、长袍、奇幻服装、游戏角色、概念设定图。
15. 输出只允许 JSON。

项目背景：
${detail.project.preface || '无'}

角色与场景资产：
${JSON.stringify(assets, null, 2)}

当前分镜：
${current.content}`
  }

  const handleImportAISegments = async () => {
    if (!detail || !current || !aiSegmentText.trim()) return
    const segments = await importDramaStoryboardSegments(detail.project.id, current.id, aiSegmentText)
    setDetail({
      ...detail,
      segments: [
        ...(detail.segments || []).filter((item) => item.storyboard_id !== current.id),
        ...segments,
      ],
    })
    setSegmentImportOpen(false)
    setAiSegmentText('')
    message.success(`已导入 ${segments.length} 个片段`)
  }

  const handleImportAIAssets = async () => {
    if (!detail || !aiAssetText.trim()) return
    const assets = await importDramaAssets(detail.project.id, aiAssetText)
    setDetail({ ...detail, assets })
    setAssetImportOpen(false)
    setAiAssetText('')
    message.success(`已导入 ${assets.length} 个资产`)
  }

  const importProps: UploadProps = {
    showUploadList: false,
    beforeUpload: async (file) => {
      const project = await importDramaProject(file)
      await loadProjects()
      await openProject(project.id)
      message.success('项目已导入')
      return Upload.LIST_IGNORE
    },
  }

  if (accessLoading) return <Empty description="正在检查权限" />
  if (!canRead) return <Empty description="你还没有短剧工坊权限，请联系管理员授予 drama:* 权限" />

  const showStoryboardNav = activeTab === 'storyboards' && !!detail?.storyboards.length
  const scrollAreaStyle: React.CSSProperties = {
    height: '100%',
    minHeight: 0,
    overflow: 'auto',
    paddingRight: 4,
    paddingBottom: 12,
  }

  return (
    <div style={{ height: '100%', minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <Row gutter={16} style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
        <Col xs={24} lg={7} xl={6} style={{ height: '100%', minHeight: 0 }}>
          <div style={{ width: '100%', height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>
            <Space.Compact style={{ width: '100%' }}>
              <Input.Search placeholder="搜索项目" value={keyword} onChange={(e) => setKeyword(e.target.value)} allowClear />
              <Select
                value={sort}
                onChange={setSort}
                style={{ width: 128 }}
                options={[
                  { value: 'updated_desc', label: '最近更新' },
                  { value: 'created_desc', label: '最新创建' },
                  { value: 'title_asc', label: '标题 A-Z' },
                ]}
              />
            </Space.Compact>
            <Space wrap>
              <Button type="primary" icon={<PlusOutlined />} disabled={!canWrite} onClick={() => setProjectModalOpen(true)}>新建</Button>
              <Upload {...importProps} disabled={!canWrite}>
                <Button icon={<ImportOutlined />} disabled={!canWrite}>导入</Button>
              </Upload>
            </Space>
            <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
              <List
                loading={loading}
                dataSource={projects}
                locale={{ emptyText: '暂无项目' }}
                renderItem={(item) => (
                  <List.Item
                    onClick={() => openProject(item.id)}
                    style={{
                      cursor: 'pointer',
                      padding: 12,
                      borderRadius: 8,
                      background: detail?.project.id === item.id ? '#fef3e7' : '#fff',
                      marginBottom: 8,
                      border: '1px solid #f0eeeb',
                    }}
                    actions={[
                      <Popconfirm key="delete" title="删除项目？" onConfirm={(e) => { e?.stopPropagation(); handleDeleteProject(item.id) }} disabled={!canWrite}>
                        <Button size="small" icon={<DeleteOutlined />} disabled={!canWrite} onClick={(e) => e.stopPropagation()} />
                      </Popconfirm>,
                    ]}
                  >
                    <List.Item.Meta title={item.title} description={item.description || '无描述'} />
                  </List.Item>
                )}
              />
            </div>
          </div>
        </Col>

        <Col xs={24} lg={17} xl={18} style={{ height: '100%', minHeight: 0 }}>
          {!detail ? (
            <Empty description="选择或新建一个短剧项目" />
          ) : (
            <div style={{ height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 16 }}>
              <Card style={{ flexShrink: 0 }}>
                <Row gutter={[12, 12]} align="middle">
                  <Col flex="auto">
                    <Title level={4} style={{ margin: 0 }}>{detail.project.title}</Title>
                    <Text type="secondary">{detail.project.description || '短剧工坊'}</Text>
                  </Col>
                  <Col>
                    <Space wrap>
                      <Tag color="blue">{detail.storyboards.length} 镜</Tag>
                      <Tag color={modifiedCount ? 'orange' : 'default'}>{modifiedCount} 已修改</Tag>
                      <Button icon={<CopyOutlined />} onClick={copyAll}>复制全部</Button>
                      <Button icon={<DownloadOutlined />} onClick={handleExport}>导出</Button>
                    </Space>
                  </Col>
                </Row>
              </Card>

              <Tabs
                className="drama-workbench-tabs"
                activeKey={activeTab}
                onChange={setActiveTab}
                style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}
                tabBarStyle={{ marginBottom: 12 }}
                items={[
                  {
                    key: 'script',
                    label: '剧本解析',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Card>
                          <Space direction="vertical" size={12} style={{ width: '100%' }}>
                            <TextArea
                              value={script}
                              onChange={(e) => setScript(e.target.value)}
                              placeholder="粘贴完整剧本，系统会按【分镜序号】：数字/总数字自动拆分"
                              autoSize={{ minRows: 12, maxRows: 24 }}
                            />
                            <Button type="primary" icon={<EditOutlined />} disabled={!canWrite} onClick={handleParse}>自动分镜解析</Button>
                          </Space>
                        </Card>
                      </div>
                    ),
                  },
                  {
                    key: 'assets',
                    label: '角色与场景资产',
                    children: (
                      <div style={{ ...scrollAreaStyle, width: '100%' }}>
                        <Space wrap>
                          <Button icon={<CopyOutlined />} onClick={() => copyText(buildAssetPrompt())}>复制 AI 资产分析提示词</Button>
                          <Button icon={<ImportOutlined />} disabled={!canWrite} onClick={() => setAssetImportOpen(true)}>粘贴 AI 结果导入</Button>
                        </Space>
                        <div style={{ marginTop: 12 }}>
                          <AssetPanel detail={detail} canWrite={canWrite} canGenerate={canGenerate} onChange={setDetail} />
                        </div>
                      </div>
                    ),
                  },
                  {
                    key: 'storyboards',
                    label: '分镜编辑',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Row gutter={[16, 16]}>
                          <Col xs={24} xl={7}>
                            <Collapse
                              items={[{
                                key: 'preface',
                                label: '前言 / 人物场景档案',
                                children: <TextArea value={detail.project.preface} autoSize={{ minRows: 8, maxRows: 18 }} onChange={(e) => setDetail({ ...detail, project: { ...detail.project, preface: e.target.value } })} onBlur={() => updateDramaProject(detail.project.id, { preface: detail.project.preface })} />,
                              }]}
                            />
                            <List
                              style={{ marginTop: 12 }}
                              dataSource={detail.storyboards}
                              renderItem={(item, index) => (
                                <List.Item
                                  onClick={() => setCurrentIndex(index)}
                                  style={{
                                    cursor: 'pointer',
                                    padding: 10,
                                    borderRadius: 8,
                                    background: index === currentIndex ? '#fef3e7' : '#fff',
                                    border: '1px solid #f0eeeb',
                                    marginBottom: 8,
                                  }}
                                >
                                  <Space>
                                    <Tag color={item.modified ? 'orange' : item.audio_file_id !== '0' ? 'green' : 'default'}>{item.seq}</Tag>
                                    <Text>{item.title}</Text>
                                  </Space>
                                </List.Item>
                              )}
                            />
                          </Col>
                          <Col xs={24} xl={17}>
                            {current ? (
                              <Card
                                title={<Space><Text strong>{current.title}</Text>{current.modified && <Tag color="orange">已修改</Tag>}</Space>}
                                extra={
                                  <Space wrap>
                                    <Upload showUploadList={false} accept="audio/*" beforeUpload={handleUploadCurrentAudio} disabled={!canWrite}>
                                      <Button icon={<UploadOutlined />} disabled={!canWrite}>导入音频</Button>
                                    </Upload>
                                    <InputNumber min={1} max={8} value={imageCount} onChange={(value) => setImageCount(Number(value || 1))} addonBefore="每镜" addonAfter="张" style={{ width: 150 }} />
                                    <Button
                                      icon={<ThunderboltOutlined />}
                                      disabled={!canGenerate}
                                      onClick={() => handleCreateTask('image', {
                                        source: 'storyboard',
                                        source_label: `分镜 ${current.seq}：${current.title}`,
                                        storyboard_count: 1,
                                        image_count: imageCount,
                                      }, [current.id])}
                                    >
                                      {current.image_file_id !== '0' ? '继续生成' : '生成图片'}
                                    </Button>
                                    <Button
                                      icon={<ThunderboltOutlined />}
                                      disabled={!canGenerate || current.image_file_id === '0'}
                                      onClick={() => handleCreateTask('video', {
                                        source: 'storyboard',
                                        source_label: `分镜 ${current.seq}：${current.title}`,
                                        storyboard_count: 1,
                                        duration_sec: 10,
                                      }, [current.id])}
                                    >
                                      生成10秒视频
                                    </Button>
                                    <Button icon={<CopyOutlined />} onClick={() => copyText(buildSegmentPrompt())}>复制生成AI提示词</Button>
                                    <Button icon={<ImportOutlined />} disabled={!canWrite} onClick={() => setSegmentImportOpen(true)}>粘贴 AI 提示词结果</Button>
                                    <Button icon={<CopyOutlined />} onClick={() => copyText(current.content)}>复制本镜</Button>
                                    <Button icon={<SaveOutlined />} type="primary" disabled={!canWrite} onClick={handleSaveStoryboard}>保存</Button>
                                  </Space>
                                }
                              >
                                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                                  {current.image_file_id !== '0' && (
                                    <Image
                                      src={getPreviewUrl(current.image_file_id)}
                                      alt={current.title}
                                      style={{ width: '100%', maxHeight: 520, objectFit: 'contain', borderRadius: 8, border: '1px solid #f0eeeb', background: '#f7f7f5' }}
                                    />
                                  )}
                                  {currentStoryboardMedia.filter((item) => item.kind === 'image').length > 0 && (
                                    <div>
                                      <Text strong>图片候选</Text>
                                      <Row gutter={[12, 12]} style={{ marginTop: 8 }}>
                                        {currentStoryboardMedia.filter((item) => item.kind === 'image').map((item, index) => (
                                          <Col xs={12} md={8} xl={6} key={item.id}>
                                            <Card size="small" bodyStyle={{ padding: 8 }} style={{ borderColor: item.selected ? '#e8964a' : '#f0eeeb' }}>
                                              <Image
                                                src={getPreviewUrl(item.file_id)}
                                                alt={`candidate-${index + 1}`}
                                                style={{ width: '100%', aspectRatio: '3 / 4', objectFit: 'cover', borderRadius: 6, background: '#f7f7f5' }}
                                              />
                                              {item.prompt && (
                                                <Text type="secondary" ellipsis={{ tooltip: item.prompt }} style={{ display: 'block', marginTop: 6, fontSize: 12 }}>
                                                  {item.prompt}
                                                </Text>
                                              )}
                                              <Space style={{ marginTop: 8, width: '100%', justifyContent: 'space-between' }}>
                                                <Tag color={item.selected ? 'orange' : 'default'}>#{index + 1}</Tag>
                                                <Button size="small" type={item.selected ? 'primary' : 'default'} disabled={item.selected || !canWrite} onClick={() => handleSelectMedia(item.id)}>
                                                  {item.selected ? '当前' : '设为当前'}
                                                </Button>
                                                <Popconfirm
                                                  title="删除这张图片？"
                                                  description="文件会移入回收站"
                                                  okText="删除"
                                                  cancelText="取消"
                                                  onConfirm={() => handleDeleteMedia(item.id)}
                                                  disabled={!canWrite}
                                                >
                                                  <Button size="small" danger icon={<DeleteOutlined />} disabled={!canWrite} />
                                                </Popconfirm>
                                              </Space>
                                            </Card>
                                          </Col>
                                        ))}
                                      </Row>
                                    </div>
                                  )}
                                  {currentSegments.length > 0 && (
                                    <StoryboardSegmentList
                                      segments={currentSegments}
                                      media={currentMedia}
                                      canGenerate={canGenerate}
                                      canWrite={canWrite}
                                      onCopy={copyText}
                                      onGenerate={handleGenerateSegmentImage}
                                      onGenerateVideo={handleGenerateSegmentVideo}
                                      onDeleteMedia={handleDeleteMedia}
                                    />
                                  )}
                                  <Space wrap>
                                    <Tag color={current.audio_file_id !== '0' ? 'green' : 'default'}>
                                      {current.audio_file_id !== '0' ? `已配音 ${Math.round((current.audio_duration_ms || 0) / 1000)}s` : '未配音'}
                                    </Tag>
                                    {current.subtitle_ass && <Button size="small" icon={<CopyOutlined />} onClick={() => copyText(current.subtitle_ass)}>复制字幕 ASS</Button>}
                                  </Space>
                                  {current.subtitle_ass && (
                                    <Collapse size="small" items={[{ key: 'subtitle', label: '字幕预览', children: <TextArea value={current.subtitle_ass} readOnly autoSize={{ minRows: 5, maxRows: 10 }} /> }]} />
                                  )}
                                  <TextArea value={current.content} onChange={(e) => updateCurrentContent(e.target.value)} autoSize={{ minRows: 22, maxRows: 38 }} />
                                </Space>
                              </Card>
                            ) : (
                              <Empty description="还没有分镜，请先解析剧本" />
                            )}
                          </Col>
                        </Row>
                      </div>
                    ),
                  },
                  {
                    key: 'tasks',
                    label: '生成任务',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Card>
                          <Space direction="vertical" size={16} style={{ width: '100%' }}>
                            <Space wrap>
                              <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => handleCreateTask('tts')}>创建 TTS 任务</Button>
                              <InputNumber min={1} max={8} value={imageCount} onChange={(value) => setImageCount(Number(value || 1))} addonBefore="每镜" addonAfter="张" style={{ width: 150 }} />
                              <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => handleCreateTask('image')}>创建图片任务</Button>
                              <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => handleCreateTask('video')}>创建视频任务</Button>
                              <Upload
                                multiple
                                showUploadList={false}
                                accept="audio/*"
                                beforeUpload={() => false}
                                customRequest={() => undefined}
                                onChange={({ fileList }) => {
                                  const files = fileList.map((item) => item.originFileObj).filter(Boolean) as File[]
                                  handleBatchAudioImport(files)
                                }}
                                disabled={!canWrite}
                              >
                                <Button icon={<ImportOutlined />} disabled={!canWrite}>批量导入音频</Button>
                              </Upload>
                            </Space>
                            <List
                              dataSource={detail.tasks}
                              locale={{ emptyText: '暂无生成任务' }}
                              renderItem={(item) => (
                                <List.Item
                                  actions={[
                                    <Button key="detail" size="small" icon={<EditOutlined />} onClick={() => setTaskDetail(item)}>详情</Button>,
                                    ...(item.status === 'pending' || item.status === 'running' ? [
                                      <Button key="cancel" size="small" danger icon={<StopOutlined />} onClick={() => handleCancelTask(item)}>取消</Button>,
                                    ] : []),
                                    ...(item.status === 'failed' || item.status === 'canceled' ? [
                                      <Button key="retry" size="small" icon={<RedoOutlined />} onClick={() => handleRetryTask(item)}>重试</Button>,
                                    ] : []),
                                  ]}
                                >
                                  <List.Item.Meta
                                    title={<Space><Tag>{getTaskTypeLabel(item.type)}</Tag><Tag color={item.status === 'failed' ? 'red' : item.status === 'done' ? 'green' : item.status === 'canceled' ? 'default' : 'blue'}>{getTaskStatusLabel(item.status)}</Tag></Space>}
                                    description={
                                      <Space direction="vertical" size={4} style={{ width: '100%' }}>
                                        <Text type="secondary">来源：{getTaskSource(item, detail)}</Text>
                                        <Text type="secondary">{item.message || `${item.progress}%`}</Text>
                                        <Progress percent={item.progress} size="small" status={item.status === 'failed' ? 'exception' : item.status === 'done' ? 'success' : 'active'} style={{ maxWidth: 360 }} />
                                        <TaskResultStrip task={item} />
                                      </Space>
                                    }
                                  />
                                </List.Item>
                              )}
                            />
                          </Space>
                        </Card>
                      </div>
                    ),
                  },
                  {
                    key: 'settings',
                    label: '系统设置',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Card>
                          <Form form={settingForm} layout="vertical" initialValues={setting || undefined}>
                            <Row gutter={16}>
                              <Col xs={24} md={18}>
                                <Form.Item name="comfyui_url" label="ComfyUI API 地址">
                                  <Input placeholder="http://comfyui:8188" />
                                </Form.Item>
                              </Col>
                              <Col xs={24} md={6} style={{ display: 'flex', alignItems: 'center' }}>
                                <Button loading={comfyChecking} onClick={handleCheckComfyUI} style={{ width: '100%', marginTop: 6 }}>检测连接与模型</Button>
                              </Col>
                              {comfyStatus && (
                                <Col xs={24}>
                                  <Alert
                                    style={{ marginBottom: 16 }}
                                    type={comfyStatus.connected && comfyStatus.checkpoints.length ? 'success' : comfyStatus.connected ? 'warning' : 'error'}
                                    showIcon
                                    message={comfyStatus.connected ? `ComfyUI 已连接，发现 ${comfyStatus.checkpoints.length} 个 Checkpoint` : 'ComfyUI 无法连接'}
                                    description={
                                      <Space direction="vertical" size={4}>
                                        <Text>IP-Adapter：{comfyStatus.ip_adapter ? '已安装' : '未安装'}；ReActor：{comfyStatus.reactor ? '已安装' : '未安装'}</Text>
                                        <ComfyModelChecklist status={comfyStatus} />
                                        {!!comfyStatus.missing?.length && <Text type="secondary">待补全：{comfyStatus.missing.join('、')}</Text>}
                                        {comfyStatus.error && <Text type="danger">{comfyStatus.error}</Text>}
                                      </Space>
                                    }
                                  />
                                </Col>
                              )}
                              <Col xs={24} md={12}>
                                <Form.Item name="checkpoint" label="基础图片模型">
                                  <Select showSearch allowClear placeholder="检测后选择 Checkpoint" options={(comfyStatus?.checkpoints || []).map((value) => ({ value, label: value }))} />
                                </Form.Item>
                              </Col>
                              <Col xs={12} md={6}><Form.Item name="width" label="图片宽度"><InputNumber min={256} max={2048} step={64} style={{ width: '100%' }} /></Form.Item></Col>
                              <Col xs={12} md={6}><Form.Item name="height" label="图片高度"><InputNumber min={256} max={2048} step={64} style={{ width: '100%' }} /></Form.Item></Col>
                              <Col xs={12} md={6}><Form.Item name="steps" label="采样步数"><InputNumber min={1} max={100} style={{ width: '100%' }} /></Form.Item></Col>
                              <Col xs={12} md={6}><Form.Item name="cfg" label="提示词强度"><InputNumber min={1} max={30} step={0.5} style={{ width: '100%' }} /></Form.Item></Col>
                              <Col xs={12} md={6}><Form.Item name="sampler" label="采样器"><Select options={[{ value: 'euler', label: 'Euler' }, { value: 'euler_ancestral', label: 'Euler a' }, { value: 'dpmpp_2m', label: 'DPM++ 2M' }]} /></Form.Item></Col>
                              <Col xs={12} md={6}><Form.Item name="scheduler" label="调度器"><Select options={[{ value: 'normal', label: 'Normal' }, { value: 'karras', label: 'Karras' }, { value: 'simple', label: 'Simple' }]} /></Form.Item></Col>
                              <Col xs={24}><Form.Item name="negative_prompt" label="默认负面提示词"><TextArea autoSize={{ minRows: 2, maxRows: 5 }} /></Form.Item></Col>
                              <Col xs={24} md={12}><Form.Item name="tts_engine" label="TTS 引擎"><Select options={[{ value: 'edge-tts', label: 'edge-tts' }, { value: 'azure', label: 'Azure Speech' }]} /></Form.Item></Col>
                              <Col xs={24} md={12}><Form.Item name="storage_root" label="云盘根目录"><Input prefix={<SettingOutlined />} /></Form.Item></Col>
                              <Col xs={24}><Form.Item name="tts_config" label="TTS 配置 JSON"><TextArea autoSize={{ minRows: 4 }} /></Form.Item></Col>
                              <Col xs={24}><Form.Item name="video_settings" label="视频默认参数 JSON"><TextArea autoSize={{ minRows: 5 }} /></Form.Item></Col>
                            </Row>
                            <Button type="primary" icon={<SaveOutlined />} disabled={!canAdmin} onClick={handleSaveSetting}>保存设置</Button>
                          </Form>
                        </Card>
                      </div>
                    ),
                  },
                ]}
              />
            </div>
          )}
        </Col>
      </Row>

      {showStoryboardNav ? (
        <div style={{ flexShrink: 0, minHeight: 56, background: '#fff', borderTop: '1px solid #f0eeeb', zIndex: 10, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12, marginTop: 12, padding: '8px 12px', flexWrap: 'wrap' }}>
          <Button icon={<LeftOutlined />} disabled={currentIndex <= 0} onClick={() => setCurrentIndex((value) => Math.max(0, value - 1))} />
          <Text>当前分镜</Text>
          <InputNumber min={1} max={detail.storyboards.length} value={currentIndex + 1} onChange={(value) => setCurrentIndex(Math.max(0, Math.min(detail.storyboards.length - 1, Number(value || 1) - 1)))} />
          <Text>/ {detail.storyboards.length}</Text>
          <Button icon={<RightOutlined />} disabled={currentIndex >= detail.storyboards.length - 1} onClick={() => setCurrentIndex((value) => Math.min(detail.storyboards.length - 1, value + 1))} />
          <Button onClick={() => setSuffixModalOpen(true)} disabled={!canWrite}>批量追加</Button>
        </div>
      ) : null}

      <Modal title="新建短剧项目" open={projectModalOpen} onOk={handleCreate} onCancel={() => setProjectModalOpen(false)}>
        <Form form={projectForm} layout="vertical">
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="批量追加内容" open={suffixModalOpen} onOk={handleAppend} onCancel={() => setSuffixModalOpen(false)}>
        <TextArea value={suffix} onChange={(e) => setSuffix(e.target.value)} autoSize={{ minRows: 5 }} />
      </Modal>

      <Modal title="粘贴 AI 资产分析结果" open={assetImportOpen} onOk={handleImportAIAssets} onCancel={() => setAssetImportOpen(false)} width={760}>
        <TextArea value={aiAssetText} onChange={(e) => setAiAssetText(e.target.value)} placeholder="粘贴 AI 输出的 JSON" autoSize={{ minRows: 14, maxRows: 24 }} />
      </Modal>

      <Modal title="粘贴片段分析结果" open={segmentImportOpen} onOk={handleImportAISegments} onCancel={() => setSegmentImportOpen(false)} width={820}>
        <TextArea value={aiSegmentText} onChange={(e) => setAiSegmentText(e.target.value)} placeholder="粘贴 AI 输出的片段 JSON" autoSize={{ minRows: 16, maxRows: 28 }} />
      </Modal>

      <Modal title="生成任务详情" open={!!taskDetail} onCancel={() => setTaskDetail(null)} footer={null} width={900}>
        {taskDetail && detail && <TaskDetailPanel task={taskDetail} detail={detail} />}
      </Modal>
    </div>
  )
}

function parseImageSettings(raw?: string) {
  const defaults = {
    checkpoint: '', width: 768, height: 1024, steps: 24, cfg: 7,
    sampler: 'euler', scheduler: 'normal', negative_prompt: '低质量，模糊，变形，多余手指，文字，水印',
  }
  try {
    return { ...defaults, ...JSON.parse(raw || '{}') }
  } catch {
    return defaults
  }
}

function ComfyModelChecklist({ status }: { status: ComfyUIStatus }) {
  const models = status.models || {}
  const items = [
    { key: 'clip_vision_sdxl', label: 'CLIP Vision' },
    { key: 'ipadapter_plus_sdxl', label: 'IPAdapter SDXL' },
    { key: 'wan22_high_noise', label: 'Wan2.2 High' },
    { key: 'wan22_low_noise', label: 'Wan2.2 Low' },
    { key: 'wan22_text_encoder', label: 'Wan 文本编码器' },
    { key: 'wan_vae', label: 'Wan VAE' },
  ]
  return (
    <Space wrap size={4}>
      {items.map((item) => (
        <Tag key={item.key} color={models[item.key] ? 'green' : 'red'}>
          {item.label}{models[item.key] ? ' 已就绪' : ' 缺失'}
        </Tag>
      ))}
    </Space>
  )
}

function getTaskTypeLabel(type: string) {
  return ({
    asset_reference: '资产参考图',
    image: '分镜图片',
    tts: '语音生成',
    video: '视频合成',
  } as Record<string, string>)[type] || type
}

function getTaskStatusLabel(status: string) {
  return ({ pending: '等待中', running: '生成中', done: '已完成', failed: '失败', canceled: '已取消' } as Record<string, string>)[status] || status
}

function getTaskSource(task: DramaTask, detail: DramaDetail) {
  try {
    const payload = JSON.parse(task.payload || '{}') as {
      source_label?: string
      asset_type?: string
      name?: string
      storyboard_count?: number
      image_count?: number
    }
    const imageSuffix = payload.image_count && task.type === 'image' ? `，每镜 ${payload.image_count} 张` : ''
    if (payload.source_label) return `${payload.source_label}${imageSuffix}`
    if (payload.name && payload.asset_type) return `${payload.asset_type === 'character' ? '角色' : '场景'}：${payload.name}`
    if (payload.storyboard_count) return `全部分镜（${payload.storyboard_count} 镜）${imageSuffix}`
  } catch {
    // Older tasks may contain an empty or non-JSON payload.
  }
  if (task.type === 'asset_reference') return '角色或场景参考图'
  return detail.storyboards.length ? `全部分镜（${detail.storyboards.length} 镜）` : '当前项目'
}

interface DramaTaskPayloadView {
  prompt?: string
  results?: Array<{
    kind?: string
    file_id?: string
    storyboard_id?: string
    segment_id?: string
    asset_id?: string
    title?: string
    prompt?: string
  }>
  prompt_log?: Array<{
    target?: string
    prompt?: string
  }>
}

function parseTaskPayload(task: DramaTask): DramaTaskPayloadView {
  try {
    return JSON.parse(task.payload || '{}') as DramaTaskPayloadView
  } catch {
    return {}
  }
}

function TaskResultStrip({ task }: { task: DramaTask }) {
  const payload = parseTaskPayload(task)
  const results = (payload.results || []).filter((item) => item.file_id && item.file_id !== '0')
  const promptLog = payload.prompt_log?.length
    ? payload.prompt_log
    : results.filter((item) => item.prompt).map((item) => ({ target: item.title, prompt: item.prompt }))
  if (!results.length && !promptLog.length) return null

  return (
    <Space direction="vertical" size={8} style={{ width: '100%', marginTop: 6 }}>
      {!!results.length && (
        <Image.PreviewGroup>
          <Space wrap size={8}>
            {results.map((item, index) => (
              <div key={`${item.file_id}-${index}`} style={{ width: item.kind?.includes('video') ? 180 : 88 }}>
                {item.kind?.includes('video') ? (
                  <video
                    src={getPreviewUrl(item.file_id || '0')}
                    controls
                    style={{ width: 180, height: 102, objectFit: 'cover', borderRadius: 6, border: '1px solid #f0eeeb', background: '#111' }}
                  />
                ) : (
                  <Image
                    src={getPreviewUrl(item.file_id || '0')}
                    alt={item.title || `result-${index + 1}`}
                    width={88}
                    height={88}
                    style={{ objectFit: 'cover', borderRadius: 6, border: '1px solid #f0eeeb', background: '#f7f7f5' }}
                  />
                )}
                <Text type="secondary" ellipsis={{ tooltip: item.title || `#${index + 1}` }} style={{ display: 'block', fontSize: 12, marginTop: 4 }}>
                  {item.title || `#${index + 1}`}
                </Text>
              </div>
            ))}
          </Space>
        </Image.PreviewGroup>
      )}
      {!!promptLog.length && (
        <Collapse
          size="small"
          ghost
          items={[{
            key: 'prompts',
            label: `提示词${promptLog.length ? `（${promptLog.length}）` : ''}`,
            children: (
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {promptLog.map((item, index) => (
                  <div key={`${item.target || 'prompt'}-${index}`}>
                    <Text type="secondary">{item.target || `Prompt ${index + 1}`}</Text>
                    <TextArea value={item.prompt || ''} readOnly autoSize={{ minRows: 3, maxRows: 10 }} style={{ marginTop: 4 }} />
                  </div>
                ))}
              </Space>
            ),
          }]}
        />
      )}
    </Space>
  )
}

function TaskDetailPanel({ task, detail }: { task: DramaTask; detail: DramaDetail }) {
  const payload = parseTaskPayload(task)
  const promptLog = payload.prompt_log?.length
    ? payload.prompt_log
    : (payload.results || []).filter((item) => item.prompt).map((item) => ({ target: item.title, prompt: item.prompt }))

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space wrap>
        <Tag color="blue">{getTaskTypeLabel(task.type)}</Tag>
        <Tag color={task.status === 'done' ? 'green' : task.status === 'failed' ? 'red' : task.status === 'running' ? 'orange' : 'default'}>
          {getTaskStatusLabel(task.status)}
        </Tag>
        <Text type="secondary">来源：{getTaskSource(task, detail)}</Text>
      </Space>
      <Progress
        percent={task.progress}
        status={task.status === 'failed' ? 'exception' : task.status === 'done' ? 'success' : task.status === 'running' ? 'active' : 'normal'}
      />
      {task.message && <Text type={task.status === 'failed' ? 'danger' : 'secondary'}>{task.message}</Text>}
      <TaskResultStrip task={task} />
      {!!promptLog.length && (
        <Collapse
          size="small"
          items={[{
            key: 'prompts',
            label: `生成提示词（${promptLog.length}）`,
            children: (
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {promptLog.map((item, index) => (
                  <div key={`${item.target || 'prompt'}-${index}`}>
                    <Text type="secondary">{item.target || `Prompt ${index + 1}`}</Text>
                    <TextArea value={item.prompt || ''} readOnly autoSize={{ minRows: 4, maxRows: 12 }} style={{ marginTop: 4 }} />
                  </div>
                ))}
              </Space>
            ),
          }]}
        />
      )}
      <Collapse
        size="small"
        items={[{
          key: 'payload',
          label: '任务原始数据',
          children: <TextArea value={task.payload || '{}'} readOnly autoSize={{ minRows: 4, maxRows: 14 }} />,
        }]}
      />
      <Space wrap size={12}>
        <Text type="secondary">创建：{task.created_at || '-'}</Text>
        <Text type="secondary">开始：{task.started_at || '-'}</Text>
        <Text type="secondary">结束：{task.finished_at || '-'}</Text>
      </Space>
    </Space>
  )
}

function StoryboardSegmentList({
  segments,
  media,
  canGenerate,
  canWrite,
  onCopy,
  onGenerate,
  onGenerateVideo,
  onDeleteMedia,
}: {
  segments: DramaStoryboardSegment[]
  media: DramaStoryboardMedia[]
  canGenerate: boolean
  canWrite: boolean
  onCopy: (text: string) => void
  onGenerate: (segment: DramaStoryboardSegment) => void
  onGenerateVideo: (segment: DramaStoryboardSegment) => void
  onDeleteMedia: (mediaId: string) => void
}) {
  return (
    <div>
      <Text strong>片段</Text>
      <List
        style={{ marginTop: 8 }}
        dataSource={segments}
        renderItem={(segment) => {
          const segmentImageMedia = media.filter((item) => item.segment_id === segment.id && item.kind === 'image')
          const segmentVideoMedia = media.filter((item) => item.segment_id === segment.id && item.kind === 'video')
          return (
          <List.Item style={{ border: '1px solid #f0eeeb', borderRadius: 8, padding: 10, marginBottom: 8 }}>
            <Space direction="vertical" size={6} style={{ width: '100%' }}>
              <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
                <Space wrap>
                  <Tag color="blue">#{segment.seq}</Tag>
                  <Text strong>{segment.title || '未命名片段'}</Text>
                  <Tag>{segment.duration_sec || 3}s</Tag>
                  {segment.scene && <Tag color="green">{segment.scene}</Tag>}
                </Space>
                <Button size="small" icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => onGenerate(segment)}>生成图片</Button>
              </Space>
              <Button size="small" icon={<VideoCameraOutlined />} disabled={!canGenerate} onClick={() => onGenerateVideo(segment)}>生成视频</Button>
              {segment.purpose && <Text type="secondary">{segment.purpose}</Text>}
              {segment.action && <Text>{segment.action}</Text>}
              {segment.shot && <Text type="secondary">镜头：{segment.shot}</Text>}
              {segment.composition_prompt && <Text type="secondary">构图：{segment.composition_prompt}</Text>}
              {segment.dialogue && <Text type="secondary">台词：{segment.dialogue}</Text>}
              {segment.reference_file_id !== '0' && (
                <Image
                  src={getPreviewUrl(segment.reference_file_id)}
                  alt={segment.title || `segment-${segment.seq}`}
                  style={{ width: 180, maxWidth: '100%', aspectRatio: '3 / 4', objectFit: 'cover', borderRadius: 6, border: '1px solid #f0eeeb', background: '#f7f7f5' }}
                />
              )}
              {segmentImageMedia.length > 0 && (
                <Image.PreviewGroup>
                  <Space wrap size={8}>
                    {segmentImageMedia.map((item, index) => (
                      <div key={item.id} style={{ width: 104 }}>
                        <Image
                          src={getPreviewUrl(item.file_id)}
                          alt={`${segment.title || `segment-${segment.seq}`}-${index + 1}`}
                          width={104}
                          height={72}
                          style={{ objectFit: 'cover', borderRadius: 6, border: '1px solid #f0eeeb', background: '#f7f7f5' }}
                        />
                        <Space size={4} style={{ marginTop: 4, width: '100%', justifyContent: 'space-between' }}>
                          <Tag>#{index + 1}</Tag>
                          <Popconfirm
                            title="删除这张片段图片？"
                            description="文件会移入回收站"
                            okText="删除"
                            cancelText="取消"
                            onConfirm={() => onDeleteMedia(item.id)}
                            disabled={!canWrite}
                          >
                            <Button size="small" danger icon={<DeleteOutlined />} disabled={!canWrite} />
                          </Popconfirm>
                        </Space>
                      </div>
                    ))}
                  </Space>
                </Image.PreviewGroup>
              )}
              {segmentVideoMedia.length > 0 && (
                <Space wrap size={8}>
                  {segmentVideoMedia.map((item, index) => (
                    <video
                      key={item.id}
                      src={getPreviewUrl(item.file_id)}
                      controls
                      preload="metadata"
                      aria-label={`${segment.title || `segment-${segment.seq}`}-video-${index + 1}`}
                      style={{ width: 240, maxWidth: '100%', aspectRatio: '16 / 9', objectFit: 'cover', borderRadius: 6, background: '#111' }}
                    />
                  ))}
                </Space>
              )}
              <Collapse
                size="small"
                ghost
                items={[
                  {
                    key: 'composition',
                    label: '构图控制提示词',
                    children: (
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <TextArea value={segment.composition_prompt || ''} readOnly autoSize={{ minRows: 2, maxRows: 6 }} />
                        <Button size="small" icon={<CopyOutlined />} onClick={() => onCopy(segment.composition_prompt || '')}>复制构图提示词</Button>
                      </Space>
                    ),
                  },
                  {
                    key: 'reference',
                    label: '参考图提示词',
                    children: (
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <TextArea value={segment.reference_prompt || ''} readOnly autoSize={{ minRows: 3, maxRows: 8 }} />
                        <Button size="small" icon={<CopyOutlined />} onClick={() => onCopy(segment.reference_prompt || '')}>复制参考图提示词</Button>
                      </Space>
                    ),
                  },
                  {
                    key: 'video',
                    label: '视频运动提示词',
                    children: (
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <TextArea value={segment.video_prompt || ''} readOnly autoSize={{ minRows: 2, maxRows: 6 }} />
                        <Button size="small" icon={<CopyOutlined />} onClick={() => onCopy(segment.video_prompt || '')}>复制视频提示词</Button>
                      </Space>
                    ),
                  },
                  {
                    key: 'negative',
                    label: '负面提示词',
                    children: <TextArea value={segment.negative_prompt || ''} readOnly autoSize={{ minRows: 2, maxRows: 6 }} />,
                  },
                ]}
              />
            </Space>
          </List.Item>
          )
        }}
      />
    </div>
  )
}

function AssetPanel({
  detail,
  canWrite,
  canGenerate,
  onChange,
}: {
  detail: DramaDetail
  canWrite: boolean
  canGenerate: boolean
  onChange: (detail: DramaDetail) => void
}) {
  const updateAsset = async (asset: DramaAsset, patch: Partial<DramaAsset>) => {
    const next = await updateDramaAsset(detail.project.id, asset.id, {
      name: patch.name ?? asset.name,
      description: patch.description ?? asset.description,
      reference_prompt: patch.reference_prompt ?? asset.reference_prompt,
      voice_name: patch.voice_name ?? asset.voice_name,
    })
    onChange({ ...detail, assets: detail.assets.map((item) => (item.id === next.id ? next : item)) })
  }

  const uploadReference = async (asset: DramaAsset, file: File) => {
    const next = await uploadDramaAssetReference(detail.project.id, asset.id, file)
    onChange({ ...detail, assets: detail.assets.map((item) => (item.id === next.id ? next : item)) })
    message.success('参考图已更新')
    return Upload.LIST_IGNORE
  }

  const createAssetImageTask = async (asset: DramaAsset) => {
    const task = await createDramaTask(detail.project.id, {
      type: 'asset_reference',
      payload: JSON.stringify({
        source: 'asset',
        source_label: `${asset.type === 'character' ? '角色' : '场景'}：${asset.name}`,
        asset_id: asset.id,
        asset_type: asset.type,
        name: asset.name,
        prompt: asset.reference_prompt || asset.description,
      }),
    })
    onChange({ ...detail, tasks: [task, ...detail.tasks] })
    message.success('参考图生成任务已创建')
  }

  if (!detail.assets.length) return <Empty description="暂无角色或场景资产" />

  return (
    <Row gutter={[16, 16]}>
      {detail.assets.map((asset) => (
        <Col xs={24} md={12} xl={8} key={asset.id}>
          <Card
            title={<Space><Tag color={asset.type === 'character' ? 'blue' : 'green'}>{asset.type === 'character' ? '角色' : '场景'}</Tag><Text strong>{asset.name}</Text></Space>}
            extra={asset.reference_file_id !== '0' ? <Tag color="orange">有参考图</Tag> : <Tag>无参考图</Tag>}
          >
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              {asset.reference_file_id !== '0' && (
                <Image
                  src={getPreviewUrl(asset.reference_file_id)}
                  alt={asset.name}
                  style={{ width: '100%', height: 180, objectFit: 'cover', borderRadius: 8, border: '1px solid #f0eeeb', background: '#f7f7f5' }}
                />
              )}
              <Input value={asset.name} disabled={!canWrite} onChange={(e) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, name: e.target.value } : item)) })} onBlur={() => updateAsset(asset, { name: asset.name })} />
              <TextArea value={asset.description} disabled={!canWrite} autoSize={{ minRows: 5, maxRows: 9 }} onChange={(e) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, description: e.target.value } : item)) })} onBlur={() => updateAsset(asset, { description: asset.description })} />
              <TextArea placeholder="参考图提示词：用于 ComfyUI 生成资产参考图" value={asset.reference_prompt || ''} disabled={!canWrite} autoSize={{ minRows: 3, maxRows: 7 }} onChange={(e) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, reference_prompt: e.target.value } : item)) })} onBlur={() => updateAsset(asset, { reference_prompt: asset.reference_prompt || '' })} />
              {asset.type === 'character' && (
                <Input placeholder="角色音色，例如 zh-CN-XiaoxiaoNeural" value={asset.voice_name} disabled={!canWrite} onChange={(e) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, voice_name: e.target.value } : item)) })} onBlur={() => updateAsset(asset, { voice_name: asset.voice_name })} />
              )}
              <Space wrap>
                <Upload showUploadList={false} accept="image/*" beforeUpload={(file) => uploadReference(asset, file)} disabled={!canWrite}>
                  <Button icon={<UploadOutlined />} disabled={!canWrite}>上传/粘贴参考图</Button>
                </Upload>
                <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => createAssetImageTask(asset)}>ComfyUI 生成</Button>
              </Space>
            </Space>
          </Card>
        </Col>
      ))}
    </Row>
  )
}
