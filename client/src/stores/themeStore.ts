import { create } from 'zustand'
import { applyTheme } from '../theme/tokens'
import type { ThemeMode } from '../theme/tokens'

const STORAGE_KEY = 'cloudnexus-theme'

function readStored(): ThemeMode {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark') return v
  } catch { /* localStorage 不可用 */ }
  try {
    if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light'
  } catch { /* matchMedia 不可用 */ }
  return 'dark'
}

function persistDOM(t: ThemeMode): void {
  document.documentElement.setAttribute('data-theme', t)
  try {
    localStorage.setItem(STORAGE_KEY, t)
  } catch { /* quota exceeded */ }
}

const initial = readStored()
applyTheme(initial)
persistDOM(initial)

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
    applyTheme(next)
    persistDOM(next)
    set({ theme: next, isDark: next === 'dark' })
  },
}))
