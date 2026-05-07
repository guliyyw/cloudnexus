import api from './api'

export interface ContainerInfo {
  id: string
  name: string
  image: string
  status: string
  ports: string[]
  created: string
}

export interface EndpointInfo {
  name: string
  host: string
  port: number
  status: string
  tls: boolean
}

export async function listEndpoints(): Promise<EndpointInfo[]> {
  const res = await api.get('/docker/endpoints')
  return res.data.data
}

export async function listContainers(all: boolean = false, endpoint: string = 'local'): Promise<ContainerInfo[]> {
  const res = await api.get('/docker/containers', { params: { all, endpoint } })
  return res.data.data
}

export async function createContainer(image: string, name: string, endpoint: string = 'local'): Promise<{ id: string }> {
  const res = await api.post('/docker/containers', { image, name }, { params: { endpoint } })
  return res.data.data
}

export async function startContainer(id: string, endpoint: string = 'local'): Promise<void> {
  await api.post(`/docker/containers/${id}/start`, null, { params: { endpoint } })
}

export async function stopContainer(id: string, endpoint: string = 'local'): Promise<void> {
  await api.post(`/docker/containers/${id}/stop`, null, { params: { endpoint } })
}

export async function restartContainer(id: string, endpoint: string = 'local'): Promise<void> {
  await api.post(`/docker/containers/${id}/restart`, null, { params: { endpoint } })
}

export async function removeContainer(id: string, force = false, endpoint: string = 'local'): Promise<void> {
  await api.delete(`/docker/containers/${id}`, { params: { force, endpoint } })
}

export async function getContainerLogs(id: string, tail = '100', endpoint: string = 'local'): Promise<string> {
  const res = await api.get(`/docker/containers/${id}/logs`, { params: { tail, endpoint }, responseType: 'text' })
  return res.data
}

// --- Image management ---

export interface ImageInfo {
  id: string
  tags: string[]
  size: number
  created: string
}

export async function listImages(endpoint: string = 'local'): Promise<ImageInfo[]> {
  const res = await api.get('/docker/images', { params: { endpoint } })
  return res.data.data
}

export async function pullImage(image: string, endpoint: string = 'local'): Promise<void> {
  await api.post('/docker/images/pull', { image }, { params: { endpoint } })
}

export async function removeImage(image: string, force = false, endpoint: string = 'local'): Promise<void> {
  await api.delete(`/docker/images/${encodeURIComponent(image)}`, { params: { force, endpoint } })
}

// --- Container stats ---

export interface ContainerStats {
  cpu_percent: number
  memory_usage: number
  memory_limit: number
  memory_percent: number
}

export async function getContainerStats(id: string, endpoint: string = 'local'): Promise<ContainerStats> {
  const res = await api.get(`/docker/containers/${id}/stats`, { params: { endpoint } })
  return res.data.data
}
