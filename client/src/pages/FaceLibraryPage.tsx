import { useState, useEffect, useCallback } from 'react'
import { Card, Table, Button, Modal, Input, Space, message, Popconfirm, Avatar } from 'antd'
import { ReloadOutlined, SmileOutlined, UserOutlined } from '@ant-design/icons'
import type { FaceProfile } from '../services/camera'
import { getFaceProfiles, updateFaceProfile, deleteFaceProfile } from '../services/camera'

export default function FaceLibraryPage() {
  const [profiles, setProfiles] = useState<FaceProfile[]>([])
  const [loading, setLoading] = useState(false)
  const [editModal, setEditModal] = useState<{ id: string; name: string } | null>(null)
  const [newName, setNewName] = useState('')

  const fetch = useCallback(async () => {
    setLoading(true)
    try {
      const list = await getFaceProfiles()
      setProfiles(list)
    } catch {
      message.error('获取人脸库失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetch() }, [fetch])

  const handleRename = async () => {
    if (!editModal || !newName.trim()) return
    try {
      await updateFaceProfile(editModal.id, newName.trim())
      message.success('改名成功')
      setEditModal(null)
      fetch()
    } catch {
      message.error('改名失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteFaceProfile(id)
      message.success('删除成功')
      fetch()
    } catch {
      message.error('删除失败')
    }
  }

  const columns = [
    {
      title: '照片', dataIndex: 'thumbnail_url', key: 'thumbnail', width: 64,
      render: (_: string, r: FaceProfile) => {
        const token = localStorage.getItem('access_token')
        const src = r.thumbnail_url
          ? `${import.meta.env.VITE_API_BASE || ''}/api/v1/faces/${r.id}/thumbnail?token=${token}`
          : undefined
        return <Avatar shape="square" size={48} src={src} icon={<UserOutlined />} />
      },
    },
    { title: '姓名', dataIndex: 'name', key: 'name' },
    {
      title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 180,
      render: (t: string) => new Date(t).toLocaleString(),
    },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: any, r: FaceProfile) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => { setEditModal({ id: r.id, name: r.name }); setNewName(r.name) }}>
            编辑
          </Button>
          <Popconfirm title="确定删除此人脸？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Card
        title={<span><SmileOutlined style={{ marginRight: 8 }} />人脸库</span>}
        extra={
          <Button icon={<ReloadOutlined />} onClick={fetch}>刷新</Button>
        }
      >
        <Table
          dataSource={profiles}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 人` }}
          locale={{ emptyText: '暂无人脸数据。请在摄像头实时画面中检测人脸并注册。' }}
        />
      </Card>

      <Modal
        title="修改姓名"
        open={!!editModal}
        onOk={handleRename}
        onCancel={() => setEditModal(null)}
        destroyOnClose
      >
        <Input
          placeholder="输入新姓名"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
        />
      </Modal>
    </div>
  )
}
