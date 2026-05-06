import { useEffect, useState } from 'react'
import { Modal, Input, Button, Space, message, Typography, List, Popconfirm, Tag, Switch } from 'antd'
import { LinkOutlined, DeleteOutlined, CopyOutlined } from '@ant-design/icons'
import * as fileApi from '../services/file'
import type { ShareInfo } from '../services/file'

const { Text } = Typography

interface Props {
  file: { id: string; name: string } | null
  open: boolean
  onClose: () => void
}

export default function ShareModal({ file, open, onClose }: Props) {
  const [shares, setShares] = useState<ShareInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [password, setPassword] = useState('')
  const [expiresIn, setExpiresIn] = useState(0)
  const [useExpiry, setUseExpiry] = useState(false)

  const fetchShares = async () => {
    if (!file) return
    try {
      const list = await fileApi.getFileShares(file.id)
      setShares(list)
    } catch { /* ignore */ }
  }

  useEffect(() => {
    if (open && file) {
      setPassword('')
      setExpiresIn(24)
      setUseExpiry(false)
      fetchShares()
    }
  }, [open, file])

  const handleCreate = async () => {
    if (!file) return
    setLoading(true)
    try {
      const share = await fileApi.createShare(file.id, password || undefined, useExpiry ? expiresIn : undefined)
      setShares((prev) => [share, ...prev])
      setPassword('')
      message.success('分享链接已创建')
    } catch {
      message.error('创建分享失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (shareId: string) => {
    try {
      await fileApi.deleteShare(shareId)
      setShares((prev) => prev.filter((s) => s.id !== shareId))
      message.success('已取消分享')
    } catch {
      message.error('取消分享失败')
    }
  }

  const handleCopyUrl = (code: string) => {
    const url = fileApi.getShareUrl(code)
    navigator.clipboard.writeText(url).then(() => message.success('链接已复制'))
  }

  const formatExpiry = (s: ShareInfo) => {
    if (!s.expires_at) return '永久'
    const d = new Date(s.expires_at)
    if (d < new Date()) return '已过期'
    return d.toLocaleString()
  }

  return (
    <Modal
      title={`分享: ${file?.name || ''}`}
      open={open}
      onCancel={onClose}
      footer={null}
      width={560}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <div>
          <Text strong>创建新分享</Text>
          <div style={{ marginTop: 8, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <Input.Password
              placeholder="访问密码（留空无需密码）"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              style={{ flex: 1, minWidth: 180 }}
            />
            <Space>
              <Switch checked={useExpiry} onChange={setUseExpiry} size="small" />
              {useExpiry && (
                <Input
                  type="number"
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(Number(e.target.value))}
                  suffix="小时"
                  style={{ width: 80 }}
                  min={1}
                />
              )}
              <Text type="secondary" style={{ fontSize: 12 }}>有效期</Text>
            </Space>
            <Button type="primary" onClick={handleCreate} loading={loading}>
              创建分享
            </Button>
          </div>
        </div>

        {shares.length > 0 && (
          <div>
            <Text strong>已有分享 ({shares.length})</Text>
            <List
              size="small"
              dataSource={shares}
              renderItem={(s) => (
                <List.Item
                  actions={[
                    <Button type="link" size="small" icon={<CopyOutlined />}
                      onClick={() => handleCopyUrl(s.share_code)}>复制链接</Button>,
                    <Popconfirm title="取消此分享？" onConfirm={() => handleDelete(s.id)}>
                      <Button type="link" size="small" danger icon={<DeleteOutlined />} />
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={<LinkOutlined style={{ fontSize: 18, color: '#e8964a' }} />}
                    title={
                      <Space size={4}>
                        <Text copyable={{ text: fileApi.getShareUrl(s.share_code) }}
                          style={{ fontSize: 12, maxWidth: 200 }} ellipsis>
                          {s.share_code}
                        </Text>
                        {s.has_password && <Tag color="orange" style={{ fontSize: 10 }}>密码保护</Tag>}
                        <Tag style={{ fontSize: 10 }}>{formatExpiry(s)}</Tag>
                      </Space>
                    }
                    description={`下载 ${s.download_count} 次 | ${new Date(s.created_at).toLocaleString()}`}
                  />
                </List.Item>
              )}
            />
          </div>
        )}
      </Space>
    </Modal>
  )
}
