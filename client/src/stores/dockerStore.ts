import { create } from 'zustand'
import * as dockerApi from '../services/docker'
import type { ContainerInfo } from '../services/docker'

interface DockerState {
  containers: ContainerInfo[]
  loading: boolean
  fetchContainers: (all?: boolean) => Promise<void>
  create: (image: string, name: string) => Promise<string>
  start: (id: string) => Promise<void>
  stop: (id: string) => Promise<void>
  restart: (id: string) => Promise<void>
  remove: (id: string, force?: boolean) => Promise<void>
}

export const useDockerStore = create<DockerState>((set, get) => ({
  containers: [],
  loading: false,

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
}))
