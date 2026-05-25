import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Typography, Row, Col, Card, Button, Modal, Input, Spin, Empty } from 'antd'
import { PlusOutlined, PictureOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { useAlbumStore } from '../stores/albumStore'
import { colors, radius, shadow } from '../theme/tokens'

const { Title, Text, Paragraph } = Typography
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

  const handleSave = async () => {
    if (!name.trim()) return
    setSaving(true)
    if (editAlbum) {
      await updateAlbum(editAlbum.id, { name: name.trim(), description: description.trim() })
    } else {
      await createAlbum(name.trim(), description.trim())
    }
    setSaving(false)
    setModalOpen(false)
    setName('')
    setDescription('')
    setEditAlbum(null)
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
      content: `确定要删除「${albumName}」吗？文件不会被删除。`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => deleteAlbum(id),
    })
  }

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <Title level={4} style={{ margin: 0 }}>相册</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
          setEditAlbum(null)
          setName('')
          setDescription('')
          setModalOpen(true)
        }}>
          新建相册
        </Button>
      </div>

      {albumLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : albums.length === 0 ? (
        <Empty description="还没有相册，创建第一个吧" />
      ) : (
        <Row gutter={[16, 16]}>
          {albums.map((album) => (
            <Col xs={24} sm={12} md={8} lg={6} key={album.id}>
              <Card
                hoverable
                style={{
                  borderRadius: radius.lg,
                  overflow: 'hidden',
                  boxShadow: shadow.card,
                }}
                bodyStyle={{ padding: 12 }}
                onClick={() => navigate(`/album/${album.id}`)}
                actions={[
                  <EditOutlined key="edit" onClick={(e) => { e.stopPropagation(); handleEdit(album) }} />,
                  <DeleteOutlined key="delete" onClick={(e) => { e.stopPropagation(); handleDelete(album.id, album.name) }} />,
                ]}
              >
                <div
                  style={{
                    width: '100%',
                    height: 140,
                    borderRadius: radius.md,
                    background: 'rgba(255,255,255,0.04)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: 10,
                    fontSize: 48,
                    color: colors.primary,
                  }}
                >
                  <PictureOutlined />
                </div>
                <Card.Meta
                  title={<Text ellipsis style={{ fontSize: 14 }}>{album.name}</Text>}
                  description={
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>{album.file_count} 张照片</Text>
                      {album.description && (
                        <Paragraph ellipsis={{ rows: 1 }} style={{ margin: 0, fontSize: 12, color: colors.textSecondary }}>
                          {album.description}
                        </Paragraph>
                      )}
                    </div>
                  }
                />
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
          <Input
            placeholder="相册名称"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={200}
          />
          <TextArea
            placeholder="描述（可选）"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={3}
            maxLength={500}
          />
        </div>
      </Modal>
    </div>
  )
}
