import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'
import { useAccess } from '../hooks/useAccess'

export function AuthGuard() {
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  if (!isLoggedIn) return <Navigate to="/login" replace />
  return <Outlet />
}

export function AdminGuard() {
  const { hasRole } = useAccess()
  if (!hasRole('admin') && !hasRole('super_admin')) return <Navigate to="/forbidden" replace />
  return <Outlet />
}
