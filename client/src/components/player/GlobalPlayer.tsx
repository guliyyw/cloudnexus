import { useEffect, useRef } from 'react'
import { usePlayerStore } from '../../stores/playerStore'
import { getStreamUrl } from '../../services/music'
import MiniPlayer from './MiniPlayer'
import FullPlayer from './FullPlayer'

export default function GlobalPlayer() {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const {
    queue, currentIndex, isPlaying, volume, isMuted, mode,
    setPlaying, setTime, setDuration, next, isMiniVisible, isFullScreen,
  } = usePlayerStore()

  const track = queue[currentIndex]

  useEffect(() => {
    const audio = audioRef.current
    if (!audio || !track) return

    audio.src = getStreamUrl(track.id, track.source)
    audio.volume = isMuted ? 0 : volume
    audio.play().catch(() => {})
  }, [track?.id])

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return
    if (isPlaying) {
      audio.play().catch(() => setPlaying(false))
    } else {
      audio.pause()
    }
  }, [isPlaying])

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return
    audio.volume = isMuted ? 0 : volume
  }, [volume, isMuted])

  return (
    <>
      <audio
        ref={audioRef}
        onTimeUpdate={() => {
          if (audioRef.current) setTime(audioRef.current.currentTime)
        }}
        onLoadedMetadata={() => {
          if (audioRef.current) setDuration(audioRef.current.duration)
        }}
        onEnded={() => {
          if (mode === 'repeat-one') {
            if (audioRef.current) {
              audioRef.current.currentTime = 0
              audioRef.current.play().catch(() => {})
            }
          } else {
            next()
          }
        }}
        onError={() => setPlaying(false)}
      />
      {isFullScreen && <FullPlayer />}
      {isMiniVisible && !isFullScreen && <MiniPlayer />}
    </>
  )
}
