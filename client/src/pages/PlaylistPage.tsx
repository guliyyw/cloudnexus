import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Row, Col, Card, Button, Modal, Input, Tag, Empty, Spin, Typography, Space } from 'antd'
import { DeleteOutlined, PlusOutlined, UnorderedListOutlined } from '@ant-design/icons'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { usePlaylistStore } from '../stores/playlistStore'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Text } = Typography

export default function PlaylistPage() {
  const navigate = useNavigate()
  const { playlists, loading, fetchPlaylists, create, remove } = usePlaylistStore()
  const [modalOpen, setModalOpen] = useState(false)
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchPlaylists()
  }, [fetchPlaylists])

  const stats = useMemo(() => {
    const totalTracks = playlists.reduce((sum, item) => sum + (item.track_count || 0), 0)
    const publicCount = playlists.filter((item) => item.is_public).length
    return { totalTracks, publicCount }
  }, [playlists])

  const openCreateModal = () => {
    setName('')
    setModalOpen(true)
  }

  const handleCreate = async () => {
    if (!name.trim()) return
    setSaving(true)
    try {
      const playlist = await create(name.trim())
      setModalOpen(false)
      setName('')
      navigate(`/playlist/${playlist.id}`)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (id: string, playlistName: string) => {
    Modal.confirm({
      title: '删除播放列表',
      content: `确定要删除「${playlistName}」吗？播放列表中的歌曲不会被删除。`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => remove(id),
    })
  }

  return (
    <div>
      <PageHeader
        eyebrow="Playlists"
        title="播放列表"
        description="把常听的歌曲整理成不同场景的播放清单，并支持导入导出。"
        actions={<Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>新建播放列表</Button>}
      />

      <MetricStrip
        items={[
          { label: '播放列表', value: playlists.length, tone: 'primary' },
          { label: '歌曲总数', value: stats.totalTracks, tone: 'success' },
          { label: '公开列表', value: stats.publicCount },
        ]}
      />

      {loading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : playlists.length === 0 ? (
        <Empty description="还没有播放列表，创建第一个吧" />
      ) : (
        <Row gutter={[16, 16]}>
          {playlists.map((playlist) => (
            <Col xs={24} sm={12} md={8} lg={6} key={playlist.id}>
              <Card
                hoverable
                style={{
                  borderRadius: radius.lg,
                  overflow: 'hidden',
                  boxShadow: shadow.card,
                  border: `1px solid ${colors.borderSubtle}`,
                }}
                styles={{ body: { padding: 14 } }}
                onClick={() => navigate(`/playlist/${playlist.id}`)}
                actions={[
                  <Button
                    key="delete"
                    type="text"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(event) => {
                      event.stopPropagation()
                      handleDelete(playlist.id, playlist.name)
                    }}
                  >
                    删除
                  </Button>,
                ]}
              >
                <div
                  style={{
                    width: '100%',
                    height: 136,
                    borderRadius: radius.md,
                    background: `linear-gradient(135deg, ${colors.primaryLight}, ${colors.surfaceMuted})`,
                    border: `1px solid ${colors.borderSubtle}`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: spacing.sm,
                    fontSize: 44,
                    color: colors.primary,
                  }}
                >
                  <UnorderedListOutlined />
                </div>
                <Space direction="vertical" size={4} style={{ width: '100%' }}>
                  <Text ellipsis style={{ fontSize: 14, fontWeight: 600 }}>{playlist.name}</Text>
                  <Space size={8} wrap>
                    <Text type="secondary" style={{ fontSize: 12 }}>{playlist.track_count} 首歌曲</Text>
                    <Tag color={playlist.is_public ? 'blue' : 'default'} style={{ fontSize: 11 }}>
                      {playlist.is_public ? '公开' : '私密'}
                    </Tag>
                  </Space>
                </Space>
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
