import { useEffect, useRef, useMemo } from 'react'
import { Typography } from 'antd'
import { colors } from '../../theme/tokens'

const { Text } = Typography

interface LyricLine {
  time: number
  text: string
}

function parseLRC(lrc: string): LyricLine[] {
  const lines = lrc.split('\n')
  const result: LyricLine[] = []
  const regex = /\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)/

  for (const line of lines) {
    const match = line.match(regex)
    if (match) {
      const minutes = parseInt(match[1], 10)
      const seconds = parseInt(match[2], 10)
      const ms = parseInt(match[3].padEnd(3, '0'), 10)
      const time = minutes * 60 + seconds + ms / 1000
      const text = match[4].trim()
      if (text) {
        result.push({ time, text })
      }
    }
  }

  return result.sort((a, b) => a.time - b.time)
}

interface Props {
  lyrics: string | null
  currentTime: number
}

export default function LyricsPanel({ lyrics, currentTime }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const activeLineRef = useRef<HTMLDivElement>(null)

  const parsedLyrics = useMemo(() => {
    if (!lyrics) return []
    return parseLRC(lyrics)
  }, [lyrics])

  const activeIndex = useMemo(() => {
    if (parsedLyrics.length === 0) return -1
    for (let i = parsedLyrics.length - 1; i >= 0; i--) {
      if (currentTime >= parsedLyrics[i].time) return i
    }
    return 0
  }, [parsedLyrics, currentTime])

  useEffect(() => {
    if (activeLineRef.current && containerRef.current) {
      const container = containerRef.current
      const line = activeLineRef.current
      const containerHeight = container.clientHeight
      const lineTop = line.offsetTop
      const lineHeight = line.clientHeight
      const scrollTarget = lineTop - containerHeight / 2 + lineHeight / 2

      container.scrollTo({
        top: scrollTarget,
        behavior: 'smooth',
      })
    }
  }, [activeIndex])

  if (!lyrics) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: 300,
      }}>
        <Text style={{ color: 'rgba(255,255,255,0.3)', fontSize: 14 }}>暂无歌词</Text>
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      style={{
        height: '60vh',
        overflowY: 'auto',
        padding: '20px 0',
        maskImage: 'linear-gradient(transparent, black 15%, black 85%, transparent)',
        WebkitMaskImage: 'linear-gradient(transparent, black 15%, black 85%, transparent)',
      }}
    >
      {parsedLyrics.map((line, idx) => (
        <div
          key={`${idx}-${line.time}`}
          ref={idx === activeIndex ? activeLineRef : undefined}
          style={{
            padding: '8px 16px',
            textAlign: 'center',
            transition: 'all 0.3s ease',
            color: idx === activeIndex ? colors.primary : 'rgba(255,255,255,0.4)',
            fontSize: idx === activeIndex ? 16 : 14,
            fontWeight: idx === activeIndex ? 600 : 400,
            transform: idx === activeIndex ? 'scale(1.05)' : 'scale(1)',
          }}
        >
          {line.text}
        </div>
      ))}
    </div>
  )
}
