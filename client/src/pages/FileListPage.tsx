import { useEffect, useState } from 'react'
import {
  Table, Button, Modal, Input, Breadcrumb, Popconfirm,
  Space, message, Tag, Typography, Tooltip, Card,
} from 'antd'
import {
  FolderAddOutlined, SearchOutlined,
  DeleteOutlined, DownloadOutlined, FolderOutlined,
  FileOutlined, HomeOutlined, ReloadOutlined, UploadOutlined,
  FileImageOutlined, PlayCircleOutlined, SoundOutlined,
  FilePdfOutlined, FileZipOutlined, EyeOutlined, ShareAltOutlined,
  SwapOutlined, CopyOutlined, HistoryOutlined,
} from '@ant-design/icons'
import { useFileStore } from '../stores/fileStore'
import UploadModal from '../components/UploadModal'
import PreviewModal from '../components/PreviewModal'
import ShareModal from '../components/ShareModal'
import DirectoryPickerModal from '../components/DirectoryPickerModal'
import FileVersionPanel from '../components/FileVersionPanel'
import { isPreviewable } from '../utils/preview'
import { getDownloadUrl } from '../services/file'
import type { FileItem } from '../services/file'
import type { ColumnsType } from 'antd/es/table'

function getFileIcon(mimeType: string, isDir: boolean) {
  if (isDir) return <FolderOutlined style={{ color: '#faad14' }} />
  if (!mimeType) return <FileOutlined />
  if (mimeType.startsWith('image/')) return <FileImageOutlined style={{ color: '#52c41a' }} />
  if (mimeType.startsWith('video/')) return <PlayCircleOutlined style={{ color: '#5b8def' }} />
  if (mimeType.startsWith('audio/')) return <SoundOutlined style={{ color: '#722ed1' }} />
  if (mimeType === 'application/pdf') return <FilePdfOutlined style={{ color: '#ff4d4f' }} />
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('compress')) return <FileZipOutlined />
  return <FileOutlined />
}

const { Text } = Typography

function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

