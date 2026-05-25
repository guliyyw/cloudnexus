import { useState, useEffect, useCallback } from 'react'
import api from '../services/api'

// SECURITY: 不再从客户端解析 JWT，改为从后端 API 获取权限
// 防止用户篡改 localStorage 中的 token 来伪造管理员权限

interface UserPermissions {
  user_id: number
  username: string
  is_admin: boolean
  roles: string[]
  permissions: string[]
}

export function useAccess() {
  const [perms, setPerms] = useState<UserPermissions | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchPermissions = useCallback(async () => {
    try {
      const res = await api.get('/user/permissions')
      setPerms(res.data.data)
    } catch {
      setPerms(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchPermissions()
  }, [fetchPermissions])

  const hasPermission = (perm: string) => {
    if (!perms) return false
    if (perms.is_admin) return true
    return perms.permissions.includes(perm) || perms.permissions.includes('*')
  }

  const hasAnyPermission = (...permsList: string[]) => {
    return permsList.some((p) => hasPermission(p))
  }

  const hasRole = (role: string) => {
    if (!perms) return false
    if (perms.is_admin) return true
    return perms.roles.includes(role)
  }

  const isAdmin = perms?.is_admin || false

  return { hasPermission, hasAnyPermission, hasRole, isAdmin, claims: perms, loading, refetch: fetchPermissions }
}
