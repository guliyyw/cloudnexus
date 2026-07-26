import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Button,
  Empty,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Typography,
  message,
} from 'antd'
import {
  ArrowRightOutlined,
  DeleteOutlined,
  EditOutlined,
  FileExcelOutlined,
  FileTextOutlined,
  FileWordOutlined,
  PlusOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import { createDocument, deleteDocument, listDocuments, type CollabDocument } from '../services/collab'
import { createOfficeDoc } from '../services/file'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Paragraph, Text, Title } = Typography

function getDocIcon(title: string) {
  const name = title.toLowerCase()
  if (name.endsWith('.doc') || name.endsWith('.docx')) return <FileWordOutlined />
  if (name.endsWith('.xls') || name.endsWith('.xlsx') || name.endsWith('.csv')) return <FileExcelOutlined />
  return <FileTextOutlined />
}

export default function DocumentListPage() {
  const [docs, setDocs] = useState<CollabDocument[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [searchValue, setSearchValue] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newDocKind, setNewDocKind] = useState<'collab' | 'word' | 'excel'>('collab')
  const [creating, setCreating] = useState(false)
  const navigate = useNavigate()
  const pageSize = 20

  const fetchDocs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listDocuments(page, pageSize, searchValue.trim())
      setDocs(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch {
      message.error('获取文档列表失败')
    } finally {
      setLoading(false)
    }
  }, [page, searchValue])

  useEffect(() => {
    fetchDocs()
  }, [fetchDocs])

  const recentDocs = docs.slice(0, 3)

  const handleSearchChange = (value: string) => {
    setSearchValue(value)
    setPage(1)
  }

  const handleCreate = async () => {
    if (!newTitle.trim()) return
    setCreating(true)
    try {
      const doc = newDocKind === 'collab'
        ? await createDocument(newTitle.trim())
        : await createOfficeDoc(newTitle.trim(), '0', newDocKind)
      message.success('在线文档已创建')
      setCreateOpen(false)
      setNewTitle('')
      setNewDocKind('collab')
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
      render: (title: string, record: CollabDocument) => (
        <button
          type="button"
          onClick={() => navigate(`/documents/${record.id}`)}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: 0,
            border: 'none',
            background: 'transparent',
            color: colors.text,
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          <span
            style={{
              width: 34,
              height: 34,
              borderRadius: radius.md,
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: colors.primaryLight,
              color: colors.primary,
              flexShrink: 0,
            }}
          >
            {getDocIcon(title)}
          </span>
          <span>{title}</span>
        </button>
      ),
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      width: 110,
      render: (version: number) => <Text style={{ color: colors.textSecondary }}>v{version}</Text>,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (updatedAt: string) => <Text style={{ color: colors.textSecondary }}>{dayjs(updatedAt).format('YYYY-MM-DD HH:mm')}</Text>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (_: unknown, record: CollabDocument) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => navigate(`/documents/${record.id}`)}>
            继续编辑
          </Button>
          <Popconfirm title="确认删除此文档？" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.lg }}>
      <section
        style={{
          borderRadius: radius.lg,
          padding: '28px 28px 24px',
          background: colors.surfaceRaised,
          border: `1px solid ${colors.borderSubtle}`,
          boxShadow: shadow.card,
        }}
      >
        <Text style={{ color: colors.primary, fontWeight: 700, letterSpacing: 0.5 }}>DOCUMENTS</Text>
        <Title level={3} style={{ margin: '10px 0 12px', color: colors.text }}>文档中心</Title>
        <Paragraph style={{ marginBottom: 0, color: colors.textSecondary, fontSize: 15, lineHeight: 1.8 }}>
          集中管理协作文档、Markdown、Word、Excel 和 PDF。搜索会在全部文档中筛选，不再只筛当前页。
        </Paragraph>
      </section>

      <section
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(0, 1.7fr) minmax(280px, 0.9fr)',
          gap: spacing.lg,
        }}
      >
        <div
          style={{
            borderRadius: radius.lg,
            padding: 24,
            background: colors.surface,
            border: `1px solid ${colors.borderSubtle}`,
            boxShadow: shadow.card,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
            <div>
              <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>主操作</Text>
              <Title level={4} style={{ margin: '6px 0 0', color: colors.text }}>快速开始新文档</Title>
            </div>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
              新建文档
            </Button>
          </div>

          <div style={{ marginTop: 18 }}>
            <Input
              placeholder="搜索全部文档标题"
              prefix={<SearchOutlined style={{ color: colors.textSecondary }} />}
              value={searchValue}
              onChange={(e) => handleSearchChange(e.target.value)}
              allowClear
            />
          </div>
        </div>

        <div
          style={{
            borderRadius: radius.lg,
            padding: 24,
            background: colors.surface,
            border: `1px solid ${colors.borderSubtle}`,
            boxShadow: shadow.card,
          }}
        >
          <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>最近编辑</Text>
          <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
            {recentDocs.length > 0 ? recentDocs.map((doc) => (
              <button
                key={doc.id}
                type="button"
                onClick={() => navigate(`/documents/${doc.id}`)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: spacing.sm,
                  padding: '14px 16px',
                  borderRadius: radius.md,
                  border: `1px solid ${colors.borderSubtle}`,
                  background: colors.surfaceMuted,
                  cursor: 'pointer',
                  color: colors.text,
                  textAlign: 'left',
                }}
              >
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontWeight: 600 }}>{doc.title}</div>
                  <Text style={{ fontSize: 12, color: colors.textSecondary }}>
                    更新于 {dayjs(doc.updated_at).format('MM-DD HH:mm')}
                  </Text>
                </div>
                <ArrowRightOutlined style={{ color: colors.primary, flexShrink: 0 }} />
              </button>
            )) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{ color: colors.textSecondary }}>还没有最近文档</span>} />
            )}
          </div>
        </div>
      </section>

      <section
        style={{
          borderRadius: radius.lg,
          padding: 24,
          background: colors.surface,
          border: `1px solid ${colors.borderSubtle}`,
          boxShadow: shadow.card,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'end', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap', marginBottom: 18 }}>
          <div>
            <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>文档列表</Text>
            <Title level={4} style={{ margin: '6px 0 0', color: colors.text }}>全部文档</Title>
          </div>
          <Text style={{ color: colors.textSecondary }}>
            当前页 {docs.length} 篇，匹配 {total} 篇
          </Text>
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
            showTotal: (value) => `共 ${value} 篇`,
          }}
          locale={{
            emptyText: (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{ color: colors.textSecondary }}>暂无匹配文档</span>} />
            ),
          }}
        />
      </section>

      <Modal
        title="新建在线文档"
        open={createOpen}
        onOk={handleCreate}
        onCancel={() => {
          setCreateOpen(false)
          setNewTitle('')
          setNewDocKind('collab')
        }}
        confirmLoading={creating}
        okText="创建"
        cancelText="取消"
      >
        <Space direction="vertical" style={{ width: '100%' }} size={12}>
          <Select
            value={newDocKind}
            onChange={setNewDocKind}
            style={{ width: '100%' }}
            options={[
              { value: 'collab', label: '协作文档' },
              { value: 'word', label: 'Word 文档 (.docx)' },
              { value: 'excel', label: 'Excel 表格 (.xlsx)' },
            ]}
          />
          <Input
            placeholder="文档名称"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            onPressEnter={handleCreate}
            autoFocus
          />
        </Space>
      </Modal>
    </div>
  )
}
