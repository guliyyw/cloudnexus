import api from './api'

export interface ContainerInfo {
  id: string
  name: string
  image: string
  status: string
  ports: string[]
  created: string
}

export async function listContainers(all: boolean = false): Promise<ContainerInfo[]> {
  const res = await api.get('/docker/containers', { params: { all } })
  return res.data.data
}

export async function createContainer(image: string, name: string): Promise<{ id: string }> {
  const res = await api.post('/docker/containers', { image, name })
  return res.data.data
}

export async function startContainer(id: string): Promise<void> {
  await api.post(`/docker/containers/${id}/start`)
}

export async function stopContainer(id: string): Promise<void> {
  await api.post(`/docker/containers/${id}/stop`)
}

export async function restartContainer(id: string): Promise<void> {
  await api.post(`/docker/containers/${id}/restart`)
}

export async function removeContainer(id: string, force = false): Promise<void> {
  await api.delete(`/docker/containers/${id}`, { params: { force } })
}

export async function getContainerLogs(id: string, tail = '100'): Promise<string> {
  const res = await api.get(`/docker/containers/${id}/logs`, { params: { tail }, responseType: 'text' })
  return res.data
}

// --- Image management ---

export interface ImageInfo {
  id: string
  tags: string[]
  size: number
  created: string
}

export async function listImages(): Promise<ImageInfo[]> {
  const res = await api.get('/docker/images')
  return res.data.data
}

export async function pullImage(image: string): Promise<void> {
  await api.post('/docker/images/pull', { image })
}

export async function removeImage(image: string, force = false): Promise<void> {
  await api.delete(`/docker/images/${encodeURIComponent(image)}`, { params: { force } })
}

// --- Container stats ---

export interface ContainerStats {
  cpu_percent: number
  memory_usage: number
  memory_limit: number
  memory_percent: number
}

export async function getContainerStats(id: string): Promise<ContainerStats> {
  const res = await api.get(`/docker/containers/${id}/stats`)
  return res.data.data
}
