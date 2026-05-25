import { useEffect, useState, useMemo } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Card, Statistic, Row, Col, Typography, Progress, Select, Modal, Form, InputNumber, Spin } from 'antd'
import { UserOutlined, CheckCircleOutlined, StopOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as adminApi from '../../services/admin'
import type { AdminUser, QuotaTier, UserQuotaInfo } from '../../services/admin'
import { formatFileSize } from '../../utils/format'
import { colors } from '../../theme/tokens'

const { Text } = Typography

export default function UserManagement() {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [quotaModalOpen, setQuotaModalOpen] = useState(false)
  const [quotaUserId, setQuotaUserId] = useState<string>('')
  const [quotaUsername, setQuotaUsername] = useState<string>('')
  const [quotaLimit, setQuotaLimit] = useState<number | null | undefined>()
  const [quotaTierId, setQuotaTierId] = useState<string | undefined>()
  const [quotaSaving, setQuotaSaving] = useState(false)
  const [quotaInfoLoading, setQuotaInfoLoading] = useState(false)
  const [currentQuotaInfo, setCurrentQuotaInfo] = useState<UserQuotaInfo | null>(null)
  const [quotaTiers, setQuotaTiers] = useState<QuotaTier[]>([])
  const tierSizeMap = useMemo(() => {
    const map: Record<string, number> = {}
    quotaTiers.forEach((t) => { map[t.id] = t.storage_limit })
    return map
  }, [quotaTiers])
  const pageSize = 20

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const res = await adminApi.getUsers(page, pageSize)
      setUsers(res.items)
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchUsers() }, [page])

  useEffect(() => {
    adminApi.getQuotaTiers().then((tiers) => {
      setQuotaTiers(tiers)
    }).catch(() => {})
  }, [])

  const handleToggleAdmin = async (id: string) => {
    const updated = await adminApi.toggleAdmin(id)
    setUsers((prev) => prev.map((u) => (u.id === id ? { ...u, is_admin: updated.is_admin } : u)))
    message.success(`已${updated.is_admin ? '设为' : '取消'}管理员`)
  }

  const handleToggleStatus = async (id: string) => {
    const updated = await adminApi.toggleStatus(id)
    setUsers((prev) => prev.map((u) => (u.id === id ? { ...u, status: updated.status } : u)))
    message.success(`已${updated.status === 1 ? '启用' : '禁用'}用户`)
  }

  const handleOpenQuota = async (userId: string, username: string) => {
    setQuotaUserId(userId)
    setQuotaUsername(username)
    setQuotaLimit(undefined)
    setQuotaTierId(undefined)
    setCurrentQuotaInfo(null)
    setQuotaModalOpen(true)
    setQuotaInfoLoading(true)
    try {
      const [info, tiers] = await Promise.all([
        adminApi.getUserQuota(userId),
        adminApi.getQuotaTiers(),
      ])
      setCurrentQuotaInfo(info)
      setQuotaTiers(tiers)
    } catch {
      // modal still usable without preloaded info
    } finally {
      setQuotaInfoLoading(false)
    }
  }

  const handleSetQuota = async () => {
    setQuotaSaving(true)
    try {
      const req: { storage_limit?: number | null; tier_id?: string } = {}
      if (quotaTierId !== undefined) {
        req.tier_id = quotaTierId
      }
      if (quotaLimit !== undefined) {
        req.storage_limit = quotaLimit
      }
      await adminApi.setUserQuota(quotaUserId, req)
      message.success('配额已更新')
      setQuotaModalOpen(false)
      fetchUsers()
    } catch (e: any) {
      message.error(e.response?.data?.message || '设置失败')
    } finally {
      setQuotaSaving(false)
    }
  }

  const activeUsers = users.filter((u) => u.status === 1).length
  const adminUsers = users.filter((u) => u.is_admin).length

  const columns: ColumnsType<AdminUser> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 180, render: (v: string) => <Text copyable={{ text: v }}>{v.slice(-8)}</Text> },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 180, ellipsis: true },
    {
      title: '角色', dataIndex: 'is_admin', key: 'is_admin', width: 80,
      render: (v: boolean) => v ? <Tag color="red">管理员</Tag> : <Tag>用户</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (v: number) => v === 1 ? <Tag color="green" icon={<CheckCircleOutlined />}>正常</Tag> : <Tag color="red" icon={<StopOutlined />}>禁用</Tag>,
    },
    {
      title: '存储用量', key: 'storage', width: 180,
      render: (_: any, record: AdminUser) => {
        const used = record.storage_used || 0
        const customLimit = record.storage_limit
        const tierDefault = (record.tier_id && tierSizeMap[record.tier_id]) ? tierSizeMap[record.tier_id] : 1073741824
        const effectiveLimit = customLimit || tierDefault
        const pct = effectiveLimit > 0 ? Math.round((used / effectiveLimit) * 100) : 0
        return (
          <div>
            <div style={{ fontSize: 12, marginBottom: 2 }}>
              {formatFileSize(used)} / {customLimit ? formatFileSize(customLimit) : (record.tier_name ? `${record.tier_name}(${formatFileSize(effectiveLimit)})` : formatFileSize(effectiveLimit))}
            </div>
            <Progress percent={pct} size="small" status={pct > 80 ? 'exception' : 'normal'} showInfo={false} />
          </div>
        )
      },
    },
    {
      title: '等级', dataIndex: 'tier_name', key: 'tier_name', width: 80,
      render: (v: string, record: AdminUser) => {
        if (record.storage_limit) return <Tag color="orange">自定义</Tag>
        return v ? <Tag color="blue">{v}</Tag> : <Tag color="default">默认</Tag>
      },
    },
    {
      title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 150,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '配额', key: 'quota', width: 100,
      render: (_: any, record: AdminUser) => (
        <Button size="small" onClick={() => handleOpenQuota(record.id, record.username)}>设置配额</Button>
      ),
    },
    {
      title: '操作', key: 'actions', width: 200,
      render: (_: any, record: AdminUser) => (
        <Space>
          <Popconfirm title={record.is_admin ? '取消管理员？' : '设为管理员？'} onConfirm={() => handleToggleAdmin(record.id)}>
            <Button size="small">{record.is_admin ? '取消管理员' : '设为管理员'}</Button>
          </Popconfirm>
          <Popconfirm
            title={record.status === 1 ? '禁用此用户？' : '启用此用户？'}
            onConfirm={() => handleToggleStatus(record.id)}
          >
            <Button size="small" danger={record.status === 1}>{record.status === 1 ? '禁用' : '启用'}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card><Statistic title="用户总数" value={total} prefix={<UserOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="活跃用户" value={activeUsers} valueStyle={{ color: '#52c41a' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="管理员" value={adminUsers} valueStyle={{ color: colors.primary }} prefix={<UserOutlined />} /></Card>
        </Col>
      </Row>

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 16 }}>用户管理</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchUsers}>刷新</Button>
      </div>

      <Table
        columns={columns}
        dataSource={users}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1300, y: 'calc(100vh - 480px)' }}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showTotal: (t) => `共 ${t} 用户`,
        }}
        size="middle"
      />

      <Modal
        title={`设置配额 — ${quotaUsername}`}
        open={quotaModalOpen}
        onOk={handleSetQuota}
        onCancel={() => setQuotaModalOpen(false)}
        confirmLoading={quotaSaving}
        width={480}
      >
        {quotaInfoLoading ? (
          <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
        ) : (
          <div>
            {currentQuotaInfo && (
              <Card size="small" style={{ marginBottom: 16, background: 'rgba(255,255,255,0.04)' }}>
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">当前用量：</Text>
                  <Text strong>{formatFileSize(currentQuotaInfo.used)}</Text>
                  <Text type="secondary"> / {formatFileSize(currentQuotaInfo.limit)}</Text>
                </div>
                <Progress
                  percent={Math.round(currentQuotaInfo.usage_percent)}
                  size="small"
                  status={currentQuotaInfo.usage_percent > 80 ? 'exception' : 'normal'}
                />
                <div style={{ marginTop: 8 }}>
                  <Text type="secondary">当前等级：</Text>
                  <Tag color="blue">{currentQuotaInfo.tier_name}</Tag>
                </div>
              </Card>
            )}

            <Form layout="vertical">
              <Form.Item label="配额等级" help="选择预设等级将清除自定义上限">
                <Select
                  allowClear
                  placeholder="选择等级（留空不修改）"
                  style={{ width: '100%' }}
                  value={quotaTierId}
                  onChange={(v) => {
                    setQuotaTierId(v)
                    if (v) setQuotaLimit(undefined)
                  }}
                  options={quotaTiers.map((t) => ({
                    value: t.id,
                    label: `${t.name} (${formatFileSize(t.storage_limit)})`,
                  }))}
                />
              </Form.Item>

              <Form.Item label="自定义存储上限 (字节)" help="设置后将覆盖等级配额；清空后可恢复等级配额">
                <InputNumber
                  min={0}
                  style={{ width: '100%' }}
                  value={quotaLimit === undefined ? undefined : (quotaLimit ?? undefined)}
                  onChange={(v) => {
                    setQuotaLimit(v === null || v === undefined ? undefined : v)
                    if (v !== undefined && v !== null) setQuotaTierId(undefined)
                  }}
                  placeholder="例如: 1073741824 (1GB)，留空不修改"
                />
              </Form.Item>
              {quotaLimit !== undefined && quotaLimit !== null && (
                <Text type="secondary">{formatFileSize(quotaLimit)}</Text>
              )}
            </Form>
          </div>
        )}
      </Modal>
    </div>
  )
}
