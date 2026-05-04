import api from './api'

export interface FileItem {
  id: number
  user_id: number
  name: string
  is_dir: boolean
  parent_id: number
  size: number
  mime_type: string
  storage_key: string
  storage_sha256: string
  is_shared: boolean
  created_at: string
  updated_at: string
}

export interface FileListResponse {
  items: FileItem[]
  total: number
  page: number
  page_size: number
}

export async function getFileList(parentId: number, page: number, pageSize: number): Promise<FileListResponse> {
  const res = await api.get('/file/list', { params: { parent_id: parentId, page, page_size: pageSize } })
  return res.data.data
}

export interface BatchUploadResult {
  files: FileItem[]
  errors: string[]
  total: number
  ok: number
}

export async function uploadFile(file: File, parentId: number): Promise<FileItem> {
  const form = new FormData()
  form.append('file', file)
  form.append('parent_id', String(parentId))
  const res = await api.post('/file/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return res.data.data
}

export async function uploadFiles(
  files: File[],
  parentId: number,
  onProgress?: (percent: number) => void,
): Promise<BatchUploadResult> {
  const form = new FormData()
  files.forEach((f) => form.append('file', f))
  form.append('parent_id', String(parentId))
  const res = await api.post('/file/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (e) => {
      if (e.total && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    },
  })
  return res.data.data
}

export async function deleteFile(id: number): Promise<void> {
  await api.delete(`/file/${id}`)
}

export async function createDirectory(name: string, parentId: number): Promise<FileItem> {
  const res = await api.post('/file/mkdir', { name, parent_id: parentId })
  return res.data.data
}

export async function searchFiles(keyword: string, page: number, pageSize: number): Promise<FileListResponse> {
  const res = await api.get('/file/search', { params: { q: keyword, page, page_size: pageSize } })
  return res.data.data
}

function getToken(): string {
  try { return localStorage.getItem('access_token') || '' } catch { return '' }
}

export function getDownloadUrl(id: number): string {
  const token = getToken()
  const sep = token ? `?token=${token}` : ''
  return `/api/v1/file/download/${id}${sep}`
}

export function getPreviewUrl(id: number): string {
  const token = getToken()
  const sep = token ? `?inline=true&token=${token}` : '?inline=true'
  return `/api/v1/file/download/${id}${sep}`
}
