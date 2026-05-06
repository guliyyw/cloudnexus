import { useEffect, useState } from 'react'
import { Modal, Breadcrumb, List, Button, Spin, Empty, message } from 'antd'
import { HomeOutlined, FolderOutlined } from '@ant-design/icons'
import * as fileApi from '../services/file'
import type { FileItem } from '../services/file'

interface Props {
  open: boolean
  title: string
  confirmText: string
  onOk: (targetDirId: string) => void
  onCancel: () => void
}

function DirectoryPickerModal({ open, title, confirmText, onOk, onCancel }: Props) {
  const [currentParentId, setCurrentParentId] = useState('0')
  const [dirs, setDirs] = useState<FileItem[]>([])
  const [loading, setLoading] = useState(false)
  const [breadcrumbs, setBreadcrumbs] = useState<{ id: string; name: string }[]>([
    { id: '0', name: '根目录' },
  ])

  const loadDirs = async (parentId: string) => {
    setLoading(true)
    try {
      const res = await fileApi.getFileList(parentId, 1, 200)
      setDirs(res.items.filter((f) => f.is_dir))
    } catch {
      message.error('加载目录失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open) {
      setCurrentParentId('0')
      setBreadcrumbs([{ id: '0', name: '根目录' }])
      loadDirs('0')
    }
  }, [open])

  const navigateTo = (dirId: string, dirName: string) => {
    const idx = breadcrumbs.findIndex((b) => b.id === dirId)
    if (idx >= 0) {
      setBreadcrumbs(breadcrumbs.slice(0, idx + 1))
    } else {
      setBreadcrumbs([...breadcrumbs, { id: dirId, name: dirName }])
    }
    setCurrentParentId(dirId)
    loadDirs(dirId)
  }

  const currentPath = breadcrumbs.map((b) => b.name).join(' / ')

  return (
    <Modal
      open={open}
      title={title}
      onCancel={onCancel}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          取消
        </Button>,
        <Button key="ok" type="primary" onClick={() => onOk(currentParentId)}>
          {confirmText}到此处
        </Button>,
      ]}
      width={480}
    >
      <div style={{ marginBottom: 12 }}>
        <Breadcrumb
          items={breadcrumbs.map((b, i) => ({
            title: i === 0 ? (
              <span style={{ cursor: 'pointer' }} onClick={() => navigateTo('0', '根目录')}>
                <HomeOutlined /> 根目录
              </span>
            ) : (
              <span style={{ cursor: 'pointer' }} onClick={() => navigateTo(b.id, b.name)}>
                {b.name}
              </span>
            ),
          }))}
        />
        <div style={{ color: '#888', fontSize: 12, marginTop: 8 }}>
          当前目录: {currentPath}
        </div>
      </div>
      <Spin spinning={loading}>
        {dirs.length === 0 ? (
          <Empty description="此目录下没有子目录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <List
            style={{ maxHeight: 320, overflow: 'auto' }}
            dataSource={dirs}
            renderItem={(item) => (
              <List.Item
                style={{ cursor: 'pointer', padding: '8px 12px' }}
                onClick={() => navigateTo(item.id, item.name)}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.background = '#f5f5f5'
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.background = ''
                }}
              >
                <FolderOutlined style={{ color: '#faad14', marginRight: 8 }} />
                {item.name}
              </List.Item>
            )}
          />
        )}
      </Spin>
    </Modal>
  )
}

export default DirectoryPickerModal
