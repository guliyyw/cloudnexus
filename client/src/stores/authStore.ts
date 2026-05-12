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
  login: (username: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string, captchaId?: string, captchaCode?: string) => Promise<void>
  logout: () => void
  fetchProfile: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoggedIn: !!localStorage.getItem('access_token'),

  login: async (username, password) => {
    const res = await api.post('/user/login', { username, password })
    const { access_token, refresh_token } = res.data.data
    localStorage.setItem('access_token', access_token)
    localStorage.setItem('refresh_token', refresh_token)
    set({ isLoggedIn: true })
  },

  register: async (username: string, email: string, password: string, captchaId?: string, captchaCode?: string) => {
    await api.post('/user/register', { username, email, password, captcha_id: captchaId, captcha_code: captchaCode })
  },

  logout: () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    set({ user: null, isLoggedIn: false })
  },

  fetchProfile: async () => {
    const res = await api.get('/user/profile')
    set({ user: res.data.data, isLoggedIn: true })
  },
}))
