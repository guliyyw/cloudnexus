import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ClipboardEvent } from 'react'
import { Button, Empty, Grid, Image, Input, InputNumber, Progress, Segmented, Select, Slider, Space, Tag, Typography, Upload, message } from 'antd'
import { DeleteOutlined, DownloadOutlined, PictureOutlined, PlayCircleOutlined, ReloadOutlined, UploadOutlined } from '@ant-design/icons'
import type { RcFile, UploadFile } from 'antd/es/upload/interface'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { getDownloadUrl, getPreviewUrl } from '../services/file'
import {
  cancelImageGeneration,
  createImageGeneration,
  listImageGenerations,
  parseImageGenerationPayload,
  retryImageGeneration,
  type ImageGenerationTask,
} from '../services/imageGeneration'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Text, Title } = Typography
const { TextArea } = Input

const SIZE_OPTIONS = [
  { value: '768x1024', label: '竖图 3:4' },
  { value: '1024x1024', label: '方图 1:1' },
  { value: '1024x768', label: '横图 4:3' },
  { value: '1344x768', label: '宽屏 16:9' },
]

const statusMeta: Record<string, { label: string; color: string }> = {
  pending: { label: '排队中', color: 'default' },
  running: { label: '生成中', color: 'processing' },
  done: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  canceled: { label: '已取消', color: 'warning' },
}

interface ReferenceUploadFile extends UploadFile {
  sourceWidth?: number
  sourceHeight?: number
}

function fitReferenceSize(sourceWidth: number, sourceHeight: number): string {
  const ratio = sourceWidth / sourceHeight
  let width: number
  let height: number
  if (ratio >= 1) {
    width = ratio >= 1.45 ? 1344 : 1024
    height = Math.round((width / ratio) / 64) * 64
  } else {
    height = ratio <= 0.69 ? 1344 : 1024
    width = Math.round((height * ratio) / 64) * 64
  }
  width = Math.min(1536, Math.max(64, width))
  height = Math.min(1536, Math.max(64, height))
  return `${width}x${height}`
}

async function readImageSize(file: File): Promise<{ width: number; height: number }> {
  if ('createImageBitmap' in window) {
    const bitmap = await createImageBitmap(file)
    const result = { width: bitmap.width, height: bitmap.height }
    bitmap.close()
    return result
  }
  const url = URL.createObjectURL(file)
  try {
    return await new Promise((resolve, reject) => {
      const image = new window.Image()
      image.onload = () => resolve({ width: image.naturalWidth, height: image.naturalHeight })
      image.onerror = reject
      image.src = url
    })
  } finally {
    URL.revokeObjectURL(url)
  }
}

function asRcFile(file: File): RcFile {
  if ('uid' in file) return file as RcFile
  const rcFile = new File([file], file.name || `pasted-${Date.now()}.png`, {
    type: file.type || 'image/png',
    lastModified: file.lastModified || Date.now(),
  }) as RcFile
  rcFile.uid = `paste-${Date.now()}-${Math.random().toString(16).slice(2)}`
  Object.defineProperty(rcFile, 'lastModifiedDate', { value: new Date(rcFile.lastModified) })
  return rcFile
}

