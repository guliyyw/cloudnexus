import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout as AntLayout, Menu, Button, theme } from 'antd'
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
} from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'

const { Header, Sider, Content } = AntLayout

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { logout, user, fetchProfile } = useAuthStore()

  useEffect(() => {
    if (!user) fetchProfile()
  }, [])
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

  const siderStyle: React.CSSProperties = {
    overflow: 'auto',
    height: '100vh',
    position: 'fixed',
    left: 0,
    top: 0,
    bottom: 0,
    background: '#fff',
    borderRight: '1px solid #f0eeeb',
  }

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
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
          onClick={({ key }) => navigate(key)}
          style={{
            background: 'transparent',
            borderInlineEnd: 'none',
            fontSize: 14,
          }}
        />
      </Sider>
      <AntLayout style={{ marginLeft: collapsed ? 80 : 220, transition: 'margin-left 0.2s' }}>
        <Header style={{
          padding: '0 24px',
          background: '#fafaf8',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          height: 56,
          borderBottom: '1px solid #f0eeeb',
        }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <span style={{ color: '#6b6b6b' }}>{user?.username}</span>
            <Button type="text" icon={<LogoutOutlined />} onClick={() => { logout(); navigate('/login') }} style={{ color: '#8c8c8c' }}>
              退出
            </Button>
          </div>
        </Header>
        <Content style={{
          margin: 20,
          padding: 24,
          background: themeToken.colorBgContainer,
          borderRadius: 12,
          minHeight: 280,
        }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}
