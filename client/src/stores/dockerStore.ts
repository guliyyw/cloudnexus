import { create } from 'zustand'
import * as dockerApi from '../services/docker'
import type { ContainerInfo, ImageInfo, ContainerStats, EndpointInfo } from '../services/docker'

interface DockerState {
  endpoint: string
  endpoints: EndpointInfo[]
  containers: ContainerInfo[]
  images: ImageInfo[]
  stats: Record<string, ContainerStats>
  loading: boolean
  imagesLoading: boolean
  setEndpoint: (name: string) => void
  fetchEndpoints: () => Promise<void>
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
  endpoint: 'local',
  endpoints: [],
  containers: [],
  images: [],
  stats: {},
  loading: false,
  imagesLoading: false,

  setEndpoint: (name: string) => {
    set({ endpoint: name, containers: [], images: [], stats: {} })
  },

  fetchEndpoints: async () => {
    try {
      const endpoints = await dockerApi.listEndpoints()
      set({ endpoints })
    } catch { /* ignore */ }
  },

  fetchContainers: async (all = false) => {
    set({ loading: true })
    try {
      const containers = await dockerApi.listContainers(all, get().endpoint)
      set({ containers })
    } finally {
      set({ loading: false })
    }
  },

  create: async (image, name) => {
    const res = await dockerApi.createContainer(image, name, get().endpoint)
    await get().fetchContainers()
    return res.id
  },

  start: async (id) => {
    await dockerApi.startContainer(id, get().endpoint)
    await get().fetchContainers()
  },

  stop: async (id) => {
    await dockerApi.stopContainer(id, get().endpoint)
    await get().fetchContainers()
  },

  restart: async (id) => {
    await dockerApi.restartContainer(id, get().endpoint)
    await get().fetchContainers()
  },

  remove: async (id, force = false) => {
    await dockerApi.removeContainer(id, force, get().endpoint)
    await get().fetchContainers()
  },

  fetchImages: async () => {
    set({ imagesLoading: true })
    try {
      const images = await dockerApi.listImages(get().endpoint)
      set({ images })
    } finally {
      set({ imagesLoading: false })
    }
  },

  pullImage: async (image) => {
    await dockerApi.pullImage(image, get().endpoint)
    await get().fetchImages()
  },

  removeImage: async (image, force = false) => {
    await dockerApi.removeImage(image, force, get().endpoint)
    await get().fetchImages()
  },

  fetchStats: async (id) => {
    try {
      const s = await dockerApi.getContainerStats(id, get().endpoint)
      set((state) => ({ stats: { ...state.stats, [id]: s } }))
    } catch { /* ignore */ }
  },
}))
