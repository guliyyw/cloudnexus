import { useEffect, useState, useCallback } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Card, Statistic, Row, Col, Typography, Tabs, Descriptions, Spin } from 'antd'
import { UserOutlined, CheckCircleOutlined, StopOutlined, ReloadOutlined, DashboardOutlined, FileTextOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as adminApi from '../services/admin'
import type { AdminUser, SystemMetrics, LogEntry } from '../services/admin'

const { Text } = Typography

function UserManagement() {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
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

  const activeUsers = users.filter((u) => u.status === 1).length
  const adminUsers = users.filter((u) => u.is_admin).length

  const columns: ColumnsType<AdminUser> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 200, render: (v: string) => <Text copyable={{ text: v }}>{v.slice(-8)}</Text> },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 150 },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 200, ellipsis: true },
    {
      title: '角色', dataIndex: 'is_admin', key: 'is_admin', width: 100,
      render: (v: boolean) => v ? <Tag color="red">管理员</Tag> : <Tag>用户</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (v: number) => v === 1 ? <Tag color="green" icon={<CheckCircleOutlined />}>正常</Tag> : <Tag color="red" icon={<StopOutlined />}>禁用</Tag>,
    },
    {
      title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 180,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
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
          <Card><Statistic title="活跃用户" value={activeUsers} valueStyle={{ color: '#3f8600' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="管理员" value={adminUsers} valueStyle={{ color: '#cf1322' }} prefix={<UserOutlined />} /></Card>
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
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showTotal: (t) => `共 ${t} 用户`,
        }}
        size="middle"
      />
    </div>
  )
}

function SystemStatus() {
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null)
  const [loading, setLoading] = useState(false)

  const fetchMetrics = useCallback(async () => {
    setLoading(true)
    try {
      const m = await adminApi.getMetrics()
      setMetrics(m)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchMetrics() }, [fetchMetrics])

  const formatUptime = (seconds: number) => {
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = seconds % 60
    const parts = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    if (m > 0) parts.push(`${m}m`)
    parts.push(`${s}s`)
    return parts.join(' ')
  }

  return (
    <Spin spinning={loading}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 16 }}>系统状态</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchMetrics}>刷新</Button>
      </div>

      {metrics && (
        <>
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}>
              <Card><Statistic title="运行时间" value={formatUptime(metrics.uptime_seconds)} /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="Goroutines" value={metrics.goroutines} /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="堆内存" value={metrics.heap_alloc_mb} suffix="MB" precision={1} /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="GC 次数" value={metrics.num_gc} /></Card>
            </Col>
          </Row>

          <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label="Go 版本">{metrics.go_version}</Descriptions.Item>
            <Descriptions.Item label="CPU 核心">{metrics.num_cpu}</Descriptions.Item>
            <Descriptions.Item label="堆系统内存">{metrics.heap_sys_mb} MB</Descriptions.Item>
            <Descriptions.Item label="栈内存">{metrics.stack_inuse_kb} KB</Descriptions.Item>
          </Descriptions>
        </>
      )}
    </Spin>
  )
}

function LogViewer() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [levelFilter, setLevelFilter] = useState<string>('')

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await adminApi.getLogs(levelFilter || undefined)
      setLogs(res.logs)
    } finally {
      setLoading(false)
    }
  }, [levelFilter])

  useEffect(() => { fetchLogs() }, [fetchLogs])

  const levelColor: Record<string, string> = {
    debug: 'default',
    info: 'blue',
    warn: 'orange',
    error: 'red',
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong style={{ fontSize: 16 }}>服务器日志</Text>
        <Space>
          <Button
            type={levelFilter === '' ? 'primary' : 'default'}
            size="small"
            onClick={() => setLevelFilter('')}
          >全部</Button>
          <Button
            type={levelFilter === 'info' ? 'primary' : 'default'}
            size="small"
            onClick={() => setLevelFilter('info')}
          >INFO</Button>
          <Button
            type={levelFilter === 'warn' ? 'primary' : 'default'}
            size="small"
            onClick={() => setLevelFilter('warn')}
          >WARN</Button>
          <Button
            type={levelFilter === 'error' ? 'primary' : 'default'}
            size="small"
            danger={levelFilter === 'error'}
            onClick={() => setLevelFilter('error')}
          >ERROR</Button>
          <Button icon={<ReloadOutlined />} onClick={fetchLogs}>刷新</Button>
        </Space>
      </div>

      <div style={{ background: '#1e1e1e', color: '#d4d4d4', borderRadius: 8, padding: 16, maxHeight: 600, overflow: 'auto', fontFamily: 'monospace', fontSize: 13 }}>
        {logs.length === 0 && !loading && (
          <div style={{ color: '#888' }}>暂无日志</div>
        )}
        {logs.map((entry, i) => (
          <div key={i} style={{ lineHeight: 1.8, whiteSpace: 'nowrap' }}>
            <span style={{ color: '#569cd6' }}>{new Date(entry.timestamp).toLocaleTimeString()}</span>
            {' '}
            <Tag color={levelColor[entry.level] || 'default'} style={{ fontSize: 11, lineHeight: '16px' }}>{entry.level.toUpperCase()}</Tag>
            {' '}
            <span style={{ color: '#888', fontSize: 12 }}>{entry.caller}</span>
            {' '}
            <span>{entry.message}</span>
          </div>
        ))}
        {loading && <div style={{ color: '#888' }}>加载中...</div>}
      </div>
    </div>
  )
}

export default function AdminPage() {
  const tabItems = [
    { key: 'users', label: <span><UserOutlined />用户管理</span>, children: <UserManagement /> },
    { key: 'status', label: <span><DashboardOutlined />系统状态</span>, children: <SystemStatus /> },
    { key: 'logs', label: <span><FileTextOutlined />服务器日志</span>, children: <LogViewer /> },
  ]

  return <Tabs defaultActiveKey="users" items={tabItems} size="large" />
}
