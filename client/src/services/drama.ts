import api from './api'

export interface DramaProject {
  id: string
  title: string
  description: string
  preface: string
  raw_script?: string
  settings: string
  created_at: string
  updated_at: string
}

export interface DramaStoryboard {
  id: string
  project_id: string
  seq: number
  title: string
  content: string
  original: string
  prompt: string
  dialogue: string
  scene_anchor: string
  plot: string
  modified: boolean
  image_file_id: string
  audio_file_id: string
  audio_duration_ms: number
  subtitle_ass: string
  video_file_id: string
}

export interface DramaStoryboardMedia {
  id: string
  project_id: string
  storyboard_id: string
  segment_id: string
  kind: 'image' | 'video'
  file_id: string
  source: string
  prompt: string
  sort_order: number
  selected: boolean
  created_at: string
}

export interface DramaStoryboardSegment {
  id: string
  project_id: string
  storyboard_id: string
  seq: number
  title: string
  duration_sec: number
  purpose: string
  characters: string
  scene: string
  dialogue: string
  action: string
  shot: string
  composition_prompt: string
  reference_prompt: string
  video_prompt: string
  negative_prompt: string
  reference_file_id: string
  video_file_id: string
}

export interface DramaAsset {
  id: string
  project_id: string
  type: 'character' | 'scene'
  name: string
  description: string
  reference_prompt: string
  voice_name: string
  reference_file_id: string
}

export interface DramaTask {
  id: string
  project_id: string
  type: string
  status: string
  progress: number
  message: string
  payload: string
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface DramaSetting {
  id: string
  comfyui_url: string
  image_settings: string
  tts_engine: string
  tts_config: string
  video_settings: string
  storage_root: string
}

export interface ComfyUIStatus {
  connected: boolean
  url: string
  checkpoints: string[]
  ip_adapter: boolean
  reactor: boolean
  missing: string[]
  error?: string
}

export interface DramaDetail {
  project: DramaProject
  storyboards: DramaStoryboard[]
  media: DramaStoryboardMedia[]
  segments: DramaStoryboardSegment[]
  assets: DramaAsset[]
  tasks: DramaTask[]
  summary: {
    storyboard_count: number
    asset_count: number
    modified_count: number
  }
}

export async function listDramaProjects(params?: { keyword?: string; sort?: string }) {
  const res = await api.get('/drama/projects', { params })
  return res.data.data as { items: DramaProject[]; total: number }
}

export async function createDramaProject(data: { title: string; description?: string }) {
  const res = await api.post('/drama/projects', data)
  return res.data.data.project as DramaProject
}

export async function getDramaProject(id: string) {
  const res = await api.get(`/drama/projects/${id}`)
  return res.data.data as DramaDetail
}

export async function updateDramaProject(id: string, data: Partial<DramaProject>) {
  const res = await api.put(`/drama/projects/${id}`, data)
  return res.data.data.project as DramaProject
}

export async function deleteDramaProject(id: string) {
  await api.delete(`/drama/projects/${id}`)
}

export async function parseDramaScript(id: string, script: string) {
  const res = await api.post(`/drama/projects/${id}/parse`, { script })
  return res.data.data as DramaDetail
}

export async function updateDramaStoryboard(projectId: string, storyboardId: string, data: { content: string; prompt?: string }) {
  const res = await api.put(`/drama/projects/${projectId}/storyboards/${storyboardId}`, data)
  return res.data.data.storyboard as DramaStoryboard
}

export async function selectDramaStoryboardMedia(projectId: string, storyboardId: string, mediaId: string) {
  const res = await api.put(`/drama/projects/${projectId}/storyboards/${storyboardId}/media/${mediaId}/select`)
  return res.data.data.storyboard as DramaStoryboard
}

export async function deleteDramaStoryboardMedia(projectId: string, storyboardId: string, mediaId: string) {
  const res = await api.delete(`/drama/projects/${projectId}/storyboards/${storyboardId}/media/${mediaId}`)
  return res.data.data as { storyboard: DramaStoryboard; media: DramaStoryboardMedia[] }
}

export async function importDramaStoryboardSegments(projectId: string, storyboardId: string, text: string) {
  const res = await api.post(`/drama/projects/${projectId}/storyboards/${storyboardId}/segments/import`, { text })
  return res.data.data.segments as DramaStoryboardSegment[]
}

export async function uploadStoryboardAudio(projectId: string, storyboardId: string, file: File, durationMs?: number) {
  const form = new FormData()
  form.append('file', file)
  if (durationMs) form.append('duration_ms', String(durationMs))
  const res = await api.post(`/drama/projects/${projectId}/storyboards/${storyboardId}/audio`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data.storyboard as DramaStoryboard
}

export async function batchImportDramaAudio(projectId: string, files: File[]) {
  const form = new FormData()
  files.forEach((file) => form.append('files', file))
  const res = await api.post(`/drama/projects/${projectId}/audio/import`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data.results as Array<{ file_name: string; matched: boolean; reason?: string; storyboard?: DramaStoryboard }>
}

export async function appendDramaStoryboards(projectId: string, suffix: string) {
  const res = await api.post(`/drama/projects/${projectId}/append`, { suffix })
  return res.data.data.storyboards as DramaStoryboard[]
}

export async function updateDramaAsset(projectId: string, assetId: string, data: { name: string; description: string; reference_prompt?: string; voice_name?: string }) {
  const res = await api.put(`/drama/projects/${projectId}/assets/${assetId}`, data)
  return res.data.data.asset as DramaAsset
}

export async function importDramaAssets(projectId: string, text: string) {
  const res = await api.post(`/drama/projects/${projectId}/assets/import`, { text })
  return res.data.data.assets as DramaAsset[]
}

export async function createDramaTask(projectId: string, data: { type: string; storyboard_ids?: string[]; payload?: string }) {
  const res = await api.post(`/drama/projects/${projectId}/tasks`, data)
  return res.data.data.task as DramaTask
}

export async function listDramaTasks(projectId: string) {
  const res = await api.get(`/drama/projects/${projectId}/tasks`)
  return res.data.data.items as DramaTask[]
}

export async function cancelDramaTask(projectId: string, taskId: string) {
  const res = await api.post(`/drama/projects/${projectId}/tasks/${taskId}/cancel`)
  return res.data.data.task as DramaTask
}

export async function retryDramaTask(projectId: string, taskId: string) {
  const res = await api.post(`/drama/projects/${projectId}/tasks/${taskId}/retry`)
  return res.data.data.task as DramaTask
}

export async function uploadDramaAssetReference(projectId: string, assetId: string, file: File) {
  const form = new FormData()
  form.append('file', file)
  const res = await api.post(`/drama/projects/${projectId}/assets/${assetId}/reference`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data.asset as DramaAsset
}

export async function importDramaProject(file: File) {
  const form = new FormData()
  form.append('file', file)
  const res = await api.post('/drama/projects/import', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data.project as DramaProject
}

export async function exportDramaProject(id: string) {
  const res = await api.get(`/drama/projects/${id}/export`, { responseType: 'blob' })
  return res.data as Blob
}

export async function getDramaSetting() {
  const res = await api.get('/drama/settings')
  return res.data.data.setting as DramaSetting
}

export async function getComfyUIStatus(url?: string) {
  const res = await api.get('/drama/settings/comfyui/status', { params: url ? { url } : undefined })
  return res.data.data.status as ComfyUIStatus
}

export async function saveDramaSetting(data: Partial<DramaSetting>) {
  const res = await api.put('/drama/settings', data)
  return res.data.data.setting as DramaSetting
}
