import { useEffect, useMemo, useState, type Key } from 'react'
import { Modal, Table, Breadcrumb, message, Space, Tag, Typography, Button } from 'antd'
import { HomeOutlined, FileOutlined, FolderOutlined } from '@ant-design/icons'
import * as fileApi from '../services/file'
import type { FileItem } from '../services/file'
import { formatFileSize } from '../utils/format'

const { Text } = Typography

interface Props {
  open: boolean
  onOk: (file: FileItem) => void
  onCancel: () => void
  onOkMultiple?: (files: FileItem[]) => void
  title?: string
  accept?: (file: FileItem) => boolean
  multiple?: boolean
}

function getFileIcon(mimeType: string) {
  if (mimeType?.startsWith('image/')) return <FileOutlined style={{ color: '#52c41a' }} />
  if (mimeType?.startsWith('video/')) return <FileOutlined style={{ color: '#5b8def' }} />
  if (mimeType?.startsWith('audio/')) return <FileOutlined style={{ color: '#722ed1' }} />
  if (mimeType === 'application/pdf') return <FileOutlined style={{ color: '#ff4d4f' }} />
  return <FileOutlined />
}

export default function FilePickerModal({
  open,
  onOk,
  onCancel,
  onOkMultiple,
  title = '选择文件',
  accept,
  multiple = false,
}: Props) {
  const [files, setFiles] = useState<FileItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [parentId, setParentId] = useState('0')
  const [breadcrumb, setBreadcrumb] = useState<{ id: string; name: string }[]>([
    { id: '0', name: '根目录' },
  ])
  const [page, setPage] = useState(1)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const pageSize = 20

  const selectableFiles = useMemo(
    () => files.filter((file) => !file.is_dir && (!accept || accept(file))),
    [accept, files],
  )
  const selectedFiles = selectableFiles.filter((file) => selectedRowKeys.includes(file.id))

  const loadFiles = async (pId: string, pg = 1) => {
    setLoading(true)
    try {
      const res = await fileApi.getFileList(pId, pg, pageSize)
      setFiles(res.items)
      setTotal(res.total)
    } catch {
      message.error('加载文件失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open) {
      setParentId('0')
      setBreadcrumb([{ id: '0', name: '根目录' }])
      setPage(1)
      setSelectedRowKeys([])
      loadFiles('0')
    }
  }, [open])

  const navigateTo = (dirId: string, dirName: string) => {
    const idx = breadcrumb.findIndex((b) => b.id === dirId)
    if (idx >= 0) {
      setBreadcrumb(breadcrumb.slice(0, idx + 1))
    } else {
      setBreadcrumb([...breadcrumb, { id: dirId, name: dirName }])
    }
    setParentId(dirId)
    setPage(1)
    setSelectedRowKeys([])
    loadFiles(dirId)
  }

  const handlePick = (file: FileItem) => {
    if (file.is_dir) return
    if (accept && !accept(file)) {
      message.warning('该文件类型不能用于当前操作')
      return
    }
    onOk(file)
  }

  return (
    <Modal
      title={title}
      open={open}
      onCancel={onCancel}
      footer={multiple ? (
        <Space>
          <Button onClick={onCancel}>取消</Button>
          <Button type="primary" disabled={!selectedFiles.length} onClick={() => onOkMultiple?.(selectedFiles)}>
            添加已选 ({selectedFiles.length})
          </Button>
        </Space>
      ) : null}
      width={680}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        <Breadcrumb
          items={breadcrumb.map((b, i) => ({
            title: i === 0 ? (
              <span style={{ cursor: 'pointer' }} onClick={() => navigateTo('0', '根目录')}>
                <HomeOutlined /> 根目录
              </span>
            ) : i < breadcrumb.length - 1 ? (
              <span style={{ cursor: 'pointer' }} onClick={() => navigateTo(b.id, b.name)}>
                {b.name}
              </span>
            ) : (
              <span>{b.name}</span>
            ),
          }))}
        />
        <Table
          dataSource={files}
          rowKey="id"
          loading={loading}
          size="small"
          rowSelection={multiple ? {
            selectedRowKeys,
            onChange: setSelectedRowKeys,
            getCheckboxProps: (record) => ({
              disabled: record.is_dir || !!(accept && !accept(record)),
            }),
          } : undefined}
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p) => { setPage(p); loadFiles(parentId, p) },
            showTotal: (t) => `共 ${t} 项`,
            showSizeChanger: false,
          }}
          columns={[
            {
              title: '名称', dataIndex: 'name', key: 'name',
              render: (name: string, record: FileItem) => (
                <Space>
                  {record.is_dir ? <FolderOutlined style={{ color: '#faad14' }} /> : getFileIcon(record.mime_type)}
                  {record.is_dir ? (
                    <a onClick={() => navigateTo(record.id, record.name)}>{name}</a>
                  ) : (
                    <a onClick={() => handlePick(record)}>{name}</a>
                  )}
                </Space>
              ),
            },
            {
              title: '大小', dataIndex: 'size', key: 'size', width: 100,
              render: (size: number, record: FileItem) => record.is_dir ? '-' : formatFileSize(size),
            },
            {
              title: '类型', dataIndex: 'mime_type', key: 'mime_type', width: 140,
              render: (v: string, r: FileItem) => r.is_dir ? <Tag>目录</Tag> : <Text type="secondary">{v || '-'}</Text>,
            },
          ]}
        />
      </Space>
    </Modal>
  )
}
