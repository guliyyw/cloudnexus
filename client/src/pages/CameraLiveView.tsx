import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Button, Space, Tag, message, Table, Switch, Descriptions, Empty, Tabs, Popconfirm } from 'antd'
import { ArrowLeftOutlined, PlayCircleOutlined, PauseCircleOutlined, CameraOutlined, ReloadOutlined, SmileOutlined, VideoCameraOutlined, DeleteOutlined } from '@ant-design/icons'
import type { Camera, RecognitionEvent, FaceRecognitionEvent, CameraRecording } from '../services/camera'
import { getCameras, startStream, stopStream, startRecognition, stopRecognition, getEvents, getFaceEvents, matchFace, clearFaceEvents, startCameraRecording, stopCameraRecording, getCameraRecordingStatus, getCameraRecordings, deleteCameraRecording, getCameraRecordingPlaybackUrl } from '../services/camera'
import { detectFaces, embeddingToArray, loadModels } from '../utils/faceDetection'
import { FaceTracker } from '../utils/faceTracker'
import FaceOverlay, { type FaceBox } from '../components/FaceOverlay'
import FaceRegisterModal from '../components/FaceRegisterModal'
import Hls from 'hls.js'

// Check if current page is loaded from a private/local network.
// Private Network Access (PNA) in browsers blocks public→private requests,
// so MJPEG direct mode only works when the page itself is on a private network.
function isPrivateNetwork(): boolean {
  const hostname = window.location.hostname
  if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '[::1]') return true
  const ip = hostname.split('.').map(Number)
  if (ip.length !== 4 || ip.some(isNaN)) return false // not an IPv4 address (domain name)

  // Check private IPv4 ranges: 10.x, 172.16-31.x, 192.168.x
  if (ip[0] === 10) return true
  if (ip[0] === 172 && ip[1] >= 16 && ip[1] <= 31) return true
  if (ip[0] === 192 && ip[1] === 168) return true
  return false
}

// Extract host:port from stream URL, used to build direct MJPEG URL
function extractHost(url: string): string {
  try {
    const u = new URL(url)
    return `${u.hostname}:${u.port || '80'}`
  } catch {
    const m = url.match(/:\/\/([^/:]+)(?::(\d+))?/)
    if (m) return `${m[1]}:${m[2] || '554'}`
    return ''
  }
}

// Build MJPEG URL from camera's host — IP Webcam serves /video as MJPEG
function buildMjpegUrl(streamUrl: string): string | null {
  if (streamUrl.startsWith('http://') || streamUrl.startsWith('https://')) {
    // Already HTTP — try known MJPEG endpoints
    const base = streamUrl.replace(/\/[^/]*$/, '') // strip last path segment
    return `${base}/video`
  }
  return null
}

function shouldUseDirectMjpeg(camera: Camera): boolean {
  const protocol = camera.protocol?.toLowerCase()
  const url = camera.stream_url.toLowerCase()
  if (protocol === 'rtsp' || protocol === 'rtmp' || protocol === 'onvif') return false
  if (url.startsWith('rtsp://') || url.startsWith('rtmp://')) return false
  return isPrivateNetwork() && (url.startsWith('http://') || url.startsWith('https://'))
}

