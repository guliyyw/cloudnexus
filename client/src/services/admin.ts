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
