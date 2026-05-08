import { useEffect, useRef } from 'react'

export interface FaceBox {
  x: number
  y: number
  width: number
  height: number
  name?: string
}

interface Props {
  faces: FaceBox[]
  videoWidth?: number
  videoHeight?: number
  displayWidth?: number
  displayHeight?: number
}

export default function FaceOverlay({
  faces,
  videoWidth = 640,
  videoHeight = 360,
  displayWidth,
  displayHeight,
}: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const dw = displayWidth || videoWidth
  const dh = displayHeight || videoHeight
  const sx = dw / videoWidth
  const sy = dh / videoHeight

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    for (const f of faces) {
      const x = f.x * sx
      const y = f.y * sy
      const w = f.width * sx
      const h = f.height * sy

      ctx.strokeStyle = f.name ? '#52c41a' : '#e8964a'
      ctx.lineWidth = 2
      ctx.strokeRect(x, y, w, h)

      if (f.name) {
        const label = `${f.name}`
        ctx.font = '13px -apple-system, sans-serif'
        const tm = ctx.measureText(label)
        const lw = tm.width + 12
        ctx.fillStyle = '#52c41a'
        ctx.fillRect(x, y - 22, lw, 22)
        ctx.fillStyle = '#fff'
        ctx.fillText(label, x + 6, y - 6)
      }
    }
  }, [faces, sx, sy])

  return (
    <canvas
      ref={canvasRef}
      width={dw}
      height={dh}
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
      }}
    />
  )
}
