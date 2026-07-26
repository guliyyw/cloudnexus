import { create } from 'zustand'
import {
  getAlbums,
  getAlbum,
  createAlbum,
  updateAlbum,
  deleteAlbum,
  getAlbumFiles,
  addFilesToAlbum,
  removeFileFromAlbum,
  type Album,
} from '../services/album'
import type { FileItem } from '../services/file'

export interface AlbumState {
  albums: Album[]
  albumLoading: boolean
  currentAlbum: Album | null
  files: FileItem[]
  filesLoading: boolean
  filesTotal: number

  fetchAlbums: () => Promise<void>
  fetchAlbum: (id: string) => Promise<void>
  createAlbum: (name: string, description: string) => Promise<Album | null>
  updateAlbum: (id: string, data: { name?: string; description?: string; cover_file_id?: string }) => Promise<void>
  deleteAlbum: (id: string) => Promise<void>
  fetchFiles: (albumId: string, page?: number) => Promise<void>
  addFiles: (albumId: string, fileIds: string[]) => Promise<void>
  removeFile: (albumId: string, fileId: string) => Promise<void>
}

export const useAlbumStore = create<AlbumState>((set, get) => ({
  albums: [],
  albumLoading: false,
  currentAlbum: null,
  files: [],
  filesLoading: false,
  filesTotal: 0,

  fetchAlbums: async () => {
    set({ albumLoading: true })
    try {
      const res = await getAlbums(1, 100)
      set({ albums: res.albums || [], albumLoading: false })
    } catch {
      set({ albumLoading: false })
    }
  },

  fetchAlbum: async (id: string) => {
    try {
      const album = await getAlbum(id)
      set({ currentAlbum: album })
    } catch { /* ignore */ }
  },

  createAlbum: async (name: string, description: string) => {
    try {
      const album = await createAlbum(name, description)
      set({ albums: [album, ...get().albums] })
      return album
    } catch {
      return null
    }
  },

  updateAlbum: async (id: string, data: { name?: string; description?: string; cover_file_id?: string }) => {
    try {
      const album = await updateAlbum(id, data)
      set({
        currentAlbum: album,
        albums: get().albums.map((a) => (a.id === id ? album : a)),
      })
    } catch { /* ignore */ }
  },

  deleteAlbum: async (id: string) => {
    try {
      await deleteAlbum(id)
      set({ albums: get().albums.filter((a) => a.id !== id), currentAlbum: null })
    } catch { /* ignore */ }
  },

  fetchFiles: async (albumId: string, page = 1) => {
    set({ filesLoading: true })
    try {
      const res = await getAlbumFiles(albumId, page, 200)
      set({ files: res.files || [], filesTotal: res.total, filesLoading: false })
    } catch {
      set({ filesLoading: false })
    }
  },

  addFiles: async (albumId: string, fileIds: string[]) => {
    await addFilesToAlbum(albumId, fileIds)
    await get().fetchFiles(albumId)
    await get().fetchAlbum(albumId)
  },

  removeFile: async (albumId: string, fileId: string) => {
    try {
      await removeFileFromAlbum(albumId, fileId)
      set({ files: get().files.filter((f) => f.id !== fileId) })
    } catch { /* ignore */ }
  },
}))
