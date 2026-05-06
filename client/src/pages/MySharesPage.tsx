import { useEffect, useState } from 'react'
import { Table, Button, Tag, message, Popconfirm, Space, Typography } from 'antd'
import { DeleteOutlined, CopyOutlined } from '@ant-design/icons'
import * as fileApi from '../services/file'
import type { ShareInfo } from '../services/file'

const { Text } = Typography

function formatExpiry(expiresAt: string | null) {
  if (!expiresAt) return '永久'
  const d = new Date(expiresAt)
  if (d < new Date()) return '已过期'
  return d.toLocaleString()
}

export default function MySharesPage() {
  const [shares, setShares] = useState<ShareInfo[]>([])
  const [loading, setLoading] = useState(false)

  const fetchShares = async () => {
    setLoading(true)
    try {
      setShares(await fileApi.getMyShares())
    } catch {
      message.error('加载分享列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchShares() }, [])

  const handleCopy = (code: string) => {
    navigator.clipboard.writeText(fileApi.getShareUrl(code)).then(() => message.success('链接已复制'))
  }

  const handleDelete = async (id: string) => {
    try {
      await fileApi.deleteShare(id)
      setShares((prev) => prev.filter((s) => s.id !== id))
      message.success('已取消分享')
    } catch {
      message.error('取消分享失败')
    }
  }

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button onClick={fetchShares} loading={loading}>刷新</Button>
        <Text type="secondary">共 {shares.length} 个分享</Text>
      </Space>
      <Table
        dataSource={shares}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        columns={[
          {
            title: '文件名', dataIndex: 'file_name', ellipsis: true,
            render: (name: string) => <Text strong>{name}</Text>,
          },
          {
            title: '大小', dataIndex: 'file_size', width: 100,
            render: (size: number) => {
              if (size >= 1 << 30) return `${(size / (1 << 30)).toFixed(1)} GB`
              if (size >= 1 << 20) return `${(size / (1 << 20)).toFixed(1)} MB`
              if (size >= 1 << 10) return `${(size / (1 << 10)).toFixed(1)} KB`
              return `${size} B`
            },
          },
          {
            title: '分享码', dataIndex: 'share_code', width: 140,
            render: (code: string) => (
              <Text copyable={{ text: fileApi.getShareUrl(code) }} style={{ fontSize: 12 }}>
                {code.slice(0, 12)}...
              </Text>
            ),
          },
          {
            title: '保护', dataIndex: 'has_password', width: 80,
            render: (v: boolean) => v ? <Tag color="orange">密码</Tag> : <Tag>公开</Tag>,
          },
          {
            title: '有效期', dataIndex: 'expires_at', width: 160,
            render: (v: string | null) => (
              <Tag color={!v ? 'green' : new Date(v) < new Date() ? 'red' : 'blue'}>
                {formatExpiry(v)}
              </Tag>
            ),
          },
          {
            title: '下载次数', dataIndex: 'download_count', width: 100,
          },
          {
            title: '创建时间', dataIndex: 'created_at', width: 170,
            render: (v: string) => new Date(v).toLocaleString(),
          },
          {
            title: '操作', width: 180,
            render: (_, record) => (
              <Space size={0}>
                <Button type="link" size="small" icon={<CopyOutlined />}
                  onClick={() => handleCopy(record.share_code)}>
                  复制链接
                </Button>
                <Popconfirm title="取消此分享？" onConfirm={() => handleDelete(record.id)}>
                  <Button type="link" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
    </div>
  )
}
