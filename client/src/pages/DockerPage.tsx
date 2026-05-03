import { useEffect, useState } from 'react'
import {
  Table, Button, Modal, Input, Space, Tag, message, Popconfirm,
  Typography, Switch,
} from 'antd'
import {
  PlusOutlined, ReloadOutlined, PlayCircleOutlined,
  PauseCircleOutlined, SyncOutlined, DeleteOutlined,
  FileTextOutlined,
} from '@ant-design/icons'
import { useDockerStore } from '../stores/dockerStore'
import * as dockerApi from '../services/docker'
import type { ContainerInfo } from '../services/docker'
import type { ColumnsType } from 'antd/es/table'

const { Text, Paragraph } = Typography

function statusColor(status: string): string {
  if (status.startsWith('Up')) return 'green'
  if (status.startsWith('Exited')) return 'red'
  return 'orange'
}

export default function DockerPage() {
  const { containers, loading, fetchContainers, create, start, stop, restart, remove } = useDockerStore()
  const [showAll, setShowAll] = useState(false)
  const [createVisible, setCreateVisible] = useState(false)
  const [createImage, setCreateImage] = useState('')
  const [createName, setCreateName] = useState('')
  const [logsVisible, setLogsVisible] = useState(false)
  const [logsContent, setLogsContent] = useState('')
  const [logsLoading, setLogsLoading] = useState(false)

  useEffect(() => { fetchContainers(showAll) }, [showAll])

  const handleViewLogs = async (id: string) => {
    setLogsVisible(true)
    setLogsLoading(true)
    try {
      const logs = await dockerApi.getContainerLogs(id, '200')
      setLogsContent(logs)
    } catch {
      setLogsContent('获取日志失败')
    } finally {
      setLogsLoading(false)
    }
  }

  const columns: ColumnsType<ContainerInfo> = [
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
      title: '操作', key: 'actions', width: 260,
      render: (_: any, record: ContainerInfo) => {
        const isRunning = record.status.startsWith('Up')
        return (
          <Space>
            {isRunning ? (
              <Button size="small" icon={<PauseCircleOutlined />} onClick={() => stop(record.id)}>停止</Button>
            ) : (
              <Button size="small" icon={<PlayCircleOutlined />} onClick={() => start(record.id)}>启动</Button>
            )}
            <Button size="small" icon={<SyncOutlined />} onClick={() => restart(record.id)}>重启</Button>
            <Button size="small" icon={<FileTextOutlined />} onClick={() => handleViewLogs(record.id)}>日志</Button>
            <Popconfirm title="确定删除？" onConfirm={() => remove(record.id, true)}>
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Space>
        )
      },
    },
  ]

  return (
    <div>
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
        columns={columns}
        dataSource={containers}
        rowKey="id"
        loading={loading}
        pagination={false}
        size="middle"
      />

      <Modal
        title="创建容器"
        open={createVisible}
        onOk={async () => {
          if (createImage.trim()) {
            try {
              await create(createImage.trim(), createName.trim() || undefined as any)
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
    </div>
  )
}
