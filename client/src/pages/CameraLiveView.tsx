import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Button, Space, Tag, message, Table, Switch, Descriptions, Empty, Tabs, Popconfirm } from 'antd'
import { ArrowLeftOutlined, PlayCircleOutlined, PauseCircleOutlined, CameraOutlined, ReloadOutlined, SmileOutlined, VideoCameraOutlined, DeleteOutlined } from '@ant-design/icons'
import type { Camera, RecognitionEvent, FaceRecognitionEvent } from '../services/camera'
import { getCameras, startStream, stopStream, startRecognition, stopRecognition, getEvents, getFaceEvents, matchFace, clearFaceEvents } from '../services/camera'
import { detectFaces, embeddingToArray, loadModels } from '../utils/faceDetection'
import { VideoRecorder } from '../utils/videoRecorder'
import { FaceTracker } from '../utils/faceTracker'
import FaceOverlay, { type FaceBox } from '../components/FaceOverlay'
import FaceRegisterModal from '../components/FaceRegisterModal'
import Hls from 'hls.js'

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
  // RTSP camera — derive HTTP URL from host
  try {
    const u = new URL(streamUrl)
    return `http://${u.hostname}:8080/video` // IP Webcam default
  } catch {
    const m = streamUrl.match(/:\/\/([^/:]+)/)
    if (m) return `http://${m[1]}:8080/video`
  }
  return null
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

  // Recording
  const [recording, setRecording] = useState(false)
  const recorderRef = useRef<VideoRecorder | null>(null)
  const recordedUrlRef = useRef('')
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

  const handleClearFaceEvents = async () => {
    if (!id) return
    try {
      const count = await clearFaceEvents(id)
      message.success(`已清空 ${count} 条人脸识别记录`)
      setFaceEvents([])
    } catch { message.error('清空失败') }
  }

  useEffect(() => { fetchCamera(); fetchEvents(); fetchFaceEvents() }, [fetchCamera, fetchEvents, fetchFaceEvents])

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
      // Stop recorder
      if (recorderRef.current) recorderRef.current.destroy()
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
      handlePlayHls()
    } finally {
      setLoading(false)
    }
  }

  const handlePlay = () => {
    if (!camera) return
    // Always try MJPEG direct first (zero server bandwidth)
    // Falls back to HLS inside handlePlayMjpeg on failure
    handlePlayMjpeg()
  }

  const handleStop = async () => {
    if (!id) return
    try {
      if (faceRecognizing) {
        if (faceTimerRef.current) clearInterval(faceTimerRef.current)
        setFaceRecognizing(false)
      }
      if (recorderRef.current) {
        recorderRef.current.destroy()
        recorderRef.current = null
        setRecording(false)
        recordedUrlRef.current = ''
        setHistoryPlaying(false)
      }
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

  // --- Recording (records from display canvas in MJPEG mode, or video element) ---
  const handleToggleRecording = (checked: boolean) => {
    const video = videoRef.current
    if (!video && !mjpegMode) { message.warning('请先开启视频流'); return }

    if (checked) {
      const rec = new VideoRecorder()
      if (mjpegMode) {
        const dc = displayCanvasRef.current
        if (dc) rec.startFromCanvas(dc)
      } else if (video) {
        rec.start(video)
      }
      recorderRef.current = rec
      setRecording(true)
      message.success('已开始保留历史视频（客户端本地录制）')
    } else {
      if (recorderRef.current) {
        recorderRef.current.destroy()
        recorderRef.current = null
      }
      setRecording(false)
      recordedUrlRef.current = ''
      setHistoryPlaying(false)
      message.success('已停止录制')
    }
  }

  const handlePlayHistory = () => {
    if (!recorderRef.current) return

    if (historyPlaying) {
      const video = videoRef.current
      if (video) {
        video.pause()
        video.src = ''
      }
      if (hlsRef.current && !mjpegMode) {
        hlsRef.current.attachMedia(video!)
      }
      setHistoryPlaying(false)
    } else {
      // Ensure video element exists for playback
      const video = videoRef.current
      if (!video) return
      const url = recorderRef.current.getBlobUrl()
      recordedUrlRef.current = url
      if (hlsRef.current && !mjpegMode) {
        hlsRef.current.detachMedia()
      }
      video.src = url
      video.play()
      setHistoryPlaying(true)
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
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => { fetchCamera(); fetchEvents(); fetchFaceEvents() }}>刷新</Button>
            <span>AI 识别: <Switch checked={recognizing} onChange={handleToggleRecognition} disabled={!playing} /></span>
            <span>人脸识别: <Switch checked={faceRecognizing} onChange={handleToggleFaceRecognition} disabled={!playing} /></span>
            <span>保留历史: <Switch checked={recording} onChange={handleToggleRecording} disabled={!playing} /></span>
            {faceRecognizing && (
              <Button type="link" icon={<SmileOutlined />} onClick={() => { facePauseRef.current = true; setRegisterModalOpen(true) }} size="small">
                注册人脸
              </Button>
            )}
            {recording && (
              <Button
                size="small"
                icon={<VideoCameraOutlined />}
                type={historyPlaying ? 'primary' : 'default'}
                onClick={handlePlayHistory}
                disabled={!recorderRef.current || recorderRef.current.getDurationMs() < 1000}
              >
                {historyPlaying ? '返回实时' : '回看历史'}
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
                    {mjpegMode && !historyPlaying && (
                      <canvas
                        ref={displayCanvasRef}
                        style={{ maxWidth: '100%', maxHeight: 480, display: 'block' }}
                      />
                    )}
                    {(mjpegMode && historyPlaying) && (
                      <video ref={videoRef} controls autoPlay muted style={{ maxWidth: '100%', maxHeight: 480, display: 'block' }} />
                    )}
                    {!mjpegMode && (
                      <video ref={videoRef} controls autoPlay muted style={{ maxWidth: '100%', maxHeight: 480, display: 'block' }} />
                    )}
                    {faceRecognizing && (
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
              <div style={{ marginTop: 8, fontSize: 12, color: '#8c8c8c', textAlign: 'center' }}>
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
