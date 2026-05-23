export type ThemeMode = 'dark' | 'light'

// ── 暗色主题 ──
const darkColors = {
  primary: '#81ecfe',
  primaryLight: 'rgba(129,236,254,0.08)',
  primaryDark: '#5ac8d8',
  bg: '#04050a',
  bgDeep: '#020308',
  bgCard: 'rgba(255,255,255,0.04)',
  text: '#f4f4f4',
  textSecondary: '#888888',
  success: '#52c41a',
  warning: '#faad14',
  error: '#ff4d4f',
  statusGreen: '#52c41a',
  statusYellow: '#faad14',
  statusRed: '#ff4d4f',
}

const darkRadius = { sm: 8, md: 12, lg: 16, xl: 500 }

const darkShadow = {
  card: '0 0 20px rgba(129,236,254,0.04)',
  hover: '0 0 40px rgba(129,236,254,0.1)',
  modal: '0 0 60px rgba(0,0,0,0.5)',
}

const darkChart = {
  gridStroke: 'rgba(255,255,255,0.06)',
  tickFill: '#888',
  tooltip: {
    fontSize: 12,
    borderRadius: 8,
    background: 'rgba(0,0,0,0.9)',
    border: '1px solid rgba(255,255,255,0.1)',
    color: '#f4f4f4',
  },
}

// ── 亮色主题 ──
const lightColors = {
  primary: '#e8964a',
  primaryLight: '#fef3e7',
  primaryDark: '#d4853a',
  bg: '#fafaf8',
  bgDeep: '#f0ede8',
  bgCard: '#ffffff',
  text: '#1a1a1a',
  textSecondary: '#8c8c8c',
  success: '#52c41a',
  warning: '#faad14',
  error: '#ff4d4f',
  statusGreen: '#52c41a',
  statusYellow: '#faad14',
  statusRed: '#ff4d4f',
}

const lightRadius = { sm: 6, md: 10, lg: 14, xl: 20 }

const lightShadow = {
  card: '0 1px 3px rgba(0,0,0,0.06)',
  hover: '0 4px 16px rgba(0,0,0,0.1)',
  modal: '0 8px 32px rgba(0,0,0,0.15)',
}

const lightChart = {
  gridStroke: 'rgba(0,0,0,0.06)',
  tickFill: '#8c8c8c',
  tooltip: {
    fontSize: 12,
    borderRadius: 8,
    background: 'rgba(255,255,255,0.95)',
    border: '1px solid rgba(0,0,0,0.1)',
    color: '#1a1a1a',
  },
}

// ── 不可变 token（两主题相同）──
export const spacing = { xs: 4, sm: 8, md: 16, lg: 24, xl: 32 }
export const motion = { fast: '0.15s', normal: '0.25s', slow: '0.4s' }

// ── 可变导出（初始化为暗色，组件静态 import 不变）──
export const colors = { ...darkColors }
export const radius = { ...darkRadius }
export const shadow = { ...darkShadow }
export const chart = { ...darkChart }

let _current: ThemeMode = 'dark'

export function getTheme(): ThemeMode {
  return _current
}

export function applyTheme(t: ThemeMode): void {
  _current = t
  if (t === 'light') {
    Object.assign(colors, lightColors)
    Object.assign(radius, lightRadius)
    Object.assign(shadow, lightShadow)
    Object.assign(chart, lightChart)
  } else {
    Object.assign(colors, darkColors)
    Object.assign(radius, darkRadius)
    Object.assign(shadow, darkShadow)
    Object.assign(chart, darkChart)
  }
}
