import { useState, useRef, useEffect } from 'react'
import { Modal, Input, message, Spin } from 'antd'
import { detectFaces, embeddingToArray } from '../utils/faceDetection'

interface Props {
  open: boolean
  videoEl: HTMLVideoElement | null
  onClose: () => void
  onRegister: (name: string, embedding: number[], thumbnailDataUrl: string) => Promise<void>
}

export default function FaceRegisterModal({ open, videoEl, onClose, onRegister }: Props) {
  const [name, setName] = useState('')
  const [loading, setLoading] = useState(false)
  const [detected, setDetected] = useState(false)
  const [embedding, setEmbedding] = useState<number[] | null>(null)
  const [thumbnail, setThumbnail] = useState<string>('')
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!open) {
      setName('')
      setDetected(false)
      setEmbedding(null)
      setThumbnail('')
      return
    }
    detectFace()
  }, [open])

  const detectFace = async () => {
    if (!videoEl) return
    setLoading(true)
    try {
      const canvas = canvasRef.current
      if (!canvas) return
      canvas.width = videoEl.videoWidth || 640
      canvas.height = videoEl.videoHeight || 360
      const ctx = canvas.getContext('2d')!
      ctx.drawImage(videoEl, 0, 0)

      const faces = await detectFaces(canvas)
      if (faces.length === 0) {
        message.warning('未检测到人脸，请面对摄像头')
        setDetected(false)
      } else {
        // Draw face crop
        const f = faces[0]
        setEmbedding(embeddingToArray(f.descriptor))
        setThumbnail(canvas.toDataURL('image/jpeg', 0.8))
        setDetected(true)
      }
    } catch {
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
    if (!embedding) {
      message.warning('人脸特征提取失败，请重试')
      return
    }
    setLoading(true)
    try {
      await onRegister(name.trim(), embedding, thumbnail)
      message.success(`已注册: ${name}`)
      onClose()
    } catch {
      message.error('注册失败')
    } finally {
      setLoading(false)
    }
  }

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
      <div style={{ textAlign: 'center', marginBottom: 16 }}>
        <canvas ref={canvasRef} style={{ display: 'none' }} />
        {detected ? (
          <div style={{ width: 200, height: 200, margin: '0 auto', background: '#1e1e1e', borderRadius: 8, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            {thumbnail ? (
              <img src={thumbnail} alt="人脸" style={{ maxWidth: '100%', maxHeight: '100%', borderRadius: 8 }} />
            ) : (
              <Spin />
            )}
          </div>
        ) : (
          <div style={{ width: 200, height: 200, margin: '0 auto', background: '#1e1e1e', borderRadius: 8, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <span style={{ color: '#8c8c8c' }}>{loading ? <Spin /> : '等待检测...'}</span>
          </div>
        )}
        <div style={{ marginTop: 8, fontSize: 12, color: '#8c8c8c' }}>
          {detected ? '已检测到人脸，请输入姓名' : '请面对摄像头，确保人脸清晰可见'}
        </div>
      </div>
      <Input
        placeholder="输入姓名"
        value={name}
        onChange={(e) => setName(e.target.value)}
        disabled={!detected}
      />
    </Modal>
  )
}
