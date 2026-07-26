import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  Dropdown,
  Empty,
  Input,
  Layout,
  List,
  Modal,
  Segmented,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
  Upload,
  message,
} from 'antd'
import type { MenuProps, UploadFile, UploadProps } from 'antd'
import {
  AppstoreOutlined,
  ClockCircleOutlined,
  CloudOutlined,
  CustomerServiceOutlined,
  DeleteOutlined,
  DownOutlined,
  ExportOutlined,
  FireOutlined,
  HeartFilled,
  HeartOutlined,
  ImportOutlined,
  PlusOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { useAccess } from '../hooks/useAccess'
import {
  exportPlaylist,
  getLibrary,
  getLikedTracks,
  getRecentTracks,
  importPlaylist,
  likeTrack,
  recordRecentTrack,
  unlikeTrack,
  uploadPublicTrack,
  type Playlist,
  type Track,
} from '../services/music'
import { usePlayerStore } from '../stores/playerStore'
import { usePlaylistStore } from '../stores/playlistStore'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Sider, Content } = Layout
const { Text, Title } = Typography

type LibrarySource = 'all' | 'public' | 'cloud'
type ViewKey = 'discover' | 'library' | 'liked' | 'recent' | 'search' | 'playlist'

function formatDuration(seconds: number): string {
  if (!seconds) return '-'
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return `${minutes}:${rest.toString().padStart(2, '0')}`
}

function formatTotalDuration(seconds: number): string {
  if (!seconds) return '0 分钟'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return hours > 0 ? `${hours} 小时 ${minutes} 分钟` : `${minutes} 分钟`
}

function trackKey(track: Pick<Track, 'id' | 'source'>): string {
  return `${track.source}:${track.id}`
}

function trackMatches(track: Track, keyword: string): boolean {
  if (!keyword.trim()) return true
  const q = keyword.trim().toLowerCase()
  return [track.title, track.artist, track.album].some((value) => (value || '').toLowerCase().includes(q))
}

function playlistMatches(playlist: Playlist, keyword: string): boolean {
  if (!keyword.trim()) return true
  return playlist.name.toLowerCase().includes(keyword.trim().toLowerCase())
}

function sourceLabel(track: Track): string {
  if (track.source === 'public') return '公共曲库'
  return track.is_uploaded ? '我的上传' : '云盘音频'
}

function mergeSavedWithLibrary(saved: Track[], library: Track[]): Track[] {
  const libraryMap = new Map(library.map((track) => [trackKey(track), track]))
  return saved.map((track) => libraryMap.get(trackKey(track)) || track)
}