export default function ImageGenerationPage() {
  const screens = Grid.useBreakpoint()
  const [prompt, setPrompt] = useState('')
  const [negativePrompt, setNegativePrompt] = useState('')
  const [size, setSize] = useState('768x1024')
  const [steps, setSteps] = useState(24)
  const [cfg, setCfg] = useState(7)
  const [imageCount, setImageCount] = useState(1)
  const [referenceWeight, setReferenceWeight] = useState(0.65)
  const [uploads, setUploads] = useState<ReferenceUploadFile[]>([])
  const [tasks, setTasks] = useState<ImageGenerationTask[]>([])
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(async (quiet = false) => {
    if (!quiet) setLoading(true)
    try {
      setTasks(await listImageGenerations())
    } catch {
      if (!quiet) message.error('读取生成记录失败')
    } finally {
      if (!quiet) setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])
  useEffect(() => {
    const active = tasks.some((task) => task.status === 'pending' || task.status === 'running')
    if (!active) return
    const timer = window.setInterval(() => load(true), 2500)
    return () => window.clearInterval(timer)
  }, [tasks, load])

  const results = useMemo(() => tasks.flatMap((task) => parseImageGenerationPayload(task).results || []), [tasks])
  const activeCount = tasks.filter((task) => task.status === 'pending' || task.status === 'running').length
  const firstReference = uploads[0]

  const addReferenceFiles = async (files: File[]) => {
    const imageFiles = files.filter((file) => file.type.startsWith('image/'))
    if (!imageFiles.length) return
    const room = Math.max(0, 4 - uploads.length)
    if (!room) {
      message.warning('最多添加 4 张参考图')
      return
    }
    const accepted = imageFiles.filter((file) => {
      if (file.size > 10 * 1024 * 1024) {
        message.error(`${file.name || '粘贴的图片'}超过 10 MB`)
        return false
      }
      return true
    }).slice(0, room)
    const nextFiles = await Promise.all(accepted.map(async (source) => {
      const file = asRcFile(source)
      const dimensions = await readImageSize(file)
      return {
        uid: file.uid,
        name: file.name,
        status: 'done' as const,
        type: file.type,
        size: file.size,
        originFileObj: file,
        thumbUrl: URL.createObjectURL(file),
        sourceWidth: dimensions.width,
        sourceHeight: dimensions.height,
      }
    }))
    if (!uploads.length && nextFiles[0]?.sourceWidth && nextFiles[0]?.sourceHeight) {
      setSize(fitReferenceSize(nextFiles[0].sourceWidth, nextFiles[0].sourceHeight))
    }
    setUploads((current) => [...current, ...nextFiles].slice(0, 4))
    if (nextFiles.length) message.success(`已添加 ${nextFiles.length} 张参考图`)
  }

  const handlePaste = (event: ClipboardEvent<HTMLDivElement>) => {
    const files = Array.from(event.clipboardData.items)
      .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
      .flatMap((item) => {
        const file = item.getAsFile()
        return file ? [file] : []
      })
    if (!files.length) return
    event.preventDefault()
    void addReferenceFiles(files)
  }

  const submit = async () => {
    if (!prompt.trim()) {
      message.warning('请输入图片提示词')
      return
    }
    const [width, height] = size.split('x').map(Number)
    setSubmitting(true)
    try {
      const task = await createImageGeneration({
        prompt: prompt.trim(), negativePrompt: negativePrompt.trim(), width, height,
        steps, cfg, imageCount, referenceWeight,
        references: uploads.flatMap((file) => file.originFileObj ? [file.originFileObj] : []),
      })
      setTasks((current) => [task, ...current])
      message.success('生成任务已加入队列')
    } catch (error: any) {
      message.error(error.response?.data?.message || '创建生成任务失败')
    } finally {
      setSubmitting(false)
    }
  }

  const removeUpload = (file: UploadFile) => {
    if (file.thumbUrl?.startsWith('blob:')) URL.revokeObjectURL(file.thumbUrl)
    const next = uploads.filter((item) => item.uid !== file.uid)
    setUploads(next)
    if (next[0]?.sourceWidth && next[0]?.sourceHeight) {
      setSize(fitReferenceSize(next[0].sourceWidth, next[0].sourceHeight))
    }
  }

  const updateTask = (next: ImageGenerationTask) => setTasks((current) => current.map((task) => task.id === next.id ? next : task))

  return (
    <div onPaste={handlePaste}>
      <PageHeader
        eyebrow="AI IMAGE"
        title="图片生成"
        description="输入画面描述，也可以加入参考图片来约束风格、人物或构图。"
        actions={<Button icon={<ReloadOutlined />} loading={loading} onClick={() => load()}>刷新记录</Button>}
      />
      <MetricStrip items={[
        { label: '生成记录', value: tasks.length },
        { label: '进行中', value: activeCount, tone: activeCount ? 'warning' : 'default' },
        { label: '已生成图片', value: results.length, tone: 'primary' },
      ]} />

      <section style={{ display: 'grid', gridTemplateColumns: screens.lg ? 'minmax(300px, 420px) minmax(0, 1fr)' : 'minmax(0, 1fr)', gap: spacing.lg, alignItems: 'start' }}>
        <div style={{ padding: spacing.lg, borderRadius: radius.lg, border: `1px solid ${colors.borderSubtle}`, background: colors.surfaceRaised, boxShadow: shadow.card }}>
          <Title level={5} style={{ marginTop: 0 }}>生成设置</Title>
          <Text strong>提示词</Text>
          <TextArea value={prompt} onChange={(event) => setPrompt(event.target.value)} rows={6} maxLength={2000} showCount placeholder="描述主体、场景、光线、镜头和风格" style={{ marginTop: 8, marginBottom: 16 }} />
          <Text strong>不希望出现的内容</Text>
          <TextArea value={negativePrompt} onChange={(event) => setNegativePrompt(event.target.value)} rows={2} maxLength={500} placeholder="例如：模糊、文字、水印" style={{ marginTop: 8, marginBottom: 16 }} />

          <Text strong>参考图片</Text>
          <Upload.Dragger
            accept="image/*"
            multiple
            maxCount={4}
            fileList={uploads}
            listType="picture"
            beforeUpload={(file) => {
              void addReferenceFiles([file])
              return false
            }}
            onRemove={(file) => { removeUpload(file); return false }}
            showUploadList={{ showRemoveIcon: true, removeIcon: <DeleteOutlined /> }}
            style={{ marginTop: 8 }}
          >
            <UploadOutlined style={{ fontSize: 22, color: colors.primary }} />
            <div style={{ marginTop: 6 }}>点击、拖入或粘贴参考图片</div>
            <Text type="secondary" style={{ fontSize: 12 }}>支持 Ctrl+V，最多 4 张，每张不超过 10 MB</Text>
          </Upload.Dragger>

          {uploads.length > 0 && (
            <div style={{ marginTop: 16 }}>
              <Text type="secondary">参考强度 {referenceWeight.toFixed(2)}</Text>
              <Slider min={0.1} max={1.2} step={0.05} value={referenceWeight} onChange={setReferenceWeight} />
            </div>
          )}

          <div style={{ marginTop: 16 }}><Text strong>画布比例</Text></div>
          <Select
            value={size}
            onChange={setSize}
            disabled={uploads.length > 0}
            options={firstReference?.sourceWidth && firstReference?.sourceHeight
              ? [{ value: size, label: `按首张参考图 ${firstReference.sourceWidth}×${firstReference.sourceHeight}` }]
              : SIZE_OPTIONS}
            style={{ width: '100%', marginTop: 8 }}
          />
          {uploads.length > 0 && <Text type="secondary" style={{ display: 'block', marginTop: 6, fontSize: 12 }}>生成尺寸 {size}，保持首张参考图比例</Text>}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: spacing.md, marginTop: 16 }}>
            <div><Text type="secondary">生成张数</Text><Segmented block value={imageCount} onChange={(value) => setImageCount(Number(value))} options={[1, 2, 3, 4]} style={{ marginTop: 8 }} /></div>
            <div><Text type="secondary">采样步数</Text><InputNumber min={1} max={100} value={steps} onChange={(value) => setSteps(value || 24)} style={{ width: '100%', marginTop: 8 }} /></div>
          </div>
          <div style={{ marginTop: 16 }}><Text type="secondary">提示词引导强度 {cfg.toFixed(1)}</Text><Slider min={1} max={20} step={0.5} value={cfg} onChange={setCfg} /></div>
          <Button type="primary" block size="large" icon={<PictureOutlined />} loading={submitting} onClick={submit}>开始生成</Button>
        </div>

        <div style={{ minWidth: 0 }}>
          <Title level={5} style={{ marginTop: 0 }}>生成记录</Title>
          {!tasks.length && !loading ? <Empty description="还没有生成记录" /> : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.md }}>
              {tasks.map((task) => {
                const payload = parseImageGenerationPayload(task)
                const taskResults = payload.results || []
                const meta = statusMeta[task.status] || statusMeta.pending
                return (
                  <article key={task.id} style={{ padding: spacing.md, border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.md, background: colors.surface }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: spacing.md, alignItems: 'flex-start' }}>
                      <div style={{ minWidth: 0 }}>
                        <Tag color={meta.color}>{meta.label}</Tag>
                        <Text strong ellipsis style={{ display: 'block', marginTop: 8 }}>{payload.prompt || '图片生成任务'}</Text>
                        <Text type="secondary" style={{ fontSize: 12 }}>{new Date(task.created_at).toLocaleString()}</Text>
                      </div>
                      <Space>
                        {(task.status === 'pending' || task.status === 'running') && <Button size="small" onClick={async () => updateTask(await cancelImageGeneration(task.id))}>取消</Button>}
                        {(task.status === 'failed' || task.status === 'canceled') && <Button size="small" icon={<PlayCircleOutlined />} onClick={async () => updateTask(await retryImageGeneration(task.id))}>重试</Button>}
                      </Space>
                    </div>
                    {(task.status === 'pending' || task.status === 'running') && <Progress percent={task.progress} status="active" style={{ marginTop: 12 }} />}
                    {task.status === 'failed' && <Text type="danger" style={{ display: 'block', marginTop: 10 }}>{task.message}</Text>}
                    {taskResults.length > 0 && (
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: spacing.sm, marginTop: spacing.md }}>
                        {taskResults.map((result) => (
                          <div key={result.file_id} style={{ position: 'relative', aspectRatio: `${payload.width || 1} / ${payload.height || 1}`, minHeight: 180, overflow: 'hidden', borderRadius: radius.sm, background: colors.surfaceMuted }}>
                            <Image src={getPreviewUrl(result.file_id)} alt={result.title} width="100%" height="100%" style={{ objectFit: 'cover' }} />
                            <Button href={getDownloadUrl(result.file_id)} icon={<DownloadOutlined />} title="下载" style={{ position: 'absolute', right: 8, bottom: 8 }} />
                          </div>
                        ))}
                      </div>
                    )}
                  </article>
                )
              })}
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
