import { useState, useEffect, useCallback } from 'react'
import { Button, Space, Typography } from 'antd'
import {
  CloseOutlined,
  LeftOutlined,
  RightOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  RotateRightOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons'
import type { FileItem } from '../../services/file'

const { Text } = Typography

interface Props {
  files: FileItem[]
  currentIndex: number
  onClose: () => void
}

function getDownloadUrl(file: FileItem): string {
  const token = localStorage.getItem('access_token')
  return `/api/v1/file/download/${file.id}?token=${token}&inline=true`
}

export default function Lightbox({ files, currentIndex: initialIndex, onClose }: Props) {
  const [index, setIndex] = useState(initialIndex)
  const [zoom, setZoom] = useState(1)
  const [rotation, setRotation] = useState(0)
  const [playing, setPlaying] = useState(false)

  const file = files[index]
  const isImage = file?.mime_type?.startsWith('image/')

  const resetTransform = useCallback(() => {
    setZoom(1)
    setRotation(0)
  }, [])

  const goNext = useCallback(() => {
    setIndex((prev) => (prev + 1) % files.length)
    resetTransform()
  }, [files.length, resetTransform])

  const goPrev = useCallback(() => {
    setIndex((prev) => (prev - 1 + files.length) % files.length)
    resetTransform()
  }, [files.length, resetTransform])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      switch (e.key) {
        case 'Escape':
          onClose()
          break
        case 'ArrowRight':
          goNext()
          break
        case 'ArrowLeft':
          goPrev()
          break
        case '+':
        case '=':
          setZoom((z) => Math.min(z + 0.25, 3))
          break
        case '-':
          setZoom((z) => Math.max(z - 0.25, 0.5))
          break
        case 'r':
          setRotation((r) => r + 90)
          break
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose, goNext, goPrev])

  useEffect(() => {
    if (!playing) return
    const timer = setInterval(goNext, 3000)
    return () => clearInterval(timer)
  }, [playing, goNext])

  if (!file) return null

  return (
    <div
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 1000,
        background: 'rgba(0,0,0,0.95)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      {/* Toolbar */}
      <div
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: 56,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 16px',
          background: 'linear-gradient(rgba(0,0,0,0.6), transparent)',
          zIndex: 2,
        }}
      >
        <Text style={{ color: '#fff', fontSize: 14, maxWidth: '60%' }} ellipsis>
          {file.name}
          <span style={{ marginLeft: 12, opacity: 0.6 }}>
            {index + 1} / {files.length}
          </span>
        </Text>
        <Space size="small">
          <Button
            type="text"
            icon={playing ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
            onClick={() => setPlaying(!playing)}
            style={{ color: '#fff' }}
          />
          <Button type="text" icon={<ZoomInOutlined />} onClick={() => setZoom((z) => Math.min(z + 0.25, 3))} style={{ color: '#fff' }} />
          <Button type="text" icon={<ZoomOutOutlined />} onClick={() => setZoom((z) => Math.max(z - 0.25, 0.5))} style={{ color: '#fff' }} />
          <Button type="text" icon={<RotateRightOutlined />} onClick={() => setRotation((r) => r + 90)} style={{ color: '#fff' }} />
          <Button type="text" icon={<CloseOutlined />} onClick={onClose} style={{ color: '#fff' }} />
        </Space>
      </div>

      {/* Navigation arrows */}
      <Button
        type="text"
        icon={<LeftOutlined style={{ fontSize: 24 }} />}
        onClick={(e) => { e.stopPropagation(); goPrev() }}
        style={{
          position: 'absolute',
          left: 16,
          top: '50%',
          transform: 'translateY(-50%)',
          color: '#fff',
          zIndex: 2,
          width: 48,
          height: 48,
        }}
      />
      <Button
        type="text"
        icon={<RightOutlined style={{ fontSize: 24 }} />}
        onClick={(e) => { e.stopPropagation(); goNext() }}
        style={{
          position: 'absolute',
          right: 16,
          top: '50%',
          transform: 'translateY(-50%)',
          color: '#fff',
          zIndex: 2,
          width: 48,
          height: 48,
        }}
      />

      {/* Content */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '100%',
          height: '100%',
          padding: '64px 80px',
        }}
      >
        {isImage ? (
          <img
            src={getDownloadUrl(file)}
            alt={file.name}
            style={{
              maxWidth: '100%',
              maxHeight: '100%',
              objectFit: 'contain',
              transform: `scale(${zoom}) rotate(${rotation}deg)`,
              transition: 'transform 0.15s ease',
              userSelect: 'none',
              pointerEvents: 'none',
            }}
          />
        ) : (
          <div style={{ color: '#fff', textAlign: 'center' }}>
            <div style={{ fontSize: 48, marginBottom: 16 }}>
              {file.mime_type?.startsWith('video/') ? '🎬' : '📄'}
            </div>
            <Text style={{ color: '#fff' }}>{file.name}</Text>
            <div style={{ marginTop: 8 }}>
              <a
                href={getDownloadUrl(file)}
                target="_blank"
                rel="noopener noreferrer"
                style={{ color: '#e8964a' }}
              >
                打开原文件
              </a>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
