import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout as AntLayout, Menu, Button, theme } from 'antd'
import {
  CloudOutlined,
  MessageOutlined,
  TeamOutlined,
  ContainerOutlined,
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
    { key: '/chat', icon: <MessageOutlined />, label: '即时通讯' },
    { key: '/friends', icon: <TeamOutlined />, label: '好友' },
    { key: '/docker', icon: <ContainerOutlined />, label: 'Docker 管理' },
  ]

  const selectedKey = '/' + location.pathname.split('/')[1]

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{ height: 48, margin: 16, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ color: '#fff', fontWeight: 'bold', fontSize: collapsed ? 14 : 18, whiteSpace: 'nowrap' }}>
            {collapsed ? 'CN' : 'CloudNexus'}
          </span>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <AntLayout>
        <Header style={{ padding: '0 24px', background: themeToken.colorBgContainer, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <span>{user?.username}</span>
            <Button type="text" icon={<LogoutOutlined />} onClick={() => { logout(); navigate('/login') }}>
              退出
            </Button>
          </div>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: themeToken.colorBgContainer, borderRadius: 8 }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}
