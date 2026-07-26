import { useEffect, useState } from 'react'
import { Slider, Button, Typography, List } from 'antd'
import {
  StepBackwardOutlined,
  StepForwardOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ShrinkOutlined,
  RetweetOutlined,
  SwapOutlined,
  CustomerServiceOutlined,
  SoundOutlined,
  DownCircleOutlined,
} from '@ant-design/icons'
import { usePlayerStore, type PlayMode } from '../../stores/playerStore'
import { colors, radius, shadow, motion } from '../../theme/tokens'
import LyricsPanel from './LyricsPanel'
import { getLyrics } from '../../services/music'

const { Text } = Typography

function formatTime(s: number): string {
  if (!s || !isFinite(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}

const modeIcons: Record<PlayMode, React.ReactNode> = {
  sequential: <DownCircleOutlined />,
  shuffle: <SwapOutlined />,
  'repeat-one': <RetweetOutlined style={{ position: 'relative' }} />,
  'repeat-all': <RetweetOutlined />,
}

const modeLabels: Record<PlayMode, string> = {
  sequential: '顺序播放',
  shuffle: '随机播放',
  'repeat-one': '单曲循环',
  'repeat-all': '列表循环',
}

const modes: PlayMode[] = ['sequential', 'repeat-one', 'repeat-all', 'shuffle']

export default function FullPlayer() {
  const {
    queue, currentIndex, isPlaying, currentTime, duration, errorMessage,
    volume, isMuted, mode,
    pause, resume, next, prev, seek, setVolume, setMode,
    toggleFullScreen,
  } = usePlayerStore()

  const [showQueue, setShowQueue] = useState(false)
  const [showLyrics, setShowLyrics] = useState(false)
  const [lyrics, setLyrics] = useState<string | null>(null)
  const [showVol, setShowVol] = useState(false)

  const track = queue[currentIndex]

  useEffect(() => {
    if (!track) {
      setLyrics(null)
      return
    }
    getLyrics(track.id, track.source)
      .then((lrc) => setLyrics(lrc || null))
      .catch(() => setLyrics(null))
  }, [track?.id])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') toggleFullScreen()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [toggleFullScreen])

  const cycleMode = () => {
    const idx = modes.indexOf(mode)
    setMode(modes[(idx + 1) % modes.length])
  }

  if (!track) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 200,
        background: 'linear-gradient(135deg, #000000 0%, #0a0a10 50%, #0a0a10 100%)',
        color: '#fff',
        display: 'flex',
        flexDirection: 'column',
        animation: `${motion.normal} ease-out`,
      }}
    >
      {/* 椤堕儴宸ュ叿鏍?*/}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '16px 24px',
        flexShrink: 0,
      }}>
        <Button
          type="text"
          icon={<ShrinkOutlined style={{ color: '#fff', fontSize: 20 }} />}
          onClick={toggleFullScreen}
          style={{ background: 'transparent' }}
        />
        <div style={{ display: 'flex', gap: 8 }}>
          <Button
            type="text"
            icon={<CustomerServiceOutlined style={{ color: showLyrics ? colors.primary : '#fff' }} />}
            onClick={() => setShowLyrics(!showLyrics)}
            style={{ background: 'transparent' }}
          />
          <Button
            type="text"
            icon={<DownCircleOutlined style={{ color: showQueue ? colors.primary : '#fff' }} />}
            onClick={() => setShowQueue(!showQueue)}
            style={{ background: 'transparent' }}
          />
        </div>
      </div>

      {/* 涓诲唴瀹瑰尯 */}
      <div style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '0 24px',
        overflow: 'hidden',
      }}>
        {showLyrics ? (
          <div style={{ width: '100%', maxWidth: 600 }}>
            <LyricsPanel lyrics={lyrics} currentTime={currentTime} />
          </div>
        ) : showQueue ? (
          <div style={{
            width: '100%',
            maxWidth: 500,
            maxHeight: '60vh',
            overflow: 'auto',
            background: 'rgba(255,255,255,0.08)',
            borderRadius: radius.lg,
            padding: 8,
          }}>
            <List
              dataSource={queue}
              renderItem={(item, idx) => (
                <div
                  key={item.id}
                  onClick={() => {
                    usePlayerStore.getState().play(item, queue)
                  }}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    padding: '8px 12px',
                    borderRadius: radius.sm,
                    cursor: 'pointer',
                    background: idx === currentIndex ? 'rgba(129,236,254,0.15)' : 'transparent',
                  }}
                >
                  <Text style={{ color: idx === currentIndex ? colors.primary : '#aaa', fontSize: 12, width: 24, textAlign: 'center', flexShrink: 0 }}>
                    {idx === currentIndex ? '▶' : idx + 1}
                  </Text>
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <Text ellipsis style={{ color: idx === currentIndex ? '#fff' : '#ccc', fontSize: 13, display: 'block' }}>
                      {item.title}
                    </Text>
                    <Text ellipsis style={{ color: '#888', fontSize: 11 }}>
                      {item.artist || '未知艺术家'}
                    </Text>
                  </div>
                </div>
              )}
            />
          </div>
        ) : (
          <div style={{ textAlign: 'center' }}>
            {/* 灏侀潰鍥?*/}
            <div style={{
              width: 280,
              height: 280,
              margin: '0 auto 24px',
              borderRadius: radius.xl,
              background: 'rgba(255,255,255,0.08)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: shadow.modal,
              overflow: 'hidden',
            }}>
              <CustomerServiceOutlined style={{ fontSize: 80, color: 'rgba(255,255,255,0.3)' }} />
            </div>
            {/* 姝屾洸淇℃伅 */}
            <Text style={{ fontSize: 22, fontWeight: 600, color: '#fff', display: 'block' }}>
              {track.title}
            </Text>
            <Text style={{ fontSize: 14, color: 'rgba(255,255,255,0.6)', display: 'block', marginTop: 4 }}>
              {track.artist || '未知艺术家'} {track.album ? `· ${track.album}` : ''}
            </Text>
            {errorMessage && (
              <Text style={{ fontSize: 13, color: colors.error, display: 'block', marginTop: 10 }}>
                {errorMessage}
              </Text>
            )}
          </div>
        )}
      </div>

      {/* 搴曢儴鎺у埗鍖?*/}
      <div style={{ padding: '16px 32px 32px', flexShrink: 0 }}>
        {/* 杩涘害鏉?*/}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
          <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.6)', width: 40, textAlign: 'right', flexShrink: 0 }}>
            {formatTime(currentTime)}
          </Text>
          <Slider
            min={0}
            max={duration || 1}
            value={currentTime}
            onChange={seek}
            tooltip={{ formatter: (v?: number) => formatTime(v || 0) }}
            styles={{
              track: { background: colors.primary },
              handle: { borderColor: colors.primary },
            }}
            style={{ margin: 0, flex: 1 }}
          />
          <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.6)', width: 40, flexShrink: 0 }}>
            {formatTime(duration)}
          </Text>
        </div>

        {/* 鎺у埗鎸夐挳 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 24 }}>
          <div style={{ position: 'relative' }}>
            <Button
              type="text"
              icon={<SoundOutlined style={{ color: '#fff', fontSize: 18 }} />}
              onClick={() => setShowVol(!showVol)}
              style={{ background: 'transparent' }}
            />
            {showVol && (
              <div style={{
                position: 'absolute',
                bottom: 36,
                left: '50%',
                transform: 'translateX(-50%)',
                background: 'rgba(255,255,255,0.12)',
                padding: '8px 4px',
                borderRadius: radius.md,
                height: 100,
              }}>
                <Slider
                  vertical
                  min={0}
                  max={1}
                  step={0.01}
                  value={isMuted ? 0 : volume}
                  onChange={(v) => setVolume(v as number)}
                  styles={{ track: { background: colors.primary }, handle: { borderColor: colors.primary } }}
                  style={{ height: 80 }}
                />
              </div>
            )}
          </div>

          <Button
            type="text"
            icon={<StepBackwardOutlined style={{ color: '#fff', fontSize: 24 }} />}
            onClick={prev}
            style={{ background: 'transparent' }}
          />
          <Button
            type="text"
            icon={isPlaying
              ? <PauseCircleOutlined style={{ fontSize: 48, color: '#fff' }} />
              : <PlayCircleOutlined style={{ fontSize: 48, color: '#fff' }} />
            }
            onClick={() => isPlaying ? pause() : resume()}
            style={{ background: 'transparent' }}
          />
          <Button
            type="text"
            icon={<StepForwardOutlined style={{ color: '#fff', fontSize: 24 }} />}
            onClick={next}
            style={{ background: 'transparent' }}
          />

          <Button
            type="text"
            icon={modeIcons[mode]}
            onClick={cycleMode}
            title={modeLabels[mode]}
            style={{ background: 'transparent', color: mode === 'sequential' ? 'rgba(255,255,255,0.4)' : colors.primary }}
          />
        </div>
      </div>
    </div>
  )
}
