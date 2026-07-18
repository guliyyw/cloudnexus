import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Layout as AntLayout, Grid } from 'antd'
import TopNav from './layout/TopNav'
import GlobalPlayer from './player/GlobalPlayer'
import { useQuotaStore } from '../stores/quotaStore'
import { colors, radius, shadow } from '../theme/tokens'

const { Content } = AntLayout
const { useBreakpoint } = Grid

export default function AppLayout() {
  const screens = useBreakpoint()
  const isMobile = !screens.md
  const location = useLocation()
  const isDramaWorkbench = location.pathname.startsWith('/drama')
  const { fetchQuota } = useQuotaStore()

  useEffect(() => {
    fetchQuota()
  }, [fetchQuota])

  const navHeight = isMobile ? 64 : 72
  const shellPadding = isMobile ? 14 : 24

  return (
    <AntLayout
      style={{
        minHeight: '100vh',
        height: isDramaWorkbench ? '100vh' : undefined,
        overflow: 'hidden',
        background: colors.bg,
        position: 'relative',
      }}
    >
      <div id="bg-vignette" />
      <div id="bg-blob" />
      <TopNav />
      <Content
        style={{
          marginTop: navHeight,
          padding: isDramaWorkbench ? (isMobile ? 8 : 16) : (isMobile ? '12px 12px 96px' : '18px 20px 108px'),
          background: 'transparent',
          flex: isDramaWorkbench ? 'none' : 1,
          height: isDramaWorkbench ? `calc(100vh - ${navHeight}px)` : undefined,
          overflow: isDramaWorkbench ? 'hidden' : 'auto',
          minHeight: 0,
          position: 'relative',
          zIndex: 1,
          boxSizing: 'border-box',
        }}
      >
        {/* Unified page shell for spacing, width and glass panel hierarchy. */}
        <div
          style={{
            maxWidth: isDramaWorkbench ? 'none' : 1480,
            height: isDramaWorkbench ? '100%' : undefined,
            minHeight: isDramaWorkbench ? 0 : '100%',
            margin: '0 auto',
            padding: isDramaWorkbench ? 0 : shellPadding,
            borderRadius: isDramaWorkbench ? 0 : (isMobile ? radius.lg : 28),
            border: isDramaWorkbench ? 'none' : `1px solid ${colors.borderSubtle}`,
            background: isDramaWorkbench ? 'transparent' : colors.surface,
            backdropFilter: isDramaWorkbench ? undefined : 'blur(18px)',
            boxShadow: isDramaWorkbench ? 'none' : shadow.shell,
          }}
        >
          <Outlet />
        </div>
      </Content>
      <GlobalPlayer />
    </AntLayout>
  )
}
