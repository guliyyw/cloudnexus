export type ThemeMode = 'dark' | 'light'

const sharedStatus = {
  success: '#52c41a',
  warning: '#faad14',
  error: '#ff4d4f',
  statusGreen: '#52c41a',
  statusYellow: '#faad14',
  statusRed: '#ff4d4f',
}

// 暗色主题强调沉浸感和高信息密度，面板层级主要靠半透明 surface 区分。
const darkColors = {
  primary: '#81ecfe',
  primaryLight: 'rgba(129,236,254,0.12)',
  primaryDark: '#5ac8d8',
  bg: '#04050a',
  bgDeep: '#020308',
  bgCard: 'rgba(255,255,255,0.04)',
  surface: 'rgba(255,255,255,0.06)',
  surfaceRaised: 'rgba(255,255,255,0.1)',
  surfaceMuted: 'rgba(255,255,255,0.03)',
  toolbarBg: 'rgba(255,255,255,0.04)',
  panelBg: 'rgba(6,8,14,0.82)',
  borderSubtle: 'rgba(255,255,255,0.08)',
  borderStrong: 'rgba(129,236,254,0.18)',
  hoverBg: 'rgba(129,236,254,0.08)',
  text: '#f4f4f4',
  textSecondary: '#9a9a9a',
  mutedText: '#666666',
  ...sharedStatus,
}

const darkRadius = { sm: 8, md: 12, lg: 18, xl: 999 }

const darkShadow = {
  card: '0 16px 40px rgba(0,0,0,0.3)',
  hover: '0 20px 48px rgba(0,0,0,0.42)',
  modal: '0 24px 64px rgba(0,0,0,0.55)',
  shell: '0 24px 80px rgba(0,0,0,0.32)',
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

// 亮色主题保持温暖底色，避免页面在大面积信息区里显得生硬惨白。
const lightColors = {
  primary: '#e8964a',
  primaryLight: '#fef3e7',
  primaryDark: '#d4853a',
  bg: '#f6f3ed',
  bgDeep: '#ede7dd',
  bgCard: '#ffffff',
  surface: 'rgba(255,255,255,0.78)',
  surfaceRaised: '#ffffff',
  surfaceMuted: '#f7f3ec',
  toolbarBg: 'rgba(255,255,255,0.72)',
  panelBg: 'rgba(255,255,255,0.92)',
  borderSubtle: 'rgba(100,77,44,0.12)',
  borderStrong: 'rgba(232,150,74,0.26)',
  hoverBg: 'rgba(232,150,74,0.08)',
  text: '#1f1a17',
  textSecondary: '#7f756c',
  mutedText: '#9f978f',
  ...sharedStatus,
}

const lightRadius = { sm: 8, md: 12, lg: 18, xl: 24 }

const lightShadow = {
  card: '0 12px 32px rgba(87,62,31,0.08)',
  hover: '0 18px 40px rgba(87,62,31,0.12)',
  modal: '0 18px 48px rgba(57,39,17,0.16)',
  shell: '0 18px 56px rgba(57,39,17,0.1)',
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

export const spacing = { xs: 4, sm: 8, md: 16, lg: 24, xl: 32, xxl: 40 }
export const motion = { fast: '0.15s', normal: '0.25s', slow: '0.4s' }

// 通过原地更新导出的 token，让静态 import 的组件在切换主题时直接读到最新值。
export const colors = { ...darkColors }
export const radius = { ...darkRadius }
export const shadow = { ...darkShadow }
export const chart = { ...darkChart, tooltip: { ...darkChart.tooltip } }

export function applyTheme(t: ThemeMode): void {
  if (t === 'light') {
    Object.assign(colors, lightColors)
    Object.assign(radius, lightRadius)
    Object.assign(shadow, lightShadow)
    Object.assign(chart, { ...lightChart, tooltip: { ...lightChart.tooltip } })
    return
  }

  Object.assign(colors, darkColors)
  Object.assign(radius, darkRadius)
  Object.assign(shadow, darkShadow)
  Object.assign(chart, { ...darkChart, tooltip: { ...darkChart.tooltip } })
}
