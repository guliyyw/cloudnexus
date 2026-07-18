import { useEffect, useState, useMemo } from 'react'
import { Typography, Table, Button, Input, Select, Spin, Empty, Modal, Upload, message, Dropdown, Space } from 'antd'
import { PlayCircleOutlined, SearchOutlined, UploadOutlined, DownOutlined } from '@ant-design/icons'
import type { UploadFile, UploadProps } from 'antd'
import { usePlayerStore } from '../stores/playerStore'
import { getLibrary, uploadPublicTrack, type Track } from '../services/music'
import { colors } from '../theme/tokens'
import { useAccess } from '../hooks/useAccess'

const { Title, Text } = Typography

function formatDuration(s: number): string {
  if (!s) return '-'
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${m}:${sec.toString().padStart(2, '0')}`
}

export default function MusicPage() {
  const { play } = usePlayerStore()
  const { isAdmin } = useAccess()
  const [tracks, setTracks] = useState<Track[]>([])
  const [loading, setLoading] = useState(false)
  const [source, setSource] = useState('all')
  const [search, setSearch] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadTitle, setUploadTitle] = useState('')
  const [uploadArtist, setUploadArtist] = useState('')
  const [uploadAlbum, setUploadAlbum] = useState('')

  const fetchLibrary = async () => {
    setLoading(true)
    try {
      const res = await getLibrary(source, 1, 200)
      setTracks(res.tracks || [])
    } catch {
      message.error('加载音乐库失败')
    }
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

  const resetUploadState = () => {
    setUploadOpen(false)
    setUploadFile(null)
    setUploadTitle('')
    setUploadArtist('')
    setUploadAlbum('')
  }

  const handleUpload = async () => {
    if (!uploadFile) {
      message.error('请选择音频文件')
      return
    }
    setUploading(true)
    try {
      await uploadPublicTrack(uploadFile, {
        title: uploadTitle.trim() || undefined,
        artist: uploadArtist.trim() || undefined,
        album: uploadAlbum.trim() || undefined,
      })
      message.success('公共歌曲上传成功')
      resetUploadState()
      if (source === 'public' || source === 'all') {
        fetchLibrary()
      }
    } catch (err: any) {
      message.error(err?.response?.data?.message || '上传公共歌曲失败')
    } finally {
      setUploading(false)
    }
  }

  const uploadProps: UploadProps = {
    accept: 'audio/*',
    beforeUpload: (file) => {
      setUploadFile(file)
      if (!uploadTitle) {
        const name = file.name.replace(/\.[^.]+$/, '')
        setUploadTitle(name)
      }
      return false
    },
    onRemove: () => {
      setUploadFile(null)
    },
    fileList: uploadFile
      ? [{ uid: uploadFile.name, name: uploadFile.name, status: 'done' as const } as UploadFile]
      : [],
    maxCount: 1,
  }

  const playTrackVariant = (record: Track, source: 'public' | 'cloud') => {
    const nextVariant = record.alternatives?.find((item) => item.source === source)
    if (!nextVariant) return

    const selectedTrack: Track = {
      ...record,
      ...nextVariant,
      alternatives: [
        {
          id: record.id,
          title: record.title,
          artist: record.artist,
          album: record.album,
          duration: record.duration,
          mime_type: record.mime_type,
          file_size: record.file_size,
          source: record.source,
          is_uploaded: record.is_uploaded,
        },
        ...(record.alternatives || []).filter((item) => !(item.id === nextVariant.id && item.source === nextVariant.source)),
      ],
    }

    const nextQueue = filteredTracks.map((track) => (
      track.id === record.id && track.source === record.source ? selectedTrack : track
    ))

    play(selectedTrack, nextQueue)
  }

  const buildSourceMenu = (record: Track) => {
    if (!record.alternatives?.length) {
      return null
    }
    return {
      items: record.alternatives.map((item) => ({
        key: `${item.source}:${item.id}`,
        label: (
          <Space size={8}>
            <span>{item.source === 'public' ? '公共库版本' : '私有版本'}</span>
            <span style={{ color: colors.textSecondary, fontSize: 12 }}>
              {item.artist || '未知艺术家'}
            </span>
          </Space>
        ),
        onClick: () => playTrackVariant(record, item.source),
      })),
    }
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
      width: 180,
      render: (v: string, record: Track) => {
        const label = v === 'public' ? '公共库' : record.is_uploaded ? '我的上传' : '云盘'
        const menu = buildSourceMenu(record)
        return (
          <Space size={6} wrap>
            <span style={{ fontSize: 11, color: v === 'public' ? colors.primary : '#52c41a' }}>
              {label}
            </span>
            {menu && (
              <Dropdown menu={menu} trigger={['click']}>
                <Button type="text" size="small" icon={<DownOutlined />} style={{ color: colors.textSecondary }}>
                  切换版本
                </Button>
              </Dropdown>
            )}
          </Space>
        )
      },
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
          onClick={() => play(record, filteredTracks)}
          style={{ color: colors.primary }}
        />
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>音乐</Title>
        {isAdmin && (
          <Button icon={<UploadOutlined />} onClick={() => setUploadOpen(true)}>
            上传公共歌曲
          </Button>
        )}
      </div>

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
            if (filteredTracks.length > 0) {
              const queue = filteredTracks.map((track) => {
                if (!track.alternatives?.length) return track
                return {
                  ...track,
                  alternatives: [
                    ...track.alternatives.filter((item) => item.source !== 'cloud'),
                    ...track.alternatives.filter((item) => item.source === 'cloud'),
                  ],
                }
              })
              play(queue[0], queue)
            }
          }}
        >
          播放全部
        </Button>
        <Button size="small" onClick={fetchLibrary}>
          刷新
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
          rowKey={(record) => `${record.source}:${record.id}`}
          size="small"
          pagination={{ pageSize: 50, size: 'small', showSizeChanger: false }}
          onRow={(record) => ({
            onDoubleClick: () => play(record, filteredTracks),
          })}
          style={{ cursor: 'pointer' }}
        />
      )}

      <Modal
        title="上传公共歌曲"
        open={uploadOpen}
        onOk={handleUpload}
        onCancel={resetUploadState}
        confirmLoading={uploading}
        okText="上传"
        cancelText="取消"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Upload {...uploadProps}>
            <Button icon={<UploadOutlined />}>选择音频文件</Button>
          </Upload>
          {uploadFile && (
            <Text style={{ fontSize: 12, color: colors.textSecondary }}>
              已选择：{uploadFile.name}
            </Text>
          )}
          <Text type="secondary" style={{ fontSize: 12 }}>
            若公共库中已存在同名歌曲（标题 + 歌手 + 专辑），上传会被拦截；若用户云盘中已有同歌，列表默认优先展示私有版本，但仍可手动切换到公共版本。
          </Text>
          <Input placeholder="标题（可选，默认取文件名或音频标签）" value={uploadTitle} onChange={(e) => setUploadTitle(e.target.value)} />
          <Input placeholder="歌手（可选）" value={uploadArtist} onChange={(e) => setUploadArtist(e.target.value)} />
          <Input placeholder="专辑（可选）" value={uploadAlbum} onChange={(e) => setUploadAlbum(e.target.value)} />
        </div>
      </Modal>
    </div>
  )
}
