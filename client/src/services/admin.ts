import api from './api'

export interface AdminUser {
  id: string
  username: string
  email: string
  avatar: string
  status: number
  is_admin: boolean
  created_at: string
  storage_used: number
  storage_limit: number | null
  tier_id: string | null
  tier_name: string
}

export interface PermissionInfo {
  id: string
  name: string
  code: string
  description: string
  group_name: string
}

export interface RoleInfo {
  id: string
  name: string
  code: string
  description: string
  is_system: boolean
  permissions: PermissionInfo[]
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
  method: string
  path: string
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

export async function getRoles(): Promise<RoleInfo[]> {
  const res = await api.get('/admin/roles')
  return res.data.data || []
}

export async function getPermissions(): Promise<PermissionInfo[]> {
  const res = await api.get('/admin/roles/permissions')
  return res.data.data || []
}

export async function createRole(req: { name: string; code: string; description?: string }): Promise<RoleInfo> {
  const res = await api.post('/admin/roles', req)
  return res.data.data
}

export async function deleteRole(roleId: string): Promise<void> {
  await api.delete(`/admin/roles/${roleId}`)
}

export async function assignRolePermissions(roleId: string, permissionIds: string[]): Promise<void> {
  await api.post(`/admin/roles/${roleId}/permissions`, { permission_ids: permissionIds })
}

export async function getUserRoles(userId: string): Promise<RoleInfo[]> {
  const res = await api.get(`/admin/users/${userId}/roles`)
  return res.data.data || []
}

export async function assignUserRole(userId: string, roleId: string): Promise<void> {
  await api.post(`/admin/users/${userId}/roles`, { role_id: roleId })
}

export async function removeUserRole(userId: string, roleId: string): Promise<void> {
  await api.delete(`/admin/users/${userId}/roles/${roleId}`)
}

export async function getDirectUserPermissions(userId: string): Promise<PermissionInfo[]> {
  const res = await api.get(`/admin/users/${userId}/permissions`)
  return res.data.data || []
}

export async function replaceDirectUserPermissions(userId: string, permissionIds: string[]): Promise<void> {
  await api.put(`/admin/users/${userId}/permissions`, { permission_ids: permissionIds })
}

export async function getMetrics(): Promise<SystemMetrics> {
  const res = await api.get('/metrics')
  return res.data.data
}

export async function getLogs(params?: { level?: string; requestId?: string; userId?: string; service?: string }): Promise<{ logs: LogEntry[]; total: number }> {
  const res = await api.get('/admin/logs', {
    params: {
      level: params?.level || undefined,
      request_id: params?.requestId || undefined,
      user_id: params?.userId || undefined,
      service: params?.service || undefined,
    },
  })
  return res.data.data
}

export async function getLogServices(): Promise<string[]> {
  const res = await api.get('/admin/logs/services')
  return res.data.data.services
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

// --- 集群节点 ---

export interface DockerNode {
  id: string
  name: string
  host: string
  port: number
  status: string
  node_type: string
  service: string
  first_seen_at: string
  total_online_seconds: number
  offline_since: string
  container_name: string
  version: string
  last_heartbeat: string
  created_at: string
  updated_at: string
}

export interface NodeOnlineSession {
  id: string
  node_name: string
  start_time: string
  end_time: string
  duration: number
  container_name: string
  version: string
}

export interface NodeFilter {
  service?: string
  host?: string
  type?: string
  status?: string
}

export async function getNodes(filter?: NodeFilter): Promise<DockerNode[]> {
  const params: Record<string, string> = {}
  if (filter?.service) params.service = filter.service
  if (filter?.host) params.host = filter.host
  if (filter?.type) params.type = filter.type
  if (filter?.status) params.status = filter.status
  const res = await api.get('/admin/nodes', { params })
  return res.data.data.nodes
}

export async function getNodeSessions(name: string): Promise<NodeOnlineSession[]> {
  const res = await api.get(`/admin/nodes/${name}/sessions`)
  return res.data.data.sessions
}

export async function addNode(req: {
  name: string; host: string; port: number;
  node_type?: string; service?: string;
  tls_cert?: string; tls_key?: string; ca_cert?: string;
}): Promise<DockerNode> {
  const res = await api.post('/admin/nodes', req)
  return res.data.data.node
}

export async function deleteNode(name: string): Promise<void> {
  await api.delete(`/admin/nodes/${name}`)
}

export interface ManagedService {
  key: string
  name: string
  service: string
  compose_name: string
  profile: string
  description: string
  required: boolean
  container_id: string
  state: string
  status_text: string
  created: boolean
  startable: boolean
}

export async function getManagedServices(): Promise<ManagedService[]> {
  const res = await api.get('/admin/services')
  return res.data.data.services || []
}

export async function startManagedService(service: string): Promise<void> {
  await api.post(`/admin/services/${service}/start`)
}

// --- 告警规则 ---

export interface AlertRule {
  id: string
  name: string
  description: string
  enabled: boolean
  node_name: string
  trigger_type: string
  condition: string
  webhook_url: string
  cooldown_seconds: number
  created_by: string
  created_at: string
  updated_at: string
}

export interface AlertHistoryItem {
  id: string
  rule_id: string
  rule_name: string
  node_name: string
  alert_type: string
  status: string
  message: string
  fired_at: string
  resolved_at: string
  webhook_url: string
  response_code: number
  error_message: string
}

export async function getAlertRules(): Promise<AlertRule[]> {
  const res = await api.get('/admin/alerts/rules')
  return res.data.data.rules
}

export async function createAlertRule(req: {
  name: string; description?: string; enabled?: boolean
  node_name?: string; trigger_type?: string; condition?: string
  webhook_url: string; cooldown_seconds?: number
}): Promise<AlertRule> {
  const res = await api.post('/admin/alerts/rules', req)
  return res.data.data.rule
}

export async function updateAlertRule(id: string, req: Record<string, unknown>): Promise<AlertRule> {
  const res = await api.put(`/admin/alerts/rules/${id}`, req)
  return res.data.data.rule
}

export async function deleteAlertRule(id: string): Promise<void> {
  await api.delete(`/admin/alerts/rules/${id}`)
}

export async function getAlertHistory(params?: {
  page?: number; page_size?: number
  rule_id?: string; node_name?: string; alert_type?: string
}): Promise<{ items: AlertHistoryItem[]; total: number; page: number; page_size: number }> {
  const res = await api.get('/admin/alerts/history', { params })
  return res.data.data
}

// ── 配额等级 ──

export interface QuotaTier {
  id: string
  name: string
  storage_limit: number
  description: string
  created_at: string
  updated_at: string
}

export async function getQuotaTiers(): Promise<QuotaTier[]> {
  const res = await api.get('/admin/quota/tiers')
  return res.data.data.tiers
}

export async function createQuotaTier(req: {
  name: string; storage_limit: number; description?: string
}): Promise<QuotaTier> {
  const res = await api.post('/admin/quota/tiers', req)
  return res.data.data
}

export async function updateQuotaTier(id: string, req: {
  name?: string; storage_limit?: number; description?: string
}): Promise<void> {
  await api.put(`/admin/quota/tiers/${id}`, req)
}

export async function deleteQuotaTier(id: string): Promise<void> {
  await api.delete(`/admin/quota/tiers/${id}`)
}

// ── 用户配额 ──

export interface UserQuotaInfo {
  used: number
  limit: number
  tier_name: string
  trash_used: number
  trash_limit: number
  usage_percent: number
}

export async function getUserQuota(userId: string): Promise<UserQuotaInfo> {
  const res = await api.get(`/admin/users/${userId}/quota`)
  return res.data.data
}

export async function setUserQuota(userId: string, req: {
  storage_limit?: number | null; tier_id?: string
}): Promise<void> {
  await api.put(`/admin/users/${userId}/quota`, req)
}

// ── 系统配置 ──

export interface SystemConfig {
  key: string
  value: string
}

export async function getSystemConfig(): Promise<SystemConfig[]> {
  const res = await api.get('/admin/config')
  return res.data.data.configs
}

export async function updateSystemConfig(key: string, value: string): Promise<void> {
  await api.put('/admin/config', { key, value })
}

// ── 管理统计 ──

export interface AdminStats {
  user_count: number
  file_count: number
  storage_used_bytes: number
  album_count: number
  album_file_count: number
  music_track_count: number
  playlist_count: number
}

export async function getAdminStats(): Promise<AdminStats> {
  const res = await api.get('/admin/stats')
  return res.data.data
}
