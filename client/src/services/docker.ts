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
