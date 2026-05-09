import { useState, useEffect, useRef } from 'react'
import { Modal, Input, message, Spin, Radio } from 'antd'
import { detectFaces, embeddingToArray } from '../utils/faceDetection'

interface DetectedPerson {
  index: number
  embedding: number[]
  thumbnail: string
  box: { x: number; y: number; width: number; height: number }
}

interface Props {
  open: boolean
  videoEl: HTMLVideoElement | HTMLCanvasElement | HTMLImageElement | null
  onClose: () => void
  onRegister: (name: string, embedding: number[], thumbnailDataUrl: string) => Promise<void>
}

export default function FaceRegisterModal({ open, videoEl, onClose, onRegister }: Props) {
  const [name, setName] = useState('')
  const [loading, setLoading] = useState(false)
  const [people, setPeople] = useState<DetectedPerson[]>([])
  const [selectedIdx, setSelectedIdx] = useState(0)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!open) {
      setName('')
      setPeople([])
      setSelectedIdx(0)
      return
    }
    detectFace()
  }, [open])

  const detectFace = async () => {
    if (!videoEl) return
    setLoading(true)
    setPeople([])
    try {
      const faces = await detectFaces(videoEl)
      if (faces.length === 0) {
        message.warning('未检测到人脸，请面对摄像头')
        setLoading(false)
        return
      }

      const canvas = canvasRef.current
      if (!canvas) { setLoading(false); return }

      const isVideo = videoEl instanceof HTMLVideoElement
      const srcW = isVideo ? (videoEl as HTMLVideoElement).videoWidth : (videoEl as HTMLCanvasElement).width
      const srcH = isVideo ? (videoEl as HTMLVideoElement).videoHeight : (videoEl as HTMLCanvasElement).height
      canvas.width = srcW || 640
      canvas.height = srcH || 360
      const ctx = canvas.getContext('2d')!
      ctx.drawImage(videoEl, 0, 0, canvas.width, canvas.height, 0, 0, canvas.width, canvas.height)

      const detected: DetectedPerson[] = []
      for (let i = 0; i < faces.length; i++) {
        const f = faces[i]
        const b = f.box
        const pad = 0.2
        const px = Math.max(0, b.x - b.width * pad)
        const py = Math.max(0, b.y - b.height * pad)
        const pw = Math.min(canvas.width - px, b.width * (1 + pad * 2))
        const ph = Math.min(canvas.height - py, b.height * (1 + pad * 2))

        const faceCanvas = document.createElement('canvas')
        faceCanvas.width = 128
        faceCanvas.height = 128
        const fctx = faceCanvas.getContext('2d')!
        fctx.drawImage(canvas, px, py, pw, ph, 0, 0, 128, 128)
        detected.push({
          index: i,
          embedding: embeddingToArray(f.descriptor),
          thumbnail: faceCanvas.toDataURL('image/jpeg', 0.8),
          box: b,
        })
      }
      setPeople(detected)
      setSelectedIdx(0)
    } catch (e) {
      console.error('Face detection error:', e)
      message.error('人脸检测失败')
    } finally {
      setLoading(false)
    }
  }

  const handleOk = async () => {
    if (!name.trim()) {
      message.warning('请输入姓名')
      return
    }
    const person = people[selectedIdx]
    if (!person) {
      message.warning('人脸特征提取失败，请重试')
      return
    }
    setLoading(true)
    try {
      await onRegister(name.trim(), person.embedding, person.thumbnail)
      message.success(`已注册: ${name}`)
      onClose()
    } catch {
      message.error('注册失败')
    } finally {
      setLoading(false)
    }
  }

  const hasFaces = people.length > 0

  return (
    <Modal
      title="注册人脸"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={loading}
      okText="注册"
      cancelText="取消"
      destroyOnClose
    >
      <canvas ref={canvasRef} style={{ display: 'none' }} />
      <div style={{ textAlign: 'center', marginBottom: 16 }}>
        {hasFaces ? (
          <>
            <div style={{ marginBottom: 8, fontSize: 13, color: '#595959' }}>
              检测到 {people.length} 人，请选择要注册的人脸
            </div>
            <Radio.Group
              value={selectedIdx}
              onChange={(e) => setSelectedIdx(e.target.value)}
              style={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'center', gap: 12 }}
            >
              {people.map((p) => (
                <Radio.Button
                  key={p.index}
                  value={p.index}
                  style={{
                    width: 100, height: 110, padding: 0, overflow: 'hidden',
                    border: selectedIdx === p.index ? '2px solid #e8964a' : '2px solid #d9d9d9',
                    borderRadius: 8, display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}
                >
                  <img src={p.thumbnail} alt={`人脸 ${p.index + 1}`}
                    style={{ width: 96, height: 106, objectFit: 'cover', borderRadius: 6 }} />
                </Radio.Button>
              ))}
            </Radio.Group>
          </>
        ) : (
          <div style={{
            width: 200, height: 200, margin: '0 auto', background: '#1e1e1e',
            borderRadius: 8, display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <span style={{ color: '#8c8c8c' }}>{loading ? <Spin /> : '等待检测...'}</span>
          </div>
        )}
        <div style={{ marginTop: 8, fontSize: 12, color: '#8c8c8c' }}>
          {hasFaces ? '已检测到人脸，请输入姓名' : '请面对摄像头，确保人脸清晰可见'}
        </div>
      </div>
      <Input
        placeholder="输入姓名"
        value={name}
        onChange={(e) => setName(e.target.value)}
        disabled={!hasFaces}
      />
    </Modal>
  )
}