export default function CameraLiveView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const videoRef = useRef<HTMLVideoElement>(null)
  const mjpegImgRef = useRef<HTMLImageElement>(null)
  const displayCanvasRef = useRef<HTMLCanvasElement>(null)
  const videoContainerRef = useRef<HTMLDivElement>(null)

  const [camera, setCamera] = useState<Camera | null>(null)
  const [playing, setPlaying] = useState(false)
  const [recognizing, setRecognizing] = useState(false)
  const [events, setEvents] = useState<RecognitionEvent[]>([])
  const [loading, setLoading] = useState(false)
  const hlsRef = useRef<Hls | null>(null)

  // MJPEG direct mode — bypasses server entirely
  const [mjpegMode, setMjpegMode] = useState(false)
  const mjpegModeRef = useRef(false)
  const mjpegAnimRef = useRef<number>(0)
  // Keep ref in sync so render loop (closure) always reads latest value
  useEffect(() => { mjpegModeRef.current = mjpegMode }, [mjpegMode])

  // Face recognition
  const [faceRecognizing, setFaceRecognizing] = useState(false)
  const [detectedFaces, setDetectedFaces] = useState<FaceBox[]>([])
  const [faceEvents, setFaceEvents] = useState<FaceRecognitionEvent[]>([])
  const [registerModalOpen, setRegisterModalOpen] = useState(false)
  const faceTimerRef = useRef<number>(0)
  const faceModelsReady = useRef(false)
  const facePauseRef = useRef(false)  // pause main loop while register modal is open
  const trackerRef = useRef(new FaceTracker())

  // Server-side recording
  const [recording, setRecording] = useState(false)
  const [recordings, setRecordings] = useState<CameraRecording[]>([])
  const [recordingLoading, setRecordingLoading] = useState(false)
  const [playbackUrl, setPlaybackUrl] = useState('')
  const segmentSeconds = 300
  const [historyPlaying, setHistoryPlaying] = useState(false)

  // Canvas dimensions for face detection input + display size for overlay scaling
  const canvasSizeRef = useRef({ w: 640, h: 360 })
  const [displaySize, setDisplaySize] = useState({ w: 640, h: 360 })

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

  const fetchFaceEvents = useCallback(async () => {
    if (!id) return
    try {
      const res = await getFaceEvents(id, 1, 50)
      setFaceEvents(res.items)
    } catch { /* ignore */ }
  }, [id])

  const fetchRecordingState = useCallback(async () => {
    if (!id) return
    try {
      const [status, list] = await Promise.all([
        getCameraRecordingStatus(id),
        getCameraRecordings(id, 1, 20),
      ])
      setRecording(status.recording)
      setRecordings(list.items)
    } catch { /* ignore */ }
  }, [id])

  const handleClearFaceEvents = async () => {
    if (!id) return
    try {
      const count = await clearFaceEvents(id)
      message.success(`已清空 ${count} 条人脸识别记录`)
      setFaceEvents([])
    } catch { message.error('清空失败') }
  }

  useEffect(() => { fetchCamera(); fetchEvents(); fetchFaceEvents(); fetchRecordingState() }, [fetchCamera, fetchEvents, fetchFaceEvents, fetchRecordingState])

  // Track actual display size of video/canvas for overlay alignment
  useEffect(() => {
    if (!playing) return
    const el = (mjpegMode && !historyPlaying) ? displayCanvasRef.current : videoRef.current
    if (!el) return
    const update = () => {
      const w = el.clientWidth || canvasSizeRef.current.w
      const h = el.clientHeight || canvasSizeRef.current.h
      if (w > 0 && h > 0) setDisplaySize({ w, h })
    }
    update()
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => ro.disconnect()
  }, [playing, mjpegMode, historyPlaying])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      // Stop MJPEG stream (clearing img src terminates the HTTP connection)
      if (mjpegAnimRef.current) cancelAnimationFrame(mjpegAnimRef.current)
      if (mjpegImgRef.current) mjpegImgRef.current.src = ''
      // Stop HLS
      if (hlsRef.current) { hlsRef.current.destroy(); hlsRef.current = null }
      if (videoRef.current) { videoRef.current.pause(); videoRef.current.src = '' }
      // Stop face detection timer
      if (faceTimerRef.current) clearInterval(faceTimerRef.current)
      // Notify server to release stream (fire-and-forget, don't block unmount)
      if (id) {
        const release = async () => {
          try { await stopStream(id) } catch { /* ignore */ }
        }
        release()
      }
    }
  }, [id])

  // --- Get display source for face detection ---
  const getDisplaySource = useCallback((): HTMLVideoElement | HTMLCanvasElement | null => {
    if (mjpegMode) return displayCanvasRef.current
    const v = videoRef.current
    if (v && v.readyState >= 2) return v
    return null
  }, [mjpegMode])

  // --- HLS Play (RTSP cameras via MediaMTX) ---
  const handlePlayHls = async () => {
    if (!id) return
    setLoading(true)
    try {
      const { hls_url } = await startStream(id)
      setPlaying(true)
      setMjpegMode(false)
      await new Promise(r => setTimeout(r, 100))

      const video = videoRef.current
      if (!video) return

      const directUrl = `${window.location.protocol}//${window.location.host}${hls_url}`

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
      message.success('视频流已开启（服务器中转）')
      fetchCamera()
    } catch (e: any) {
      setPlaying(false)
      message.error(e?.response?.data?.message || '开启视频流失败')
    } finally {
      setLoading(false)
    }
  }

  // --- MJPEG Direct Play (zero server bandwidth) ---
  const handlePlayMjpeg = async () => {
    if (!camera) return
    const mjpegUrl = buildMjpegUrl(camera.stream_url)
    if (!mjpegUrl) { message.error('无法获取直连 MJPEG 地址'); return }

    setLoading(true)
    try {
      // Test if MJPEG URL is reachable
      const testRes = await fetch(mjpegUrl, { mode: 'cors', signal: AbortSignal.timeout(5000) })
      if (!testRes.ok) throw new Error(`HTTP ${testRes.status}`)
      // Don't read body — just check connectivity

      setPlaying(true)
      setMjpegMode(true)

      // Wait for React to render the img
      await new Promise(r => setTimeout(r, 100))

      const img = mjpegImgRef.current
      const canvas = displayCanvasRef.current
      if (!img || !canvas) return

      img.src = mjpegUrl

      // Render loop: draw img → canvas for display + face detection
      const render = () => {
        if (!mjpegModeRef.current) return
        const c = displayCanvasRef.current
        const i = mjpegImgRef.current
        if (c && i && i.naturalWidth > 0) {
          c.width = i.naturalWidth
          c.height = i.naturalHeight
          canvasSizeRef.current = { w: i.naturalWidth, h: i.naturalHeight }
          const ctx = c.getContext('2d')
          if (ctx) ctx.drawImage(i, 0, 0)
        }
        mjpegAnimRef.current = requestAnimationFrame(render)
      }
      mjpegAnimRef.current = requestAnimationFrame(render)

      message.success('视频流已开启（直连摄像头，零服务器带宽）')
    } catch {
      // MJPEG direct failed — fall back to HLS
      message.info('直连失败，尝试服务器中转...')
      await handlePlayHls()
    } finally {
      setLoading(false)
    }
  }

  const handlePlay = () => {
    if (!camera) return
    if (shouldUseDirectMjpeg(camera)) {
      handlePlayMjpeg()
    } else {
      handlePlayHls()
    }
  }

  const handleStop = async () => {
    if (!id) return
    try {
      if (faceRecognizing) {
        if (faceTimerRef.current) clearInterval(faceTimerRef.current)
        setFaceRecognizing(false)
      }
      setPlaybackUrl('')
      setHistoryPlaying(false)
      if (mjpegAnimRef.current) {
        cancelAnimationFrame(mjpegAnimRef.current)
        mjpegAnimRef.current = 0
      }
      if (mjpegMode) {
        // MJPEG mode — just clear img src, no server call needed
        if (mjpegImgRef.current) mjpegImgRef.current.src = ''
        const dc = displayCanvasRef.current
        if (dc) {
          const ctx = dc.getContext('2d')
          if (ctx) ctx.clearRect(0, 0, dc.width, dc.height)
        }
        setMjpegMode(false)
      } else {
        // HLS mode — stop MediaMTX
        if (hlsRef.current) {
          hlsRef.current.destroy()
          hlsRef.current = null
        }
        if (videoRef.current) {
          videoRef.current.pause()
          videoRef.current.src = ''
        }
        await stopStream(id)
      }
      if (recognizing) {
        try { await stopRecognition(id) } catch { /* ignore */ }
        setRecognizing(false)
      }
      setPlaying(false)
      setDetectedFaces([])
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

  // --- Face Recognition ---
  const handleToggleFaceRecognition = async (checked: boolean) => {
    if (!id) return
    if (checked) {
      try {
        if (!faceModelsReady.current) {
          await loadModels()
          faceModelsReady.current = true
        }
      } catch { message.error('人脸模型加载失败'); return }
      setFaceRecognizing(true)
      message.success('人脸识别已开启')
      trackerRef.current.reset()

      faceTimerRef.current = window.setInterval(async () => {
        const source = getDisplaySource()
        if (!source) return

        if (facePauseRef.current) return
        try {
          const faces = await detectFaces(source)
          const now = Date.now()
          const { tracks, newTrackIds, newTrackDetIdx } = trackerRef.current.update(
            faces.map((f) => f.box),
            now,
          )

          // Only call match API for new tracks (first appearance)
          for (const trackId of newTrackIds) {
            const detIdx = newTrackDetIdx.get(trackId)
            if (detIdx === undefined) continue
            const emb = embeddingToArray(faces[detIdx].descriptor)
            try {
              const result = await matchFace({ embedding: emb, camera_id: id! })
              trackerRef.current.setName(trackId, result.matched ? result.name : undefined)
            } catch { /* match failed */ }
          }

          const boxes: FaceBox[] = tracks.map((t) => ({ ...t.box, name: t.name }))
          setDetectedFaces(boxes)
        } catch { /* detection failed, skip frame */ }
      }, 2500)
    } else {
      if (faceTimerRef.current) clearInterval(faceTimerRef.current)
      setFaceRecognizing(false)
      setDetectedFaces([])
      message.success('人脸识别已停止')
    }
  }

  const handleToggleRecording = async (checked: boolean) => {
    if (!id) return
    setRecordingLoading(true)
    try {
      if (checked) {
        const status = await startCameraRecording(id, {
          segment_seconds: segmentSeconds,
          retention_days: 0,
          max_storage_mb: 0,
        })
        setRecording(status.recording)
        message.success('Recording started')
      } else {
        await stopCameraRecording(id)
        setRecording(false)
        message.success('Recording stopped')
      }
      await fetchRecordingState()
    } catch (e: any) {
      message.error(e?.response?.data?.message || 'Recording operation failed')
    } finally {
      setRecordingLoading(false)
    }
  }

  const handlePlayRecording = (rec: CameraRecording) => {
    if (!id) return
    if (hlsRef.current && !mjpegMode) {
      hlsRef.current.detachMedia()
    }
    setPlaybackUrl(getCameraRecordingPlaybackUrl(id, rec.id))
    setHistoryPlaying(true)
    setPlaying(true)
  }

  const handleReturnLive = async () => {
    setPlaybackUrl('')
    setHistoryPlaying(false)
    if (!playing) return
    if (mjpegMode) return
    const video = videoRef.current
    if (video) {
      video.pause()
      video.removeAttribute('src')
      video.load()
    }
    if (hlsRef.current && video) {
      hlsRef.current.attachMedia(video)
    } else if (id) {
      await handlePlayHls()
    }
  }

  const handleDeleteRecording = async (rec: CameraRecording) => {
    if (!id) return
    try {
      await deleteCameraRecording(id, rec.id)
      message.success('Recording deleted')
      await fetchRecordingState()
    } catch (e: any) {
      message.error(e?.response?.data?.message || 'Delete failed')
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

  const faceEventColumns = [
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: (t: string) => new Date(t).toLocaleString() },
    {
      title: '姓名', dataIndex: 'face_name', key: 'face_name', width: 100,
      render: (n: string) => n ? <Tag color="green">{n}</Tag> : <Tag>未识别</Tag>,
    },
    {
      title: '相似度', dataIndex: 'confidence', key: 'confidence', width: 80,
      render: (c: number) => `${(c * 100).toFixed(1)}%`,
    },
  ]

  const tabs = [
    {
      key: 'object',
      label: 'AI 检测',
      children: (
        <Table
          dataSource={events}
          columns={eventColumns}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 10, size: 'small' }}
          locale={{ emptyText: '暂无检测事件' }}
        />
      ),
    },
    {
      key: 'face',
      label: '人脸识别',
      children: (
        <div>
          {faceEvents.length > 0 && (
            <div style={{ marginBottom: 8, textAlign: 'right' }}>
              <Popconfirm title="确定清空此摄像头所有识别记录？考勤记录不受影响" onConfirm={handleClearFaceEvents}>
                <Button size="small" icon={<DeleteOutlined />} danger>清空</Button>
              </Popconfirm>
            </div>
          )}
          <Table
            dataSource={faceEvents}
            columns={faceEventColumns}
            rowKey="id"
            size="small"
            pagination={{ pageSize: 10, size: 'small' }}
            locale={{ emptyText: '暂无人脸识别事件' }}
          />
        </div>
      ),
    },
  ]

  const recordingColumns = [
    {
      title: 'Start',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 160,
      render: (t: string) => new Date(t).toLocaleString(),
    },
    {
      title: 'Duration',
      dataIndex: 'duration_seconds',
      key: 'duration_seconds',
      width: 90,
      render: (v: number) => `${v}s`,
    },
    {
      title: 'Size',
      dataIndex: 'size_bytes',
      key: 'size_bytes',
      width: 90,
      render: (v: number) => `${(v / 1024 / 1024).toFixed(1)} MB`,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_: unknown, rec: CameraRecording) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => handlePlayRecording(rec)}>Play</Button>
          <Popconfirm title="Delete this recording?" onConfirm={() => handleDeleteRecording(rec)}>
            <Button type="link" size="small" danger>Delete</Button>
          </Popconfirm>
        </Space>
      ),
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
            {mjpegMode && <Tag color="green">直连</Tag>}
          </Space>
        }
        extra={
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={() => { fetchCamera(); fetchEvents(); fetchFaceEvents(); fetchRecordingState() }}>刷新</Button>
            <span>AI 识别: <Switch checked={recognizing} onChange={handleToggleRecognition} disabled={!playing || historyPlaying} /></span>
            <span>人脸识别: <Switch checked={faceRecognizing} onChange={handleToggleFaceRecognition} disabled={!playing || historyPlaying} /></span>
            <span>监控录像: <Switch checked={recording} onChange={handleToggleRecording} loading={recordingLoading} /></span>
            {faceRecognizing && (
              <Button type="link" icon={<SmileOutlined />} onClick={() => { facePauseRef.current = true; setRegisterModalOpen(true) }} size="small">
                注册人脸
              </Button>
            )}
            {historyPlaying && (
              <Button
                size="small"
                icon={<VideoCameraOutlined />}
                type="primary"
                onClick={handleReturnLive}
              >
                返回实时
              </Button>
            )}
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
            <div ref={videoContainerRef} style={{
              background: '#1e1e1e', borderRadius: 12, overflow: 'hidden',
              minHeight: 360, display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              {playing ? (
                <>
                  <img
                    ref={mjpegImgRef}
                    crossOrigin="anonymous"
                    style={{ display: 'none' }}
                    alt=""
                  />
                  <div style={{ position: 'relative', lineHeight: 0 }}>
                    {historyPlaying && (
                      <video src={playbackUrl} controls autoPlay style={{ maxWidth: '100%', maxHeight: 480, display: 'block' }} />
                    )}
                    {mjpegMode && !historyPlaying && (
                      <canvas
                        ref={displayCanvasRef}
                        style={{ maxWidth: '100%', maxHeight: 480, display: 'block' }}
                      />
                    )}
                    {!mjpegMode && !historyPlaying && (
                      <video ref={videoRef} controls autoPlay muted style={{ maxWidth: '100%', maxHeight: 480, display: 'block' }} />
                    )}
                    {faceRecognizing && !historyPlaying && (
                      <FaceOverlay
                        faces={detectedFaces}
                        videoWidth={canvasSizeRef.current.w}
                        videoHeight={canvasSizeRef.current.h}
                        displayWidth={displaySize.w}
                        displayHeight={displaySize.h}
                      />
                    )}
                  </div>
                </>
              ) : (
                <Empty description="点击「播放」开始实时画面" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </div>
            {mjpegMode && (
              <div style={{ marginTop: 8, fontSize: 12, color: '#888', textAlign: 'center' }}>
                直连摄像头 MJPEG 流，视频数据不经过服务器
              </div>
            )}
          </div>
          <div style={{ flex: '0 0 280px', minWidth: 240 }}>
            {camera && (() => {
              let host = extractHost(camera.stream_url)
              return (
              <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
                <Descriptions.Item label="协议">{camera.protocol?.toUpperCase()}</Descriptions.Item>
                <Descriptions.Item label="IP:端口"><code>{host}</code></Descriptions.Item>
                <Descriptions.Item label="地址" styles={{ content: { wordBreak: 'break-all' } }}>
                  {camera.stream_url}
                </Descriptions.Item>
                <Descriptions.Item label="最近在线">{camera.last_seen_at ? new Date(camera.last_seen_at).toLocaleString() : '-'}</Descriptions.Item>
              </Descriptions>
              )
            })()}
            <Card
              title="录像回放"
              size="small"
              style={{ marginBottom: 16 }}
              extra={<Button size="small" onClick={fetchRecordingState}>刷新</Button>}
            >
              <Space direction="vertical" size="small" style={{ width: '100%', marginBottom: 12 }}>
                <Space wrap size="small">
                  <Tag color="blue">每 5 分钟保存一个片段</Tag>
                  <Tag color="green">永久保留</Tag>
                </Space>
                {recording && <Tag color="red">正在录像，停止后仍会保留，需手动删除</Tag>}
              </Space>
              <Table
                dataSource={recordings}
                columns={recordingColumns}
                rowKey="id"
                size="small"
                pagination={{ pageSize: 5, size: 'small' }}
                locale={{ emptyText: '暂无录像' }}
              />
            </Card>
            <Card title="识别记录" size="small" extra={<Button size="small" onClick={() => { fetchEvents(); fetchFaceEvents() }}>刷新</Button>}>
              <Tabs defaultActiveKey="object" size="small" items={tabs} />
            </Card>
          </div>
        </div>
      </Card>

      <FaceRegisterModal
        open={registerModalOpen}
        videoEl={mjpegMode ? displayCanvasRef.current : videoRef.current}
        onClose={() => { facePauseRef.current = false; setRegisterModalOpen(false) }}
        onRegister={async (name, embedding, thumbnailDataUrl) => {
          const { createFaceProfile } = await import('../services/camera')
          await createFaceProfile({ name, embedding, thumbnail_url: thumbnailDataUrl })
        }}
      />
    </div>
  )
}
