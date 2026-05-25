import { create } from 'zustand'
import {
  getPlaylists,
  createPlaylist,
  getPlaylist,
  updatePlaylist,
  deletePlaylist,
  addTrackToPlaylist,
  removeTrackFromPlaylist,
  type Playlist,
  type PlaylistTrack,
} from '../services/music'

interface PlaylistState {
  playlists: Playlist[]
  currentPlaylist: Playlist | null
  currentTracks: PlaylistTrack[]
  loading: boolean

  fetchPlaylists: () => Promise<void>
  fetchPlaylist: (id: string) => Promise<void>
  create: (name: string, isPublic?: boolean) => Promise<Playlist>
  update: (id: string, data: { name?: string; is_public?: boolean }) => Promise<void>
  remove: (id: string) => Promise<void>
  addTrack: (playlistId: string, trackId: string, source: string) => Promise<void>
  removeTrack: (playlistId: string, trackId: string) => Promise<void>
}

export const usePlaylistStore = create<PlaylistState>((set) => ({
  playlists: [],
  currentPlaylist: null,
  currentTracks: [],
  loading: false,

  fetchPlaylists: async () => {
    set({ loading: true })
    try {
      const playlists = await getPlaylists()
      set({ playlists })
    } catch { /* ignore */ }
    set({ loading: false })
  },

  fetchPlaylist: async (id: string) => {
    set({ loading: true })
    try {
      const data = await getPlaylist(id)
      set({ currentPlaylist: data.playlist, currentTracks: data.tracks })
    } catch { /* ignore */ }
    set({ loading: false })
  },

  create: async (name: string, isPublic = false) => {
    const pl = await createPlaylist(name, isPublic)
    set((s) => ({ playlists: [pl, ...s.playlists] }))
    return pl
  },

  update: async (id: string, data: { name?: string; is_public?: boolean }) => {
    const pl = await updatePlaylist(id, data)
    set((s) => ({
      playlists: s.playlists.map((p) => (p.id === id ? pl : p)),
      currentPlaylist: s.currentPlaylist?.id === id ? pl : s.currentPlaylist,
    }))
  },

  remove: async (id: string) => {
    await deletePlaylist(id)
    set((s) => ({
      playlists: s.playlists.filter((p) => p.id !== id),
      currentPlaylist: s.currentPlaylist?.id === id ? null : s.currentPlaylist,
    }))
  },

  addTrack: async (playlistId: string, trackId: string, source: string) => {
    await addTrackToPlaylist(playlistId, trackId, source)
  },

  removeTrack: async (playlistId: string, trackId: string) => {
    await removeTrackFromPlaylist(playlistId, trackId)
    set((s) => ({
      currentTracks: s.currentTracks.filter((t) => t.track_id !== trackId),
    }))
  },
}))
