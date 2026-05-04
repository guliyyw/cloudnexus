import api from './api'

export interface AdminUser {
  id: string
  username: string
  email: string
  avatar: string
  status: number
  is_admin: boolean
  created_at: string
}

export interface AdminUserListResponse {
  items: AdminUser[]
  total: number
  page: number
  page_size: number
}

export interface SystemMetrics {
  uptime_seconds: number
  goroutines: number
  heap_alloc_mb: number
  heap_sys_mb: number
  stack_inuse_kb: number
  num_gc: number
  go_version: string
  num_cpu: number
}

export interface ResourceMetrics {
  cpu_percent: number
  mem_total_mb: number
  mem_used_mb: number
  mem_percent: number
  disk_total_mb: number
  disk_used_mb: number
  disk_percent: number
  disk_path: string
  net_bytes_recv: number
  net_bytes_sent: number
  net_packets_recv: number
  net_packets_sent: number
}

export interface LogEntry {
  timestamp: string
  level: string
  message: string
  caller: string
  service: string
  request_id: string
  user_id: string
  stack: string
}

export async function getUsers(page: number, pageSize: number): Promise<AdminUserListResponse> {
  const res = await api.get('/admin/users', { params: { page, page_size: pageSize } })
  return res.data.data
}

export async function toggleAdmin(id: string): Promise<AdminUser> {
  const res = await api.put(`/admin/users/${id}/toggle-admin`)
  return res.data.data
}

export async function toggleStatus(id: string): Promise<AdminUser> {
  const res = await api.put(`/admin/users/${id}/toggle-status`)
  return res.data.data
}

export async function getMetrics(): Promise<SystemMetrics> {
  const res = await api.get('/metrics')
  return res.data.data
}

export async function getLogs(level?: string): Promise<{ logs: LogEntry[]; total: number }> {
  const res = await api.get('/admin/logs', { params: { level } })
  return res.data.data
}

export async function getResourceMetrics(): Promise<ResourceMetrics> {
  const res = await api.get('/admin/metrics/resources')
  return res.data.data
}

export interface LogFileInfo {
  date: string
  size: number
}

export async function getLogFiles(): Promise<LogFileInfo[]> {
  const res = await api.get('/admin/logs/files')
  return res.data.data.files
}

export function getLogDownloadUrl(date: string): string {
  const token = localStorage.getItem('access_token') || ''
  return `/api/v1/admin/logs/download?date=${date}&token=${token}`
}

export interface MetricSnapshot {
  timestamp: string
  uptime_seconds: number
  goroutines: number
  heap_alloc_mb: number
  cpu_percent: number
  mem_percent: number
}

export interface MetricsHistoryResponse {
  snapshots: MetricSnapshot[]
}

export async function getMetricsHistory(n?: number): Promise<MetricsHistoryResponse> {
  const res = await api.get('/admin/metrics/history', { params: { n } })
  return res.data.data
}
