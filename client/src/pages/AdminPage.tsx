import { useEffect, useState, useCallback, useRef } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Card, Statistic, Row, Col, Typography, Tabs, Descriptions, Spin, Progress, Input, Select, Modal, Form, InputNumber } from 'antd'
import { UserOutlined, CheckCircleOutlined, StopOutlined, ReloadOutlined, DashboardOutlined, FileTextOutlined, CloudServerOutlined, AreaChartOutlined, DownloadOutlined, ClusterOutlined, PlusOutlined, DeleteOutlined, ClockCircleOutlined, HistoryOutlined } from '@ant-design/icons'
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
          <Card><Statistic title="活跃用户" value={activeUsers} valueStyle={{ color: '#52c41a' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="管理员" value={adminUsers} valueStyle={{ color: '#e8964a' }} prefix={<UserOutlined />} /></Card>
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
            <Line type="monotone" dataKey="cpu_percent" stroke="#e8964a" dot={false} strokeWidth={2} />
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
                <Line type="monotone" dataKey="mem_percent" stroke="#d4a06a" dot={false} strokeWidth={2} />
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
                <Line yAxisId="left" type="monotone" dataKey="goroutines" stroke="#c49a5c" dot={false} strokeWidth={2} name="Goroutines" />
                <Line yAxisId="right" type="monotone" dataKey="heap_alloc_mb" stroke="#8c6a4a" dot={false} strokeWidth={2} name="堆内存(MB)" />
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
  const [requestIdFilter, setRequestIdFilter] = useState<string>('')
  const [requestIdInput, setRequestIdInput] = useState<string>('')
  const [userIdFilter, setUserIdFilter] = useState<string>('')
  const [serviceFilter, setServiceFilter] = useState<string>('')
  const [services, setServices] = useState<string[]>([])
  const [expandedStacks, setExpandedStacks] = useState<Set<number>>(new Set())
  const intRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await adminApi.getLogs({
        level: levelFilter || undefined,
        requestId: requestIdFilter || undefined,
        userId: userIdFilter || undefined,
        service: serviceFilter || undefined,
      })
      setLogs(res.logs)
    } finally {
      setLoading(false)
    }
  }, [levelFilter, requestIdFilter, userIdFilter, serviceFilter])

  useEffect(() => {
    adminApi.getLogServices().then(setServices).catch(() => {})
  }, [])

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

  const methodColor: Record<string, string> = {
    GET: 'green',
    POST: 'blue',
    PUT: 'orange',
    DELETE: 'red',
    PATCH: 'purple',
  }

  const logDownloadUrl = adminApi.getLogDownloadUrl(new Date().toISOString().slice(0, 10))

  const handleRequestIdSearch = () => {
    setRequestIdFilter(requestIdInput.trim())
  }

  const handleClickRequestId = (rid: string) => {
    setRequestIdFilter(rid)
    setRequestIdInput(rid)
  }

  const handleClearRequestId = () => {
    setRequestIdFilter('')
    setRequestIdInput('')
  }

  const handleClickUserId = (uid: string) => {
    setUserIdFilter(uid)
  }

  const handleClearUserId = () => {
    setUserIdFilter('')
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8 }}>
        <Text strong style={{ fontSize: 16 }}>服务器日志</Text>
        <Space wrap>
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
          {services.length > 0 && (
            <Select
              size="small"
              placeholder="服务"
              value={serviceFilter || undefined}
              onChange={(v) => setServiceFilter(v || '')}
              allowClear
              style={{ width: 130 }}
              options={[{ label: '当前(实时)', value: '' }, ...services.map((s) => ({ label: s, value: s }))]}
            />
          )}
          <Input
            size="small"
            placeholder="Request ID"
            value={requestIdInput}
            onChange={(e) => setRequestIdInput(e.target.value)}
            onPressEnter={handleRequestIdSearch}
            style={{ width: 110 }}
            allowClear
            onClear={handleClearRequestId}
          />
          {requestIdFilter && (
            <Tag color="geekblue" closable onClose={handleClearRequestId}>
              请求: {requestIdFilter.slice(0, 8)}
            </Tag>
          )}
          {userIdFilter && (
            <Tag color="purple" closable onClose={handleClearUserId}>
              uid: {userIdFilter.slice(-8)}
            </Tag>
          )}
          <Button icon={<ReloadOutlined />} onClick={fetchLogs}>刷新</Button>
          <Button icon={<DownloadOutlined />} size="small" href={logDownloadUrl}>下载日志</Button>
        </Space>
      </div>

      <div style={{ background: '#1e1e1e', color: '#d4d4d4', borderRadius: 8, padding: 16, maxHeight: 600, overflow: 'auto', fontFamily: 'monospace', fontSize: 13 }}>
        {logs.length === 0 && !loading && (
          <div style={{ color: '#888' }}>暂无日志</div>
        )}
        {logs.map((entry, i) => {
          const hasStack = !!entry.stack
          const isExpanded = expandedStacks.has(i)
          return (
            <div key={i}>
              <div style={{ lineHeight: 1.8, whiteSpace: 'nowrap', cursor: hasStack ? 'pointer' : 'default' }}
                   onClick={() => {
                     if (hasStack) {
                       setExpandedStacks(prev => {
                         const next = new Set(prev)
                         if (next.has(i)) next.delete(i)
                         else next.add(i)
                         return next
                       })
                     }
                   }}>
                <span style={{ color: '#569cd6' }}>{new Date(entry.timestamp).toLocaleTimeString()}</span>
                {' '}
                <Tag color={levelColor[entry.level] || 'default'} style={{ fontSize: 11, lineHeight: '16px' }}>{entry.level.toUpperCase()}</Tag>
                {' '}
                {entry.service && <Tag style={{ fontSize: 11, lineHeight: '16px' }}>{entry.service}</Tag>}
                {' '}
                {entry.method && <Tag color={methodColor[entry.method] || 'default'} style={{ fontSize: 10, lineHeight: '15px' }}>{entry.method}</Tag>}
                {' '}
                {entry.path && <span style={{ color: '#ce9178', fontSize: 12 }} title={entry.path}>{entry.path.length > 40 ? entry.path.slice(0, 40) + '...' : entry.path}</span>}
                {' '}
                {entry.request_id && (
                  <Tag
                    color="geekblue"
                    style={{ fontSize: 10, lineHeight: '15px', cursor: 'pointer', maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis' }}
                    title={`点击追踪请求 ${entry.request_id}`}
                    onClick={(e) => { e.stopPropagation(); handleClickRequestId(entry.request_id) }}
                  >{entry.request_id.slice(0, 8)}</Tag>
                )}
                {' '}
                {entry.user_id && (
                  <Tag
                    color="purple"
                    style={{ fontSize: 10, lineHeight: '15px', cursor: 'pointer' }}
                    title="点击按用户过滤"
                    onClick={(e) => { e.stopPropagation(); handleClickUserId(entry.user_id) }}
                  >uid:{entry.user_id}</Tag>
                )}
                {' '}
                <span style={{ color: '#888', fontSize: 12 }}>{entry.caller}</span>
                {' '}
                <span>{entry.message}</span>
                {hasStack && <span style={{ color: '#f14c4c', marginLeft: 4, fontSize: 11 }}>{isExpanded ? '[-]' : '[+]'}</span>}
              </div>
              {hasStack && isExpanded && (
                <pre style={{ background: '#2d2d2d', color: '#ce9178', margin: '4px 0 4px 20px', padding: 8, borderRadius: 4, fontSize: 11, whiteSpace: 'pre-wrap', wordBreak: 'break-all', maxHeight: 200, overflow: 'auto' }}>
                  {entry.stack}
                </pre>
              )}
            </div>
          )
        })}
        {loading && <div style={{ color: '#888' }}>加载中...</div>}
      </div>
    </div>
  )
}

