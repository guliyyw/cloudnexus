import { useState } from 'react'
import { Modal, Upload, Button, Progress, message } from 'antd'
import { InboxOutlined, FolderOutlined } from '@ant-design/icons'
import { useFileStore } from '../stores/fileStore'

interface Props {
  open: boolean
  targetDirId: number
  targetDirName: string
  onClose: () => void
}

export default function UploadModal({ open, targetDirId, targetDirName, onClose }: Props) {
  const { upload, breadcrumb, currentParentId } = useFileStore()
  const [files, setFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)

  // The actual parent to upload into: if targetDirId provided, use it, else current dir
  const parentId = targetDirId !== undefined ? targetDirId : currentParentId
  const dirLabel = targetDirName || breadcrumb[breadcrumb.length - 1]?.name || '根目录'

  const handleUpload = async () => {
    if (files.length === 0) return
    setUploading(true)
    setProgress(0)
    try {
      const result = await upload(files, parentId, (pct) => setProgress(pct))
      if (result.errors?.length > 0) {
        message.warning(`部分文件上传失败: ${result.errors.join(', ')}`)
      } else {
        message.success(`成功上传 ${result.ok} 个文件`)
      }
      setFiles([])
      setProgress(0)
      onClose()
    } catch {
      message.error('上传失败')
    } finally {
      setUploading(false)
    }
  }

  return (
    <Modal
      title={
        <span>
          上传文件到 <FolderOutlined style={{ color: '#faad14' }} /> {dirLabel}
        </span>
      }
      open={open}
      onCancel={() => { if (!uploading) { setFiles([]); onClose() } }}
      footer={[
        <Button key="cancel" onClick={() => { setFiles([]); onClose() }} disabled={uploading}>
          取消
        </Button>,
        <Button key="upload" type="primary" onClick={handleUpload} loading={uploading} disabled={files.length === 0}>
          上传 ({files.length})
        </Button>,
      ]}
      width={560}
      maskClosable={false}
    >
      <Upload.Dragger
        multiple
        showUploadList={true}
        fileList={files.map((f, i) => ({
          uid: `${Date.now()}-${i}-${f.name}`,
          name: f.name,
          size: f.size,
          status: 'done' as const,
        }))}
        beforeUpload={(file) => {
          setFiles((prev) => [...prev, file])
          return false
        }}
        onRemove={(f) => {
          setFiles((prev) => prev.filter((_, i) => `${Date.now()}-${i}-${prev[i].name}` !== f.uid))
        }}
        disabled={uploading}
      >
        <p className="ant-upload-drag-icon"><InboxOutlined /></p>
        <p className="ant-upload-text">点击或拖拽文件到此区域</p>
        <p className="ant-upload-hint">支持批量上传，单个文件最大 100MB</p>
      </Upload.Dragger>

      {uploading && (
        <div style={{ marginTop: 16 }}>
          <Progress percent={progress} status="active" />
        </div>
      )}
    </Modal>
  )
}
