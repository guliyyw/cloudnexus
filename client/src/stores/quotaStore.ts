import { create } from 'zustand'
import { getQuota, QuotaInfo } from '../services/file'

interface QuotaState {
  quota: QuotaInfo | null
  loading: boolean
  fetchQuota: () => Promise<void>
}

export const useQuotaStore = create<QuotaState>((set) => ({
  quota: null,
  loading: false,
  fetchQuota: async () => {
    set({ loading: true })
    try {
      const q = await getQuota()
      set({ quota: q, loading: false })
    } catch {
      set({ loading: false })
    }
  },
}))
