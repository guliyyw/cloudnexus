import { useEffect, useRef, useState } from 'react'
import { Modal, Upload, Button, Progress, message, List, Tag, Space } from 'antd'
import { FolderOutlined, InboxOutlined, WarningOutlined } from '@ant-design/icons'
import { useFileStore } from '../stores/fileStore'
import useChunkUpload from '../hooks/useChunkUpload'
import * as fileApi from '../services/file'
import { formatFileSize } from '../utils/format'
import { colors } from '../theme/tokens'

const CHUNK_THRESHOLD = 100 * 1024 * 1024

interface FileEntry {
  uid: string
  file: File
}

interface Props {
  open: boolean
  targetDirId: string
  targetDirName: string
  onClose: () => void
  accept?: string
  onUploaded?: (files: fileApi.FileItem[]) => void | Promise<void>
}

export default function UploadModal({ open, targetDirId, targetDirName, onClose, accept, onUploaded }: Props) {
  const { upload, breadcrumb, currentParentId } = useFileStore()
  const chunkUpload = useChunkUpload()
  const [entries, setEntries] = useState<FileEntry[]>([])
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [resumeList, setResumeList] = useState<fileApi.ChunkUploadInfo[]>([])
  const [showResume, setShowResume] = useState(false)
  const [resumingId, setResumingId] = useState<string | null>(null)
  const uidCounter = useRef(0)

  const parentId = targetDirId !== undefined ? targetDirId : currentParentId
  const dirLabel = targetDirName || breadcrumb[breadcrumb.length - 1]?.name || '根目录'

  useEffect(() => {
    if (open) {
      fileApi.listIncompleteUploads().then((uploads) => {
        if (uploads.length > 0) setResumeList(uploads)
      }).catch(() => {})
    } else {
      setResumeList([])
      setShowResume(false)
      setResumingId(null)
    }
  }, [open])

  const reset = () => {
    setEntries([])
    setProgress(0)
    setResumingId(null)
    chunkUpload.reset()
  }

  const handleSmallUpload = async () => {
    if (entries.length === 0) return
    setUploading(true)
    setProgress(0)
    try {
      const files = entries.map((entry) => entry.file)
      const result = await upload(files, parentId, (pct) => setProgress(pct))
      if (result.errors?.length > 0) {
        message.warning(`部分文件上传失败：${result.errors.join(', ')}`)
      } else {
        message.success(`成功上传 ${result.ok} 个文件`)
      }
      await onUploaded?.(result.files || [])
      reset()
      onClose()
    } catch (error: any) {
      message.error(error.response?.data?.message || '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const handleChunkUpload = async () => {
    if (entries.length === 0) return
    setUploading(true)
    try {
      const file = entries[0].file
      await chunkUpload.startUpload(file, parentId, async (item) => {
        message.success(`文件 ${file.name} 上传完成`)
        await onUploaded?.([item])
        reset()
        onClose()
      })
    } catch (error: any) {
      message.error(error.response?.data?.message || '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const handleCancelResume = async (uploadId: string) => {
    try {
      setResumingId(uploadId)
      await fileApi.cancelChunkUpload(uploadId)
      setResumeList((prev) => prev.filter((item) => item.upload_id !== uploadId))
      message.success('已取消未完成的上传')
    } catch {
      message.error('取消失败')
    } finally {
      setResumingId(null)
    }
  }

  const handleUpload = () => {
    const hasLarge = entries.some((entry) => entry.file.size > CHUNK_THRESHOLD)
    if (hasLarge && entries.length > 1) {
      message.warning('大文件请单独上传')
      return
    }
    if (hasLarge) {
      handleChunkUpload()
    } else {
      handleSmallUpload()
    }
  }

  const effectiveProgress = chunkUpload.active ? chunkUpload.progress.percent : progress
  const isChunkActive = !!chunkUpload.active

  const handleClose = () => {
    if (!uploading) {
      reset()
      onClose()
    }
  }

  return (
    <>
      <Modal
        title={<span>上传到 <FolderOutlined style={{ color: '#faad14' }} /> {dirLabel}</span>}
        open={open && !showResume}
        onCancel={handleClose}
        footer={[
          <Button key="cancel" onClick={handleClose} disabled={uploading}>取消</Button>,
          ...(isChunkActive && uploading ? [
            <Button key="cancelChunk" danger onClick={() => { chunkUpload.cancel(); setUploading(false); reset() }}>
              取消上传
            </Button>,
          ] : []),
          <Button key="upload" type="primary" onClick={handleUpload} loading={uploading && !isChunkActive} disabled={entries.length === 0 || uploading}>
            上传 ({entries.length})
          </Button>,
        ]}
        width={560}
        maskClosable={false}
      >
        <Upload.Dragger
          accept={accept}
          multiple
          showUploadList
          fileList={entries.map((entry) => ({
            uid: entry.uid,
            name: entry.file.name,
            size: entry.file.size,
            status: 'done' as const,
          }))}
          beforeUpload={(file) => {
            uidCounter.current += 1
            const uid = `upload-${uidCounter.current}-${file.name}`
            setEntries((prev) => [...prev, { uid, file }])
            return false
          }}
          onRemove={(file) => {
            setEntries((prev) => prev.filter((entry) => entry.uid !== file.uid))
          }}
          disabled={uploading}
        >
          <p className="ant-upload-drag-icon"><InboxOutlined /></p>
          <p className="ant-upload-text">点击或拖拽文件到这里</p>
          <p className="ant-upload-hint">支持批量上传；超过 100MB 的文件会自动使用分片上传。</p>
        </Upload.Dragger>

        {uploading && (
          <div style={{ marginTop: 16 }}>
            {isChunkActive ? (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span style={{ fontSize: 13, color: colors.textSecondary }}>分片上传中：{chunkUpload.active?.fileName}</span>
                  <span style={{ fontSize: 13, color: colors.textSecondary }}>{chunkUpload.progress.completed} / {chunkUpload.progress.total} 片</span>
                </div>
                <Progress percent={chunkUpload.progress.percent} status="active" strokeColor={{ from: colors.primary, to: 'rgba(129,236,254,0.2)' }} />
                {chunkUpload.progress.currentChunk >= 0 && (
                  <div style={{ fontSize: 11, color: colors.textSecondary, marginTop: 2 }}>
                    已完成分片 {chunkUpload.progress.currentChunk + 1}
                  </div>
                )}
              </div>
            ) : (
              <Progress percent={effectiveProgress} status="active" />
            )}
          </div>
        )}
      </Modal>

      <Modal
        title={<span><WarningOutlined style={{ color: '#faad14' }} /> 检测到未完成的上传</span>}
        open={showResume}
        onCancel={() => { setShowResume(false); setResumeList([]) }}
        footer={[
          <Button key="ignore" onClick={() => { setShowResume(false); setResumeList([]) }}>忽略</Button>,
        ]}
        width={500}
      >
        <p style={{ color: colors.textSecondary, marginBottom: 12 }}>以下文件上传未完成，可以取消它们后重新上传。</p>
        <List
          dataSource={resumeList}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button key="cancel" size="small" danger loading={resumingId === item.upload_id} onClick={() => handleCancelResume(item.upload_id)}>
                  取消
                </Button>,
              ]}
            >
              <List.Item.Meta
                title={item.file_name}
                description={
                  <Space size={4}>
                    <Tag color="orange">未完成</Tag>
                    <span>{formatFileSize(item.file_size)}</span>
                    <span>· 已完成 {item.completed?.length || 0}/{item.total_chunks} 片</span>
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      </Modal>
    </>
  )
}
