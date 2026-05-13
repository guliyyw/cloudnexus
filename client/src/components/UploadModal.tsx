import { useState, useRef } from 'react'
import { Modal, Upload, Button, Progress, message } from 'antd'
import { InboxOutlined, FolderOutlined } from '@ant-design/icons'
import { useFileStore } from '../stores/fileStore'

interface FileEntry {
  uid: string
  file: File
}

interface Props {
  open: boolean
  targetDirId: string
  targetDirName: string
  onClose: () => void
}

export default function UploadModal({ open, targetDirId, targetDirName, onClose }: Props) {
  const { upload, breadcrumb, currentParentId } = useFileStore()
  const [entries, setEntries] = useState<FileEntry[]>([])
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const uidCounter = useRef(0)

  const parentId = targetDirId !== undefined ? targetDirId : currentParentId
  const dirLabel = targetDirName || breadcrumb[breadcrumb.length - 1]?.name || '根目录'

  const reset = () => {
    setEntries([])
    setProgress(0)
  }

  const handleUpload = async () => {
    if (entries.length === 0) return
    setUploading(true)
    setProgress(0)
    try {
      const files = entries.map((e) => e.file)
      const result = await upload(files, parentId, (pct) => setProgress(pct))
      if (result.errors?.length > 0) {
        message.warning(`部分文件上传失败: ${result.errors.join(', ')}`)
      } else {
        message.success(`成功上传 ${result.ok} 个文件`)
      }
      reset()
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
      onCancel={() => { if (!uploading) { reset(); onClose() } }}
      footer={[
        <Button key="cancel" onClick={() => { reset(); onClose() }} disabled={uploading}>
          取消
        </Button>,
        <Button key="upload" type="primary" onClick={handleUpload} loading={uploading} disabled={entries.length === 0}>
          上传 ({entries.length})
        </Button>,
      ]}
      width={560}
      maskClosable={false}
    >
      <Upload.Dragger
        multiple
        showUploadList={true}
        fileList={entries.map((e) => ({
          uid: e.uid,
          name: e.file.name,
          size: e.file.size,
          status: 'done' as const,
        }))}
        beforeUpload={(file) => {
          uidCounter.current += 1
          const uid = `upload-${uidCounter.current}-${file.name}`
          setEntries((prev) => [...prev, { uid, file }])
          return false
        }}
        onRemove={(f) => {
          setEntries((prev) => prev.filter((e) => e.uid !== f.uid))
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
