import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Avatar, Button, Drawer, Dropdown, Grid, Menu, Typography } from 'antd'
import {
  CloudOutlined,
  ContainerOutlined,
  CustomerServiceOutlined,
  DashboardOutlined,
  DeleteOutlined,
  FileTextOutlined,
  LogoutOutlined,
  MenuOutlined,
  MessageOutlined,
  MoonOutlined,
  PictureOutlined,
  SettingOutlined,
  ShareAltOutlined,
  SunOutlined,
  TeamOutlined,
  UnorderedListOutlined,
  UserOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../../stores/authStore'
import { useThemeStore } from '../../stores/themeStore'
import { useAccess } from '../../hooks/useAccess'
import { colors, motion, radius } from '../../theme/tokens'

const { useBreakpoint } = Grid
const { Text } = Typography

interface NavItem {
  key: string
  icon: React.ReactNode
  label: string
  path: string
}

interface RouteContext {
  key: string
  title: string
  description: string
  match: (pathname: string) => boolean
}

const baseNavItems: NavItem[] = [
  { key: 'dashboard', icon: <DashboardOutlined />, label: '首页', path: '/dashboard' },
  { key: 'files', icon: <CloudOutlined />, label: '文件', path: '/files' },
  { key: 'shares', icon: <ShareAltOutlined />, label: '我的分享', path: '/shares' },
  { key: 'trash', icon: <DeleteOutlined />, label: '回收站', path: '/trash' },
  { key: 'chat', icon: <MessageOutlined />, label: '聊天', path: '/chat' },
  { key: 'friends', icon: <TeamOutlined />, label: '好友', path: '/friends' },
  { key: 'docker', icon: <ContainerOutlined />, label: 'Docker', path: '/docker' },
  { key: 'camera', icon: <VideoCameraOutlined />, label: '摄像头', path: '/cameras' },
  { key: 'album', icon: <PictureOutlined />, label: '相册', path: '/album' },
  { key: 'music', icon: <CustomerServiceOutlined />, label: '音乐', path: '/music' },
  { key: 'playlist', icon: <UnorderedListOutlined />, label: '播放列表', path: '/playlist' },
  { key: 'docs', icon: <FileTextOutlined />, label: '文档', path: '/documents' },
]

const adminNavItems: NavItem[] = [
  { key: 'admin', icon: <SettingOutlined />, label: '管理后台', path: '/admin' },
  { key: 'status', icon: <DashboardOutlined />, label: '系统状态', path: '/status' },
]

const routeContexts: RouteContext[] = [
  {
    key: '/dashboard',
    title: '系统概览',
    description: '从首页进入协作、媒体与系统模块，并快速查看当前服务健康状态。',
    match: (pathname) => pathname === '/dashboard',
  },
  {
    key: '/files',
    title: '在线文档',
    description: '从文件工作台打开的协作文档会保留当前文件上下文。',
    match: (pathname) => pathname.startsWith('/files/') && pathname.endsWith('/edit'),
  },
  {
    key: '/files',
    title: '文件工作台',
    description: '统一管理目录、上传、分享、版本与协作入口。',
    match: (pathname) => pathname.startsWith('/files'),
  },
  {
    key: '/documents',
    title: '在线文档',
    description: '集中处理协作文档的浏览、进入与编辑状态。',
    match: (pathname) => pathname.startsWith('/documents/'),
  },
  {
    key: '/documents',
    title: '文档中心',
    description: '浏览、创建并进入实时协作的在线文档。',
    match: (pathname) => pathname.startsWith('/documents'),
  },
  {
    key: '/chat',
    title: '即时通讯',
    description: '会话、成员和实时消息流都围绕当前上下文集中展示。',
    match: (pathname) => pathname.startsWith('/chat'),
  },
  {
    key: '/friends',
    title: '好友关系',
    description: '管理联系人、邀请关系和私聊入口。',
    match: (pathname) => pathname.startsWith('/friends'),
  },
  {
    key: '/docker',
    title: 'Docker 管理',
    description: '查看容器、执行操作并区分可控与受限资源。',
    match: (pathname) => pathname.startsWith('/docker'),
  },
  {
    key: '/cameras',
    title: '摄像头中心',
    description: '集中查看摄像头、识别结果、人脸库与考勤状态。',
    match: (pathname) => pathname.startsWith('/cameras'),
  },
  {
    key: '/album',
    title: '相册',
    description: '按时间线、文件夹和媒体视角管理影像内容。',
    match: (pathname) => pathname.startsWith('/album'),
  },
  {
    key: '/music',
    title: '音乐',
    description: '浏览曲库、管理播放与全局播放器体验。',
    match: (pathname) => pathname.startsWith('/music'),
  },
  {
    key: '/playlist',
    title: '播放列表',
    description: '管理歌单、排序和导入导出。',
    match: (pathname) => pathname.startsWith('/playlist'),
  },
  {
    key: '/shares',
    title: '我的分享',
    description: '集中查看已创建的公开分享与访问策略。',
    match: (pathname) => pathname.startsWith('/shares'),
  },
  {
    key: '/trash',
    title: '回收站',
    description: '回溯误删文件并执行恢复或彻底清理。',
    match: (pathname) => pathname.startsWith('/trash'),
  },
  {
    key: '/admin',
    title: '管理后台',
    description: '权限、日志、配额和系统规则等后台配置集中在这里。',
    match: (pathname) => pathname.startsWith('/admin'),
  },
  {
    key: '/status',
    title: '系统状态',
    description: '查看服务与资源状态，快速定位异常节点。',
    match: (pathname) => pathname.startsWith('/status'),
  },
  {
    key: '/settings',
    title: '个人设置',
    description: '管理当前账号的基础资料、偏好和安全设置。',
    match: (pathname) => pathname.startsWith('/settings'),
  },
]

export default function TopNav() {
  const navigate = useNavigate()
  const location = useLocation()
  const { logout, user, fetchProfile } = useAuthStore()
  const { isDark, toggleTheme } = useThemeStore()
  const { isAdmin } = useAccess()
  const screens = useBreakpoint()
  const isMobile = !screens.md
  const [drawerOpen, setDrawerOpen] = useState(false)

  useEffect(() => {
    if (!user) {
      fetchProfile().catch(() => undefined)
    }
  }, [fetchProfile, user])

  const navItems = useMemo(() => (
    isAdmin ? [...baseNavItems, ...adminNavItems] : baseNavItems
  ), [isAdmin])

  const currentContext = useMemo(() => {
    return routeContexts.find((context) => context.match(location.pathname)) ?? routeContexts[0]
  }, [location.pathname])

  const userMenuItems = [
    {
      key: 'profile',
      label: (
        <div style={{ padding: '4px 0' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Avatar size={40} icon={<UserOutlined />} style={{ backgroundColor: colors.primary }} />
            <div style={{ minWidth: 0 }}>
              <div style={{ fontWeight: 600, fontSize: 14, color: colors.text }}>{user?.username || '未登录'}</div>
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
      return
    }

    if (key === 'logout') {
      logout()
      navigate('/login')
    }
  }

  const handleNavClick = (path: string) => {
    navigate(path)
    if (isMobile) setDrawerOpen(false)
  }

  const navHeight = isMobile ? 64 : 72

  return (
    <>
      <div
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          height: navHeight,
          background: colors.panelBg,
          backdropFilter: 'blur(18px)',
          borderBottom: `1px solid ${colors.borderSubtle}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: isMobile ? '0 14px' : '0 28px',
          zIndex: 100,
          boxShadow: '0 10px 32px rgba(0,0,0,0.12)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: isMobile ? 10 : 20, minWidth: 0 }}>
          {isMobile && (
            <Button
              type="text"
              icon={<MenuOutlined />}
              onClick={() => setDrawerOpen(true)}
              style={{ color: colors.text, borderRadius: radius.md }}
            />
          )}

          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              cursor: 'pointer',
              minWidth: 0,
            }}
            onClick={() => navigate('/dashboard')}
          >
            <span
              style={{
                color: colors.primary,
                fontWeight: 700,
                fontSize: isMobile ? 18 : 20,
                lineHeight: 1.1,
                letterSpacing: -0.5,
              }}
            >
              CloudNexus
            </span>
            <Text style={{ fontSize: 11, color: colors.textSecondary }}>
              协作 · 媒体 · 系统
            </Text>
          </div>

          {!isMobile && (
            <>
              <div
                style={{
                  width: 1,
                  height: 32,
                  background: colors.borderSubtle,
                  flexShrink: 0,
                }}
              />
              {/* 桌面端不铺全局导航按钮，改用当前模块上下文提示来保持顶栏克制。 */}
              <div style={{ minWidth: 0 }}>
                <div style={{ fontSize: 16, fontWeight: 600, color: colors.text }}>{currentContext.title}</div>
                <Text style={{ fontSize: 12, color: colors.textSecondary }} ellipsis>
                  {currentContext.description}
                </Text>
              </div>
            </>
          )}

          {isMobile && (
            <Text
              style={{
                fontSize: 13,
                fontWeight: 600,
                color: colors.text,
                maxWidth: 120,
              }}
              ellipsis
            >
              {currentContext.title}
            </Text>
          )}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}>
          <Button
            type="text"
            icon={isDark ? <SunOutlined /> : <MoonOutlined />}
            onClick={toggleTheme}
            title={isDark ? '切换到亮色模式' : '切换到暗色模式'}
            style={{
              color: colors.textSecondary,
              fontSize: 18,
              borderRadius: radius.md,
            }}
          />
          <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenuClick }} trigger={['click']}>
            <Avatar
              size={isMobile ? 34 : 38}
              icon={<UserOutlined />}
              style={{
                backgroundColor: colors.primary,
                cursor: 'pointer',
                flexShrink: 0,
                transition: `transform ${motion.fast} ease`,
              }}
            />
          </Dropdown>
        </div>
      </div>

      <Drawer
        title="CloudNexus"
        placement="left"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={280}
        styles={{ body: { padding: 0 } }}
      >
        <div
          style={{
            padding: '0 16px 16px',
            borderBottom: `1px solid ${colors.borderSubtle}`,
            marginBottom: 8,
          }}
        >
          <div style={{ fontSize: 15, fontWeight: 600, color: colors.text }}>{currentContext.title}</div>
          <Text style={{ fontSize: 12, color: colors.textSecondary }}>{currentContext.description}</Text>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[currentContext.key]}
          style={{ borderInlineEnd: 'none', fontSize: 14, background: 'transparent' }}
          items={navItems.map((item) => ({
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
