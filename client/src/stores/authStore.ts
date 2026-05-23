import { create } from 'zustand'
import api from '../services/api'

interface User {
  id: string
  username: string
  email: string
  avatar: string
  is_admin: boolean
}

interface AuthState {
  user: User | null
  isLoggedIn: boolean
  isLoading: boolean
  hasChecked: boolean
  login: (username: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string, captchaId?: string, captchaCode?: string) => Promise<void>
  logout: () => Promise<void>
  fetchProfile: () => Promise<void>
  checkAuth: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isLoggedIn: false,
  isLoading: true,
  hasChecked: false,

  login: async (username, password) => {
    await api.post('/user/login', { username, password })
    set({ isLoggedIn: true })
    try {
      const profileRes = await api.get('/user/profile')
      set({ user: profileRes.data.data })
    } catch { /* profile fetch failure doesn't block login */ }
  },

  register: async (username: string, email: string, password: string, captchaId?: string, captchaCode?: string) => {
    await api.post('/user/register', { username, email, password, captcha_id: captchaId, captcha_code: captchaCode })
  },

  logout: async () => {
    try {
      await api.post('/user/logout')
    } catch { /* ignore logout API errors */ }
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    set({ user: null, isLoggedIn: false, hasChecked: false })
  },

  fetchProfile: async () => {
    const res = await api.get('/user/profile')
    set({ user: res.data.data, isLoggedIn: true })
  },

  checkAuth: async () => {
    if (get().hasChecked) return
    set({ isLoading: true })
    try {
      const res = await api.get('/user/profile')
      set({ user: res.data.data, isLoggedIn: true, isLoading: false, hasChecked: true })
    } catch {
      set({ user: null, isLoggedIn: false, isLoading: false, hasChecked: true })
    }
  },
}))
