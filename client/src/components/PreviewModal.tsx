import { Modal, Image } from 'antd'
import type { FileItem } from '../services/file'
import { getPreviewUrl, getDownloadUrl } from '../services/file'

interface Props {
  file: FileItem | null
  open: boolean
  onClose: () => void
}

export default function PreviewModal({ file, open, onClose }: Props) {
  if (!file) return null

  const isImage = file.mime_type?.startsWith('image/')
  const isVideo = file.mime_type?.startsWith('video/')
  const isAudio = file.mime_type?.startsWith('audio/')
  const isPdf = file.mime_type === 'application/pdf'

  return (
    <Modal
      title={file.name}
      open={open}
      onCancel={onClose}
      footer={null}
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
          <a href={getDownloadUrl(file.id)} download={file.name}>点击下载</a>
        </div>
      )}
    </Modal>
  )
}
