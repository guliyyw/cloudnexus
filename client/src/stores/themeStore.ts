import { create } from 'zustand'
import { applyTheme } from '../theme/tokens'
import type { ThemeMode } from '../theme/tokens'

const STORAGE_KEY = 'cloudnexus-theme'

function readStored(): ThemeMode {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark') return v
  } catch {
    // localStorage 不可用时回退到系统偏好
  }

  try {
    if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light'
  } catch {
    // matchMedia 不可用时继续回退到暗色默认值
  }

  return 'dark'
}

function persistDOM(t: ThemeMode): void {
  document.documentElement.setAttribute('data-theme', t)
  try {
    localStorage.setItem(STORAGE_KEY, t)
  } catch {
    // quota exceeded 时不阻塞界面切换
  }
}

function applyAndPersistTheme(t: ThemeMode): void {
  applyTheme(t)
  persistDOM(t)
}

const initial = readStored()

// 在应用第一次渲染前就同步 token 和 data-theme，避免主题切换闪烁。
applyAndPersistTheme(initial)

interface ThemeState {
  theme: ThemeMode
  isDark: boolean
  toggleTheme: () => void
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  theme: initial,
  isDark: initial === 'dark',

  toggleTheme: () => {
    const next: ThemeMode = get().theme === 'dark' ? 'light' : 'dark'
    applyAndPersistTheme(next)
    set({ theme: next, isDark: next === 'dark' })
  },
}))
