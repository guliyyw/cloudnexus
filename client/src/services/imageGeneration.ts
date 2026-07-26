import api from './api'

export interface ImageGenerationResult {
  kind: string
  file_id: string
  title: string
  prompt: string
}

export interface ImageGenerationPayload {
  prompt?: string
  negative_prompt?: string
  width?: number
  height?: number
  image_count?: number
  reference_file_ids?: string[]
  results?: ImageGenerationResult[]
}

export interface ImageGenerationTask {
  id: string
  status: 'pending' | 'running' | 'done' | 'failed' | 'canceled'
  progress: number
  message: string
  payload: string
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface CreateImageGenerationInput {
  prompt: string
  negativePrompt?: string
  width: number
  height: number
  steps: number
  cfg: number
  imageCount: number
  referenceWeight: number
  references: File[]
}

export function parseImageGenerationPayload(task: ImageGenerationTask): ImageGenerationPayload {
  try {
    return JSON.parse(task.payload || '{}') as ImageGenerationPayload
  } catch {
    return {}
  }
}

export async function listImageGenerations(): Promise<ImageGenerationTask[]> {
  const res = await api.get('/image-generation')
  return res.data.data.items || []
}

export async function createImageGeneration(input: CreateImageGenerationInput): Promise<ImageGenerationTask> {
  const form = new FormData()
  form.append('prompt', input.prompt)
  form.append('negative_prompt', input.negativePrompt || '')
  form.append('width', String(input.width))
  form.append('height', String(input.height))
  form.append('steps', String(input.steps))
  form.append('cfg', String(input.cfg))
  form.append('image_count', String(input.imageCount))
  form.append('reference_weight', String(input.referenceWeight))
  input.references.forEach((file) => form.append('references', file))
  const res = await api.post('/image-generation', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return res.data.data.task
}

export async function cancelImageGeneration(taskId: string): Promise<ImageGenerationTask> {
  const res = await api.post(`/image-generation/${taskId}/cancel`)
  return res.data.data.task
}

export async function retryImageGeneration(taskId: string): Promise<ImageGenerationTask> {
  const res = await api.post(`/image-generation/${taskId}/retry`)
  return res.data.data.task
}
