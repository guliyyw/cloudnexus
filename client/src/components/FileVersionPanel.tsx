import { useEffect, useState } from 'react'
import { Drawer, Table, Button, Space, message, Popconfirm, Typography, Empty } from 'antd'
import { ReloadOutlined, HistoryOutlined, DownloadOutlined, SwapOutlined } from '@ant-design/icons'
import * as fileApi from '../services/file'
import type { FileVersion } from '../services/file'

const { Text } = Typography

interface Props {
  file: { id: string; name: string } | null
  open: boolean
  onClose: () => void
}

export default function FileVersionPanel({ file, open, onClose }: Props) {
  const [versions, setVersions] = useState<FileVersion[]>([])
  const [loading, setLoading] = useState(false)

  const fetchVersions = async () => {
    if (!file) return
    setLoading(true)
    try {
      const res = await fileApi.getVersions(file.id, 1, 50)
      setVersions(res.items)
    } catch {
      message.error('获取版本列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { if (open) fetchVersions() }, [open, file?.id])

  const handleRestore = async (versionId: string) => {
    if (!file) return
    try {
      await fileApi.restoreVersion(file.id, versionId)
      message.success('版本已恢复，当前文件已更新')
      fetchVersions()
    } catch (e: any) {
      message.error(e?.response?.data?.message || '恢复失败')
    }
  }

  const columns = [
    { title: '版本', dataIndex: 'version_num', key: 'version_num', width: 60, render: (n: number) => `v${n}` },
    {
      title: '大小', dataIndex: 'size', key: 'size', width: 80,
      render: (s: number) => {
        if (s === 0) return '-'
        const units = ['B', 'KB', 'MB', 'GB']
        let i = 0; let size = s
        while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
        return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
      },
    },
    {
      title: '校验', dataIndex: 'sha256', key: 'sha256', width: 100,
      render: (h: string) => h ? <Text code style={{ fontSize: 10 }}>{h.substring(0, 8)}</Text> : '-',
    },
    { title: '说明', dataIndex: 'message', key: 'message', ellipsis: true },
    {
      title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160,
      render: (t: string) => new Date(t).toLocaleString(),
    },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: any, r: FileVersion) => (
        <Space size="small">
          <Button type="link" size="small" icon={<DownloadOutlined />}
            href={fileApi.getVersionDownloadUrl(file!.id, r.id)} target="_blank">
            下载
          </Button>
          <Popconfirm title="恢复此版本将覆盖当前文件" onConfirm={() => handleRestore(r.id)}>
            <Button type="link" size="small" icon={<SwapOutlined />}>恢复</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <Drawer
      title={
        <Space>
          <HistoryOutlined />
          <span>版本历史 — {file?.name}</span>
        </Space>
      }
      open={open}
      onClose={onClose}
      width={680}
      extra={<Button icon={<ReloadOutlined />} onClick={fetchVersions}>刷新</Button>}
    >
      {versions.length === 0 && !loading ? (
        <Empty description="暂无历史版本，覆盖上传后自动保存旧版本" />
      ) : (
        <Table
          dataSource={versions}
          columns={columns}
          rowKey="id"
          loading={loading}
          size="small"
          pagination={{ pageSize: 50, size: 'small' }}
        />
      )}
    </Drawer>
  )
}
