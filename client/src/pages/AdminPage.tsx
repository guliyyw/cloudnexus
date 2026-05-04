import { useEffect, useState, useCallback, useRef } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Card, Statistic, Row, Col, Typography, Tabs, Descriptions, Spin, Progress } from 'antd'
import { UserOutlined, CheckCircleOutlined, StopOutlined, ReloadOutlined, DashboardOutlined, FileTextOutlined, CloudServerOutlined, AreaChartOutlined, DownloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import * as adminApi from '../services/admin'
import type { AdminUser, SystemMetrics, LogEntry, ResourceMetrics, MetricSnapshot } from '../services/admin'

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
        scroll={{ y: 'calc(100vh - 480px)' }}
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
  const [resMetrics, setResMetrics] = useState<ResourceMetrics | null>(null)
  const [loading, setLoading] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchMetrics = useCallback(async () => {
    try {
      const [m, rm] = await Promise.all([
        adminApi.getMetrics(),
        adminApi.getResourceMetrics(),
      ])
      setMetrics(m)
      setResMetrics(rm)
    } catch {
      // resource metrics may not be available
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    setLoading(true)
    fetchMetrics()
    intervalRef.current = setInterval(fetchMetrics, 5000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [fetchMetrics])

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

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes < 0) return '—'
    if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB/s`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB/s`
    return `${bytes} B/s`
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

          <Descriptions bordered size="small" column={2} style={{ marginBottom: 24 }}>
            <Descriptions.Item label="Go 版本">{metrics.go_version}</Descriptions.Item>
            <Descriptions.Item label="CPU 核心">{metrics.num_cpu}</Descriptions.Item>
            <Descriptions.Item label="堆系统内存">{metrics.heap_sys_mb} MB</Descriptions.Item>
            <Descriptions.Item label="栈内存">{metrics.stack_inuse_kb} KB</Descriptions.Item>
          </Descriptions>
        </>
      )}

      {resMetrics && (
        <>
          <div style={{ marginBottom: 12 }}>
            <Text strong style={{ fontSize: 14 }}><CloudServerOutlined /> 服务器资源</Text>
          </div>

          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}>
              <Card size="small">
                <Statistic title="CPU 使用率" value={resMetrics.cpu_percent} suffix="%" precision={1} />
                <Progress percent={resMetrics.cpu_percent} size="small" status={resMetrics.cpu_percent > 80 ? 'exception' : 'normal'} showInfo={false} />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title="内存" value={resMetrics.mem_percent} suffix="%" precision={1} />
                <Progress percent={resMetrics.mem_percent} size="small" status={resMetrics.mem_percent > 80 ? 'exception' : 'normal'} showInfo={false} />
                <Text type="secondary" style={{ fontSize: 11 }}>{resMetrics.mem_used_mb} / {resMetrics.mem_total_mb} MB</Text>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title={`磁盘 ${resMetrics.disk_path}`} value={resMetrics.disk_percent} suffix="%" precision={1} />
                <Progress percent={resMetrics.disk_percent} size="small" status={resMetrics.disk_percent > 80 ? 'exception' : 'normal'} showInfo={false} />
                <Text type="secondary" style={{ fontSize: 11 }}>{resMetrics.disk_used_mb} / {resMetrics.disk_total_mb} MB</Text>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title="网络 接收" value={formatBytes(resMetrics.net_bytes_recv)} />
                <Text type="secondary" style={{ fontSize: 11 }}>发送: {formatBytes(resMetrics.net_bytes_sent)}</Text>
              </Card>
            </Col>
          </Row>
        </>
      )}
    </Spin>
  )
}

function HistoricalMetrics() {
  const [snapshots, setSnapshots] = useState<MetricSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const intRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchHistory = useCallback(async () => {
    try {
      const res = await adminApi.getMetricsHistory(60)
      setSnapshots(res.snapshots || [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    setLoading(true)
    fetchHistory()
    intRef.current = setInterval(fetchHistory, 10000)
    return () => { if (intRef.current) clearInterval(intRef.current) }
  }, [fetchHistory])

  const chartData = snapshots.map((s) => ({
    ...s,
    time: new Date(s.timestamp).toLocaleTimeString(),
  }))

  return (
    <Spin spinning={loading}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 16 }}>历史指标 (最近 10 分钟)</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchHistory}>刷新</Button>
      </div>

      <Card title="CPU 使用率 (%)" size="small" style={{ marginBottom: 16 }}>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" fontSize={11} />
            <YAxis domain={[0, 100]} fontSize={11} />
            <Tooltip />
            <Line type="monotone" dataKey="cpu_percent" stroke="#1677ff" dot={false} strokeWidth={2} />
          </LineChart>
        </ResponsiveContainer>
      </Card>

      <Row gutter={16}>
        <Col span={12}>
          <Card title="内存使用率 (%)" size="small">
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" fontSize={11} />
                <YAxis domain={[0, 100]} fontSize={11} />
                <Tooltip />
                <Line type="monotone" dataKey="mem_percent" stroke="#52c41a" dot={false} strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="Goroutines / 堆内存 (MB)" size="small">
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" fontSize={11} />
                <YAxis yAxisId="left" fontSize={11} />
                <YAxis yAxisId="right" orientation="right" fontSize={11} />
                <Tooltip />
                <Legend />
                <Line yAxisId="left" type="monotone" dataKey="goroutines" stroke="#faad14" dot={false} strokeWidth={2} name="Goroutines" />
                <Line yAxisId="right" type="monotone" dataKey="heap_alloc_mb" stroke="#722ed1" dot={false} strokeWidth={2} name="堆内存(MB)" />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </Spin>
  )
}

function LogViewer() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [levelFilter, setLevelFilter] = useState<string>('')
  const intRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await adminApi.getLogs(levelFilter || undefined)
      setLogs(res.logs)
    } finally {
      setLoading(false)
    }
  }, [levelFilter])

  useEffect(() => {
    fetchLogs()
    intRef.current = setInterval(fetchLogs, 3000)
    return () => { if (intRef.current) clearInterval(intRef.current) }
  }, [fetchLogs])

  const levelColor: Record<string, string> = {
    debug: 'default',
    info: 'blue',
    warn: 'orange',
    error: 'red',
  }

  const logDownloadUrl = adminApi.getLogDownloadUrl(new Date().toISOString().slice(0, 10))

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
          <Button icon={<DownloadOutlined />} size="small" href={logDownloadUrl}>下载日志</Button>
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
            {entry.service && <Tag style={{ fontSize: 11, lineHeight: '16px' }}>{entry.service}</Tag>}
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
    { key: 'history', label: <span><AreaChartOutlined />历史指标</span>, children: <HistoricalMetrics /> },
    { key: 'logs', label: <span><FileTextOutlined />服务器日志</span>, children: <LogViewer /> },
  ]

  return <Tabs defaultActiveKey="users" items={tabItems} size="large" />
}
