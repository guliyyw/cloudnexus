import api from './api'

export interface HealthSnapshot {
  timestamp: string
  data: string // JSON string of module status array
}

export interface ResourcePoint {
  timestamp: string
  cpu_percent: number
  memory_used: number
  memory_total: number
}

export interface ResourceHistoryResponse {
  services: Record<string, ResourcePoint[]>
}

export async function getHealthHistory(range = '24h'): Promise<{ snapshots: HealthSnapshot[] }> {
  const res = await api.get('/admin/status/history', { params: { range } })
  return res.data.data
}

export async function getResourceHistory(range = '24h', service = 'all'): Promise<ResourceHistoryResponse> {
  const res = await api.get('/admin/status/resources', { params: { range, service } })
  return res.data.data
}
