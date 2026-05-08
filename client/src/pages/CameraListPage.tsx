import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Button, Modal, Form, Input, Select, Space, message, Tag, Popconfirm } from 'antd'
import { PlusOutlined, VideoCameraOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import type { Camera } from '../services/camera'
import { getCameras, createCamera, updateCamera, deleteCamera } from '../services/camera'

export default function CameraListPage() {
  const [cameras, setCameras] = useState<Camera[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingCam, setEditingCam] = useState<Camera | null>(null)
  const [form] = Form.useForm()
  const navigate = useNavigate()

  const fetchCameras = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getCameras(page, 10)
      setCameras(res.items)
      setTotal(res.total)
    } catch {
      message.error('获取摄像头列表失败')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => { fetchCameras() }, [fetchCameras])

  const handleSave = async () => {
    const values = await form.validateFields()
    try {
      if (editingCam) {
        await updateCamera(editingCam.id, values)
        message.success('更新成功')
      } else {
        await createCamera(values)
        message.success('添加成功')
      }
      setModalOpen(false)
      form.resetFields()
      setEditingCam(null)
      fetchCameras()
    } catch (e: any) {
      message.error(e?.response?.data?.message || '操作失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteCamera(id)
      message.success('删除成功')
      fetchCameras()
    } catch {
      message.error('删除失败')
    }
  }

  const openEdit = (cam: Camera) => {
    setEditingCam(cam)
    form.setFieldsValue(cam)
    setModalOpen(true)
  }

  const openCreate = () => {
    setEditingCam(null)
    form.resetFields()
    form.setFieldsValue({ protocol: 'rtsp' })
    setModalOpen(true)
  }

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '协议', dataIndex: 'protocol', key: 'protocol', width: 80,
      render: (p: string) => <Tag>{p.toUpperCase()}</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) => (
        <Tag color={s === 'online' ? 'green' : 'default'}>{s === 'online' ? '在线' : '离线'}</Tag>
      ),
    },
    {
      title: '操作', key: 'actions', width: 260,
      render: (_: any, r: Camera) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => navigate(`/cameras/${r.id}`)} icon={<PlayCircleOutlined />}>
            查看
          </Button>
          <Button type="link" size="small" onClick={() => openEdit(r)}>编辑</Button>
          <Popconfirm title="确定删除此摄像头？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Card
        title={<span><VideoCameraOutlined style={{ marginRight: 8 }} />摄像头管理</span>}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchCameras}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>添加摄像头</Button>
          </Space>
        }
      >
        <Table
          dataSource={cameras}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page, total, pageSize: 10,
            onChange: (p) => setPage(p),
            showTotal: (t) => `共 ${t} 个摄像头`,
          }}
          locale={{ emptyText: '暂无摄像头，点击右上角"添加摄像头"开始' }}
        />
      </Card>

      <Modal
        title={editingCam ? '编辑摄像头' : '添加摄像头'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => { setModalOpen(false); form.resetFields(); setEditingCam(null) }}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：大门摄像头" />
          </Form.Item>
          <Form.Item name="stream_url" label="RTSP 地址" rules={[{ required: true, message: '请输入 RTSP 地址' }]}>
            <Input placeholder="rtsp://admin:password@192.168.1.126:554/udp/av0_0" />
          </Form.Item>
          <Form.Item name="protocol" label="协议">
            <Select>
              <Select.Option value="rtsp">RTSP</Select.Option>
              <Select.Option value="rtmp">RTMP</Select.Option>
              <Select.Option value="onvif">ONVIF</Select.Option>
            </Select>
          </Form.Item>
        </Form>
        <div style={{ fontSize: 12, color: '#8c8c8c', marginTop: 8 }}>
          提示：默认账号 admin，密码 888888。RTSP 格式示例：
          <br />
          rtsp://admin:888888@IP:554/udp/av0_0 (主码流)
          <br />
          rtsp://admin:888888@IP:554/udp/av0_1 (子码流)
        </div>
      </Modal>
    </div>
  )
}
