import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Button, Space, Tag, message, Table, Switch, Descriptions, Empty } from 'antd'
import { ArrowLeftOutlined, PlayCircleOutlined, PauseCircleOutlined, CameraOutlined, ReloadOutlined } from '@ant-design/icons'
import type { Camera, RecognitionEvent } from '../services/camera'
import { getCameras, startStream, stopStream, startRecognition, stopRecognition, getEvents } from '../services/camera'
import Hls from 'hls.js'

export default function CameraLiveView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const videoRef = useRef<HTMLVideoElement>(null)

  const [camera, setCamera] = useState<Camera | null>(null)
  const [playing, setPlaying] = useState(false)
  const [recognizing, setRecognizing] = useState(false)
  const [events, setEvents] = useState<RecognitionEvent[]>([])
  const [loading, setLoading] = useState(false)
  const hlsRef = useRef<Hls | null>(null)

  const fetchCamera = useCallback(async () => {
    if (!id) return
    try {
      const res = await getCameras(1, 100)
      const cam = res.items.find((c: Camera) => c.id === id)
      if (cam) setCamera(cam)
    } catch { message.error('获取摄像头信息失败') }
  }, [id])

  const fetchEvents = useCallback(async () => {
    if (!id) return
    try {
      const res = await getEvents(id, 1, 50)
      setEvents(res.items)
    } catch { /* ignore */ }
  }, [id])

  useEffect(() => { fetchCamera(); fetchEvents() }, [fetchCamera, fetchEvents])

  // Cleanup HLS on unmount
  useEffect(() => {
    return () => {
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
    }
  }, [])

  const handlePlay = async () => {
    if (!id) return
    setLoading(true)
    try {
      const { hls_url } = await startStream(id)
      // Render the video element first
      setPlaying(true)
      // Wait for React to render the video element
      await new Promise(r => setTimeout(r, 100))

      const video = videoRef.current
      if (!video) return

      // Use direct HLS port to avoid nginx cookie-check redirect loop
      const directUrl = `http://${window.location.hostname}:8888${hls_url}`

      if (Hls.isSupported()) {
        const hls = new Hls()
        hls.loadSource(directUrl)
        hls.attachMedia(video)
        hls.on(Hls.Events.MANIFEST_PARSED, () => video.play())
        hlsRef.current = hls
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = directUrl
        video.play()
      }
      message.success('视频流已开启')
      fetchCamera()
    } catch (e: any) {
      setPlaying(false)
      message.error(e?.response?.data?.message || '开启视频流失败')
    } finally {
      setLoading(false)
    }
  }

  const handleStop = async () => {
    if (!id) return
    try {
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
      if (videoRef.current) {
        videoRef.current.pause()
        videoRef.current.src = ''
      }
      await stopStream(id)
      if (recognizing) {
        try { await stopRecognition(id) } catch { /* ignore */ }
        setRecognizing(false)
      }
      setPlaying(false)
      message.success('视频流已停止')
      fetchCamera()
    } catch { message.error('停止视频流失败') }
  }

  const handleToggleRecognition = async (checked: boolean) => {
    if (!id) return
    try {
      if (checked) {
        await startRecognition(id)
        setRecognizing(true)
        message.success('AI 识别已开启')
      } else {
        await stopRecognition(id)
        setRecognizing(false)
        message.success('AI 识别已停止')
      }
    } catch (e: any) {
      message.error(e?.response?.data?.message || '操作失败')
    }
  }

  const statusColor = camera?.status === 'online' ? 'green' : 'default'
  const statusText = camera?.status === 'online' ? '在线' : '离线'

  const eventColumns = [
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: (t: string) => new Date(t).toLocaleString() },
    {
      title: '类型', dataIndex: 'event_type', key: 'event_type', width: 100,
      render: (t: string) => <Tag color="orange">{t}</Tag>,
    },
    {
      title: '置信度', dataIndex: 'confidence', key: 'confidence', width: 80,
      render: (c: number) => `${(c * 100).toFixed(1)}%`,
    },
  ]

  return (
    <div>
      <Card
        title={
          <Space>
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/cameras')} />
            <CameraOutlined />
            <span>{camera?.name || '加载中...'}</span>
            <Tag color={statusColor}>{statusText}</Tag>
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => { fetchCamera(); fetchEvents() }}>刷新</Button>
            <span>AI 识别: <Switch checked={recognizing} onChange={handleToggleRecognition} disabled={!playing} /></span>
            {playing ? (
              <Button danger icon={<PauseCircleOutlined />} onClick={handleStop} loading={loading}>停止</Button>
            ) : (
              <Button type="primary" icon={<PlayCircleOutlined />} onClick={handlePlay} loading={loading}>播放</Button>
            )}
          </Space>
        }
      >
        <div style={{ display: 'flex', gap: 20, flexWrap: 'wrap' }}>
          <div style={{ flex: '1 1 640px', minWidth: 320 }}>
            <div style={{
              background: '#1e1e1e', borderRadius: 12, overflow: 'hidden',
              minHeight: 360, display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              {playing ? (
                <video ref={videoRef} controls autoPlay muted style={{ width: '100%', maxHeight: 480 }} />
              ) : (
                <Empty description="点击「播放」开始实时画面" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </div>
          </div>
          <div style={{ flex: '0 0 280px', minWidth: 240 }}>
            {camera && (
              <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
                <Descriptions.Item label="协议">{camera.protocol?.toUpperCase()}</Descriptions.Item>
                <Descriptions.Item label="地址" styles={{ content: { wordBreak: 'break-all' } }}>
                  {camera.stream_url}
                </Descriptions.Item>
                <Descriptions.Item label="最近在线">{camera.last_seen_at ? new Date(camera.last_seen_at).toLocaleString() : '-'}</Descriptions.Item>
              </Descriptions>
            )}
            <Card title="AI 识别记录" size="small" extra={<Button size="small" onClick={fetchEvents}>刷新</Button>}>
              <Table
                dataSource={events}
                columns={eventColumns}
                rowKey="id"
                size="small"
                pagination={{ pageSize: 10, size: 'small' }}
                locale={{ emptyText: '暂无识别事件' }}
              />
            </Card>
          </div>
        </div>
      </Card>
    </div>
  )
}
