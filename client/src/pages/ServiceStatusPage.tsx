import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { Button, Card, Col, Row, Select, Space, Spin, Switch, Table, Tabs, Tag, Typography, message } from 'antd'
import {
  CloudOutlined,
  ClusterOutlined,
  ContainerOutlined,
  FileTextOutlined,
  MessageOutlined,
  PlaySquareOutlined,
  ReloadOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons'
import ModuleCard from '../components/dashboard/ModuleCard'
import StatusTimeline from '../components/status/StatusTimeline'
import ResourceChart from '../components/status/ResourceChart'
import SystemStatus from '../components/admin/SystemStatus'
import HistoricalMetrics from '../components/admin/HistoricalMetrics'
import { useDashboardStore } from '../stores/dashboardStore'
import { useStatusStore } from '../stores/statusStore'
import { getManagedServices, startManagedService, type ManagedService } from '../services/admin'
import { colors, radius, spacing } from '../theme/tokens'

const { Title, Text } = Typography

const iconMap: Record<string, ReactNode> = {
  files: <CloudOutlined />,
  im: <MessageOutlined />,
  docker: <ContainerOutlined />,
  camera: <VideoCameraOutlined />,
  cameras: <VideoCameraOutlined />,
  collab: <FileTextOutlined />,
  drama: <PlaySquareOutlined />,
  infra: <ClusterOutlined />,
}

const RANGE_OPTIONS = [
  { value: '1h', label: '最近 1 小时' },
  { value: '6h', label: '最近 6 小时' },
  { value: '24h', label: '最近 24 小时' },
  { value: '7d', label: '最近 7 天' },
]

export default function ServiceStatusPage() {
  const { modules, summary, fetchStatus } = useDashboardStore()
  const { snapshots, resources, historyLoading, resourceLoading, fetchHistory, fetchResources } = useStatusStore()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [range, setRange] = useState('1h')
  const [serviceFilter, setServiceFilter] = useState('all')
  const [services, setServices] = useState<ManagedService[]>([])
  const [servicesLoading, setServicesLoading] = useState(false)
  const [startingService, setStartingService] = useState<string | null>(null)

  const loadServices = useCallback(async () => {
    setServicesLoading(true)
    try {
      setServices(await getManagedServices())
    } catch {
      message.error('服务管理信息加载失败')
    } finally {
      setServicesLoading(false)
    }
  }, [])

  const loadAll = useCallback(() => {
    fetchStatus()
    fetchHistory(range)
    fetchResources(range, 'all')
    loadServices()
  }, [fetchStatus, fetchHistory, fetchResources, range, loadServices])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    if (!autoRefresh) return
    const timer = setInterval(loadAll, 15000)
    return () => clearInterval(timer)
  }, [autoRefresh, loadAll])

  const serviceNames = useMemo(() => Object.keys(resources).sort(), [resources])

  const handleStartService = useCallback(async (service: string) => {
    setStartingService(service)
    try {
      await startManagedService(service)
      message.success('已请求启动服务')
      await Promise.all([fetchStatus(), loadServices()])
    } catch (err: any) {
      message.error(err?.response?.data?.message || '启动服务失败')
    } finally {
      setStartingService(null)
    }
  }, [fetchStatus, loadServices])

  const summaryItems = useMemo(() => {
    if (!summary) return []
    return [
      { label: '全部服务', value: summary.total, color: colors.text },
      { label: '正常', value: summary.healthy, color: colors.statusGreen },
      { label: '告警', value: summary.warning, color: colors.statusYellow },
      { label: '异常', value: summary.error, color: colors.statusRed },
    ]
  }, [summary])

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md, marginBottom: spacing.md, flexWrap: 'wrap' }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>系统状态</Title>
          <Text type="secondary">查看服务健康、历史变化和实时资源占用。</Text>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: spacing.sm, flexWrap: 'wrap' }}>
          <Select size="small" value={range} onChange={setRange} options={RANGE_OPTIONS} style={{ width: 132 }} />
          <Text type="secondary" style={{ fontSize: 12 }}>自动刷新</Text>
          <Switch size="small" checked={autoRefresh} onChange={setAutoRefresh} />
          <ReloadOutlined
            spin={resourceLoading}
            title="刷新数据"
            style={{ cursor: resourceLoading ? 'default' : 'pointer', color: colors.primary, fontSize: 16 }}
            onClick={resourceLoading ? undefined : loadAll}
          />
        </div>
      </div>

      {summaryItems.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: spacing.sm, marginBottom: spacing.md }}>
          {summaryItems.map((item) => (
            <Card key={item.label} size="small" style={{ borderRadius: radius.md, borderColor: colors.borderSubtle }}>
              <Text type="secondary" style={{ fontSize: 12 }}>{item.label}</Text>
              <div style={{ color: item.color, fontSize: 26, fontWeight: 750, lineHeight: 1.4 }}>{item.value}</div>
            </Card>
          ))}
        </div>
      )}

      <Tabs
        defaultActiveKey="resources"
        items={[
          {
            key: 'resources',
            label: '资源使用',
            children: (
              <div>
                {serviceNames.length > 1 && (
                  <div style={{ marginBottom: spacing.md }}>
                    <Select
                      size="small"
                      value={serviceFilter}
                      onChange={setServiceFilter}
                      options={[
                        { value: 'all', label: '全部服务' },
                        ...serviceNames.map((name) => ({ value: name, label: name })),
                      ]}
                      style={{ width: 180 }}
                    />
                  </div>
                )}
                {resourceLoading && serviceNames.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
                ) : (
                  <ResourceChart services={resources} serviceFilter={serviceFilter} />
                )}
              </div>
            ),
          },
          {
            key: 'status',
            label: '服务总览',
            children: (
              <Row gutter={[16, 16]}>
                {modules.map((mod) => (
                  <Col xs={24} sm={12} lg={8} key={mod.key}>
                    <ModuleCard
                      icon={iconMap[mod.key]}
                      name={mod.name}
                      status={mod.status}
                      detail={mod.detail}
                      metric={mod.status === 'green' ? '运行正常' : '需要关注'}
                    />
                  </Col>
                ))}
              </Row>
            ),
          },
          {
            key: 'manage',
            label: '服务管理',
            children: (
              <Table
                rowKey="service"
                size="small"
                loading={servicesLoading}
                dataSource={services}
                pagination={false}
                scroll={{ x: 900 }}
                columns={[
                  {
                    title: '服务',
                    dataIndex: 'name',
                    render: (_: string, record: ManagedService) => (
                      <Space direction="vertical" size={0}>
                        <Text strong>{record.name}</Text>
                        <Text type="secondary" style={{ fontSize: 12 }}>{record.service}</Text>
                        <Text type="secondary" style={{ fontSize: 12, maxWidth: 520 }}>{record.description}</Text>
                      </Space>
                    ),
                  },
                  {
                    title: '启动组',
                    dataIndex: 'profile',
                    render: (profile: string, record: ManagedService) => record.required ? <Tag>minimal</Tag> : <Tag color="blue">{profile}</Tag>,
                  },
                  {
                    title: '容器',
                    dataIndex: 'created',
                    render: (created: boolean) => created ? <Tag color="green">已创建</Tag> : <Tag color="default">未创建</Tag>,
                  },
                  {
                    title: '状态',
                    dataIndex: 'state',
                    render: (state: string, record: ManagedService) => {
                      if (!record.created) return <Tag color="default">未创建</Tag>
                      if (state === 'running') return <Tag color="green">运行中</Tag>
                      if (state === 'exited') return <Tag color="red">已停止</Tag>
                      return <Tag color="orange">{state || '未知'}</Tag>
                    },
                  },
                  {
                    title: '操作',
                    key: 'actions',
                    align: 'right',
                    render: (_: unknown, record: ManagedService) => (
                      <Button
                        size="small"
                        type="primary"
                        disabled={record.required || !record.startable}
                        loading={startingService === record.service}
                        onClick={() => handleStartService(record.service)}
                      >
                        启动
                      </Button>
                    ),
                  },
                ]}
              />
            ),
          },
          {
            key: 'history',
            label: '健康历史',
            children: historyLoading ? (
              <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
            ) : (
              <StatusTimeline snapshots={snapshots} />
            ),
          },
          {
            key: 'runtime',
            label: '主服务指标',
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.lg }}>
                <SystemStatus />
                <HistoricalMetrics />
              </div>
            ),
          },
        ]}
      />
    </div>
  )
}
