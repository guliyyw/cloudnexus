import { useEffect, useState } from 'react'
import { Table, Button, Space, message, Popconfirm, Typography, Card, Row, Col, Statistic } from 'antd'
import {
  DeleteOutlined, UndoOutlined, ExclamationCircleOutlined,
  ReloadOutlined, FolderOutlined, FileOutlined, FileImageOutlined,
  PlayCircleOutlined, SoundOutlined, FilePdfOutlined, FileZipOutlined,
  InboxOutlined,
} from '@ant-design/icons'
import * as fileApi from '../services/file'
import type { FileItem } from '../services/file'
import type { ColumnsType } from 'antd/es/table'
import { formatFileSize } from '../utils/format'

const { Text } = Typography

function getFileIcon(mimeType: string, isDir: boolean) {
  if (isDir) return <FolderOutlined style={{ color: '#faad14' }} />
  if (!mimeType) return <FileOutlined />
  if (mimeType.startsWith('image/')) return <FileImageOutlined style={{ color: '#52c41a' }} />
  if (mimeType.startsWith('video/')) return <PlayCircleOutlined style={{ color: '#5b8def' }} />
  if (mimeType.startsWith('audio/')) return <SoundOutlined style={{ color: '#722ed1' }} />
  if (mimeType === 'application/pdf') return <FilePdfOutlined style={{ color: '#ff4d4f' }} />
  if (mimeType.includes('zip') || mimeType.includes('rar')) return <FileZipOutlined />
  return <FileOutlined />
}

export default function RecycleBinPage() {
  const [files, setFiles] = useState<FileItem[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [quota, setQuota] = useState<fileApi.QuotaInfo | null>(null)
  const pageSize = 20

  const fetchTrash = async () => {
    setLoading(true)
    try {
      const res = await fileApi.getTrashList(page, pageSize)
      setFiles(res.items)
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }

  const fetchQuota = async () => {
    try {
      setQuota(await fileApi.getQuota())
    } catch { /* ignore */ }
  }

  useEffect(() => {
    fetchTrash()
    fetchQuota()
  }, [page])

  const handleRestore = async (id: string) => {
    try {
      await fileApi.restoreFromTrash(id)
      message.success('已恢复')
      fetchTrash()
      fetchQuota()
    } catch (e: any) {
      message.error(e.response?.data?.message || '恢复失败')
    }
  }

  const handlePermanentDelete = async (id: string) => {
    try {
      await fileApi.permanentDelete(id)
      message.success('已彻底删除')
      fetchTrash()
      fetchQuota()
    } catch (e: any) {
      message.error(e.response?.data?.message || '删除失败')
    }
  }

  const handleEmptyTrash = async () => {
    try {
      const result = await fileApi.emptyTrash()
      message.success(`已清空回收站，共删除 ${result.deleted} 个文件`)
      setFiles([])
      setTotal(0)
      fetchQuota()
    } catch (e: any) {
      message.error(e.response?.data?.message || '清空失败')
    }
  }

  const columns: ColumnsType<FileItem> = [
    {
      title: '名称', dataIndex: 'name', key: 'name',
      render: (name: string, record: FileItem) => (
        <Space>
          {getFileIcon(record.mime_type, record.is_dir)}
          <Text>{name}</Text>
        </Space>
      ),
    },
    {
      title: '大小', dataIndex: 'size', key: 'size', width: 100,
      render: (v: number) => v > 0 ? formatFileSize(v) : '-',
    },
    {
      title: '删除时间', dataIndex: 'updated_at', key: 'updated_at', width: 170,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '操作', key: 'actions', width: 200,
      render: (_: unknown, record: FileItem) => (
        <Space>
          <Button size="small" icon={<UndoOutlined />} onClick={() => handleRestore(record.id)}>
            恢复
          </Button>
          <Popconfirm
            title="彻底删除后将无法恢复，确认删除？"
            icon={<ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />}
            onConfirm={() => handlePermanentDelete(record.id)}
          >
            <Button size="small" danger icon={<DeleteOutlined />}>
              彻底删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const trashUsed = quota?.trash_used || 0
  const trashLimit = quota?.trash_limit || 1073741824
  const trashPercent = trashLimit > 0 ? Math.round((trashUsed / trashLimit) * 100) : 0

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="回收站占用"
              value={formatFileSize(trashUsed)}
              suffix={`/ ${formatFileSize(trashLimit)}`}
              prefix={<DeleteOutlined />}
            />
            <div style={{ marginTop: 8 }}>
              <div style={{ height: 6, background: 'rgba(255,255,255,0.06)', borderRadius: 3, overflow: 'hidden' }}>
                <div style={{ height: '100%', width: `${Math.min(trashPercent, 100)}%`, background: trashPercent > 80 ? '#ff4d4f' : '#81ecfe', borderRadius: 3, transition: 'width 0.3s' }} />
              </div>
              <Text type="secondary" style={{ fontSize: 11 }}>{trashPercent}% 已用</Text>
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="回收站文件数" value={total} prefix={<InboxOutlined />} />
          </Card>
        </Col>
      </Row>

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong style={{ fontSize: 16 }}><DeleteOutlined /> 回收站</Text>
        <Space>
          <Popconfirm
            title="清空回收站后将无法恢复，确认清空？"
            icon={<ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />}
            onConfirm={handleEmptyTrash}
          >
            <Button danger icon={<DeleteOutlined />} disabled={total === 0}>
              清空回收站
            </Button>
          </Popconfirm>
          <Button icon={<ReloadOutlined />} onClick={() => { fetchTrash(); fetchQuota() }}>
            刷新
          </Button>
        </Space>
      </div>

      <Text type="secondary" style={{ display: 'block', marginBottom: 12, fontSize: 12 }}>
        文件删除后保留 30 天，过期将自动清理
      </Text>

      <Table
        columns={columns}
        dataSource={files}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showTotal: (t) => `共 ${t} 项`,
        }}
        size="middle"
      />
    </div>
  )
}
