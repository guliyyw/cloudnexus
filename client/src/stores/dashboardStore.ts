import { create } from 'zustand'
import api from '../services/api'

interface ModuleStatus {
  key: string
  name: string
  icon: string
  status: 'green' | 'yellow' | 'red'
  detail: string
}

interface Summary {
  total: number
  healthy: number
  warning: number
  error: number
}

interface DashboardState {
  modules: ModuleStatus[]
  summary: Summary | null
  loading: boolean
  fetchStatus: () => Promise<void>
}

export const useDashboardStore = create<DashboardState>((set) => ({
  modules: [],
  summary: null,
  loading: false,
  fetchStatus: async () => {
    set({ loading: true })
    try {
      const res = await api.get('/dashboard/status')
      set({
        modules: res.data.data.modules,
        summary: res.data.data.summary,
        loading: false,
      })
    } catch {
      set({ loading: false })
    }
  },
}))
