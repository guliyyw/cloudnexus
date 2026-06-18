import { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
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
          padding: isMobile ? '12px 12px 96px' : '18px 20px 108px',
          background: 'transparent',
          flex: 1,
          overflow: 'auto',
          minHeight: 0,
          position: 'relative',
          zIndex: 1,
        }}
      >
        {/* 统一页面壳负责留白、最大宽度和玻璃面板层级，各业务页只专注内容结构。 */}
        <div
          style={{
            maxWidth: 1480,
            minHeight: '100%',
            margin: '0 auto',
            padding: shellPadding,
            borderRadius: isMobile ? radius.lg : 28,
            border: `1px solid ${colors.borderSubtle}`,
            background: colors.surface,
            backdropFilter: 'blur(18px)',
            boxShadow: shadow.shell,
          }}
        >
          <Outlet />
        </div>
      </Content>
      <GlobalPlayer />
    </AntLayout>
  )
}
