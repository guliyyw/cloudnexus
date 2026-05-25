import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Table, Button, Modal, Input, message, Popconfirm, Space } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, FileTextOutlined } from '@ant-design/icons'
import { listDocuments, createDocument, deleteDocument, CollabDocument } from '../services/collab'
import dayjs from 'dayjs'

export default function DocumentListPage() {
  const [docs, setDocs] = useState<CollabDocument[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [creating, setCreating] = useState(false)
  const navigate = useNavigate()
  const pageSize = 20

  const fetchDocs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listDocuments(page, pageSize)
      setDocs(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch {
      message.error('获取文档列表失败')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => { fetchDocs() }, [fetchDocs])

  const handleCreate = async () => {
    if (!newTitle.trim()) return
    setCreating(true)
    try {
      const doc = await createDocument(newTitle.trim())
      message.success('创建成功')
      setCreateOpen(false)
      setNewTitle('')
      navigate(`/documents/${doc.id}`)
    } catch {
      message.error('创建失败')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteDocument(id)
      message.success('已删除')
      fetchDocs()
    } catch {
      message.error('删除失败')
    }
  }

  const columns = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (t: string, r: CollabDocument) => (
        <a onClick={() => navigate(`/documents/${r.id}`)} style={{ fontWeight: 500 }}>
          <FileTextOutlined style={{ marginRight: 8, color: '#81ecfe' }} />
          {t}
        </a>
      ),
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      width: 80,
      render: (v: number) => `v${v}`,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_: unknown, r: CollabDocument) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />}
            onClick={() => navigate(`/documents/${r.id}`)}>
            编辑
          </Button>
          <Popconfirm title="确认删除此文档？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>在线文档</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新建文档
        </Button>
      </div>

      <Table
        dataSource={docs}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: setPage,
          showTotal: (t) => `共 ${t} 篇`,
        }}
        locale={{ emptyText: '暂无文档，点击"新建文档"开始' }}
      />

      <Modal
        title="新建文档"
        open={createOpen}
        onOk={handleCreate}
        onCancel={() => { setCreateOpen(false); setNewTitle('') }}
        confirmLoading={creating}
        okText="创建"
        cancelText="取消"
      >
        <Input
          placeholder="输入文档标题"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          onPressEnter={handleCreate}
          autoFocus
        />
      </Modal>
    </div>
  )
}
