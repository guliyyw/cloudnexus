import api from './api'

export interface CollabDocument {
  id: string
  title: string
  content?: string  // base64-encoded Yjs state (not used by frontend)
  owner_id: string
  last_editor: string
  version: number
  created_at: string
  updated_at: string
}

export interface CollabListResponse {
  data: CollabDocument[]
  total: number
  page: number
  page_size: number
}

export async function listDocuments(page = 1, pageSize = 20): Promise<CollabListResponse> {
  const { data } = await api.get('/collab', { params: { page, page_size: pageSize } })
  return data.data
}

export async function createDocument(title: string): Promise<CollabDocument> {
  const { data } = await api.post('/collab', { title })
  return data.data
}

export async function getDocument(id: string): Promise<CollabDocument> {
  const { data } = await api.get(`/collab/${id}`)
  return data.data
}

export async function updateDocument(id: string, title: string): Promise<CollabDocument> {
  const { data } = await api.put(`/collab/${id}`, { title })
  return data.data
}

export async function deleteDocument(id: string): Promise<void> {
  await api.delete(`/collab/${id}`)
}
