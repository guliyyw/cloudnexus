import { Outlet } from 'react-router-dom'
import { Layout as AntLayout, Grid } from 'antd'
import TopNav from './layout/TopNav'
import { useQuotaStore } from '../stores/quotaStore'
import { useEffect } from 'react'

const { Content } = AntLayout
const { useBreakpoint } = Grid

export default function AppLayout() {
  const screens = useBreakpoint()
  const isMobile = !screens.md
  const { fetchQuota } = useQuotaStore()

  useEffect(() => {
    fetchQuota()
  }, [])

  return (
    <AntLayout style={{ height: '100vh', overflow: 'hidden', background: '#fafaf8' }}>
      <TopNav />
      <Content style={{
        marginTop: 56,
        margin: isMobile ? '68px 8px 8px' : '68px 16px 16px',
        padding: isMobile ? 16 : 24,
        background: '#ffffff',
        borderRadius: 12,
        flex: 1,
        overflow: 'auto',
        minHeight: 0,
      }}>
        <Outlet />
      </Content>
    </AntLayout>
  )
}
