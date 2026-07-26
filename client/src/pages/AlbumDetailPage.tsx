import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Button, Modal, Spin, Tabs, message } from 'antd'
import {
  AppstoreOutlined,
  ArrowLeftOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  DownloadOutlined,
  FolderOutlined,
  PlusOutlined,
  StarOutlined,
} from '@ant-design/icons'
import AlbumFolder from '../components/album/AlbumFolder'
import AlbumGrid from '../components/album/AlbumGrid'
import AlbumTimeline from '../components/album/AlbumTimeline'
import Lightbox from '../components/album/Lightbox'
import FilePickerModal from '../components/FilePickerModal'
import UploadModal from '../components/UploadModal'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { useAlbumStore } from '../stores/albumStore'
import { batchDownloadFiles, type FileItem } from '../services/file'
import { colors, spacing } from '../theme/tokens'

const mediaExtensions = /\.(jpe?g|png|gif|webp|bmp|svg|heic|heif|mp4|mov|m4v|webm|avi|mkv)$/i

function isAlbumMedia(file: FileItem) {
  const mimeType = file.mime_type || ''
  return mimeType.startsWith('image/') || mimeType.startsWith('video/') || mediaExtensions.test(file.name || '')
}

export default function AlbumDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentAlbum, files, filesLoading, fetchAlbum, fetchFiles, addFiles, removeFile, updateAlbum } = useAlbumStore()
  const [viewMode, setViewMode] = useState<string>('grid')
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [uploadOpen, setUploadOpen] = useState(false)

  useEffect(() => {
    if (id) {
      fetchAlbum(id)
      fetchFiles(id)
    }
  }, [id, fetchAlbum, fetchFiles])

  const mediaFiles = files.filter(isAlbumMedia)
  const videoCount = mediaFiles.filter((file) => file.mime_type?.startsWith('video/') || /\.(mp4|mov|m4v|webm|avi|mkv)$/i.test(file.name || '')).length
  const imageCount = mediaFiles.length - videoCount

  const handleSelect = useCallback((fileId: string, checked: boolean) => {
    setSelectedIds((prev) => (checked ? [...prev, fileId] : prev.filter((item) => item !== fileId)))
  }, [])

  const handlePreview = useCallback((_file: FileItem, index: number) => {
    setLightboxIndex(index)
  }, [])

  const handleBatchDelete = () => {
    if (!id || !selectedIds.length) return
    Modal.confirm({
      title: '移出相册',
      content: `确定要将 ${selectedIds.length} 个文件移出相册吗？原文件不会被删除。`,
      okText: '移出',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        for (const fileId of selectedIds) {
          await removeFile(id, fileId)
        }
        await fetchFiles(id)
        await fetchAlbum(id)
        setSelectedIds([])
        message.success('已移出相册')
      },
    })
  }

  const handleAddPickedFiles = async (picked: FileItem[]) => {
    if (!id || !picked.length) return
    const pickedMedia = picked.filter(isAlbumMedia)
    if (!pickedMedia.length) {
      message.warning('请选择图片或视频文件')
      return
    }
    try {
      await addFiles(id, pickedMedia.map((file) => file.id))
      setPickerOpen(false)
      setSelectedIds([])
      message.success(`已添加 ${pickedMedia.length} 个文件`)
    } catch (error: any) {
      message.error(error?.response?.data?.message || '添加到相册失败')
    }
  }

  const handleUploadedFiles = async (uploadedFiles: FileItem[]) => {
    if (!id || !uploadedFiles.length) return
    const uploadedMedia = uploadedFiles.filter(isAlbumMedia)
    if (!uploadedMedia.length) {
      setUploadOpen(false)
      message.warning('已上传，但没有可加入相册的图片或视频')
      return
    }
    await addFiles(id, uploadedMedia.map((file) => file.id))
    await fetchFiles(id)
    await fetchAlbum(id)
    setSelectedIds([])
    setUploadOpen(false)
    message.success(`已上传并加入相册 ${uploadedMedia.length} 个文件`)
  }

  const handleSetCover = async () => {
    if (!id || selectedIds.length !== 1) return
    await updateAlbum(id, { cover_file_id: selectedIds[0] })
    await fetchAlbum(id)
    message.success('已设置相册封面')
  }

  const handleBatchDownload = async () => {
    if (!selectedIds.length) return
    await batchDownloadFiles(selectedIds)
  }

  return (
    <div>
      <PageHeader
        eyebrow="Album"
        title={currentAlbum?.name || '相册详情'}
        actions={
          <>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/album')}>返回相册</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setUploadOpen(true)}>上传图片/视频</Button>
            <Button icon={<PlusOutlined />} onClick={() => setPickerOpen(true)}>从云盘添加</Button>
            {selectedIds.length === 1 && <Button icon={<StarOutlined />} onClick={handleSetCover}>设为封面</Button>}
            {selectedIds.length > 0 && (
              <>
                <Button icon={<DownloadOutlined />} onClick={handleBatchDownload}>下载 {selectedIds.length}</Button>
                <Button icon={<DeleteOutlined />} danger onClick={handleBatchDelete}>移出 {selectedIds.length}</Button>
              </>
            )}
          </>
        }
      />

      <MetricStrip
        items={[
          { label: '全部媒体', value: mediaFiles.length, tone: 'primary' },
          { label: '照片', value: imageCount },
          { label: '视频', value: videoCount, tone: 'success' },
          { label: '已选择', value: selectedIds.length, tone: selectedIds.length ? 'warning' : 'default' },
        ]}
      />

      <Tabs
        activeKey={viewMode}
        onChange={setViewMode}
        size="small"
        style={{ marginBottom: spacing.md }}
        items={[
          { key: 'grid', label: <span><AppstoreOutlined style={{ marginRight: 4 }} />网格</span> },
          { key: 'timeline', label: <span><ClockCircleOutlined style={{ marginRight: 4 }} />时间线</span> },
          { key: 'folder', label: <span><FolderOutlined style={{ marginRight: 4 }} />文件夹</span> },
        ]}
      />

      {filesLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : (
        <>
          {viewMode === 'grid' && <AlbumGrid files={mediaFiles} selectedIds={selectedIds} onSelect={handleSelect} onPreview={handlePreview} />}
          {viewMode === 'timeline' && <AlbumTimeline files={mediaFiles} onPreview={handlePreview} />}
          {viewMode === 'folder' && <AlbumFolder files={mediaFiles} onPreview={handlePreview} />}
          {mediaFiles.length === 0 && <div style={{ textAlign: 'center', padding: 60, color: colors.textSecondary }}>这个相册还没有内容</div>}
        </>
      )}

      {lightboxIndex !== null && lightboxIndex < mediaFiles.length && (
        <Lightbox files={mediaFiles} currentIndex={lightboxIndex} onClose={() => setLightboxIndex(null)} />
      )}

      <FilePickerModal
        open={pickerOpen}
        title="添加图片或视频到相册"
        multiple
        accept={isAlbumMedia}
        onOk={(file) => handleAddPickedFiles([file])}
        onOkMultiple={handleAddPickedFiles}
        onCancel={() => setPickerOpen(false)}
      />

      <UploadModal
        open={uploadOpen}
        targetDirId="0"
        targetDirName="根目录"
        accept="image/*,video/*"
        onUploaded={handleUploadedFiles}
        onClose={() => setUploadOpen(false)}
      />
    </div>
  )
}
