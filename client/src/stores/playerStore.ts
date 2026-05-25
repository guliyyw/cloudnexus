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

  play: (track?: Track, queue?: Track[]) => void
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

  play: (track?: Track, queue?: Track[]) => {
    if (queue) {
      const idx = track ? queue.findIndex((t) => t.id === track.id) : 0
      set({ queue, currentIndex: idx >= 0 ? idx : 0, isPlaying: true, isMiniVisible: true })
    } else if (track) {
      set({ queue: [track], currentIndex: 0, isPlaying: true, isMiniVisible: true })
    } else {
      set({ isPlaying: true })
    }
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
    set({ currentIndex: next, isPlaying: true, currentTime: 0 })
  },

  prev: () => {
    const { queue, currentIndex } = get()
    if (queue.length === 0) return
    const prev = currentIndex > 0 ? currentIndex - 1 : queue.length - 1
    set({ currentIndex: prev, isPlaying: true, currentTime: 0 })
  },

  seek: (time: number) => set({ currentTime: time }),
  setVolume: (v: number) => set({ volume: Math.max(0, Math.min(1, v)), isMuted: false }),
  toggleMute: () => set((s) => ({ isMuted: !s.isMuted })),
  setMode: (mode: PlayMode) => set({ mode }),
  setPlaying: (playing: boolean) => set({ isPlaying: playing }),
  setTime: (time: number) => set({ currentTime: time }),
  setDuration: (d: number) => set({ duration: d }),
  showMini: () => set({ isMiniVisible: true }),
  hideMini: () => set({ isMiniVisible: false }),
  toggleFullScreen: () => set((s) => ({ isFullScreen: !s.isFullScreen })),
}))
