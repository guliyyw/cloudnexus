import { useEffect, useMemo, useState } from 'react'
import { Button, Checkbox, Form, Input, Modal, Popconfirm, Select, Tag, Typography, message } from 'antd'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import * as adminApi from '../../services/admin'
import type { PermissionInfo, RoleInfo } from '../../services/admin'
import { colors, spacing } from '../../theme/tokens'

const { Text } = Typography

export default function RoleManagementModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [roles, setRoles] = useState<RoleInfo[]>([])
  const [permissions, setPermissions] = useState<PermissionInfo[]>([])
  const [roleId, setRoleId] = useState<string>()
  const [moduleIds, setModuleIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [form] = Form.useForm()

  const load = async () => {
    const [nextRoles, nextPermissions] = await Promise.all([adminApi.getRoles(), adminApi.getPermissions()])
    setRoles(nextRoles)
    setPermissions(nextPermissions)
    const selected = nextRoles.find((role) => role.id === roleId) || nextRoles[0]
    if (selected) {
      setRoleId(selected.id)
      setModuleIds((selected.permissions || []).filter((permission) => permission.code.startsWith('module:')).map((permission) => permission.id))
    }
  }

  useEffect(() => {
    if (open) load().catch(() => message.error('读取权限组失败'))
  }, [open])

  const modulePermissions = useMemo(
    () => permissions.filter((permission) => permission.code.startsWith('module:')),
    [permissions],
  )
  const selectedRole = roles.find((role) => role.id === roleId)

  const selectRole = (nextRoleId: string) => {
    const role = roles.find((item) => item.id === nextRoleId)
    setRoleId(nextRoleId)
    setModuleIds((role?.permissions || []).filter((permission) => permission.code.startsWith('module:')).map((permission) => permission.id))
  }

  const save = async () => {
    if (!selectedRole) return
    setSaving(true)
    try {
      const nonModuleIds = (selectedRole.permissions || [])
        .filter((permission) => !permission.code.startsWith('module:'))
        .map((permission) => permission.id)
      await adminApi.assignRolePermissions(selectedRole.id, [...nonModuleIds, ...moduleIds])
      message.success('权限组模块已更新，组内用户重新登录后完全生效')
      await load()
    } catch (error: any) {
      message.error(error.response?.data?.message || '保存权限组失败')
    } finally {
      setSaving(false)
    }
  }

  const create = async () => {
    const values = await form.validateFields()
    await adminApi.createRole(values)
    message.success('权限组已创建')
    setCreateOpen(false)
    form.resetFields()
    await load()
  }

  const remove = async () => {
    if (!selectedRole || selectedRole.is_system) return
    try {
      await adminApi.deleteRole(selectedRole.id)
      message.success('权限组已删除')
      setRoleId(undefined)
      await load()
    } catch (error: any) {
      message.error(error.response?.data?.message || '删除权限组失败')
    }
  }

  return (
    <>
      <Modal title="权限组管理" open={open} onCancel={onClose} onOk={save} confirmLoading={saving} width={680}>
        <div style={{ display: 'flex', gap: spacing.sm, marginBottom: spacing.md }}>
          <Select
            value={roleId}
            onChange={selectRole}
            options={roles.map((role) => ({ value: role.id, label: `${role.name} (${role.code})` }))}
            style={{ flex: 1 }}
          />
          {selectedRole?.is_system && <Tag color="blue" style={{ margin: '8px 0' }}>系统内置</Tag>}
          <Button icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>新建</Button>
          <Popconfirm
            title="删除权限组"
            description="删除后，该组会从所有用户的权限配置中移除。"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            disabled={!selectedRole || selectedRole.is_system}
            onConfirm={remove}
          >
            <Button danger icon={<DeleteOutlined />} disabled={!selectedRole || selectedRole.is_system} title={selectedRole?.is_system ? '系统内置权限组不能删除' : '删除权限组'} />
          </Popconfirm>
        </div>
        <Text type="secondary">勾选该权限组可以进入的模块。操作权限保持原有配置不变。</Text>
        <Checkbox.Group value={moduleIds} onChange={(values) => setModuleIds(values as string[])} style={{ width: '100%', marginTop: spacing.md }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: spacing.sm }}>
            {modulePermissions.map((permission) => (
              <div key={permission.id} style={{ padding: 12, border: `1px solid ${colors.borderSubtle}`, borderRadius: 8 }}>
                <Checkbox value={permission.id}>{permission.name}</Checkbox>
              </div>
            ))}
          </div>
        </Checkbox.Group>
      </Modal>
      <Modal title="新建权限组" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={create}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input placeholder="例如：媒体运营" /></Form.Item>
          <Form.Item name="code" label="标识" rules={[{ required: true, pattern: /^[a-z][a-z0-9_]*$/ }]}><Input placeholder="例如：media_operator" /></Form.Item>
          <Form.Item name="description" label="说明"><Input placeholder="说明该权限组适用的人群" /></Form.Item>
        </Form>
      </Modal>
    </>
  )
}