function ClusterNodes() {
  const [nodes, setNodes] = useState<adminApi.DockerNode[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [addLoading, setAddLoading] = useState(false)
  const [sessionsCache, setSessionsCache] = useState<Record<string, adminApi.NodeOnlineSession[]>>({})
  const [filterService, setFilterService] = useState<string>('')
  const [filterHost, setFilterHost] = useState<string>('')
  const [filterType, setFilterType] = useState<string>('')
  const [form] = Form.useForm()
  const intRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchNodes = useCallback(async () => {
    setLoading(true)
    try {
      const filter: adminApi.NodeFilter = {}
      if (filterService) filter.service = filterService
      if (filterHost) filter.host = filterHost
      if (filterType) filter.type = filterType
      const list = await adminApi.getNodes(filter)
      setNodes(list)
      // Auto-fetch sessions for online nodes so "本次上线" shows correctly
      const online = list.filter((n) => n.status === 'healthy' || n.status === 'unresponsive')
      const sessionsPromises = online.map(async (n) => {
        try {
          const s = await adminApi.getNodeSessions(n.name)
          return { name: n.name, sessions: s }
        } catch { return { name: n.name, sessions: [] } }
      })
      const results = await Promise.all(sessionsPromises)
      setSessionsCache((prev) => {
        const next = { ...prev }
        for (const r of results) { next[r.name] = r.sessions }
        return next
      })
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [filterService, filterHost, filterType])

  useEffect(() => {
    fetchNodes()
    intRef.current = setInterval(fetchNodes, 5000)
    return () => { if (intRef.current) clearInterval(intRef.current) }
  }, [fetchNodes])

  const handleAdd = async () => {
    try {
      const values = await form.validateFields()
      setAddLoading(true)
      await adminApi.addNode(values)
      message.success('节点添加成功')
      setModalOpen(false)
      form.resetFields()
      fetchNodes()
    } catch {
      // validation error or API error
    } finally {
      setAddLoading(false)
    }
  }

  const handleDelete = async (name: string) => {
    await adminApi.deleteNode(name)
    message.success('节点已删除')
    setSessionsCache((prev) => { const n = { ...prev }; delete n[name]; return n })
    fetchNodes()
  }

  const handleExpandRow = async (name: string) => {
    if (sessionsCache[name]) return
    try {
      const sessions = await adminApi.getNodeSessions(name)
      setSessionsCache((prev) => ({ ...prev, [name]: sessions }))
    } catch {
      // ignore
    }
  }

  const formatDuration = (seconds: number) => {
    if (!seconds || seconds <= 0) return '—'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    if (h > 0) return `${h}h ${m}m`
    if (m > 0) return `${m}m`
    return `${seconds}s`
  }

  const serviceColor: Record<string, string> = {
    'user-file-svc': 'blue',
    'im-svc': 'green',
    'docker-svc': 'orange',
    'postgres': 'purple',
    'redis': 'red',
    'minio': 'cyan',
  }

  const onlineNodes = nodes.filter((n) => n.status === 'healthy' || n.status === 'unresponsive')
  const offlineNodes = nodes.filter((n) => n.status === 'offline')
  const services = [...new Set(nodes.map((n) => n.service).filter(Boolean))]
  const hosts = [...new Set(nodes.map((n) => n.host).filter(Boolean))]
  const nodeTypes = [...new Set(nodes.map((n) => n.node_type).filter(Boolean))]

  const typeColor: Record<string, string> = {
    service: 'blue',
    infrastructure: 'purple',
  }

  const onlineColumns: ColumnsType<adminApi.DockerNode> = [
    { title: '节点名称', dataIndex: 'name', key: 'name', width: 150, ellipsis: true },
    {
      title: '类型', dataIndex: 'node_type', key: 'type', width: 90,
      render: (v: string) => {
        const label = v === 'infrastructure' ? '基础设施' : '服务'
        return <Tag color={typeColor[v] || 'default'}>{label}</Tag>
      },
    },
    {
      title: '服务', dataIndex: 'service', key: 'service', width: 130,
      render: (v: string) => v ? <Tag color={serviceColor[v] || 'default'}>{v}</Tag> : <Tag>未知</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 90,
      render: (v: string) => {
        if (v === 'unresponsive') return <Tag color="orange" icon={<StopOutlined />}>未响应</Tag>
        return <Tag color="green" icon={<CheckCircleOutlined />}>在线</Tag>
      },
    },
    {
      title: '累计在线', dataIndex: 'total_online_seconds', key: 'total', width: 120,
      render: (v: number) => <span><ClockCircleOutlined style={{ marginRight: 4 }} />{formatDuration(v)}</span>,
    },
    {
      title: '本次上线', dataIndex: 'last_heartbeat', key: 'session', width: 170,
      render: (_v: string, record: adminApi.DockerNode) => {
        const sessions = sessionsCache[record.name]
        const openSession = sessions?.find((s) => !s.end_time)
        if (openSession) return new Date(openSession.start_time).toLocaleString()
        return <span style={{ color: '#888' }}>—</span>
      },
    },
    {
      title: '主机:端口', key: 'addr', width: 170,
      render: (_: any, r: adminApi.DockerNode) => <Text type="secondary">{r.host}:{r.port}</Text>,
    },
  ]

  const offlineColumns: ColumnsType<adminApi.DockerNode> = [
    { title: '节点名称', dataIndex: 'name', key: 'name', width: 150, ellipsis: true },
    {
      title: '类型', dataIndex: 'node_type', key: 'type', width: 90,
      render: (v: string) => {
        const label = v === 'infrastructure' ? '基础设施' : '服务'
        return <Tag color={typeColor[v] || 'default'}>{label}</Tag>
      },
    },
    {
      title: '服务', dataIndex: 'service', key: 'service', width: 130,
      render: (v: string) => v ? <Tag color={serviceColor[v] || 'default'}>{v}</Tag> : <Tag>未知</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 90,
      render: () => <Tag color="red" icon={<StopOutlined />}>离线</Tag>,
    },
    {
      title: '累计在线', dataIndex: 'total_online_seconds', key: 'total', width: 120,
      render: (v: number) => <span><ClockCircleOutlined style={{ marginRight: 4 }} />{formatDuration(v)}</span>,
    },
    {
      title: '下线时间', dataIndex: 'offline_since', key: 'offline', width: 170,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '首次注册', dataIndex: 'first_seen_at', key: 'first', width: 170,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '操作', key: 'actions', width: 60,
      render: (_: any, record: adminApi.DockerNode) => (
        <Popconfirm title="确定删除此节点及历史？" onConfirm={() => handleDelete(record.name)}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card><Statistic title="在线节点" value={onlineNodes.length} valueStyle={{ color: '#52c41a' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="历史节点" value={offlineNodes.length} valueStyle={{ color: '#999' }} prefix={<HistoryOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="未响应" value={nodes.filter((n) => n.status === 'unresponsive').length} valueStyle={{ color: '#fa8c16' }} prefix={<StopOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="节点总计" value={nodes.length} prefix={<CloudServerOutlined />} /></Card>
        </Col>
      </Row>

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong style={{ fontSize: 16 }}>集群节点</Text>
        <Space>
          <Select
            allowClear
            placeholder="服务"
            style={{ width: 140 }}
            value={filterService || undefined}
            onChange={(v) => setFilterService(v || '')}
            options={services.map((s) => ({ label: s, value: s }))}
          />
          <Select
            allowClear
            placeholder="主机"
            style={{ width: 140 }}
            value={filterHost || undefined}
            onChange={(v) => setFilterHost(v || '')}
            options={hosts.map((h) => ({ label: h, value: h }))}
          />
          <Select
            allowClear
            placeholder="类型"
            style={{ width: 120 }}
            value={filterType || undefined}
            onChange={(v) => setFilterType(v || '')}
            options={nodeTypes.filter(t => t === 'service' || t === 'infrastructure').map((t) => ({ label: t === 'infrastructure' ? '基础设施' : '服务', value: t }))}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchNodes}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>添加节点</Button>
        </Space>
      </div>

      {/* 当前在线 / 未响应节点 */}
      <div style={{ marginBottom: 8 }}>
        <Text type="secondary" style={{ fontSize: 13 }}><CheckCircleOutlined style={{ color: '#52c41a', marginRight: 4 }} />当前在线 / 未响应</Text>
      </div>
      <Table
        columns={onlineColumns}
        dataSource={onlineNodes}
        rowKey="id"
        loading={loading}
        pagination={false}
        size="middle"
        style={{ marginBottom: 24 }}
        locale={{ emptyText: '暂无在线节点' }}
      />

      {/* 历史节点（可展开在线时间表） */}
      {offlineNodes.length > 0 && (
        <>
          <div style={{ marginBottom: 8 }}>
            <Text type="secondary" style={{ fontSize: 13 }}><HistoryOutlined style={{ color: '#999', marginRight: 4 }} />历史节点</Text>
          </div>
          <Table
            columns={offlineColumns}
            dataSource={offlineNodes}
            rowKey="id"
            pagination={false}
            size="middle"
            expandable={{
              expandedRowRender: (record) => {
                const sessions = sessionsCache[record.name]
                if (!sessions) return <Spin size="small" />
                if (sessions.length === 0) return <Text type="secondary">暂无在线记录</Text>
                return (
                  <div style={{ maxHeight: 200, overflow: 'auto' }}>
                    <table style={{ width: '100%', fontSize: 12, borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>上线时间</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>下线时间</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>持续时长</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>容器名称</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>版本</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sessions.map((s) => (
                          <tr key={s.id} style={{ borderBottom: '1px solid #fafafa' }}>
                            <td style={{ padding: '4px 12px' }}>{new Date(s.start_time).toLocaleString()}</td>
                            <td style={{ padding: '4px 12px' }}>{s.end_time ? new Date(s.end_time).toLocaleString() : <Tag color="green" style={{ fontSize: 11 }}>进行中</Tag>}</td>
                            <td style={{ padding: '4px 12px' }}>{formatDuration(s.duration)}</td>
                            <td style={{ padding: '4px 12px', fontFamily: 'monospace', fontSize: 11 }}>{s.container_name || '-'}</td>
                            <td style={{ padding: '4px 12px' }}>{s.version ? <Tag style={{ fontSize: 11 }}>{s.version}</Tag> : '-'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )
              },
              rowExpandable: () => true,
              expandIcon: ({ expanded, onExpand, record }) => (
                <Button
                  type="text"
                  size="small"
                  icon={expanded ? <StopOutlined /> : <ClockCircleOutlined />}
                  onClick={(e) => { onExpand(record, e); handleExpandRow(record.name) }}
                />
              ),
            }}
          />
        </>
      )}

      <Modal
        title="添加节点"
        open={modalOpen}
        onOk={handleAdd}
        onCancel={() => { setModalOpen(false); form.resetFields() }}
        confirmLoading={addLoading}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="节点名称" rules={[{ required: true, message: '请输入节点名称' }]}>
            <Input placeholder="例如: node-1" />
          </Form.Item>
          <Form.Item name="host" label="主机地址" rules={[{ required: true, message: '请输入主机地址' }]}>
            <Input placeholder="例如: 192.168.1.10" />
          </Form.Item>
          <Form.Item name="port" label="端口" initialValue={2376}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default function AdminPage() {
  const tabItems = [
    { key: 'users', label: <span><UserOutlined />用户管理</span>, children: <UserManagement /> },
    { key: 'status', label: <span><DashboardOutlined />系统状态</span>, children: <SystemStatus /> },
    { key: 'nodes', label: <span><ClusterOutlined />集群节点</span>, children: <ClusterNodes /> },
    { key: 'history', label: <span><AreaChartOutlined />历史指标</span>, children: <HistoricalMetrics /> },
    { key: 'logs', label: <span><FileTextOutlined />服务器日志</span>, children: <LogViewer /> },
  ]

  return <Tabs defaultActiveKey="users" items={tabItems} size="large" />
}
