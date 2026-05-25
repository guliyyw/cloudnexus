import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Typography, Button, Space, Tabs, Spin, Modal } from 'antd'
import {
  ArrowLeftOutlined,
  AppstoreOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  FolderOutlined,
} from '@ant-design/icons'
import AlbumGrid from '../components/album/AlbumGrid'
import AlbumTimeline from '../components/album/AlbumTimeline'
import AlbumFolder from '../components/album/AlbumFolder'
import Lightbox from '../components/album/Lightbox'
import { useAlbumStore } from '../stores/albumStore'
import type { FileItem } from '../services/file'

const { Title, Text } = Typography

export default function AlbumDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentAlbum, files, filesLoading, fetchAlbum, fetchFiles, removeFile } = useAlbumStore()
  const [viewMode, setViewMode] = useState<string>('grid')
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)

  useEffect(() => {
    if (id) {
      fetchAlbum(id)
      fetchFiles(id)
    }
  }, [id, fetchAlbum, fetchFiles])

  const handleSelect = useCallback((fileId: string, checked: boolean) => {
    setSelectedIds((prev) =>
      checked ? [...prev, fileId] : prev.filter((f) => f !== fileId)
    )
  }, [])

  const handlePreview = useCallback((_file: FileItem, index: number) => {
    setLightboxIndex(index)
  }, [])

  const handleBatchDelete = () => {
    if (!id || !selectedIds.length) return
    Modal.confirm({
      title: '移出相册',
      content: `确定要将 ${selectedIds.length} 个文件移出相册吗？`,
      okText: '确定',
      cancelText: '取消',
      onOk: async () => {
        for (const fileId of selectedIds) {
          await removeFile(id, fileId)
        }
        setSelectedIds([])
      },
    })
  }

  const imageFiles = files.filter((f) => f.mime_type?.startsWith('image/') || f.mime_type?.startsWith('video/'))

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <Space>
          <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/album')} />
          <Title level={4} style={{ margin: 0 }}>{currentAlbum?.name || '相册详情'}</Title>
          {currentAlbum?.description && (
            <Text type="secondary" style={{ fontSize: 13 }}>{currentAlbum.description}</Text>
          )}
        </Space>
        <Space>
          {selectedIds.length > 0 && (
            <Button icon={<DeleteOutlined />} danger onClick={handleBatchDelete}>
              移出相册 ({selectedIds.length})
            </Button>
          )}
          <Tabs
            activeKey={viewMode}
            onChange={setViewMode}
            size="small"
            style={{ marginBottom: 0 }}
            items={[
              { key: 'grid', label: <span><AppstoreOutlined style={{ marginRight: 4 }} />网格</span> },
              { key: 'timeline', label: <span><ClockCircleOutlined style={{ marginRight: 4 }} />时间线</span> },
              { key: 'folder', label: <span><FolderOutlined style={{ marginRight: 4 }} />文件夹</span> },
            ]}
          />
        </Space>
      </div>

      {filesLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : (
        <>
          {viewMode === 'grid' && (
            <AlbumGrid
              files={imageFiles}
              selectedIds={selectedIds}
              onSelect={handleSelect}
              onPreview={handlePreview}
            />
          )}
          {viewMode === 'timeline' && (
            <AlbumTimeline
              files={imageFiles}
              onPreview={handlePreview}
            />
          )}
          {viewMode === 'folder' && (
            <AlbumFolder files={imageFiles} onPreview={handlePreview} />
          )}
          {imageFiles.length === 0 && (
            <div style={{ textAlign: 'center', padding: 60, color: '#888' }}>
              相册中还没有内容
            </div>
          )}
        </>
      )}

      {lightboxIndex !== null && lightboxIndex < imageFiles.length && (
        <Lightbox
          files={imageFiles}
          currentIndex={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
        />
      )}
    </div>
  )
}
