import { useEffect, useState, useMemo } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Button, Dropdown, Avatar, Drawer, Menu, Grid, Typography } from 'antd'
import {
  DashboardOutlined,
  CloudOutlined,
  MessageOutlined,
  TeamOutlined,
  ContainerOutlined,
  VideoCameraOutlined,
  FileTextOutlined,
  PictureOutlined,
  CustomerServiceOutlined,
  SettingOutlined,
  LogoutOutlined,
  UserOutlined,
  MenuOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../../stores/authStore'
import { useAccess } from '../../hooks/useAccess'
import { colors } from '../../theme/tokens'

const { useBreakpoint } = Grid
const { Text } = Typography

interface NavItem {
  key: string
  icon: React.ReactNode
  label: string
  path: string
  adminOnly?: boolean
}

const sectionLabels: Record<string, string> = {
  '/dashboard': '',
  '/files': '文件管理',
  '/shares': '我的分享',
  '/chat': '即时通讯',
  '/friends': '好友',
  '/docker': 'Docker 管理',
  '/cameras': '摄像头管理',
  '/album': '相册',
  '/music': '音乐',
  '/documents': '在线文档',
  '/trash': '回收站',
  '/settings': '个人设置',
  '/admin': '管理后台',
  '/status': '系统状态',
}

export default function TopNav() {
  const navigate = useNavigate()
  const location = useLocation()
  const { logout, user, fetchProfile } = useAuthStore()
  const { isAdmin } = useAccess()
  const screens = useBreakpoint()
  const isMobile = !screens.md
  const [drawerOpen, setDrawerOpen] = useState(false)

  useEffect(() => {
    if (!user) fetchProfile()
  }, [])

  const currentSection = useMemo(() => {
    const base = '/' + location.pathname.split('/')[1]
    return sectionLabels[base] || ''
  }, [location.pathname])

  const allNavItems: NavItem[] = [
    { key: 'dashboard', icon: <DashboardOutlined />, label: '首页', path: '/dashboard' },
    { key: 'files', icon: <CloudOutlined />, label: '文件', path: '/files' },
    { key: 'chat', icon: <MessageOutlined />, label: '聊天', path: '/chat' },
    { key: 'friends', icon: <TeamOutlined />, label: '好友', path: '/friends' },
    { key: 'docker', icon: <ContainerOutlined />, label: 'Docker', path: '/docker' },
    { key: 'camera', icon: <VideoCameraOutlined />, label: '摄像头', path: '/cameras' },
    { key: 'album', icon: <PictureOutlined />, label: '相册', path: '/album' },
    { key: 'music', icon: <CustomerServiceOutlined />, label: '音乐', path: '/music' },
    { key: 'docs', icon: <FileTextOutlined />, label: '文档', path: '/documents' },
    ...(isAdmin ? [
      { key: 'admin', icon: <SettingOutlined />, label: '管理后台', path: '/admin', adminOnly: true },
      { key: 'status', icon: <DashboardOutlined />, label: '系统状态', path: '/status', adminOnly: true },
    ] : []),
  ]

  const selectedKey = '/' + location.pathname.split('/')[1]

  const userMenuItems = [
    {
      key: 'profile',
      label: (
        <div style={{ padding: '4px 0' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Avatar size={40} icon={<UserOutlined />} style={{ backgroundColor: colors.primary }} />
            <div>
              <div style={{ fontWeight: 600, fontSize: 14 }}>{user?.username}</div>
              <Text type="secondary" style={{ fontSize: 12 }}>{user?.email}</Text>
            </div>
          </div>
        </div>
      ),
      disabled: true,
    },
    { type: 'divider' as const },
    { key: 'settings', icon: <SettingOutlined />, label: '个人设置' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true },
  ]

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'settings') {
      navigate('/settings')
    } else if (key === 'logout') {
      logout()
      navigate('/login')
    }
  }

  const handleNavClick = (path: string) => {
    navigate(path)
    if (isMobile) setDrawerOpen(false)
  }

  return (
    <>
      <div
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          height: 56,
          background: '#fff',
          borderBottom: '1px solid #f0eeeb',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: isMobile ? '0 12px' : '0 24px',
          zIndex: 100,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: isMobile ? 8 : 16 }}>
          {isMobile && (
            <Button
              type="text"
              icon={<MenuOutlined />}
              onClick={() => setDrawerOpen(true)}
            />
          )}
          <span
            style={{
              color: colors.primary,
              fontWeight: 700,
              fontSize: 20,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              letterSpacing: -0.5,
            }}
            onClick={() => navigate('/dashboard')}
          >
            CloudNexus
          </span>
          {!isMobile && currentSection && (
            <Text type="secondary" style={{ fontSize: 13, fontWeight: 500 }}>
              {currentSection}
            </Text>
          )}
        </div>
        <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenuClick }} trigger={['click']}>
          <Avatar
            size={isMobile ? 32 : 36}
            icon={<UserOutlined />}
            style={{ backgroundColor: colors.primary, cursor: 'pointer', flexShrink: 0 }}
          />
        </Dropdown>
      </div>

      <Drawer
        title="CloudNexus"
        placement="left"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={260}
        styles={{ body: { padding: 0 } }}
      >
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          style={{ borderInlineEnd: 'none', fontSize: 14 }}
          items={allNavItems.map((item) => ({
            key: item.path,
            icon: item.icon,
            label: item.label,
          }))}
          onClick={({ key }) => handleNavClick(key)}
        />
      </Drawer>
    </>
  )
}
