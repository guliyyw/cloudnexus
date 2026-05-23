import { useEffect, useState, useCallback, useRef } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Card, Statistic, Row, Col, Typography, Spin, Input, Select, Modal, Form, InputNumber } from 'antd'
import { CheckCircleOutlined, StopOutlined, ReloadOutlined, CloudServerOutlined, PlusOutlined, DeleteOutlined, ClockCircleOutlined, HistoryOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as adminApi from '../../services/admin'

const { Text } = Typography

export default function ClusterNodes() {
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
    try {
      await adminApi.deleteNode(name)
      message.success('节点已删除')
      setSessionsCache((prev) => { const n = { ...prev }; delete n[name]; return n })
      fetchNodes()
    } catch (err: any) {
      message.error(err.response?.data?.message || '删除失败')
    }
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
    {
      title: '操作', key: 'actions', width: 60,
      render: (_: any, record: adminApi.DockerNode) => (
        <Popconfirm title="确定删除此节点及历史？" onConfirm={() => handleDelete(record.name)}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
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
          <Card><Statistic title="历史节点" value={offlineNodes.length} valueStyle={{ color: '#888' }} prefix={<HistoryOutlined />} /></Card>
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
                        <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>上线时间</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>下线时间</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>持续时长</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>容器名称</th>
                          <th style={{ padding: '4px 12px', textAlign: 'left' }}>版本</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sessions.map((s) => (
                          <tr key={s.id} style={{ borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
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
          <Form.Item name="node_type" label="节点类型" initialValue="docker_endpoint">
            <Select options={[
              { value: 'docker_endpoint', label: 'Docker 端点' },
              { value: 'service', label: '服务节点' },
              { value: 'infrastructure', label: '基础设施' },
            ]} />
          </Form.Item>
          <Form.Item name="service" label="服务名" tooltip="docker 端点默认使用 docker，留空自动填充">
            <Input placeholder="例如: docker" />
          </Form.Item>
          <Form.Item name="tls_cert" label="TLS 客户端证书 (PEM)">
            <Input.TextArea rows={3} placeholder="-----BEGIN CERTIFICATE-----" />
          </Form.Item>
          <Form.Item name="tls_key" label="TLS 客户端密钥 (PEM)">
            <Input.TextArea rows={3} placeholder="-----BEGIN PRIVATE KEY-----" />
          </Form.Item>
          <Form.Item name="ca_cert" label="CA 证书 (PEM)">
            <Input.TextArea rows={3} placeholder="-----BEGIN CERTIFICATE-----" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
