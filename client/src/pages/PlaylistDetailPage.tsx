import { useEffect, useState, useMemo, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Table,
  Button,
  Space,
  Switch,
  Modal,
  Input,
  Spin,
  Empty,
  Dropdown,
  Upload,
  message,
  Tag,
  Typography,
} from 'antd'
import {
  ArrowLeftOutlined,
  PlusOutlined,
  PlayCircleOutlined,
  DeleteOutlined,
  SearchOutlined,
  ExportOutlined,
  ImportOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { usePlaylistStore } from '../stores/playlistStore'
import { usePlayerStore } from '../stores/playerStore'
import { getLibrary, exportPlaylist, importPlaylist, type Track } from '../services/music'
import { colors, radius, spacing } from '../theme/tokens'

const { Text } = Typography

function formatDuration(s: number): string {
  if (!s) return '-'
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${m}:${sec.toString().padStart(2, '0')}`
}

function formatTotalDuration(seconds: number): string {
  if (!seconds) return '0 分钟'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return hours > 0 ? `${hours} 小时 ${minutes} 分钟` : `${minutes} 分钟`
}

export default function PlaylistDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentPlaylist, currentTracks, loading, fetchPlaylist, update, addTrack, removeTrack } = usePlaylistStore()
  const { play } = usePlayerStore()

  const [addModalOpen, setAddModalOpen] = useState(false)
  const [librarySearch, setLibrarySearch] = useState('')
  const [libraryTracks, setLibraryTracks] = useState<Track[]>([])
  const [libraryLoading, setLibraryLoading] = useState(false)
  const [addingTrackId, setAddingTrackId] = useState<string | null>(null)
  const [importing, setImporting] = useState(false)
  const [trackDetails, setTrackDetails] = useState<Track[]>([])

  const playlistId = id || ''

  useEffect(() => {
    if (id) fetchPlaylist(id)
  }, [id, fetchPlaylist])

  useEffect(() => {
    if (currentTracks.length === 0) {
      setTrackDetails([])
      return
    }
    const loadDetails = async () => {
      try {
        const res = await getLibrary('all', 1, 500)
        const libraryMap = new Map<string, Track>()
        for (const track of res.tracks || []) {
          libraryMap.set(`${track.source}:${track.id}`, track)
        }
        const details = currentTracks
          .map((track) => libraryMap.get(`${track.source}:${track.track_id}`))
          .filter(Boolean) as Track[]
        setTrackDetails(details)
      } catch {
        setTrackDetails([])
      }
    }
    loadDetails()
  }, [currentTracks])

  const stats = useMemo(() => {
    const publicCount = trackDetails.filter((track) => track.source === 'public').length
    const cloudCount = trackDetails.length - publicCount
    const totalDuration = trackDetails.reduce((sum, track) => sum + (track.duration || 0), 0)
    return { publicCount, cloudCount, totalDuration }
  }, [trackDetails])

  const handlePlay = useCallback((track: Track, index: number) => {
    play(track, trackDetails.slice(index))
  }, [play, trackDetails])

  const handleRemoveTrack = useCallback((trackId: string) => {
    if (!playlistId) return
    removeTrack(playlistId, trackId)
  }, [playlistId, removeTrack])

  const handleTogglePublic = useCallback(async (checked: boolean) => {
    if (!playlistId) return
    await update(playlistId, { is_public: checked })
  }, [playlistId, update])

  const openAddModal = async () => {
    setAddModalOpen(true)
    setLibrarySearch('')
    setLibraryLoading(true)
    try {
      const res = await getLibrary('all', 1, 200)
      setLibraryTracks(res.tracks || [])
    } catch {
      message.error('加载音乐库失败')
    } finally {
      setLibraryLoading(false)
    }
  }

  const handleAddTrack = async (track: Track) => {
    if (!playlistId) return
    setAddingTrackId(track.id)
    try {
      await addTrack(playlistId, track.id, track.source)
      message.success('已添加到播放列表')
      fetchPlaylist(playlistId)
    } catch {
      message.error('添加失败')
    } finally {
      setAddingTrackId(null)
    }
  }

  const filteredLibrary = useMemo(() => {
    if (!librarySearch.trim()) return libraryTracks
    const q = librarySearch.toLowerCase()
    return libraryTracks.filter((track) =>
      track.title.toLowerCase().includes(q) ||
      track.artist.toLowerCase().includes(q) ||
      track.album.toLowerCase().includes(q)
    )
  }, [libraryTracks, librarySearch])

  const existingTrackIds = useMemo(() => {
    return new Set(currentTracks.map((track) => `${track.source}:${track.track_id}`))
  }, [currentTracks])

  const handleExport = async (format: 'json' | 'm3u') => {
    if (!playlistId) return
    try {
      const blob = await exportPlaylist(playlistId, format)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${currentPlaylist?.name || 'playlist'}.${format}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      message.success('导出成功')
    } catch {
      message.error('导出失败')
    }
  }

  const exportMenuItems: MenuProps['items'] = [
    { key: 'json', label: 'JSON', onClick: () => handleExport('json') },
    { key: 'm3u', label: 'M3U', onClick: () => handleExport('m3u') },
  ]

  const handleImport = async (file: File) => {
    if (!playlistId) return false
    const ext = file.name.split('.').pop()?.toLowerCase()
    if (ext !== 'json' && ext !== 'm3u') {
      message.error('仅支持 .json 或 .m3u 文件')
      return false
    }
    setImporting(true)
    try {
      await importPlaylist(playlistId, file, ext)
      message.success('导入成功')
      fetchPlaylist(playlistId)
    } catch {
      message.error('导入失败')
    } finally {
      setImporting(false)
    }
    return false
  }

  const columns = [
    {
      title: '#',
      dataIndex: 'index',
      width: 50,
      render: (_: unknown, __: unknown, idx: number) => (
        <span style={{ color: colors.textSecondary, fontSize: 12 }}>{idx + 1}</span>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      render: (text: string, record: Track) => (
        <div>
          <div style={{ fontWeight: 600, fontSize: 13 }}>{text}</div>
          <span style={{ fontSize: 11, color: colors.textSecondary }}>{record.artist || '未知歌手'}</span>
        </div>
      ),
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 92,
      render: (v: string) => (
        <Tag color={v === 'public' ? 'orange' : 'green'}>{v === 'public' ? '公共库' : '云盘'}</Tag>
      ),
    },
    {
      title: '时长',
      dataIndex: 'duration',
      width: 80,
      render: (v: number) => formatDuration(v),
    },
    {
      title: '',
      key: 'actions',
      width: 90,
      render: (_: unknown, record: Track, idx: number) => (
        <Space size={4}>
          <Button type="text" size="small" icon={<PlayCircleOutlined />} onClick={() => handlePlay(record, idx)} style={{ color: colors.primary }} />
          <Button type="text" size="small" icon={<DeleteOutlined />} onClick={() => handleRemoveTrack(record.id)} style={{ color: colors.error }} />
        </Space>
      ),
    },
  ]

  return (
    <div>
      <PageHeader
        eyebrow="Playlist"
        title={currentPlaylist?.name || '播放列表'}
        description="维护这个播放列表的歌曲顺序、公开状态，并可导入或导出为通用格式。"
        actions={
          <>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/playlist')}>返回列表</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openAddModal}>添加歌曲</Button>
            <Dropdown menu={{ items: exportMenuItems }} placement="bottomRight">
              <Button icon={<ExportOutlined />}>导出</Button>
            </Dropdown>
            <Upload accept=".json,.m3u" showUploadList={false} beforeUpload={handleImport}>
              <Button icon={<ImportOutlined />} loading={importing}>导入</Button>
            </Upload>
          </>
        }
      />

      <div style={{ marginBottom: spacing.md }}>
        <Space size={8}>
          <Text type="secondary" style={{ fontSize: 13 }}>公开播放列表</Text>
          <Switch size="small" checked={!!currentPlaylist?.is_public} onChange={handleTogglePublic} disabled={!currentPlaylist} />
        </Space>
      </div>

      <MetricStrip
        items={[
          { label: '歌曲数量', value: trackDetails.length, tone: 'primary' },
          { label: '公共库', value: stats.publicCount },
          { label: '云盘', value: stats.cloudCount, tone: 'success' },
          { label: '总时长', value: formatTotalDuration(stats.totalDuration), tone: 'warning' },
        ]}
      />

      {loading ? (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
      ) : trackDetails.length === 0 ? (
        <Empty description="播放列表为空，点击“添加歌曲”开始整理" />
      ) : (
        <Table
          dataSource={trackDetails}
          columns={columns}
          rowKey={(record) => `${record.source}:${record.id}`}
          size="small"
          pagination={{ pageSize: 50, size: 'small', showSizeChanger: false }}
          onRow={(record, idx) => ({
            onDoubleClick: () => handlePlay(record, idx || 0),
          })}
          style={{ cursor: 'pointer' }}
        />
      )}

      <Modal
        title="添加歌曲"
        open={addModalOpen}
        onCancel={() => setAddModalOpen(false)}
        footer={null}
        width={680}
      >
        <Input
          placeholder="搜索歌曲、歌手、专辑..."
          prefix={<SearchOutlined />}
          value={librarySearch}
          onChange={(e) => setLibrarySearch(e.target.value)}
          style={{ marginBottom: 12 }}
        />
        {libraryLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : filteredLibrary.length === 0 ? (
          <Empty description="没有找到歌曲" />
        ) : (
          <div style={{ maxHeight: 420, overflowY: 'auto' }}>
            {filteredLibrary.map((track) => {
              const key = `${track.source}:${track.id}`
              const exists = existingTrackIds.has(key)
              return (
                <div
                  key={key}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: spacing.sm,
                    padding: '10px 8px',
                    borderBottom: `1px solid ${colors.borderSubtle}`,
                    borderRadius: radius.sm,
                    background: exists ? colors.surfaceMuted : 'transparent',
                    opacity: exists ? 0.62 : 1,
                  }}
                >
                  <div style={{ minWidth: 0 }}>
                    <div style={{ fontWeight: 600, fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{track.title}</div>
                    <div style={{ fontSize: 11, color: colors.textSecondary }}>
                      {track.artist || '未知歌手'} {track.album ? `· ${track.album}` : ''}
                    </div>
                  </div>
                  <Space size={8}>
                    <Tag color={track.source === 'public' ? 'orange' : 'green'} style={{ fontSize: 11 }}>
                      {track.source === 'public' ? '公共库' : '云盘'}
                    </Tag>
                    {exists ? (
                      <Text type="secondary" style={{ fontSize: 12 }}>已添加</Text>
                    ) : (
                      <Button type="link" size="small" loading={addingTrackId === track.id} onClick={() => handleAddTrack(track)}>
                        添加
                      </Button>
                    )}
                  </Space>
                </div>
              )
            })}
          </div>
        )}
      </Modal>
    </div>
  )
}
