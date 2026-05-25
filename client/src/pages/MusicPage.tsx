import { useEffect, useState, useMemo } from 'react'
import { Typography, Table, Button, Input, Select, Spin, Empty } from 'antd'
import { PlayCircleOutlined, SearchOutlined } from '@ant-design/icons'
import { usePlayerStore } from '../stores/playerStore'
import { getLibrary, type Track } from '../services/music'
import { colors } from '../theme/tokens'

const { Title } = Typography

function formatDuration(s: number): string {
  if (!s) return '-'
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${m}:${sec.toString().padStart(2, '0')}`
}

export default function MusicPage() {
  const { play } = usePlayerStore()
  const [tracks, setTracks] = useState<Track[]>([])
  const [loading, setLoading] = useState(false)
  const [source, setSource] = useState('all')
  const [search, setSearch] = useState('')

  const fetchLibrary = async () => {
    setLoading(true)
    try {
      const res = await getLibrary(source, 1, 200)
      setTracks(res.tracks || [])
    } catch { /* ignore */ }
    setLoading(false)
  }

  useEffect(() => {
    fetchLibrary()
  }, [source])

  const filteredTracks = useMemo(() => {
    if (!search.trim()) return tracks
    const q = search.toLowerCase()
    return tracks.filter((t) =>
      t.title.toLowerCase().includes(q) ||
      t.artist.toLowerCase().includes(q) ||
      t.album.toLowerCase().includes(q)
    )
  }, [tracks, search])

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
          <div style={{ fontWeight: 500, fontSize: 13 }}>{text}</div>
          <span style={{ fontSize: 11, color: colors.textSecondary }}>{record.artist || '未知艺术家'}</span>
        </div>
      ),
    },
    {
      title: '专辑',
      dataIndex: 'album',
      width: 150,
      render: (text: string) => text || '-',
    },
    {
      title: '时长',
      dataIndex: 'duration',
      width: 80,
      render: (v: number) => formatDuration(v),
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 80,
      render: (v: string) => (
        <span style={{ fontSize: 11, color: v === 'public' ? colors.primary : '#52c41a' }}>
          {v === 'public' ? '公共库' : '云盘'}
        </span>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 40,
      render: (_: unknown, record: Track) => (
        <Button
          type="text"
          size="small"
          icon={<PlayCircleOutlined />}
          onClick={() => play(record, tracks)}
          style={{ color: colors.primary }}
        />
      ),
    },
  ]

  return (
    <div>
      <Title level={4} style={{ marginBottom: 16 }}>音乐</Title>

      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <Select
          value={source}
          onChange={setSource}
          size="small"
          style={{ width: 120 }}
          options={[
            { value: 'all', label: '全部来源' },
            { value: 'public', label: '公共库' },
            { value: 'cloud', label: '我的云盘' },
          ]}
        />
        <Input
          size="small"
          placeholder="搜索歌曲..."
          prefix={<SearchOutlined />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ width: 200 }}
        />
        <Button
          size="small"
          icon={<PlayCircleOutlined />}
          onClick={() => {
            if (filteredTracks.length > 0) play(filteredTracks[0], filteredTracks)
          }}
        >
          播放全部
        </Button>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
      ) : filteredTracks.length === 0 ? (
        <Empty description="没有找到音乐" />
      ) : (
        <Table
          dataSource={filteredTracks}
          columns={columns}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 50, size: 'small', showSizeChanger: false }}
          onRow={(record) => ({
            onDoubleClick: () => play(record, filteredTracks),
          })}
          style={{ cursor: 'pointer' }}
        />
      )}
    </div>
  )
}
