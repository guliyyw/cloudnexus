import { Outlet, useLocation } from 'react-router-dom'
import { Layout as AntLayout, Grid } from 'antd'
import TopNav from './layout/TopNav'
import GlobalPlayer from './player/GlobalPlayer'
import { useQuotaStore } from '../stores/quotaStore'
import { useEffect } from 'react'
import { colors } from '../theme/tokens'

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
  }, [])

  return (
    <AntLayout style={{ height: '100vh', overflow: 'hidden', background: colors.bg }}>
      <TopNav />
      <Content style={{
        marginTop: 56,
        margin: isMobile ? '68px 8px 8px' : '68px 16px 16px',
        padding: isMobile ? 16 : 24,
        background: 'transparent',
        flex: isDramaWorkbench ? 'none' : 1,
        height: isDramaWorkbench ? `calc(100vh - ${isMobile ? 76 : 84}px)` : undefined,
        boxSizing: 'border-box',
        overflow: isDramaWorkbench ? 'hidden' : 'auto',
        minHeight: 0,
        position: 'relative',
        zIndex: 1,
      }}>
        <Outlet />
      </Content>
      <GlobalPlayer />
    </AntLayout>
  )
}
