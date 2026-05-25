import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Typography, Row, Col, Card, Button, Modal, Input, Tag, Empty, Spin } from 'antd'
import { PlusOutlined, UserOutlined } from '@ant-design/icons'
import { usePlaylistStore } from '../stores/playlistStore'
import { colors, radius, shadow } from '../theme/tokens'

const { Title, Text } = Typography

export default function PlaylistPage() {
  const navigate = useNavigate()
  const { playlists, loading, fetchPlaylists, create, remove } = usePlaylistStore()
  const [modalOpen, setModalOpen] = useState(false)
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchPlaylists()
  }, [fetchPlaylists])

  const handleCreate = async () => {
    if (!name.trim()) return
    setSaving(true)
    try {
      const pl = await create(name.trim())
      setModalOpen(false)
      setName('')
      navigate(`/playlist/${pl.id}`)
    } catch { /* ignore */ }
    setSaving(false)
  }

  const handleDelete = (id: string, playlistName: string) => {
    Modal.confirm({
      title: '删除播放列表',
      content: `确定要删除「${playlistName}」吗？`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => remove(id),
    })
  }

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <Title level={4} style={{ margin: 0 }}>播放列表</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
          setName('')
          setModalOpen(true)
        }}>
          新建播放列表
        </Button>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : playlists.length === 0 ? (
        <Empty description="还没有播放列表，创建第一个吧" />
      ) : (
        <Row gutter={[16, 16]}>
          {playlists.map((pl) => (
            <Col xs={24} sm={12} md={8} lg={6} key={pl.id}>
              <Card
                hoverable
                style={{
                  borderRadius: radius.lg,
                  overflow: 'hidden',
                  boxShadow: shadow.card,
                }}
                bodyStyle={{ padding: 12 }}
                onClick={() => navigate(`/playlist/${pl.id}`)}
                actions={[
                  <span
                    key="delete"
                    onClick={(e) => { e.stopPropagation(); handleDelete(pl.id, pl.name) }}
                    style={{ color: colors.error, fontSize: 14 }}
                  >
                    删除
                  </span>,
                ]}
              >
                <div
                  style={{
                    width: '100%',
                    height: 140,
                    borderRadius: radius.md,
                    background: `linear-gradient(135deg, ${colors.primaryLight}, ${colors.primary}22)`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: 10,
                    fontSize: 48,
                    color: colors.primary,
                  }}
                >
                  <UserOutlined />
                </div>
                <Card.Meta
                  title={<Text ellipsis style={{ fontSize: 14 }}>{pl.name}</Text>}
                  description={
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>{pl.track_count} 首歌曲</Text>
                      <Tag color={pl.is_public ? 'blue' : 'default'} style={{ fontSize: 11 }}>
                        {pl.is_public ? '公开' : '私密'}
                      </Tag>
                    </div>
                  }
                />
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Modal
        title="新建播放列表"
        open={modalOpen}
        onOk={handleCreate}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
        okText="创建"
        cancelText="取消"
      >
        <div style={{ marginTop: 8 }}>
          <Input
            placeholder="播放列表名称"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={200}
            onPressEnter={handleCreate}
          />
        </div>
      </Modal>
    </div>
  )
}
