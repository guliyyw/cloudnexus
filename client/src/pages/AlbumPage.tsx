import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Typography, Row, Col, Card, Button, Modal, Input, Spin, Empty, Space, Tooltip } from 'antd'
import { DeleteOutlined, EditOutlined, PictureOutlined, PlusOutlined } from '@ant-design/icons'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { useAlbumStore } from '../stores/albumStore'
import { getPreviewUrl } from '../services/file'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Text, Paragraph } = Typography
const { TextArea } = Input

export default function AlbumPage() {
  const navigate = useNavigate()
  const { albums, albumLoading, fetchAlbums, createAlbum, updateAlbum, deleteAlbum } = useAlbumStore()
  const [modalOpen, setModalOpen] = useState(false)
  const [editAlbum, setEditAlbum] = useState<{ id: string; name: string; description: string } | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchAlbums()
  }, [fetchAlbums])

  const metrics = useMemo(() => {
    const totalFiles = albums.reduce((sum, album) => sum + (album.file_count || 0), 0)
    const withCover = albums.filter((album) => album.cover_file_id && album.cover_file_id !== '0').length
    return [
      { label: '相册', value: albums.length, tone: 'primary' as const },
      { label: '媒体文件', value: totalFiles },
      { label: '已有封面', value: withCover, tone: 'success' as const },
    ]
  }, [albums])

  const openCreate = () => {
    setEditAlbum(null)
    setName('')
    setDescription('')
    setModalOpen(true)
  }

  const handleSave = async () => {
    if (!name.trim()) return
    setSaving(true)
    try {
      if (editAlbum) {
        await updateAlbum(editAlbum.id, { name: name.trim(), description: description.trim() })
      } else {
        await createAlbum(name.trim(), description.trim())
      }
      setModalOpen(false)
      setName('')
      setDescription('')
      setEditAlbum(null)
    } finally {
      setSaving(false)
    }
  }

  const handleEdit = (album: { id: string; name: string; description: string }) => {
    setEditAlbum(album)
    setName(album.name)
    setDescription(album.description)
    setModalOpen(true)
  }

  const handleDelete = (id: string, albumName: string) => {
    Modal.confirm({
      title: '删除相册',
      content: `确定要删除「${albumName}」吗？相册里的原文件不会被删除。`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => deleteAlbum(id),
    })
  }

  return (
    <div>
      <PageHeader
        eyebrow="Media Library"
        title="相册"
        description="整理云盘里的图片和视频。"
        actions={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建相册</Button>}
      />

      <MetricStrip items={metrics} />

      {albumLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : albums.length === 0 ? (
        <Empty description="还没有相册，创建第一个吧" />
      ) : (
        <Row gutter={[18, 18]}>
          {albums.map((album) => (
            <Col xs={24} sm={12} md={8} xl={6} key={album.id}>
              <Card
                hoverable
                style={{
                  borderRadius: radius.lg,
                  overflow: 'hidden',
                  boxShadow: shadow.card,
                  background: colors.surface,
                  border: `1px solid ${colors.borderSubtle}`,
                }}
                styles={{ body: { padding: 14 } }}
                onClick={() => navigate(`/album/${album.id}`)}
              >
                {album.cover_file_id && album.cover_file_id !== '0' ? (
                  <img
                    src={getPreviewUrl(album.cover_file_id)}
                    alt={album.name}
                    style={{ width: '100%', height: 154, borderRadius: radius.md, objectFit: 'cover', marginBottom: 12, background: colors.surfaceMuted }}
                  />
                ) : (
                  <div
                    style={{
                      width: '100%',
                      height: 154,
                      borderRadius: radius.md,
                      background: colors.surfaceMuted,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      marginBottom: 12,
                      fontSize: 46,
                      color: colors.primary,
                    }}
                  >
                    <PictureOutlined />
                  </div>
                )}
                <Space direction="vertical" size={spacing.xs} style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 8 }}>
                    <Tooltip title={album.name}>
                      <Text
                        style={{
                          fontSize: 15,
                          fontWeight: 700,
                          color: colors.text,
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                          lineHeight: 1.35,
                          minHeight: 40,
                        }}
                      >
                        {album.name}
                      </Text>
                    </Tooltip>
                    <Space size={2} onClick={(event) => event.stopPropagation()}>
                      <Tooltip title="编辑相册">
                        <Button type="text" size="small" icon={<EditOutlined />} onClick={() => handleEdit(album)} />
                      </Tooltip>
                      <Tooltip title="删除相册">
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(album.id, album.name)} />
                      </Tooltip>
                    </Space>
                  </div>
                  <Text type="secondary" style={{ fontSize: 12 }}>{album.file_count} 个媒体文件</Text>
                  {album.description && (
                    <Paragraph ellipsis={{ rows: 2 }} style={{ margin: 0, fontSize: 12, color: colors.textSecondary }}>
                      {album.description}
                    </Paragraph>
                  )}
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Modal
        title={editAlbum ? '编辑相册' : '新建相册'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginTop: 8 }}>
          <Input placeholder="相册名称" value={name} onChange={(event) => setName(event.target.value)} maxLength={200} />
          <TextArea placeholder="描述（可选）" value={description} onChange={(event) => setDescription(event.target.value)} rows={3} maxLength={500} />
        </div>
      </Modal>
    </div>
  )
}
