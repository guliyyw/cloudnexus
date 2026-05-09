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
  videoWidth: number
  videoHeight: number
  displayWidth: number
  displayHeight: number
}

export default function FaceOverlay({
  faces,
  videoWidth,
  videoHeight,
  displayWidth,
  displayHeight,
}: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const sx = displayWidth / videoWidth
  const sy = displayHeight / videoHeight

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
        const label = f.name
        const fontSize = Math.max(12, Math.min(16, h * 0.25))
        ctx.font = `${fontSize}px -apple-system, sans-serif`
        const tm = ctx.measureText(label)
        const lw = tm.width + 12
        const lh = fontSize + 8
        const lx = x
        const ly = y + h + 2

        ctx.fillStyle = '#52c41a'
        ctx.fillRect(lx, ly, lw, lh)
        ctx.fillStyle = '#fff'
        ctx.fillText(label, lx + 6, ly + fontSize - 1)
      }
    }
  }, [faces, sx, sy])

  return (
    <canvas
      ref={canvasRef}
      width={displayWidth}
      height={displayHeight}
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: displayWidth,
        height: displayHeight,
        pointerEvents: 'none',
      }}
    />
  )
}
