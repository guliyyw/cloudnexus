import api from './api'

export interface Camera {
  id: string
  owner_id: string
  name: string
  stream_url: string
  protocol: string
  status: string
  last_seen_at: string | null
  created_at: string
  updated_at: string
}

export interface RecognitionEvent {
  id: string
  camera_id: string
  event_type: string
  confidence: number
  snapshot_url: string
  metadata: string
  created_at: string
}

export interface DetectedObject {
  class: string
  confidence: number
  x1: number
  y1: number
  x2: number
  y2: number
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export async function getCameras(page = 1, pageSize = 10): Promise<PaginatedResponse<Camera>> {
  const { data } = await api.get('/cameras', { params: { page, page_size: pageSize } })
  return data.data
}

export async function createCamera(params: { name: string; stream_url: string; protocol?: string }): Promise<Camera> {
  const { data } = await api.post('/cameras', params)
  return data.data.camera
}

export async function updateCamera(id: string, params: { name: string; stream_url: string; protocol?: string }): Promise<void> {
  await api.put(`/cameras/${id}`, params)
}

export async function deleteCamera(id: string): Promise<void> {
  await api.delete(`/cameras/${id}`)
}

export async function startStream(id: string): Promise<{ hls_url: string; webrtc_url: string }> {
  const { data } = await api.post(`/cameras/${id}/stream/start`)
  return data.data
}

export async function stopStream(id: string): Promise<void> {
  await api.post(`/cameras/${id}/stream/stop`)
}

export async function startRecognition(id: string): Promise<void> {
  await api.post(`/cameras/${id}/recognition/start`)
}

export async function stopRecognition(id: string): Promise<void> {
  await api.post(`/cameras/${id}/recognition/stop`)
}

export async function getEvents(id: string, page = 1, pageSize = 20): Promise<PaginatedResponse<RecognitionEvent>> {
  const { data } = await api.get(`/cameras/${id}/events`, { params: { page, page_size: pageSize } })
  return data.data
}

export interface DiscoverRequest {
  subnet: string
  ports?: number[]
}

export interface DiscoveredCamera {
  ip: string
  port: number
  rtsp_url: string
  source: string
}

export interface DiscoverResponse {
  cameras: DiscoveredCamera[]
  scan_duration_ms: number
  total_scanned: number
  open_ports: number
}

export async function discoverCameras(params: DiscoverRequest): Promise<DiscoverResponse> {
  const { data } = await api.post('/cameras/discover', params)
  return data.data
}

export async function detectImage(file: File): Promise<DetectedObject[]> {
  const form = new FormData()
  form.append('image', file)
  const { data } = await api.post('/detect-image', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data.data.objects
}
