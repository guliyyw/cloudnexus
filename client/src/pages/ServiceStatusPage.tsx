import { useEffect, useState, useCallback, useMemo } from 'react'
import { Typography, Row, Col, Switch, Select, Spin, Tabs } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import ModuleCard from '../components/dashboard/ModuleCard'
import StatusTimeline from '../components/status/StatusTimeline'
import ResourceChart from '../components/status/ResourceChart'
import { useDashboardStore } from '../stores/dashboardStore'
import { useStatusStore } from '../stores/statusStore'
import { colors } from '../theme/tokens'
import {
  CloudOutlined,
  MessageOutlined,
  ContainerOutlined,
  VideoCameraOutlined,
  FileTextOutlined,
  ClusterOutlined,
} from '@ant-design/icons'

const { Title, Text } = Typography

const iconMap: Record<string, React.ReactNode> = {
  files: <CloudOutlined />,
  im: <MessageOutlined />,
  docker: <ContainerOutlined />,
  camera: <VideoCameraOutlined />,
  collab: <FileTextOutlined />,
  infra: <ClusterOutlined />,
}

const RANGE_OPTIONS = [
  { value: '1h', label: '最近1小时' },
  { value: '6h', label: '最近6小时' },
  { value: '24h', label: '最近24小时' },
  { value: '7d', label: '最近7天' },
]

export default function ServiceStatusPage() {
  const { modules, summary, fetchStatus } = useDashboardStore()
  const { snapshots, resources, historyLoading, resourceLoading, fetchHistory, fetchResources } = useStatusStore()
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [range, setRange] = useState('24h')
  const [serviceFilter, setServiceFilter] = useState('all')

  const loadAll = useCallback(() => {
    fetchStatus()
    fetchHistory(range)
    fetchResources(range, serviceFilter)
  }, [fetchStatus, fetchHistory, fetchResources, range, serviceFilter])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    if (!autoRefresh) return
    const timer = setInterval(loadAll, 30000)
    return () => clearInterval(timer)
  }, [autoRefresh, loadAll])

  const serviceNames = useMemo(() => {
    const names = Object.keys(resources)
    if (names.length === 0) return []
    return names
  }, [resources])

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <Title level={4} style={{ margin: 0 }}>系统状态</Title>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Select
            size="small"
            value={range}
            onChange={setRange}
            options={RANGE_OPTIONS}
            style={{ width: 130 }}
          />
          <Text type="secondary" style={{ fontSize: 12 }}>自动刷新</Text>
          <Switch size="small" checked={autoRefresh} onChange={setAutoRefresh} />
          <ReloadOutlined
            style={{ cursor: 'pointer', color: colors.primary, fontSize: 16 }}
            onClick={loadAll}
          />
        </div>
      </div>

      <Tabs
        defaultActiveKey="status"
        items={[
          {
            key: 'status',
            label: '服务状态',
            children: (
              <div>
                {summary && (
                  <div style={{ marginBottom: 16, color: '#8c8c8c', fontSize: 12 }}>
                    共 {summary.total} 个模块，{summary.healthy} 个正常
                    {summary.warning > 0 && `，${summary.warning} 个警告`}
                    {summary.error > 0 && `，${summary.error} 个异常`}
                  </div>
                )}
                <Row gutter={[16, 16]}>
                  {modules.map((mod) => (
                    <Col xs={24} sm={12} lg={8} key={mod.key}>
                      <ModuleCard
                        icon={iconMap[mod.key]}
                        name={mod.name}
                        status={mod.status}
                        detail={mod.detail}
                        onClick={() => {}}
                      />
                    </Col>
                  ))}
                </Row>
              </div>
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
            key: 'resources',
            label: '资源使用',
            children: (
              <div>
                {serviceNames.length > 1 && (
                  <div style={{ marginBottom: 16 }}>
                    <Select
                      size="small"
                      value={serviceFilter}
                      onChange={setServiceFilter}
                      options={[
                        { value: 'all', label: '全部服务' },
                        ...serviceNames.map((n) => ({ value: n, label: n })),
                      ]}
                      style={{ width: 180 }}
                    />
                  </div>
                )}
                {resourceLoading ? (
                  <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
                ) : (
                  <ResourceChart services={resources} serviceFilter={serviceFilter} />
                )}
              </div>
            ),
          },
        ]}
      />
    </div>
  )
}
