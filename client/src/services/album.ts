import api from './api'
import type { FileItem } from './file'

export interface Album {
  id: string
  owner_id: string
  name: string
  description: string
  cover_file_id: string
  file_count: number
  created_at: string
  updated_at: string
}

export interface AlbumListResponse {
  albums: Album[]
  total: number
}

export interface AlbumFilesResponse {
  files: FileItem[]
  total: number
}

export async function getAlbums(page = 1, pageSize = 20): Promise<AlbumListResponse> {
  const res = await api.get('/albums', { params: { page, page_size: pageSize } })
  return res.data.data
}

export async function getAlbum(id: string): Promise<Album> {
  const res = await api.get(`/albums/${id}`)
  return res.data.data
}

export async function createAlbum(name: string, description: string): Promise<Album> {
  const res = await api.post('/albums', { name, description })
  return res.data.data
}

export async function updateAlbum(id: string, data: { name?: string; description?: string; cover_file_id?: string }): Promise<Album> {
  const res = await api.put(`/albums/${id}`, data)
  return res.data.data
}

export async function deleteAlbum(id: string): Promise<void> {
  await api.delete(`/albums/${id}`)
}

export async function getAlbumFiles(id: string, page = 1, pageSize = 50): Promise<AlbumFilesResponse> {
  const res = await api.get(`/albums/${id}/files`, { params: { page, page_size: pageSize } })
  return res.data.data
}

export async function addFilesToAlbum(albumId: string, fileIds: string[]): Promise<void> {
  await api.post(`/albums/${albumId}/files`, { file_ids: fileIds.map(String) })
}

export async function removeFileFromAlbum(albumId: string, fileId: string): Promise<void> {
  await api.delete(`/albums/${albumId}/files/${fileId}`)
}
