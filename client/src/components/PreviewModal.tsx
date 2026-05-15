import { useState, useEffect } from 'react'
import { Modal, Image, Button, Dropdown, Space, List, message, Typography } from 'antd'
import { EllipsisOutlined, PictureOutlined, CustomerServiceOutlined } from '@ant-design/icons'
import type { FileItem } from '../services/file'
import { getPreviewUrl, getDownloadUrl } from '../services/file'
import { getAlbums, addFilesToAlbum } from '../services/album'
import type { Album } from '../services/album'
import { usePlayerStore } from '../stores/playerStore'
import type { Track } from '../services/music'

const { Text } = Typography

interface Props {
  file: FileItem | null
  open: boolean
  onClose: () => void
}

function fileToTrack(f: FileItem): Track {
  return {
    id: f.id,
    title: f.name,
    artist: '',
    album: '',
    duration: 0,
    source: 'cloud',
    mime_type: f.mime_type,
    file_size: f.size,
  }
}

export default function PreviewModal({ file, open, onClose }: Props) {
  const [albumPickerOpen, setAlbumPickerOpen] = useState(false)
  const [albums, setAlbums] = useState<Album[]>([])
  const [loadingAlbums, setLoadingAlbums] = useState(false)
  const play = usePlayerStore((s) => s.play)

  useEffect(() => {
    if (albumPickerOpen) {
      setLoadingAlbums(true)
      getAlbums(1, 100)
        .then((res) => setAlbums(res.albums))
        .catch(() => message.error('加载相册失败'))
        .finally(() => setLoadingAlbums(false))
    }
  }, [albumPickerOpen])

  const handleAddToAlbum = async (albumId: string) => {
    if (!file) return
    try {
      await addFilesToAlbum(albumId, [file.id])
      message.success('已添加到相册')
      setAlbumPickerOpen(false)
    } catch {
      message.error('添加失败')
    }
  }

  const handlePlayAudio = () => {
    if (!file) return
    play(fileToTrack(file))
    onClose()
  }

  if (!file) return null

  const isImage = file.mime_type?.startsWith('image/')
  const isVideo = file.mime_type?.startsWith('video/')
  const isAudio = file.mime_type?.startsWith('audio/')
  const isPdf = file.mime_type === 'application/pdf'

  const moreItems = []
  if (isImage) {
    moreItems.push({ key: 'album', icon: <PictureOutlined />, label: '添加到相册', onClick: () => setAlbumPickerOpen(true) })
  }
  if (isAudio) {
    moreItems.push({ key: 'play', icon: <CustomerServiceOutlined />, label: '在音乐中播放', onClick: handlePlayAudio })
  }

  return (
    <>
      <Modal
        title={
          <Space>
            <span>{file.name}</span>
            {moreItems.length > 0 && (
              <Dropdown menu={{ items: moreItems.map((item) => ({ key: item.key, icon: item.icon, label: item.label, onClick: item.onClick })) }}>
                <Button type="text" size="small" icon={<EllipsisOutlined />} />
              </Dropdown>
            )}
          </Space>
        }
        open={open}
        onCancel={onClose}
        footer={
          <Button href={getDownloadUrl(file.id)} download={file.name}>下载</Button>
        }
        width={isImage ? 'auto' : 800}
        centered
        destroyOnClose
      >
        {isImage && (
          <Image src={getPreviewUrl(file.id)} alt={file.name} style={{ maxHeight: '70vh' }} />
        )}
        {isVideo && (
          <video controls style={{ width: '100%', maxHeight: '70vh' }}>
            <source src={getPreviewUrl(file.id)} type={file.mime_type} />
          </video>
        )}
        {isAudio && (
          <audio controls style={{ width: '100%' }}>
            <source src={getPreviewUrl(file.id)} type={file.mime_type} />
          </audio>
        )}
        {isPdf && (
          <iframe src={getPreviewUrl(file.id)} style={{ width: '100%', height: '70vh', border: 'none' }} />
        )}
        {!isImage && !isVideo && !isAudio && !isPdf && (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            <p>此文件类型不支持在线预览</p>
            <Button type="link" href={getDownloadUrl(file.id)} download={file.name}>点击下载</Button>
          </div>
        )}
      </Modal>

      <Modal
        title="选择相册"
        open={albumPickerOpen}
        onCancel={() => setAlbumPickerOpen(false)}
        footer={null}
        width={400}
      >
        <List
          loading={loadingAlbums}
          dataSource={albums}
          locale={{ emptyText: '暂无相册' }}
          renderItem={(album) => (
            <List.Item
              style={{ cursor: 'pointer', padding: '8px 12px', borderRadius: 6 }}
              onClick={() => handleAddToAlbum(album.id)}
            >
              <List.Item.Meta
                avatar={<PictureOutlined style={{ fontSize: 20 }} />}
                title={<Text strong>{album.name}</Text>}
                description={`${album.file_count} 个文件`}
              />
            </List.Item>
          )}
        />
      </Modal>
    </>
  )
}