export default function FileListPage() {
  const {
    files, total, page, pageSize, breadcrumb, loading, searchMode, searchKeyword,
    fetchFiles, remove, batchRemove, batchDownload, moveItem, copyItem, mkdir, search, navigateTo, setPage, currentParentId,
  } = useFileStore()

  const [mkdirVisible, setMkdirVisible] = useState(false)
  const [mkdirName, setMkdirName] = useState('')
  const [searchValue, setSearchValue] = useState('')
  const [uploadModalOpen, setUploadModalOpen] = useState(false)
  const [uploadTargetDir, setUploadTargetDir] = useState({ id: '0', name: '根目录' })
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null)
  const [shareFile, setShareFile] = useState<FileItem | null>(null)
  const [versionFile, setVersionFile] = useState<FileItem | null>(null)
  const [dropDirId, setDropDirId] = useState<string | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [pickerOpen, setPickerOpen] = useState<'move' | 'copy' | null>(null)
  const [pendingMoveCopyIds, setPendingMoveCopyIds] = useState<string[]>([])

  useEffect(() => { fetchFiles() }, [])

  // Open upload modal, optionally targeting a specific directory
  const openUploadModal = (dirId = currentParentId, dirName?: string) => {
    const name = dirName || breadcrumb.find((b) => b.id === dirId)?.name || '根目录'
    setUploadTargetDir({ id: dirId, name })
    setUploadModalOpen(true)
  }

  const handleDirDragOver = (e: React.DragEvent, dirId: string) => {
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'copy'
    setDropDirId(dirId)
  }

  const handleDirDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDropDirId(null)
  }

  const handleDirDrop = (e: React.DragEvent, dirId: string, dirName: string) => {
    e.preventDefault()
    e.stopPropagation()
    setDropDirId(null)

    const moveId = e.dataTransfer.getData('application/cloudnexus-move')
    if (moveId) {
      if (moveId === dirId) {
        message.warning('不能将目录移动到自身')
        return
      }
      Modal.confirm({
        title: '确认移动',
        content: `移动到 "${dirName}"？`,
        okText: '移动',
        cancelText: '取消',
        onOk: async () => {
          try {
            await moveItem(moveId, dirId)
            message.success('移动成功')
          } catch {
            message.error('移动失败')
          }
        },
      })
      return
    }

    if (e.dataTransfer.files.length > 0) {
      openUploadModal(dirId, dirName)
    }
  }

  const handleBatchDelete = () => {
    Modal.confirm({
      title: `确认删除 ${selectedRowKeys.length} 个文件？`,
      content: '此操作不可恢复',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const result = await batchRemove(selectedRowKeys)
          setSelectedRowKeys([])
          if (result.errors.length > 0) {
            message.warning(`部分删除失败: ${result.errors.join(', ')}`)
          } else {
            message.success(`已删除 ${result.deleted} 个文件`)
          }
        } catch {
          message.error('批量删除失败，请重试')
        }
      },
    })
  }

  const handleBatchDownload = async () => {
    const fileIds = selectedRowKeys.filter((id) => {
      const f = files.find((item) => item.id === id)
      return f && !f.is_dir
    })
    if (fileIds.length === 0) {
      message.warning('所选均为目录，无可下载文件')
      return
    }
    try {
      await batchDownload(fileIds)
      message.success(`正在下载 ${fileIds.length} 个文件`)
    } catch {
      message.error('批量下载失败')
    }
  }

  const handlePickerOk = async (targetDirId: string) => {
    const ids = pendingMoveCopyIds
    setPickerOpen(null)
    setPendingMoveCopyIds([])

    if (ids.length === 0) return

    if (pickerOpen === 'move') {
      let ok = 0
      const errs: string[] = []
      for (const id of ids) {
        try {
          await moveItem(id, targetDirId)
          ok++
        } catch (e: any) {
          errs.push(e?.response?.data?.message || '未知错误')
        }
      }
      if (errs.length > 0) message.warning(`部分移动失败: ${errs.join(', ')}`)
      else message.success(`已移动 ${ok} 项`)
    } else {
      let ok = 0
      const errs: string[] = []
      for (const id of ids) {
        try {
          await copyItem(id, targetDirId)
          ok++
        } catch (e: any) {
          errs.push(e?.response?.data?.message || '未知错误')
        }
      }
      if (errs.length > 0) message.warning(`部分复制失败: ${errs.join(', ')}`)
      else message.success(`已复制 ${ok} 项`)
    }
    setSelectedRowKeys([])
  }

  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys as string[]),
  }

  const columns: ColumnsType<FileItem> = [
    {
      title: '名称', dataIndex: 'name', key: 'name', width: 300,
      render: (name: string, record: FileItem) => (
        <Space>
          {getFileIcon(record.mime_type, record.is_dir)}
          {record.is_dir ? (
            <a
              onClick={() => navigateTo(record.id, record.name)}
              onDragOver={(e) => handleDirDragOver(e, record.id)}
              onDragLeave={handleDirDragLeave}
              onDrop={(e) => handleDirDrop(e, record.id, record.name)}
              style={{
                padding: '4px 8px',
                borderRadius: 4,
                background: dropDirId === record.id ? '#fef3e7' : undefined,
                outline: dropDirId === record.id ? '2px dashed #e8964a' : undefined,
              }}
            >
              {name}
            </a>
          ) : isPreviewable(record.mime_type) ? (
            <a onClick={() => setPreviewFile(record)}>{name}</a>
          ) : (
            <a href={getDownloadUrl(record.id)} download={name}>{name}</a>
          )}
        </Space>
      ),
    },
    {
      title: '大小', dataIndex: 'size', key: 'size', width: 100,
      render: (size: number, record: FileItem) => record.is_dir ? '-' : formatSize(size),
    },
    {
      title: '类型', dataIndex: 'mime_type', key: 'mime_type', width: 150,
      render: (v: string, r: FileItem) => r.is_dir ? <Tag>目录</Tag> : <Text type="secondary">{v || '-'}</Text>,
    },
    {
      title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '操作', key: 'actions', width: 120,
      render: (_: any, record: FileItem) => (
        <Space>
          {record.is_dir && (
            <Tooltip title="上传到此处">
              <Button type="link" size="small" icon={<UploadOutlined />}
                onClick={() => openUploadModal(record.id, record.name)} />
            </Tooltip>
          )}
          {!record.is_dir && isPreviewable(record.mime_type) && (
            <Tooltip title="预览">
              <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => setPreviewFile(record)} />
            </Tooltip>
          )}
          {!record.is_dir && (
            <Tooltip title="下载">
              <Button type="link" size="small" icon={<DownloadOutlined />} href={getDownloadUrl(record.id)} download={record.name} />
            </Tooltip>
          )}
          {!record.is_dir && (
            <Tooltip title="分享">
              <Button type="link" size="small" icon={<ShareAltOutlined />} onClick={() => setShareFile(record)} />
            </Tooltip>
          )}
          {!record.is_dir && (
            <Tooltip title="版本">
              <Button type="link" size="small" icon={<HistoryOutlined />} onClick={() => setVersionFile(record)} />
            </Tooltip>
          )}
          <Popconfirm title="确定删除？" onConfirm={() => remove(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8 }}>
        <Space>
          <Button type="primary" icon={<UploadOutlined />}
            onClick={() => openUploadModal()}>
            上传文件
          </Button>
          <Button icon={<FolderAddOutlined />} onClick={() => setMkdirVisible(true)}>新建目录</Button>
          <Button icon={<ReloadOutlined />} onClick={() => fetchFiles()}>刷新</Button>
          <Text type="secondary" style={{ marginLeft: 8 }}>
            拖拽文件到目录名即可上传到该目录
          </Text>
        </Space>
        <Space>
          <Input.Search
            placeholder="搜索文件..."
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            onSearch={(v) => search(v)}
            enterButton={<SearchOutlined />}
            allowClear
            style={{ width: 250 }}
          />
        </Space>
      </div>

      {selectedRowKeys.length > 0 && (
        <Card size="small" style={{ marginBottom: 16, background: '#fef3e7', borderColor: '#f5d5b0' }}>
          <Space>
            <Text strong>已选择 {selectedRowKeys.length} 项</Text>
            <Button type="primary" size="small" icon={<SwapOutlined />}
              onClick={() => {
                setPendingMoveCopyIds([...selectedRowKeys])
                setPickerOpen('move')
              }}>
              移动到...
            </Button>
            <Button size="small" icon={<CopyOutlined />}
              onClick={() => {
                setPendingMoveCopyIds([...selectedRowKeys])
                setPickerOpen('copy')
              }}>
              复制到...
            </Button>
            <Button type="primary" danger size="small" icon={<DeleteOutlined />}
              onClick={handleBatchDelete}>
              批量删除
            </Button>
            <Button type="primary" size="small" icon={<DownloadOutlined />}
              onClick={handleBatchDownload}>
              批量下载
            </Button>
            <Button size="small" onClick={() => setSelectedRowKeys([])}>取消选择</Button>
          </Space>
        </Card>
      )}

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={breadcrumb.map((b, i) => ({
          title: i === 0 ? <><HomeOutlined /> {b.name}</> : b.name,
          ...(i < breadcrumb.length - 1 ? { onClick: () => navigateTo(b.id, b.name) } : {}),
        }))}
      />

      {searchMode && (
        <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
          搜索 "{searchKeyword}" 的结果，共 {total} 项
          <Button type="link" size="small" onClick={() => fetchFiles()}>返回全部</Button>
        </Text>
      )}

      <Table
        rowSelection={rowSelection}
        columns={columns}
        dataSource={files}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showTotal: (t) => `共 ${t} 项`,
          showSizeChanger: false,
        }}
        size="middle"
        onRow={(record) => {
          const base: any = {
            draggable: true,
            onDragStart: (e: React.DragEvent) => {
              e.dataTransfer.setData('application/cloudnexus-move', record.id)
              e.dataTransfer.effectAllowed = 'move'
            },
          }
          if (record.is_dir) {
            return {
              ...base,
              onDragOver: (e: React.DragEvent) => handleDirDragOver(e, record.id),
              onDragLeave: handleDirDragLeave,
              onDrop: (e: React.DragEvent) => handleDirDrop(e, record.id, record.name),
              style: {
                background: dropDirId === record.id ? '#fef3e7' : undefined,
                outline: dropDirId === record.id ? '2px dashed #e8964a' : undefined,
                cursor: 'grab',
              },
            }
          }
          return { ...base, style: { cursor: 'grab' } }
        }}
      />

      <Modal
        title="新建目录"
        open={mkdirVisible}
        onOk={async () => {
          if (mkdirName.trim()) {
            await mkdir(mkdirName.trim())
            setMkdirName('')
            setMkdirVisible(false)
            message.success('目录已创建')
          }
        }}
        onCancel={() => setMkdirVisible(false)}
      >
        <Input placeholder="目录名称" value={mkdirName} onChange={(e) => setMkdirName(e.target.value)} />
      </Modal>

      <UploadModal
        open={uploadModalOpen}
        targetDirId={uploadTargetDir.id}
        targetDirName={uploadTargetDir.name}
        onClose={() => setUploadModalOpen(false)}
      />

      <PreviewModal
        file={previewFile}
        open={!!previewFile}
        onClose={() => setPreviewFile(null)}
      />

      <ShareModal
        file={shareFile}
        open={!!shareFile}
        onClose={() => setShareFile(null)}
      />

      <FileVersionPanel
        file={versionFile}
        open={!!versionFile}
        onClose={() => setVersionFile(null)}
      />

      <DirectoryPickerModal
        open={pickerOpen !== null}
        title={pickerOpen === 'move' ? '移动到...' : '复制到...'}
        confirmText={pickerOpen === 'move' ? '移动' : '复制'}
        onOk={handlePickerOk}
        onCancel={() => { setPickerOpen(null); setPendingMoveCopyIds([]) }}
      />
    </div>
  )
}
