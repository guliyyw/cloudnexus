import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Button, Modal, Form, Input, Select, Space, message, Tag, Popconfirm, List, Progress, Alert } from 'antd'
import { PlusOutlined, VideoCameraOutlined, PlayCircleOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { Camera } from '../services/camera'
import { getCameras, createCamera, updateCamera, deleteCamera, discoverCameras } from '../services/camera'
import { detectLocalIP, generateSubnetIPs, discoverCamerasLocally, subnetFromIP, type LocalDiscoveredCamera } from '../utils/cameraDiscovery'

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

  // --- Discovery ---
  const [discoverModalOpen, setDiscoverModalOpen] = useState(false)
  const [discoverSubnet, setDiscoverSubnet] = useState('192.168.1.0/24')
  const [discovering, setDiscovering] = useState(false)
  const [discoveredCameras, setDiscoveredCameras] = useState<LocalDiscoveredCamera[]>([])
  const [discoverProgress, setDiscoverProgress] = useState({ scanned: 0, total: 0 })
  const [discoverSource, setDiscoverSource] = useState<'client' | 'server'>('client')

  const openDiscover = async () => {
    setDiscoveredCameras([])
    setDiscoverProgress({ scanned: 0, total: 0 })
    const ip = await detectLocalIP()
    if (ip) {
      setDiscoverSubnet(subnetFromIP(ip))
    }
    setDiscoverSource('client')
    setDiscoverModalOpen(true)
  }

  const handleDiscover = async () => {
    setDiscovering(true)
    setDiscoveredCameras([])
    setDiscoverProgress({ scanned: 0, total: 0 })
    setDiscoverSource('client')
    try {
      const raw = discoverSubnet.replace('/24', '').split('/')[0] || '192.168.1.0'
      const ips = generateSubnetIPs(raw)
      if (ips.length === 0) {
        message.error('无效的子网范围')
        setDiscovering(false)
        return
      }
      setDiscoverProgress({ scanned: 0, total: ips.length * 8 })
      const localResults = await discoverCamerasLocally(ips, (scanned, total) => {
        setDiscoverProgress({ scanned, total })
      })
      if (localResults.length > 0) {
        setDiscoveredCameras(localResults)
        message.success(`客户端发现 ${localResults.length} 个摄像头`)
      }
    } catch {
      message.error('扫描失败')
    } finally {
      setDiscovering(false)
    }
  }

  const handleServerDiscover = async () => {
    setDiscovering(true)
    setDiscoveredCameras([])
    setDiscoverProgress({ scanned: 0, total: 0 })
    setDiscoverSource('server')
    try {
      const res = await discoverCameras({ subnet: discoverSubnet })
      const cameras: LocalDiscoveredCamera[] = (res.cameras || []).map((c) => ({
        ip: c.ip,
        port: c.port,
        path: '',
        source: c.source,
      }))
      setDiscoveredCameras(cameras)
      if (cameras.length === 0) {
        message.info('未发现摄像头')
      } else {
        message.success(`服务器发现 ${cameras.length} 个摄像头`)
      }
    } catch (e: any) {
      message.error(e?.response?.data?.message || '服务器扫描失败')
    } finally {
      setDiscovering(false)
    }
  }

  const handleAddDiscovered = (cam: LocalDiscoveredCamera) => {
    setDiscoverModalOpen(false)
    const name = `摄像头-${cam.ip}`
    form.setFieldsValue({
      name,
      stream_url: `rtsp://${cam.ip}:554/`,
      protocol: 'rtsp',
    })
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
            <Button icon={<SearchOutlined />} onClick={openDiscover}>发现摄像头</Button>
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
            <Input placeholder="rtsp://admin:888888@192.168.1.32:554/udp/av0_0" />
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

      {/* Discovery Modal */}
      <Modal
        title="发现摄像头"
        open={discoverModalOpen}
        onCancel={() => { setDiscoverModalOpen(false); setDiscoveredCameras([]) }}
        footer={null}
        width={680}
        destroyOnClose
      >
        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>子网范围</div>
          <Input
            value={discoverSubnet}
            onChange={(e) => setDiscoverSubnet(e.target.value)}
            placeholder="192.168.1.0/24"
            disabled={discovering}
          />
        </div>

        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button
              type="primary"
              onClick={handleDiscover}
              loading={discovering && discoverSource === 'client'}
              icon={<SearchOutlined />}
            >
              局域网扫描
            </Button>
            <Button
              onClick={handleServerDiscover}
              loading={discovering && discoverSource === 'server'}
            >
              服务器深度扫描
            </Button>
          </Space>
        </div>

        {discovering && discoverProgress.total > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Progress
              percent={Math.round((discoverProgress.scanned / discoverProgress.total) * 100)}
              format={() => `${discoverProgress.scanned}/${discoverProgress.total}`}
              size="small"
            />
          </div>
        )}

        {discoveredCameras.length > 0 && (
          <div>
            <div style={{ fontWeight: 500, marginBottom: 8 }}>
              发现的摄像头 ({discoveredCameras.length})
            </div>
            <List
              dataSource={discoveredCameras}
              renderItem={(cam) => (
                <List.Item
                  actions={[
                    <Button
                      type="link"
                      onClick={() => handleAddDiscovered(cam)}
                      icon={<PlusOutlined />}
                    >
                      添加
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={
                      <Space>
                        <span>{cam.ip}:{cam.port}</span>
                        {cam.source === 'onvif' && <Tag color="blue">ONVIF</Tag>}
                        {cam.source === 'http' && <Tag color="green">HTTP</Tag>}
                        {cam.source === 'rtsp_probe' && <Tag color="orange">RTSP</Tag>}
                      </Space>
                    }
                    description={
                      <span>
                        建议 RTSP: <code>rtsp://{cam.ip}:554/</code>
                      </span>
                    }
                  />
                </List.Item>
              )}
            />
          </div>
        )}

        {!discovering && discoveredCameras.length === 0 && (
          <Alert
            message="点击「局域网扫描」从浏览器直接探测本地网络。如果摄像头与服务器在同一网络，也可尝试「服务器深度扫描」。"
            type="info"
            showIcon
          />
        )}
      </Modal>
    </div>
  )
}
