import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout as AntLayout, Menu, Button, theme, Grid } from 'antd'
import {
  ShareAltOutlined,
  CloudOutlined,
  MessageOutlined,
  TeamOutlined,
  ContainerOutlined,
  SettingOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'

const { Header, Sider, Content } = AntLayout
const { useBreakpoint } = Grid

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { logout, user, fetchProfile } = useAuthStore()
  const screens = useBreakpoint()
  const isMobile = !screens.md

  useEffect(() => {
    if (!user) fetchProfile()
  }, [])
  useEffect(() => { setCollapsed(!!isMobile) }, [isMobile])
  const { token: themeToken } = theme.useToken()

  const menuItems = [
    { key: '/files', icon: <CloudOutlined />, label: '文件管理' },
    { key: '/shares', icon: <ShareAltOutlined />, label: '我的分享' },
    { key: '/chat', icon: <MessageOutlined />, label: '即时通讯' },
    { key: '/friends', icon: <TeamOutlined />, label: '好友' },
    { key: '/docker', icon: <ContainerOutlined />, label: 'Docker 管理' },
    ...(user?.is_admin ? [{ key: '/admin', icon: <SettingOutlined />, label: '管理后台' }] : []),
  ]

  const selectedKey = '/' + location.pathname.split('/')[1]
  const siderWidth = collapsed ? 80 : 220

  const siderStyle: React.CSSProperties = {
    overflow: 'auto',
    height: '100vh',
    position: 'fixed',
    left: 0,
    top: 0,
    bottom: 0,
    zIndex: 10,
    background: '#fff',
    borderRight: '1px solid #f0eeeb',
  }

  return (
    <AntLayout style={{ height: '100vh', overflow: 'hidden' }}>
      {isMobile && !collapsed && (
        <div
          onClick={() => setCollapsed(true)}
          style={{
            position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.3)', zIndex: 9,
          }}
        />
      )}
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={220}
        style={siderStyle}
      >
        <div style={{ height: 48, margin: '16px 16px 8px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ color: '#e8964a', fontWeight: 700, fontSize: collapsed ? 16 : 20, whiteSpace: 'nowrap', letterSpacing: -0.5 }}>
            {collapsed ? 'CN' : 'CloudNexus'}
          </span>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => { navigate(key); if (isMobile) setCollapsed(true) }}
          style={{
            background: 'transparent',
            borderInlineEnd: 'none',
            fontSize: 14,
          }}
        />
      </Sider>
      <AntLayout style={{
        marginLeft: siderWidth,
        transition: 'margin-left 0.2s',
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
      }}>
        <Header style={{
          padding: isMobile ? '0 12px' : '0 24px',
          background: '#fafaf8',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          height: 56,
          borderBottom: '1px solid #f0eeeb',
          flexShrink: 0,
        }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: isMobile ? 4 : 16 }}>
            {!isMobile && (
              <Button type="text" icon={<UserOutlined />}
                onClick={() => navigate('/settings')}
                style={{ color: '#6b6b6b' }}>
                {user?.username}
              </Button>
            )}
            <Button type="text" icon={<LogoutOutlined />}
              onClick={() => { logout(); navigate('/login') }}
              style={{ color: '#8c8c8c' }}>
              {!isMobile && '退出'}
            </Button>
          </div>
        </Header>
        <Content style={{
          margin: isMobile ? 12 : 20,
          padding: isMobile ? 16 : 24,
          background: themeToken.colorBgContainer,
          borderRadius: 12,
          flex: 1,
          overflow: 'auto',
          minHeight: 0,
        }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}
