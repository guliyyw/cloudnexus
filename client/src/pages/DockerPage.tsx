import { useEffect, useState } from 'react'
import {
  Table, Button, Modal, Input, Space, Tag, message, Popconfirm,
  Typography, Switch, Tabs, Progress, Select,
} from 'antd'
import {
  PlusOutlined, ReloadOutlined, PlayCircleOutlined,
  PauseCircleOutlined, SyncOutlined, DeleteOutlined,
  FileTextOutlined, CloudDownloadOutlined, BlockOutlined,
  EnvironmentOutlined,
} from '@ant-design/icons'
import { useDockerStore } from '../stores/dockerStore'
import * as dockerApi from '../services/docker'
import type { ContainerInfo, ImageInfo } from '../services/docker'
import type { ColumnsType } from 'antd/es/table'

const { Text, Paragraph } = Typography

function statusColor(status: string): string {
  if (status.startsWith('Up')) return 'green'
  if (status.startsWith('Exited')) return 'red'
  return 'orange'
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

export default function DockerPage() {
  const {
    endpoint, endpoints, setEndpoint,
    containers, images, stats, loading, imagesLoading,
    fetchEndpoints, fetchContainers, create, start, stop, restart, remove,
    fetchImages, pullImage, removeImage, fetchStats,
  } = useDockerStore()

  const [showAll, setShowAll] = useState(false)
  const [createVisible, setCreateVisible] = useState(false)
  const [createImage, setCreateImage] = useState('')
  const [createName, setCreateName] = useState('')
  const [logsVisible, setLogsVisible] = useState(false)
  const [logsContent, setLogsContent] = useState('')
  const [logsLoading, setLogsLoading] = useState(false)
  const [pullVisible, setPullVisible] = useState(false)
  const [pullImageName, setPullImageName] = useState('')
  const [pullLoading, setPullLoading] = useState(false)
  const [expandedStats, setExpandedStats] = useState<Set<string>>(new Set())

  useEffect(() => { fetchEndpoints() }, [fetchEndpoints])
  useEffect(() => { fetchContainers(showAll) }, [showAll, endpoint, fetchContainers])

  const handleViewLogs = async (id: string) => {
    setLogsVisible(true)
    setLogsLoading(true)
    try {
      const logs = await dockerApi.getContainerLogs(id, '200', endpoint)
      setLogsContent(logs)
    } catch {
      setLogsContent('获取日志失败')
    } finally {
      setLogsLoading(false)
    }
  }

  const toggleStats = (id: string) => {
    const next = new Set(expandedStats)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
      fetchStats(id)
    }
    setExpandedStats(next)
  }

  const containerColumns: ColumnsType<ContainerInfo> = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 200,
      render: (v: string) => v || <Text type="secondary">(未命名)</Text>,
    },
    { title: '镜像', dataIndex: 'image', key: 'image', width: 200 },
    { title: '容器ID', dataIndex: 'id', key: 'id', width: 120,
      render: (v: string) => <Text code>{v}</Text>,
    },
    { title: '状态', dataIndex: 'status', key: 'status', width: 180,
      render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    { title: '创建时间', dataIndex: 'created', key: 'created', width: 180,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作', key: 'actions', width: 320,
      render: (_: any, record: ContainerInfo) => {
        const isRunning = record.status.startsWith('Up')
        return (
          <Space wrap size={4}>
            {isRunning ? (
              <Button size="small" icon={<PauseCircleOutlined />} onClick={() => stop(record.id)}>停止</Button>
            ) : (
              <Button size="small" icon={<PlayCircleOutlined />} onClick={() => start(record.id)}>启动</Button>
            )}
            <Button size="small" icon={<SyncOutlined />} onClick={() => restart(record.id)}>重启</Button>
            <Button size="small" icon={<FileTextOutlined />} onClick={() => handleViewLogs(record.id)}>日志</Button>
            {isRunning && (
              <Button size="small" icon={<BlockOutlined />}
                onClick={() => toggleStats(record.id)}
                type={expandedStats.has(record.id) ? 'primary' : 'default'}>
                监控
              </Button>
            )}
            <Popconfirm title="确定删除？" onConfirm={() => remove(record.id, true)}>
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Space>
        )
      },
    },
  ]

  const imageColumns: ColumnsType<ImageInfo> = [
    { title: '标签', dataIndex: 'tags', key: 'tags', width: 280,
      render: (tags: string[]) => tags.map((t) => <Tag key={t} color="blue">{t}</Tag>),
    },
    { title: '镜像ID', dataIndex: 'id', key: 'id', width: 140,
      render: (v: string) => <Text code>{v}</Text>,
    },
    { title: '大小', dataIndex: 'size', key: 'size', width: 100,
      render: (v: number) => formatBytes(v),
    },
    { title: '创建时间', dataIndex: 'created', key: 'created', width: 180,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作', key: 'actions', width: 100,
      render: (_: any, record: ImageInfo) => (
        <Popconfirm title="确定删除该镜像？" onConfirm={() => removeImage(record.tags[0] || record.id, false)}>
          <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
        <Space>
          <EnvironmentOutlined />
          <Select
            value={endpoint}
            onChange={(v) => {
              setEndpoint(v)
              fetchContainers(showAll)
            }}
            style={{ minWidth: 240 }}
            options={endpoints.map(ep => ({
              value: ep.name,
              label: (
                <span>
                  {ep.name} <Text type="secondary" style={{ fontSize: 12 }}>
                    ({ep.host}{ep.port ? ':' + ep.port : ''})
                  </Text>
                  <Tag style={{ marginLeft: 4, fontSize: 10 }}
                    color={ep.status === 'healthy' ? 'green' : ep.status === 'unresponsive' ? 'orange' : 'red'}>
                    {ep.status}
                  </Tag>
                </span>
              ),
            }))}
          />
          <Button size="small" icon={<ReloadOutlined />}
            onClick={fetchEndpoints} title="刷新端点列表" />
        </Space>
      </div>

      <Tabs
        defaultActiveKey="containers"
        onChange={(key) => {
          if (key === 'images') fetchImages()
        }}
        items={[
          {
            key: 'containers',
            label: '容器',
            children: (
              <>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Space>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>创建容器</Button>
                    <Button icon={<ReloadOutlined />} onClick={() => fetchContainers(showAll)} loading={loading}>刷新</Button>
                  </Space>
                  <Space>
                    <Text>显示全部</Text>
                    <Switch checked={showAll} onChange={(v) => setShowAll(v)} />
                  </Space>
                </div>

                <Table
                  columns={containerColumns}
                  dataSource={containers}
                  rowKey="id"
                  loading={loading}
                  pagination={false}
                  size="middle"
                  expandable={{
                    expandedRowRender: (record) => {
                      const s = stats[record.id]
                      if (!s) return <Text type="secondary">点击"监控"按钮加载资源使用情况</Text>
                      return (
                        <Space direction="vertical" style={{ width: '100%', padding: '8px 0' }}>
                          <Space size="large">
                            <div>
                              <Text strong style={{ fontSize: 12 }}>CPU</Text>
                              <Progress
                                type="circle"
                                percent={Math.min(s.cpu_percent, 100)}
                                size={60}
                                strokeColor="#81ecfe"
                                format={(p) => `${p?.toFixed(1)}%`}
                              />
                            </div>
                            <div>
                              <Text strong style={{ fontSize: 12 }}>内存</Text>
                              <Progress
                                type="circle"
                                percent={Math.min(s.memory_percent, 100)}
                                size={60}
                                strokeColor="#5ac8d8"
                                format={(p) => `${p?.toFixed(1)}%`}
                              />
                            </div>
                            <div style={{ fontSize: 12, color: '#888' }}>
                              <div>内存使用: {formatBytes(s.memory_usage)}</div>
                              <div>内存限制: {s.memory_limit > 0 ? formatBytes(s.memory_limit) : '无限制'}</div>
                            </div>
                          </Space>
                        </Space>
                      )
                    },
                    expandedRowKeys: [...expandedStats].filter((id) => stats[id]),
                    onExpand: (expanded, record) => {
                      if (expanded) {
                        fetchStats(record.id)
                        setExpandedStats((prev) => new Set(prev).add(record.id))
                      } else {
                        setExpandedStats((prev) => {
                          const next = new Set(prev)
                          next.delete(record.id)
                          return next
                        })
                      }
                    },
                    expandIcon: () => null,
                  }}
                />
              </>
            ),
          },
          {
            key: 'images',
            label: '镜像',
            children: (
              <>
                <div style={{ marginBottom: 16 }}>
                  <Space>
                    <Button type="primary" icon={<CloudDownloadOutlined />}
                      onClick={() => setPullVisible(true)}>拉取镜像</Button>
                    <Button icon={<ReloadOutlined />} onClick={fetchImages} loading={imagesLoading}>刷新</Button>
                  </Space>
                </div>

                <Table
                  columns={imageColumns}
                  dataSource={images}
                  rowKey="id"
                  loading={imagesLoading}
                  pagination={false}
                  size="middle"
                />
              </>
            ),
          },
        ]}
      />

      {/* Create Container Modal */}
      <Modal
        title="创建容器"
        open={createVisible}
        onOk={async () => {
          if (createImage.trim()) {
            try {
              await create(createImage.trim(), createName.trim())
              setCreateImage('')
              setCreateName('')
              setCreateVisible(false)
              message.success('容器已创建并启动')
            } catch (err: any) {
              message.error(err.response?.data?.message || '创建失败')
            }
          }
        }}
        onCancel={() => setCreateVisible(false)}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input placeholder="镜像名称 (如 nginx:alpine)" value={createImage}
            onChange={(e) => setCreateImage(e.target.value)} />
          <Input placeholder="容器名称 (可选)" value={createName}
            onChange={(e) => setCreateName(e.target.value)} />
        </div>
      </Modal>

      {/* Logs Modal */}
      <Modal
        title="容器日志"
        open={logsVisible}
        onCancel={() => setLogsVisible(false)}
        footer={null}
        width={700}
      >
        <Paragraph style={{ maxHeight: 400, overflow: 'auto', background: '#1e1e1e', color: '#d4d4d4', padding: 12, borderRadius: 4, fontFamily: 'monospace', fontSize: 12, whiteSpace: 'pre-wrap' }}>
          {logsLoading ? '加载中...' : logsContent || '(无日志)'}
        </Paragraph>
      </Modal>

      {/* Pull Image Modal */}
      <Modal
        title="拉取镜像"
        open={pullVisible}
        onOk={async () => {
          if (pullImageName.trim()) {
            setPullLoading(true)
            try {
              await pullImage(pullImageName.trim())
              setPullImageName('')
              setPullVisible(false)
              message.success(`镜像 ${pullImageName} 拉取完成`)
            } catch (err: any) {
              message.error(err.response?.data?.message || '拉取失败')
            } finally {
              setPullLoading(false)
            }
          }
        }}
        onCancel={() => setPullVisible(false)}
        confirmLoading={pullLoading}
      >
        <Input placeholder="镜像名称 (如 nginx:alpine)" value={pullImageName}
          onChange={(e) => setPullImageName(e.target.value)} />
      </Modal>
    </div>
  )
}