export default function MusicPage() {
  const { isAdmin } = useAccess()
  const { play } = usePlayerStore()
  const {
    playlists,
    currentPlaylist,
    currentTracks,
    loading: playlistLoading,
    fetchPlaylists,
    fetchPlaylist,
    create,
    update,
    remove,
    addTrack,
    removeTrack,
  } = usePlaylistStore()

  const [view, setView] = useState<ViewKey>('discover')
  const [source, setSource] = useState<LibrarySource>('all')
  const [search, setSearch] = useState('')
  const [tracks, setTracks] = useState<Track[]>([])
  const [likedTracks, setLikedTracks] = useState<Track[]>([])
  const [recentTracks, setRecentTracks] = useState<Track[]>([])
  const [libraryLoading, setLibraryLoading] = useState(false)
  const [selectedPlaylistId, setSelectedPlaylistId] = useState<string | null>(null)
  const [playlistTracks, setPlaylistTracks] = useState<Track[]>([])

  const [createOpen, setCreateOpen] = useState(false)
  const [playlistName, setPlaylistName] = useState('')
  const [playlistPublic, setPlaylistPublic] = useState(false)
  const [creating, setCreating] = useState(false)

  const [addOpen, setAddOpen] = useState(false)
  const [addSearch, setAddSearch] = useState('')
  const [addingKey, setAddingKey] = useState<string | null>(null)
  const [importing, setImporting] = useState(false)

  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadTitle, setUploadTitle] = useState('')
  const [uploadArtist, setUploadArtist] = useState('')
  const [uploadAlbum, setUploadAlbum] = useState('')

  const fetchLibrary = useCallback(async (nextSource = source) => {
    setLibraryLoading(true)
    try {
      const res = await getLibrary(nextSource, 1, 500)
      setTracks(res.tracks || [])
    } catch {
      message.error('加载音乐库失败')
    } finally {
      setLibraryLoading(false)
    }
  }, [source])

  const fetchUserMusicState = useCallback(async () => {
    try {
      const [liked, recent] = await Promise.all([getLikedTracks(), getRecentTracks(50)])
      setLikedTracks(liked.tracks || [])
      setRecentTracks(recent.tracks || [])
    } catch {
      message.error('加载喜欢和最近播放失败')
    }
  }, [])

  useEffect(() => {
    fetchPlaylists()
    fetchUserMusicState()
  }, [fetchPlaylists, fetchUserMusicState])

  useEffect(() => {
    fetchLibrary(source)
  }, [fetchLibrary, source])

  useEffect(() => {
    if (tracks.length === 0) return
    setLikedTracks((items) => mergeSavedWithLibrary(items, tracks))
    setRecentTracks((items) => mergeSavedWithLibrary(items, tracks))
  }, [tracks])

  useEffect(() => {
    if (!selectedPlaylistId) return
    fetchPlaylist(selectedPlaylistId)
  }, [fetchPlaylist, selectedPlaylistId])

  useEffect(() => {
    if (currentTracks.length === 0) {
      setPlaylistTracks([])
      return
    }
    const libraryMap = new Map(tracks.map((track) => [trackKey(track), track]))
    setPlaylistTracks(
      currentTracks
        .map((item) => libraryMap.get(`${item.source}:${item.track_id}`))
        .filter(Boolean) as Track[],
    )
  }, [currentTracks, tracks])

  const likedKeys = useMemo(() => new Set(likedTracks.map(trackKey)), [likedTracks])
  const filteredTracks = useMemo(() => tracks.filter((track) => trackMatches(track, search)), [tracks, search])
  const filteredPlaylistTracks = useMemo(() => playlistTracks.filter((track) => trackMatches(track, search)), [playlistTracks, search])
  const filteredLikedTracks = useMemo(() => likedTracks.filter((track) => trackMatches(track, search)), [likedTracks, search])
  const filteredRecentTracks = useMemo(() => recentTracks.filter((track) => trackMatches(track, search)), [recentTracks, search])
  const searchedTracks = useMemo(() => tracks.filter((track) => trackMatches(track, search)), [tracks, search])
  const searchedPlaylists = useMemo(() => playlists.filter((playlist) => playlistMatches(playlist, search)), [playlists, search])
  const addableTracks = useMemo(() => tracks.filter((track) => trackMatches(track, addSearch)), [tracks, addSearch])
  const existingPlaylistKeys = useMemo(() => new Set(currentTracks.map((track) => `${track.source}:${track.track_id}`)), [currentTracks])

  const libraryStats = useMemo(() => {
    const playlistTrackCount = playlists.reduce((sum, playlist) => sum + (playlist.track_count || 0), 0)
    return { playlistTrackCount }
  }, [playlists])

  const selectedQueue = useMemo(() => {
    if (view === 'playlist') return filteredPlaylistTracks
    if (view === 'liked') return filteredLikedTracks
    if (view === 'recent') return filteredRecentTracks
    if (view === 'search') return searchedTracks
    return filteredTracks
  }, [filteredLikedTracks, filteredPlaylistTracks, filteredRecentTracks, filteredTracks, searchedTracks, view])

  const heroTracks = useMemo(() => tracks.slice(0, 6), [tracks])

  const setGlobalSearch = (value: string) => {
    setSearch(value)
    if (value.trim()) {
      setView('search')
      setSelectedPlaylistId(null)
    } else if (view === 'search') {
      setView('discover')
    }
  }

  const recordRecent = useCallback(async (track: Track) => {
    setRecentTracks((items) => [track, ...items.filter((item) => trackKey(item) !== trackKey(track))].slice(0, 50))
    try {
      await recordRecentTrack(track.id, track.source)
    } catch {
      message.error('同步最近播放失败')
    }
  }, [])

  const playMusic = useCallback((track?: Track, queue?: Track[]) => {
    if (track) recordRecent(track)
    play(track, queue)
  }, [play, recordRecent])

  const toggleLike = async (track: Track) => {
    const key = trackKey(track)
    const liked = likedKeys.has(key)
    setLikedTracks((items) => liked ? items.filter((item) => trackKey(item) !== key) : [track, ...items])
    try {
      if (liked) {
        await unlikeTrack(track.id, track.source)
      } else {
        await likeTrack(track.id, track.source)
      }
    } catch {
      setLikedTracks((items) => liked ? [track, ...items.filter((item) => trackKey(item) !== key)] : items.filter((item) => trackKey(item) !== key))
      message.error('同步喜欢失败')
    }
  }

  const showPlaylist = (playlist: Playlist) => {
    setSelectedPlaylistId(playlist.id)
    setView('playlist')
    setSearch('')
  }

  const openCreateModal = () => {
    setPlaylistName('')
    setPlaylistPublic(false)
    setCreateOpen(true)
  }

  const handleCreate = async () => {
    if (!playlistName.trim()) {
      message.warning('请输入歌单名称')
      return
    }
    setCreating(true)
    try {
      const playlist = await create(playlistName.trim(), playlistPublic)
      setCreateOpen(false)
      showPlaylist(playlist)
      message.success('歌单已创建')
    } catch {
      message.error('创建歌单失败')
    } finally {
      setCreating(false)
    }
  }

  const handleDeletePlaylist = (playlist: Playlist) => {
    Modal.confirm({
      title: '删除歌单',
      content: `确定删除「${playlist.name}」吗？歌单里的歌曲不会被删除。`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        await remove(playlist.id)
        if (selectedPlaylistId === playlist.id) {
          setSelectedPlaylistId(null)
          setView('library')
        }
      },
    })
  }

  const handleAddTrack = async (track: Track) => {
    if (!selectedPlaylistId) return
    const key = trackKey(track)
    setAddingKey(key)
    try {
      await addTrack(selectedPlaylistId, track.id, track.source)
      await fetchPlaylist(selectedPlaylistId)
      await fetchPlaylists()
      message.success('已添加到歌单')
    } catch {
      message.error('添加失败')
    } finally {
      setAddingKey(null)
    }
  }

  const handleRemoveTrack = async (track: Track) => {
    if (!selectedPlaylistId) return
    await removeTrack(selectedPlaylistId, track.id)
    await fetchPlaylist(selectedPlaylistId)
    await fetchPlaylists()
  }

  const handleTogglePublic = async (checked: boolean) => {
    if (!selectedPlaylistId) return
    await update(selectedPlaylistId, { is_public: checked })
  }

  const handleExport = async (format: 'json' | 'm3u') => {
    if (!selectedPlaylistId) return
    try {
      const blob = await exportPlaylist(selectedPlaylistId, format)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${currentPlaylist?.name || 'playlist'}.${format}`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
      message.success('导出成功')
    } catch {
      message.error('导出失败')
    }
  }

  const handleImport = async (file: File) => {
    if (!selectedPlaylistId) return false
    const format = file.name.split('.').pop()?.toLowerCase()
    if (format !== 'json' && format !== 'm3u') {
      message.error('仅支持 .json 或 .m3u 文件')
      return false
    }
    setImporting(true)
    try {
      await importPlaylist(selectedPlaylistId, file, format)
      await fetchPlaylist(selectedPlaylistId)
      await fetchPlaylists()
      message.success('导入成功')
    } catch {
      message.error('导入失败')
    } finally {
      setImporting(false)
    }
    return false
  }

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
      fetchLibrary(source)
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
      if (!uploadTitle) setUploadTitle(file.name.replace(/\.[^.]+$/, ''))
      return false
    },
    onRemove: () => setUploadFile(null),
    fileList: uploadFile ? [{ uid: uploadFile.name, name: uploadFile.name, status: 'done' as const } as UploadFile] : [],
    maxCount: 1,
  }

  const playTrackVariant = (record: Track, nextSource: 'public' | 'cloud', queue: Track[]) => {
    const nextVariant = record.alternatives?.find((item) => item.source === nextSource)
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
    const nextQueue = queue.map((track) => (trackKey(track) === trackKey(record) ? selectedTrack : track))
    playMusic(selectedTrack, nextQueue)
  }

  const buildSourceMenu = (record: Track, queue: Track[]) => {
    if (!record.alternatives?.length) return undefined
    return {
      items: record.alternatives.map((item) => ({
        key: `${item.source}:${item.id}`,
        label: item.source === 'public' ? '公共曲库版本' : '云盘版本',
        onClick: () => playTrackVariant(record, item.source, queue),
      })),
    }
  }

  const renderTrackTitle = (text: string, record: Track) => (
    <div style={{ minWidth: 0 }}>
      <div style={{ fontWeight: 650, fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{text}</div>
      <div style={{ fontSize: 12, color: colors.textSecondary, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {record.artist || '未知歌手'}{record.album ? ` · ${record.album}` : ''}
      </div>
    </div>
  )

  const makeTrackColumns = (queue: Track[], inPlaylist = false) => [
    { title: '', dataIndex: 'index', width: 42, render: (_: unknown, __: Track, idx: number) => <Text type="secondary" style={{ fontSize: 12 }}>{idx + 1}</Text> },
    { title: '歌曲', dataIndex: 'title', render: renderTrackTitle },
    {
      title: '来源',
      dataIndex: 'source',
      width: 150,
      render: (_: string, record: Track) => {
        const menu = buildSourceMenu(record, queue)
        return (
          <Space size={6}>
            <Tag color={record.source === 'public' ? 'orange' : 'green'}>{sourceLabel(record)}</Tag>
            {menu && (
              <Dropdown menu={menu} trigger={['click']}>
                <Button type="text" size="small" icon={<DownOutlined />} />
              </Dropdown>
            )}
          </Space>
        )
      },
    },
    { title: '时长', dataIndex: 'duration', width: 80, render: (value: number) => formatDuration(value) },
    {
      title: '',
      key: 'actions',
      width: inPlaylist ? 136 : 96,
      render: (_: unknown, record: Track, idx: number) => {
        const liked = likedKeys.has(trackKey(record))
        return (
          <Space size={4}>
            <Tooltip title={liked ? '取消喜欢' : '喜欢'}>
              <Button type="text" size="small" icon={liked ? <HeartFilled /> : <HeartOutlined />} onClick={() => toggleLike(record)} style={{ color: liked ? colors.error : colors.textSecondary }} />
            </Tooltip>
            <Tooltip title="播放">
              <Button type="text" size="small" icon={<PlayCircleOutlined />} onClick={() => playMusic(record, queue.slice(idx))} style={{ color: colors.primary }} />
            </Tooltip>
            {inPlaylist && (
              <Tooltip title="移出歌单">
                <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => handleRemoveTrack(record)} />
              </Tooltip>
            )}
          </Space>
        )
      },
    },
  ]

  const exportMenuItems: MenuProps['items'] = [
    { key: 'json', label: '导出 JSON', onClick: () => handleExport('json') },
    { key: 'm3u', label: '导出 M3U', onClick: () => handleExport('m3u') },
  ]

  const mainTitle = view === 'playlist' && currentPlaylist ? currentPlaylist.name : view === 'liked' ? '我喜欢' : view === 'recent' ? '最近播放' : view === 'search' ? `搜索：${search.trim()}` : view === 'library' ? '全部音乐' : '发现音乐'
  const mainDescription = view === 'playlist'
    ? `${playlistTracks.length} 首 · ${formatTotalDuration(playlistTracks.reduce((sum, track) => sum + (track.duration || 0), 0))}`
    : view === 'liked'
      ? `${likedTracks.length} 首收藏歌曲`
      : view === 'recent'
        ? `${recentTracks.length} 首最近播放`
        : view === 'search'
          ? `${searchedTracks.length} 首歌曲 · ${searchedPlaylists.length} 个歌单`
          : '公共曲库和你的云盘音频会统一出现在这里。'

  const sideItems = [
    { key: 'discover', title: '发现音乐', icon: <FireOutlined /> },
    { key: 'library', title: '全部音乐', icon: <CustomerServiceOutlined /> },
    { key: 'liked', title: '我喜欢', icon: <HeartOutlined /> },
    { key: 'recent', title: '最近播放', icon: <ClockCircleOutlined /> },
  ]

  const renderTrackCards = (items: Track[]) => (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: spacing.md }}>
      {items.map((track, index) => {
        const liked = likedKeys.has(trackKey(track))
        return (
          <div key={trackKey(track)} onDoubleClick={() => playMusic(track, items)} style={{ border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.md, padding: spacing.md, background: colors.surface, cursor: 'pointer', minHeight: 140, display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
            <Space align="start" style={{ justifyContent: 'space-between', width: '100%' }}>
              <Avatar size={46} icon={<CustomerServiceOutlined />} style={{ background: colors.primaryLight, color: colors.primary }} />
              <Text type="secondary" style={{ fontSize: 12 }}>{String(index + 1).padStart(2, '0')}</Text>
            </Space>
            <div>
              <div style={{ fontWeight: 750, fontSize: 15, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{track.title}</div>
              <Text type="secondary" style={{ fontSize: 12 }}>{track.artist || '未知歌手'}</Text>
            </div>
            <Space style={{ justifyContent: 'space-between', width: '100%' }}>
              <Tag color={track.source === 'public' ? 'orange' : 'green'}>{sourceLabel(track)}</Tag>
              <Space size={4}>
                <Button type="text" icon={liked ? <HeartFilled /> : <HeartOutlined />} onClick={() => toggleLike(track)} style={{ color: liked ? colors.error : colors.textSecondary }} />
                <Button type="text" icon={<PlayCircleOutlined />} onClick={() => playMusic(track, items)} style={{ color: colors.primary }} />
              </Space>
            </Space>
          </div>
        )
      })}
    </div>
  )

  return (
    <div>
      <PageHeader
        eyebrow="Music"
        title="音乐中心"
        description="集中发现、搜索、收藏、回看最近播放，并管理自己的歌单。"
        actions={<Space wrap>{isAdmin && <Button icon={<UploadOutlined />} onClick={() => setUploadOpen(true)}>上传公共歌曲</Button>}<Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>新建歌单</Button></Space>}
      />

      <MetricStrip items={[
        { label: '曲库歌曲', value: tracks.length, tone: 'primary' },
        { label: '我喜欢', value: likedTracks.length },
        { label: '最近播放', value: recentTracks.length, tone: 'success' },
        { label: '歌单收录', value: libraryStats.playlistTrackCount, tone: 'warning' },
      ]} />

      <Layout style={{ background: 'transparent', gap: spacing.md }}>
        <Sider width={260} breakpoint="lg" collapsedWidth={0} style={{ background: colors.surfaceRaised, border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.lg, boxShadow: shadow.card, overflow: 'hidden' }}>
          <div style={{ padding: spacing.md, borderBottom: `1px solid ${colors.borderSubtle}` }}>
            <Input prefix={<SearchOutlined />} placeholder="搜索歌曲 / 歌单" value={search} onChange={(event) => setGlobalSearch(event.target.value)} allowClear />
          </div>
          <div style={{ padding: spacing.sm }}>
            <List
              size="small"
              dataSource={sideItems}
              renderItem={(item) => (
                <List.Item
                  onClick={() => {
                    setView(item.key as ViewKey)
                    setSelectedPlaylistId(null)
                    setSearch('')
                  }}
                  style={{ cursor: 'pointer', borderRadius: radius.md, padding: '10px 12px', border: 0, background: view === item.key ? colors.primaryLight : 'transparent', color: view === item.key ? colors.primary : colors.text, fontWeight: view === item.key ? 700 : 500 }}
                >
                  <Space>{item.icon}<span>{item.title}</span></Space>
                </List.Item>
              )}
            />
          </div>
          <div style={{ padding: `0 ${spacing.md}px ${spacing.sm}px`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>我的歌单</Text>
            <Tooltip title="新建歌单"><Button type="text" size="small" icon={<PlusOutlined />} onClick={openCreateModal} /></Tooltip>
          </div>
          <List
            size="small"
            loading={playlistLoading && playlists.length === 0}
            dataSource={playlists}
            locale={{ emptyText: '还没有歌单' }}
            style={{ padding: `0 ${spacing.sm}px ${spacing.md}px` }}
            renderItem={(playlist) => (
              <List.Item onClick={() => showPlaylist(playlist)} actions={[<Button key="delete" type="text" size="small" danger icon={<DeleteOutlined />} onClick={(event) => { event.stopPropagation(); handleDeletePlaylist(playlist) }} />]} style={{ cursor: 'pointer', borderRadius: radius.md, padding: '9px 8px', border: 0, background: selectedPlaylistId === playlist.id ? colors.primaryLight : 'transparent' }}>
                <List.Item.Meta avatar={<Avatar size={32} icon={<AppstoreOutlined />} style={{ background: colors.primaryLight, color: colors.primary }} />} title={<Text ellipsis style={{ maxWidth: 132, fontWeight: 650 }}>{playlist.name}</Text>} description={<Text type="secondary" style={{ fontSize: 12 }}>{playlist.track_count || 0} 首</Text>} />
              </List.Item>
            )}
          />
        </Sider>

        <Content>
          <div style={{ minHeight: 620, background: colors.surfaceRaised, border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.lg, boxShadow: shadow.card, overflow: 'hidden' }}>
            <div style={{ padding: spacing.md, borderBottom: `1px solid ${colors.borderSubtle}` }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
                <div>
                  <Title level={4} style={{ margin: 0 }}>{mainTitle}</Title>
                  <Text type="secondary">{mainDescription}</Text>
                </div>
                <Space wrap>
                  {view === 'playlist' && currentPlaylist && (
                    <>
                      <Space size={6}><Text type="secondary" style={{ fontSize: 13 }}>公开</Text><Switch size="small" checked={!!currentPlaylist.is_public} onChange={handleTogglePublic} /></Space>
                      <Button icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>添加歌曲</Button>
                      <Dropdown menu={{ items: exportMenuItems }} placement="bottomRight"><Button icon={<ExportOutlined />}>导出</Button></Dropdown>
                      <Upload accept=".json,.m3u" showUploadList={false} beforeUpload={handleImport}><Button icon={<ImportOutlined />} loading={importing}>导入</Button></Upload>
                    </>
                  )}
                  {view !== 'playlist' && view !== 'liked' && view !== 'recent' && (
                    <>
                      <Segmented value={source} onChange={(value) => setSource(value as LibrarySource)} options={[{ value: 'all', label: '全部' }, { value: 'public', label: '公共' }, { value: 'cloud', label: '云盘' }]} />
                      <Button icon={<ReloadOutlined />} onClick={() => fetchLibrary(source)}>刷新</Button>
                    </>
                  )}
                </Space>
              </div>
              <div style={{ marginTop: spacing.md, display: 'flex', gap: spacing.sm, flexWrap: 'wrap' }}>
                <Button type="primary" icon={<PlayCircleOutlined />} disabled={selectedQueue.length === 0} onClick={() => selectedQueue.length > 0 && playMusic(selectedQueue[0], selectedQueue)}>播放全部</Button>
                {view !== 'playlist' && view !== 'liked' && view !== 'recent' && view !== 'search' && (
                  <Select value={source} onChange={(value) => setSource(value)} style={{ width: 132 }} options={[{ value: 'all', label: '全部来源' }, { value: 'public', label: '公共曲库' }, { value: 'cloud', label: '我的云盘' }]} suffixIcon={source === 'cloud' ? <CloudOutlined /> : undefined} />
                )}
              </div>
            </div>

            <div style={{ padding: spacing.md }}>
              {view === 'discover' ? (
                libraryLoading ? <div style={{ textAlign: 'center', padding: 80 }}><Spin /></div> : heroTracks.length === 0 ? <Empty description="没有找到音乐" /> : (
                  <Space direction="vertical" size={spacing.lg} style={{ width: '100%' }}>
                    <section><Title level={5} style={{ marginTop: 0 }}>今日推荐</Title>{renderTrackCards(heroTracks)}</section>
                    <section><Title level={5}>最近播放</Title>{recentTracks.length === 0 ? <Empty description="播放歌曲后会出现在这里" /> : renderTrackCards(recentTracks.slice(0, 6))}</section>
                  </Space>
                )
              ) : view === 'search' ? (
                <Space direction="vertical" size={spacing.lg} style={{ width: '100%' }}>
                  {searchedPlaylists.length > 0 && (
                    <section>
                      <Title level={5} style={{ marginTop: 0 }}>歌单</Title>
                      <List dataSource={searchedPlaylists} renderItem={(playlist) => (
                        <List.Item onClick={() => showPlaylist(playlist)} style={{ cursor: 'pointer', borderRadius: radius.md, padding: '10px 8px' }}>
                          <List.Item.Meta avatar={<Avatar icon={<AppstoreOutlined />} style={{ background: colors.primaryLight, color: colors.primary }} />} title={playlist.name} description={`${playlist.track_count || 0} 首 · ${playlist.is_public ? '公开' : '私密'}`} />
                        </List.Item>
                      )} />
                    </section>
                  )}
                  <section>
                    <Title level={5} style={{ marginTop: 0 }}>歌曲</Title>
                    {searchedTracks.length === 0 ? <Empty description="没有找到歌曲" /> : (
                      <Table dataSource={searchedTracks} columns={makeTrackColumns(searchedTracks)} rowKey={trackKey} size="small" pagination={{ pageSize: 50, size: 'small', showSizeChanger: false }} onRow={(record, index) => ({ onDoubleClick: () => playMusic(record, searchedTracks.slice(index || 0)) })} style={{ cursor: 'pointer' }} />
                    )}
                  </section>
                </Space>
              ) : (libraryLoading || playlistLoading) && selectedQueue.length === 0 ? (
                <div style={{ textAlign: 'center', padding: 80 }}><Spin /></div>
              ) : selectedQueue.length === 0 ? (
                <Empty description={view === 'playlist' ? '这个歌单还没有歌曲' : view === 'liked' ? '还没有喜欢的歌曲' : view === 'recent' ? '还没有最近播放' : '没有找到音乐'} />
              ) : (
                <Table dataSource={selectedQueue} columns={makeTrackColumns(selectedQueue, view === 'playlist')} rowKey={trackKey} size="small" pagination={{ pageSize: 50, size: 'small', showSizeChanger: false }} onRow={(record, index) => ({ onDoubleClick: () => playMusic(record, selectedQueue.slice(index || 0)) })} style={{ cursor: 'pointer' }} />
              )}
            </div>
          </div>
        </Content>
      </Layout>

      <Modal title="新建歌单" open={createOpen} onOk={handleCreate} onCancel={() => setCreateOpen(false)} confirmLoading={creating} okText="创建" cancelText="取消">
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Input placeholder="歌单名称" value={playlistName} onChange={(event) => setPlaylistName(event.target.value)} maxLength={200} onPressEnter={handleCreate} />
          <Space><Switch checked={playlistPublic} onChange={setPlaylistPublic} /><Text>公开歌单</Text></Space>
        </Space>
      </Modal>

      <Modal title="添加歌曲" open={addOpen} onCancel={() => setAddOpen(false)} footer={null} width={720}>
        <Input prefix={<SearchOutlined />} placeholder="搜索歌曲、歌手、专辑" value={addSearch} onChange={(event) => setAddSearch(event.target.value)} allowClear style={{ marginBottom: spacing.md }} />
        <div style={{ maxHeight: 440, overflowY: 'auto' }}>
          {addableTracks.length === 0 ? <Empty description="没有找到歌曲" /> : addableTracks.map((track) => {
            const key = trackKey(track)
            const exists = existingPlaylistKeys.has(key)
            return (
              <div key={key} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: spacing.sm, padding: '10px 8px', borderBottom: `1px solid ${colors.borderSubtle}`, background: exists ? colors.surfaceMuted : 'transparent' }}>
                {renderTrackTitle(track.title, track)}
                <Space><Tag color={track.source === 'public' ? 'orange' : 'green'}>{sourceLabel(track)}</Tag>{exists ? <Text type="secondary">已添加</Text> : <Button type="link" loading={addingKey === key} onClick={() => handleAddTrack(track)}>添加</Button>}</Space>
              </div>
            )
          })}
        </div>
      </Modal>

      <Modal title="上传公共歌曲" open={uploadOpen} onOk={handleUpload} onCancel={resetUploadState} confirmLoading={uploading} okText="上传" cancelText="取消">
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Upload {...uploadProps}><Button icon={<UploadOutlined />}>选择音频文件</Button></Upload>
          {uploadFile && <Text type="secondary" style={{ fontSize: 12 }}>已选择：{uploadFile.name}</Text>}
          <Input placeholder="标题（可选）" value={uploadTitle} onChange={(event) => setUploadTitle(event.target.value)} />
          <Input placeholder="歌手（可选）" value={uploadArtist} onChange={(event) => setUploadArtist(event.target.value)} />
          <Input placeholder="专辑（可选）" value={uploadAlbum} onChange={(event) => setUploadAlbum(event.target.value)} />
        </Space>
      </Modal>
    </div>
  )
}
