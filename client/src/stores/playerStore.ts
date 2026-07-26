import { create } from 'zustand'
import type { Track } from '../services/music'

export type PlayMode = 'sequential' | 'shuffle' | 'repeat-one' | 'repeat-all'

export interface PlayerState {
  queue: Track[]
  currentIndex: number
  isPlaying: boolean
  currentTime: number
  duration: number
  volume: number
  isMuted: boolean
  mode: PlayMode
  isMiniVisible: boolean
  isFullScreen: boolean
  errorMessage: string | null

  play: (track?: Track, queue?: Track[]) => void
  switchTrackSource: (trackId: string, source: 'public' | 'cloud') => void
  pause: () => void
  resume: () => void
  next: () => void
  prev: () => void
  seek: (time: number) => void
  setVolume: (v: number) => void
  toggleMute: () => void
  setMode: (mode: PlayMode) => void
  setPlaying: (playing: boolean) => void
  setTime: (time: number) => void
  setDuration: (d: number) => void
  setError: (message: string | null) => void
  showMini: () => void
  hideMini: () => void
  toggleFullScreen: () => void
}

export const usePlayerStore = create<PlayerState>((set, get) => ({
  queue: [],
  currentIndex: -1,
  isPlaying: false,
  currentTime: 0,
  duration: 0,
  volume: 0.7,
  isMuted: false,
  mode: 'sequential',
  isMiniVisible: false,
  isFullScreen: false,
  errorMessage: null,

  play: (track?: Track, queue?: Track[]) => {
    if (queue) {
      const idx = track ? queue.findIndex((t) => t.id === track.id && t.source === track.source) : 0
      set({ queue, currentIndex: idx >= 0 ? idx : 0, isPlaying: true, isMiniVisible: true, errorMessage: null })
    } else if (track) {
      set({ queue: [track], currentIndex: 0, isPlaying: true, isMiniVisible: true, errorMessage: null })
    } else {
      set({ isPlaying: true, errorMessage: null })
    }
  },

  switchTrackSource: (trackId: string, source: 'public' | 'cloud') => {
    const { queue, currentIndex } = get()
    const nextQueue = queue.map((track, idx) => {
      if (idx !== currentIndex || track.id !== trackId || !track.alternatives?.length) {
        return track
      }
      const nextVariant = track.alternatives.find((item) => item.source === source)
      if (!nextVariant) {
        return track
      }
      const remainingAlternatives = [
        {
          id: track.id,
          title: track.title,
          artist: track.artist,
          album: track.album,
          duration: track.duration,
          mime_type: track.mime_type,
          file_size: track.file_size,
          source: track.source,
          is_uploaded: track.is_uploaded,
        },
        ...track.alternatives.filter((item) => !(item.id === nextVariant.id && item.source === nextVariant.source)),
      ]
      return {
        ...track,
        ...nextVariant,
        alternatives: remainingAlternatives,
      }
    })
    set({ queue: nextQueue, isPlaying: true, currentTime: 0, errorMessage: null })
  },

  pause: () => set({ isPlaying: false }),
  resume: () => set({ isPlaying: true }),

  next: () => {
    const { queue, currentIndex, mode } = get()
    if (queue.length === 0) return
    let next = currentIndex + 1
    if (next >= queue.length) {
      next = mode === 'repeat-all' ? 0 : currentIndex
    }
    if (mode === 'shuffle') {
      next = Math.floor(Math.random() * get().queue.length)
    }
    set({ currentIndex: next, isPlaying: true, currentTime: 0, errorMessage: null })
  },

  prev: () => {
    const { queue, currentIndex } = get()
    if (queue.length === 0) return
    const prev = currentIndex > 0 ? currentIndex - 1 : queue.length - 1
    set({ currentIndex: prev, isPlaying: true, currentTime: 0, errorMessage: null })
  },

  seek: (time: number) => set({ currentTime: time }),
  setVolume: (v: number) => set({ volume: Math.max(0, Math.min(1, v)), isMuted: false }),
  toggleMute: () => set((s) => ({ isMuted: !s.isMuted })),
  setMode: (mode: PlayMode) => set({ mode }),
  setPlaying: (playing: boolean) => set({ isPlaying: playing }),
  setTime: (time: number) => set({ currentTime: time }),
  setDuration: (d: number) => set({ duration: d }),
  setError: (message: string | null) => set({ errorMessage: message, isPlaying: message ? false : get().isPlaying }),
  showMini: () => set({ isMiniVisible: true }),
  hideMini: () => set({ isMiniVisible: false }),
  toggleFullScreen: () => set((s) => ({ isFullScreen: !s.isFullScreen })),
}))
