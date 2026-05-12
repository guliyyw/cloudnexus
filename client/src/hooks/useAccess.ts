import { useMemo } from 'react'

interface JWTClaims {
  user_id: number
  username: string
  is_admin: boolean
  roles: string[]
  permissions: string[]
  jti: string
}

function parseJWT(): JWTClaims | null {
  const token = localStorage.getItem('access_token')
  if (!token) return null
  try {
    const payload = token.split('.')[1]
    const decoded = JSON.parse(atob(payload))
    return {
      user_id: decoded.user_id,
      username: decoded.username,
      is_admin: decoded.is_admin || false,
      roles: decoded.roles || [],
      permissions: decoded.permissions || [],
      jti: decoded.jti,
    }
  } catch {
    return null
  }
}

export function useAccess() {
  const claims = useMemo(() => parseJWT(), [])

  const hasPermission = (perm: string) => {
    if (!claims) return false
    if (claims.is_admin) return true
    return claims.permissions.includes(perm) || claims.permissions.includes('*')
  }

  const hasAnyPermission = (...perms: string[]) => {
    return perms.some((p) => hasPermission(p))
  }

  const hasRole = (role: string) => {
    if (!claims) return false
    if (claims.is_admin) return true
    return claims.roles.includes(role)
  }

  const isAdmin = claims?.is_admin || false

  return { hasPermission, hasAnyPermission, hasRole, isAdmin, claims }
}
