import { ReactNode } from 'react'
import { useAccess } from '../hooks/useAccess'

interface Props {
  permission?: string
  permissions?: string[]
  role?: string
  children: ReactNode
  fallback?: ReactNode
}

export default function AccessControl({ permission, permissions, role, children, fallback = null }: Props) {
  const { hasPermission, hasAnyPermission, hasRole } = useAccess()

  if (permission && !hasPermission(permission)) return <>{fallback}</>
  if (permissions && !hasAnyPermission(...permissions)) return <>{fallback}</>
  if (role && !hasRole(role)) return <>{fallback}</>

  return <>{children}</>
}
