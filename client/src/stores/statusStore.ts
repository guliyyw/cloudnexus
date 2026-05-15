import { create } from 'zustand'
import { getHealthHistory, getResourceHistory, type HealthSnapshot, type ResourcePoint } from '../services/status'

export interface ParsedSnapshot {
  timestamp: string
  modules: { name: string; status: string }[]
}

export interface StatusState {
  snapshots: ParsedSnapshot[]
  resources: Record<string, ResourcePoint[]>
  historyLoading: boolean
  resourceLoading: boolean

  fetchHistory: (range?: string) => Promise<void>
  fetchResources: (range?: string, service?: string) => Promise<void>
}

function parseSnapshots(raw: HealthSnapshot[]): ParsedSnapshot[] {
  return raw.map((s) => {
    let modules: { name: string; status: string }[] = []
    try {
      modules = JSON.parse(s.data)
    } catch { /* ignore parse errors */ }
    return { timestamp: s.timestamp, modules }
  })
}

export const useStatusStore = create<StatusState>((set) => ({
  snapshots: [],
  resources: {},
  historyLoading: false,
  resourceLoading: false,

  fetchHistory: async (range = '24h') => {
    set({ historyLoading: true })
    try {
      const res = await getHealthHistory(range)
      set({ snapshots: parseSnapshots(res.snapshots || []), historyLoading: false })
    } catch {
      set({ historyLoading: false })
    }
  },

  fetchResources: async (range = '24h', service = 'all') => {
    set({ resourceLoading: true })
    try {
      const res = await getResourceHistory(range, service)
      set({ resources: res.services || {}, resourceLoading: false })
    } catch {
      set({ resourceLoading: false })
    }
  },
}))
