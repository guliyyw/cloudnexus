import { create } from 'zustand'
import * as fileApi from '../services/file'
import type { FileItem } from '../services/file'

interface FileState {
  files: FileItem[]
  total: number
  page: number
  pageSize: number
  currentParentId: number
  breadcrumb: { id: number; name: string }[]
  loading: boolean
  searchMode: boolean
  searchKeyword: string

  fetchFiles: (parentId?: number, page?: number) => Promise<void>
  upload: (files: File[]) => Promise<void>
  remove: (id: number) => Promise<void>
  mkdir: (name: string) => Promise<void>
  search: (keyword: string) => Promise<void>
  navigateTo: (parentId: number, name: string) => void
  setPage: (page: number) => void
}

export const useFileStore = create<FileState>((set, get) => ({
  files: [],
  total: 0,
  page: 1,
  pageSize: 20,
  currentParentId: 0,
  breadcrumb: [{ id: 0, name: '根目录' }],
  loading: false,
  searchMode: false,
  searchKeyword: '',

  fetchFiles: async (parentId?: number, page?: number) => {
    const state = get()
    const pid = parentId ?? state.currentParentId
    const pg = page ?? 1
    set({ loading: true, searchMode: false, searchKeyword: '' })
    try {
      const res = await fileApi.getFileList(pid, pg, state.pageSize)
      set({ files: res.items, total: res.total, page: pg, currentParentId: pid })
    } finally {
      set({ loading: false })
    }
  },

  upload: async (files: File[]) => {
    const state = get()
    const result = await fileApi.uploadFiles(files, state.currentParentId)
    if (result.errors.length > 0) {
      console.warn('部分文件上传失败:', result.errors)
    }
    await get().fetchFiles()
  },

  remove: async (id: number) => {
    await fileApi.deleteFile(id)
    await get().fetchFiles()
  },

  mkdir: async (name: string) => {
    const state = get()
    await fileApi.createDirectory(name, state.currentParentId)
    await get().fetchFiles()
  },

  search: async (keyword: string) => {
    if (!keyword.trim()) {
      get().fetchFiles()
      return
    }
    set({ loading: true, searchMode: true, searchKeyword: keyword })
    try {
      const res = await fileApi.searchFiles(keyword, 1, get().pageSize)
      set({ files: res.items, total: res.total, page: 1 })
    } finally {
      set({ loading: false })
    }
  },

  navigateTo: (parentId: number, name: string) => {
    const state = get()
    const idx = state.breadcrumb.findIndex((b) => b.id === parentId)
    if (idx >= 0) {
      set({ breadcrumb: state.breadcrumb.slice(0, idx + 1) })
    } else {
      set({ breadcrumb: [...state.breadcrumb, { id: parentId, name }] })
    }
    set({ currentParentId: parentId })
    get().fetchFiles(parentId, 1)
  },

  setPage: (page: number) => {
    get().fetchFiles(undefined, page)
  },
}))
