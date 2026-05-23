import { useEffect } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { Spin } from 'antd'
import { useAuthStore } from '../stores/authStore'
import { useAccess } from '../hooks/useAccess'

export function AuthGuard() {
  const { isLoggedIn, isLoading, checkAuth } = useAuthStore()

  useEffect(() => {
    // 组件挂载时检查登录状态
    checkAuth()
  }, [])

  // 加载中显示 spinner
  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" />
      </div>
    )
  }

  if (!isLoggedIn) return <Navigate to="/login" replace />
  return <Outlet />
}

export function AdminGuard() {
  const { hasRole, loading } = useAccess()
  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}><Spin size="large" /></div>
  }
  if (!hasRole('admin') && !hasRole('super_admin')) return <Navigate to="/forbidden" replace />
  return <Outlet />
}
