import { useState } from 'react'
import { Slider, Button, Typography } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  SoundOutlined,
  CloseOutlined,
  ExpandOutlined,
} from '@ant-design/icons'
import { usePlayerStore } from '../../stores/playerStore'
import { colors, radius, shadow } from '../../theme/tokens'

const { Text } = Typography

function formatTime(s: number): string {
  if (!s || !isFinite(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}

export default function MiniPlayer() {
  const {
    queue, currentIndex, isPlaying, currentTime, duration, errorMessage,
    volume, isMuted, pause, resume, next, prev, seek, setVolume,
    hideMini, toggleFullScreen,
  } = usePlayerStore()
  const [showVol, setShowVol] = useState(false)

  const track = queue[currentIndex]
  if (!track) return null

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        right: 0,
        height: 64,
        background: 'rgba(0,0,0,0.9)',
        backdropFilter: 'blur(12px)',
        borderTop: '1px solid rgba(255,255,255,0.06)',
        display: 'flex',
        alignItems: 'center',
        padding: '0 16px',
        gap: 12,
        zIndex: 99,
        boxShadow: '0 -2px 12px rgba(0,0,0,0.4)',
      }}
    >
      <div style={{ flex: '0 0 auto', minWidth: 0, maxWidth: 200 }}>
        <Text strong ellipsis style={{ fontSize: 13, display: 'block' }}>
          {track.title}
        </Text>
        <Text type="secondary" ellipsis style={{ fontSize: 11, color: errorMessage ? colors.error : undefined }}>
          {errorMessage || track.artist || '未知艺术家'}
        </Text>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}>
        <Button type="text" size="small" icon={<StepBackwardOutlined />} onClick={prev} />
        <Button
          type="text"


          icon={isPlaying ? <PauseCircleOutlined style={{ fontSize: 28 }} /> : <PlayCircleOutlined style={{ fontSize: 28 }} />}
          onClick={() => isPlaying ? pause() : resume()}
          style={{ color: colors.primary }}
        />
        <Button type="text" size="small" icon={<StepForwardOutlined />} onClick={next} />
      </div>

      <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
        <Text style={{ fontSize: 10, color: colors.textSecondary, flexShrink: 0, width: 36, textAlign: 'right' }}>
          {formatTime(currentTime)}
        </Text>
        <Slider
          min={0}
          max={duration || 1}
          value={currentTime}
          onChange={seek}
          tooltip={{ formatter: (v?: number) => formatTime(v || 0) }}
          style={{ margin: 0 }}


        />
        <Text style={{ fontSize: 10, color: colors.textSecondary, flexShrink: 0, width: 36 }}>
          {formatTime(duration)}
        </Text>
      </div>

      <div style={{ position: 'relative', flexShrink: 0 }}>
        <Button
          type="text"


          icon={<SoundOutlined />}
          onClick={() => setShowVol(!showVol)}
        />
        {showVol && (
          <div
            style={{
              position: 'absolute',
              bottom: 40,
              left: '50%',
              transform: 'translateX(-50%)',
              background: 'rgba(30,30,30,0.95)',
              padding: '8px 4px',
              borderRadius: radius.md,
              boxShadow: shadow.modal,
              height: 100,
            }}
          >
            <Slider
              vertical
              min={0}
              max={1}
              step={0.01}
              value={isMuted ? 0 : volume}
              onChange={(v) => setVolume(v as number)}
              style={{ height: 80 }}
    

            />
          </div>
        )}
      </div>

      <Button type="text" size="small" icon={<CloseOutlined />} onClick={hideMini} />
      <Button type="text" size="small" icon={<ExpandOutlined />} onClick={toggleFullScreen} style={{ color: colors.primary }} />
    </div>
  )
}
