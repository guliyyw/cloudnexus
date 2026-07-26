import { useEffect, useMemo, useState } from 'react'
import { Alert, Checkbox, Divider, Modal, Spin, Switch, Tag, Typography, message } from 'antd'
import * as adminApi from '../../services/admin'
import type { PermissionInfo, RoleInfo } from '../../services/admin'
import { colors, spacing } from '../../theme/tokens'

const { Text } = Typography

interface Props {
  userId: string
  username: string
  isAdmin: boolean
  open: boolean
  onClose: () => void
  onSaved: () => void
}

export default function UserAccessModal({ userId, username, isAdmin, open, onClose, onSaved }: Props) {
  const [roles, setRoles] = useState<RoleInfo[]>([])
  const [permissions, setPermissions] = useState<PermissionInfo[]>([])
  const [initialRoleIds, setInitialRoleIds] = useState<string[]>([])
  const [roleIds, setRoleIds] = useState<string[]>([])
  const [directPermissions, setDirectPermissions] = useState<PermissionInfo[]>([])
  const [directModuleIds, setDirectModuleIds] = useState<string[]>([])
  const [adminEnabled, setAdminEnabled] = useState(isAdmin)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open) return
    setAdminEnabled(isAdmin)
    setLoading(true)
    Promise.all([
      adminApi.getRoles(),
      adminApi.getPermissions(),
      adminApi.getUserRoles(userId),
      adminApi.getDirectUserPermissions(userId),
    ]).then(([allRoles, allPermissions, userRoles, direct]) => {
      const assigned = userRoles.map((role) => role.id)
      setRoles(allRoles)
      setPermissions(allPermissions)
      setInitialRoleIds(assigned)
      setRoleIds(assigned)
      setDirectPermissions(direct)
      setDirectModuleIds(direct.filter((permission) => permission.code.startsWith('module:')).map((permission) => permission.id))
    }).catch(() => message.error('读取用户权限失败')).finally(() => setLoading(false))
  }, [open, userId, isAdmin])

  const modulePermissions = useMemo(
    () => permissions.filter((permission) => permission.code.startsWith('module:')),
    [permissions],
  )

  const inheritedCodes = useMemo(() => {
    const selected = new Set(roleIds)
    return new Set(
      roles
        .filter((role) => selected.has(role.id))
        .flatMap((role) => role.permissions || [])
        .map((permission) => permission.code),
    )
  }, [roleIds, roles])

  const handleSave = async () => {
    setSaving(true)
    try {
      const initial = new Set(initialRoleIds)
      const next = new Set(roleIds)
      const additions = roleIds.filter((id) => !initial.has(id))
      const removals = initialRoleIds.filter((id) => !next.has(id))
      await Promise.all([
        ...additions.map((roleId) => adminApi.assignUserRole(userId, roleId)),
        ...removals.map((roleId) => adminApi.removeUserRole(userId, roleId)),
      ])

      const nonModuleIds = directPermissions
        .filter((permission) => !permission.code.startsWith('module:'))
        .map((permission) => permission.id)
      await adminApi.replaceDirectUserPermissions(userId, [...nonModuleIds, ...directModuleIds])
      if (adminEnabled !== isAdmin) {
        await adminApi.toggleAdmin(userId)
      }
      onSaved()
      message.success('访问权限已更新，用户重新登录后完全生效')
      onClose()
    } catch (error: any) {
      message.error(error.response?.data?.message || '保存访问权限失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={`访问权限 - ${username}`} open={open} onCancel={onClose} onOk={handleSave} confirmLoading={saving} width={680}>
      <Spin spinning={loading}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md, padding: 12, marginBottom: spacing.md, border: `1px solid ${colors.borderSubtle}`, borderRadius: 8 }}>
          <div>
            <Text strong>管理员</Text>
            <div><Text type="secondary" style={{ fontSize: 12 }}>允许进入管理后台并管理用户、权限和系统配置。</Text></div>
          </div>
          <Switch checked={adminEnabled} onChange={setAdminEnabled} />
        </div>
        <Alert
          type="info"
          showIcon
          message="权限组提供基础权限，用户专属授权用于额外开放模块。管理员始终拥有全部权限。"
          style={{ marginBottom: spacing.md }}
        />

        <Text strong>权限组</Text>
        <Checkbox.Group value={roleIds} onChange={(values) => setRoleIds(values as string[])} style={{ width: '100%', marginTop: spacing.sm }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: spacing.sm }}>
            {roles.map((role) => (
              <div key={role.id} style={{ padding: 12, border: `1px solid ${colors.borderSubtle}`, borderRadius: 8 }}>
                <Checkbox value={role.id}>{role.name}</Checkbox>
                <div style={{ marginTop: 4 }}><Text type="secondary" style={{ fontSize: 12 }}>{role.description || role.code}</Text></div>
              </div>
            ))}
          </div>
        </Checkbox.Group>

        <Divider />
        <Text strong>用户专属模块</Text>
        <div style={{ marginTop: 4, marginBottom: spacing.sm }}>
          <Text type="secondary">已由权限组开放的模块会标记为“组内已有”，无需重复选择。</Text>
        </div>
        <Checkbox.Group value={directModuleIds} onChange={(values) => setDirectModuleIds(values as string[])} style={{ width: '100%' }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: spacing.sm }}>
            {modulePermissions.map((permission) => {
              const inherited = inheritedCodes.has(permission.code)
              return (
                <div key={permission.id} style={{ padding: 12, border: `1px solid ${colors.borderSubtle}`, borderRadius: 8 }}>
                  <Checkbox value={permission.id} disabled={inherited}>{permission.name}</Checkbox>
                  {inherited && <Tag color="green" style={{ marginLeft: 8 }}>组内已有</Tag>}
                </div>
              )
            })}
          </div>
        </Checkbox.Group>
      </Spin>
    </Modal>
  )
}
