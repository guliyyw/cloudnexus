import api from './api'

export interface Track {
  id: string
  title: string
  artist: string
  album: string
  duration: number
  mime_type: string
  file_size: number
  source: 'public' | 'cloud'
}

export interface LibraryResponse {
  tracks: Track[]
  total: number
}

export interface Playlist {
  id: string
  owner_id: string
  name: string
  is_public: boolean
  track_count: number
  created_at: string
  updated_at: string
}

export interface PlaylistTrack {
  playlist_id: string
  track_id: string
  source: string
  sort_order: number
}

export async function getLibrary(source = 'all', page = 1, pageSize = 50): Promise<LibraryResponse> {
  const res = await api.get('/music/library', { params: { source, page, page_size: pageSize } })
  return res.data.data
}

export async function getPlaylists(): Promise<Playlist[]> {
  const res = await api.get('/music/playlists')
  return res.data.data
}

export async function createPlaylist(name: string, isPublic = false): Promise<Playlist> {
  const res = await api.post('/music/playlists', { name, is_public: isPublic })
  return res.data.data
}

export async function getPlaylist(id: string): Promise<{ playlist: Playlist; tracks: PlaylistTrack[] }> {
  const res = await api.get(`/music/playlists/${id}`)
  return res.data.data
}

export async function updatePlaylist(id: string, data: { name?: string; is_public?: boolean }): Promise<Playlist> {
  const res = await api.put(`/music/playlists/${id}`, data)
  return res.data.data
}

export async function deletePlaylist(id: string): Promise<void> {
  await api.delete(`/music/playlists/${id}`)
}

export async function addTrackToPlaylist(playlistId: string, trackId: string, source: string): Promise<void> {
  await api.post(`/music/playlists/${playlistId}/tracks`, { track_id: trackId, source })
}

export async function removeTrackFromPlaylist(playlistId: string, trackId: string): Promise<void> {
  await api.delete(`/music/playlists/${playlistId}/tracks/${trackId}`)
}

export function getStreamUrl(trackId: string, source: string): string {
  const token = localStorage.getItem('access_token')
  return `/api/v1/music/tracks/${trackId}/stream?source=${source}&token=${token}`
}
