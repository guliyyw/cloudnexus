import { create } from 'zustand'
import * as dockerApi from '../services/docker'
import type { ContainerInfo, ImageInfo, ContainerStats } from '../services/docker'

interface DockerState {
  containers: ContainerInfo[]
  images: ImageInfo[]
  stats: Record<string, ContainerStats>
  loading: boolean
  imagesLoading: boolean
  fetchContainers: (all?: boolean) => Promise<void>
  create: (image: string, name: string) => Promise<string>
  start: (id: string) => Promise<void>
  stop: (id: string) => Promise<void>
  restart: (id: string) => Promise<void>
  remove: (id: string, force?: boolean) => Promise<void>
  fetchImages: () => Promise<void>
  pullImage: (image: string) => Promise<void>
  removeImage: (image: string, force?: boolean) => Promise<void>
  fetchStats: (id: string) => Promise<void>
}

export const useDockerStore = create<DockerState>((set, get) => ({
  containers: [],
  images: [],
  stats: {},
  loading: false,
  imagesLoading: false,

  fetchContainers: async (all = false) => {
    set({ loading: true })
    try {
      const containers = await dockerApi.listContainers(all)
      set({ containers })
    } finally {
      set({ loading: false })
    }
  },

  create: async (image, name) => {
    const res = await dockerApi.createContainer(image, name)
    await get().fetchContainers()
    return res.id
  },

  start: async (id) => {
    await dockerApi.startContainer(id)
    await get().fetchContainers()
  },

  stop: async (id) => {
    await dockerApi.stopContainer(id)
    await get().fetchContainers()
  },

  restart: async (id) => {
    await dockerApi.restartContainer(id)
    await get().fetchContainers()
  },

  remove: async (id, force = false) => {
    await dockerApi.removeContainer(id, force)
    await get().fetchContainers()
  },

  fetchImages: async () => {
    set({ imagesLoading: true })
    try {
      const images = await dockerApi.listImages()
      set({ images })
    } finally {
      set({ imagesLoading: false })
    }
  },

  pullImage: async (image) => {
    await dockerApi.pullImage(image)
    await get().fetchImages()
  },

  removeImage: async (image, force = false) => {
    await dockerApi.removeImage(image, force)
    await get().fetchImages()
  },

  fetchStats: async (id) => {
    try {
      const s = await dockerApi.getContainerStats(id)
      set((state) => ({ stats: { ...state.stats, [id]: s } }))
    } catch { /* ignore — container may not be running */ }
  },
}))
