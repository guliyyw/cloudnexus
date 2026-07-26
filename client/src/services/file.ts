import axios from 'axios'
import api from './api'

export interface FileItem {
  id: string
  user_id: string
  name: string
  is_dir: boolean
  parent_id: string
  size: number
  mime_type: string
  storage_key: string
  storage_sha256: string
  is_shared: boolean
  collab_type: string
  created_at: string
  updated_at: string
}

export interface FileListResponse {
  items: FileItem[]
  total: number
  page: number
  page_size: number
}

export async function getFileList(parentId: string, page: number, pageSize: number): Promise<FileListResponse> {
  const res = await api.get('/file/list', { params: { parent_id: parentId, page, page_size: pageSize } })
  return res.data.data
}

export interface BatchUploadResult {
  files: FileItem[]
  errors: string[]
  total: number
  ok: number
}

export async function uploadFile(file: File, parentId: string): Promise<FileItem> {
  const form = new FormData()
  form.append('file', file)
  form.append('parent_id', parentId)
  const res = await api.post('/file/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  const data = res.data.data
  return data.files ? data.files[0] : data
}

export async function uploadFiles(
  files: File[],
  parentId: string,
  onProgress?: (percent: number) => void,
): Promise<BatchUploadResult> {
  const form = new FormData()
  files.forEach((f) => form.append('file', f))
  form.append('parent_id', parentId)
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

export async function deleteFile(id: string): Promise<void> {
  await api.delete(`/file/${id}`)
}

export async function createDirectory(name: string, parentId: string): Promise<FileItem> {
  const res = await api.post('/file/mkdir', { name, parent_id: parentId })
  return res.data.data
}

export async function searchFiles(keyword: string, page: number, pageSize: number): Promise<FileListResponse> {
  const res = await api.get('/file/search', { params: { q: keyword, page, page_size: pageSize } })
  return res.data.data
}

export interface BatchDeleteResult {
  deleted: number
  errors: string[]
}

export async function batchDeleteFiles(ids: string[]): Promise<BatchDeleteResult> {
  const res = await api.post('/file/batch-delete', { ids })
  return res.data.data
}

export async function batchDownloadFiles(ids: string[]): Promise<void> {
  const res = await api.post('/file/batch-download', { ids }, { responseType: 'blob' })
  const url = URL.createObjectURL(res.data)
  const a = document.createElement('a')
  a.href = url
  const disposition = res.headers['content-disposition']
  const match = disposition?.match(/filename="?(.+?)"?$/)
  a.download = match?.[1] || 'files.zip'
  a.click()
  URL.revokeObjectURL(url)
}

export async function moveFile(id: string, targetParentId: string): Promise<FileItem> {
  const res = await api.post('/file/move', { id, target_parent_id: targetParentId })
  return res.data.data
}

export async function copyFile(id: string, targetParentId: string): Promise<FileItem> {
  const res = await api.post('/file/copy', { id, target_parent_id: targetParentId })
  return res.data.data
}

function getToken(): string {
  try { return localStorage.getItem('access_token') || '' } catch { return '' }
}

export function getDownloadUrl(id: string): string {
  // JWT 不再通过 URL 参数传递，使用 Cookie 或 Header
  return `/api/v1/file/download/${id}`
}

export function getPreviewUrl(id: string): string {
  return `/api/v1/file/download/${id}?inline=true`
}

export function getWordPdfUrl(id: string): string {
  return `/api/v1/file/${id}/convert/pdf`
}

export async function downloadFileBlob(id: string): Promise<Blob> {
  const res = await api.get(`/file/download/${id}`, { responseType: 'blob' })
  return res.data
}

export async function saveTextFile(id: string, content: string, versionMessage = 'online edit'): Promise<FileItem> {
  const res = await api.put(`/file/${id}/text`, { content, version_message: versionMessage })
  return res.data.data
}

export async function saveFileContent(id: string, blob: Blob, filename: string, versionMessage = 'online edit'): Promise<FileItem> {
  const form = new FormData()
  form.append('file', blob, filename)
  form.append('version_message', versionMessage)
  const res = await api.put(`/file/${id}/content`, form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return res.data.data
}

export async function saveWordHtml(id: string, html: string, versionMessage = 'Word online edit'): Promise<FileItem> {
  const res = await api.put(`/file/${id}/word`, { html, version_message: versionMessage })
  return res.data.data
}

export async function exportWordHtml(id: string, html: string): Promise<Blob> {
  const res = await api.post(`/file/${id}/convert/docx`, { html }, { responseType: 'blob' })
  return res.data
}

export interface ShareInfo {
  id: string
  file_id: string
  owner_id: string
  share_code: string
  expires_at: string | null
  download_limit: number
  download_count: number
  created_at: string
  file_name: string
  file_size: number
  mime_type: string
  has_password: boolean
}

export async function createShare(fileId: string, password?: string, expiresIn?: number): Promise<ShareInfo> {
  const res = await api.post(`/file/${fileId}/share`, { password, expires_in: expiresIn })
  return res.data.data
}

export async function getFileShares(fileId: string): Promise<ShareInfo[]> {
  const res = await api.get(`/file/${fileId}/shares`)
  return res.data.data
}

export async function getMyShares(): Promise<ShareInfo[]> {
  const res = await api.get('/shares/my')
  return res.data.data
}

export async function deleteShare(shareId: string): Promise<void> {
  await api.delete(`/shares/${shareId}`)
}

export async function getShareByCode(code: string): Promise<ShareInfo> {
  const res = await axios.get(`/api/v1/share/${code}`)
  return res.data.data
}

export async function verifySharePassword(code: string, password: string): Promise<void> {
  await axios.post(`/api/v1/share/${code}/verify`, { password })
}

export function getShareUrl(code: string): string {
  return `${window.location.origin}/s/${code}`
}

export function getShareDownloadUrl(code: string): string {
  // 密码不再通过 URL 参数传递，使用 POST body 或 Header
  return `/api/v1/share/${code}/download`
}

export function getSharePreviewUrl(code: string): string {
  return `/api/v1/share/${code}/download?inline=true`
}

// ── 协作文档 ──

export async function createCollabDoc(title: string, parentId: string, collabType: string = 'doc'): Promise<FileItem> {
  const res = await api.post('/file/collab', { title, parent_id: parentId, collab_type: collabType })
  return res.data.data
}

export async function createOfficeDoc(title: string, parentId: string, kind: 'word' | 'excel'): Promise<FileItem> {
  const res = await api.post('/file/office', { title, parent_id: parentId, kind })
  return res.data.data
}

export async function getFileMeta(id: string): Promise<FileItem> {
  const res = await api.get(`/file/${id}/meta`)
  return res.data.data
}

// ── 文件版本 ──

export interface FileVersion {
  id: string
  file_id: string
  version_num: number
  storage_key: string
  size: number
  sha256: string
  message: string
  created_at: string
}

export interface VersionListResponse {
  items: FileVersion[]
  total: number
  page: number
  page_size: number
}

export async function getVersions(fileId: string, page: number, pageSize: number): Promise<VersionListResponse> {
  const res = await api.get(`/file/${fileId}/versions`, { params: { page, page_size: pageSize } })
  return res.data.data
}

export async function restoreVersion(fileId: string, versionId: string): Promise<FileItem> {
  const res = await api.post(`/file/${fileId}/versions/${versionId}/restore`)
  return res.data.data
}

export function getVersionDownloadUrl(fileId: string, versionId: string): string {
  const token = getToken()
  const sep = token ? `?token=${token}` : ''
  return `/api/v1/file/${fileId}/versions/${versionId}/download${sep}`
}

// ── 分块上传 ──

export interface ChunkInitResponse {
  upload_id: string
  chunk_size: number
  total_chunks: number
}

export interface ChunkUploadInfo {
  upload_id: string
  file_name: string
  file_size: number
  total_chunks: number
  completed: number[]
  chunk_size: number
  status: string
  created_at: string
}

export async function initChunkUpload(params: {
  file_name: string; file_size: number; parent_id: string; mime_type?: string
}): Promise<ChunkInitResponse> {
  const res = await api.post('/file/chunk/init', params)
  return res.data.data
}

export async function uploadChunk(
  uploadId: string, chunkIndex: number, chunk: Blob,
  onProgress?: (pct: number) => void,
): Promise<{ chunk_index: number; completed: number; total_chunks: number }> {
  const form = new FormData()
  form.append('upload_id', uploadId)
  form.append('chunk_index', String(chunkIndex))
  form.append('chunk', chunk, `chunk_${chunkIndex}`)
  const res = await api.post('/file/chunk/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (e) => {
      if (e.total && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    },
  })
  return { ...res.data.data, chunk_index: chunkIndex }
}

export async function getChunkUploadStatus(uploadId: string): Promise<ChunkUploadInfo> {
  const res = await api.get(`/file/chunk/status/${uploadId}`)
  return res.data.data
}

export async function completeChunkUpload(uploadId: string, versionMessage?: string): Promise<FileItem> {
  const res = await api.post('/file/chunk/complete', { upload_id: uploadId, version_message: versionMessage })
  return res.data.data
}

export async function cancelChunkUpload(uploadId: string): Promise<void> {
  await api.delete(`/file/chunk/cancel/${uploadId}`)
}

export async function listIncompleteUploads(): Promise<ChunkUploadInfo[]> {
  const res = await api.get('/file/chunk/incomplete')
  return res.data.data.uploads
}

// ── 回收站 ──

export async function getTrashList(page: number, pageSize: number): Promise<FileListResponse> {
  const res = await api.get('/file/trash/', { params: { page, page_size: pageSize } })
  return res.data.data
}

export async function restoreFromTrash(id: string): Promise<void> {
  await api.post(`/file/trash/${id}/restore`)
}

export async function permanentDelete(id: string): Promise<void> {
  await api.delete(`/file/trash/${id}`)
}

export async function emptyTrash(): Promise<{ deleted: number }> {
  const res = await api.delete('/file/trash/')
  return res.data.data
}

// ── 配额 ──

export interface QuotaInfo {
  used: number
  limit: number
  tier_name: string
  trash_used: number
  trash_limit: number
  usage_percent: number
}

export async function getQuota(): Promise<QuotaInfo> {
  const res = await api.get('/user/quota')
  return res.data.data
}
