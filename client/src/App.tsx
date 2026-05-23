import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider, theme as antdTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useThemeStore } from './stores/themeStore'
import { AuthGuard, AdminGuard } from './components/Guard'
import AppLayout from './components/Layout'
import PageTransition from './components/PageTransition'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import ForbiddenPage from './pages/ForbiddenPage'
import ShareAccessPage from './pages/ShareAccessPage'
import Dashboard from './pages/Dashboard'
import FileListPage from './pages/FileListPage'
import ChatPage from './pages/ChatPage'
import FriendPage from './pages/FriendPage'
import DockerPage from './pages/DockerPage'
import AdminPage from './pages/AdminPage'
import MySharesPage from './pages/MySharesPage'
import UserSettingsPage from './pages/UserSettingsPage'
import CameraPage from './pages/CameraPage'
import CameraLiveView from './pages/CameraLiveView'
import DocumentListPage from './pages/DocumentListPage'
import DocumentEditorPage from './pages/DocumentEditorPage'
import RecycleBinPage from './pages/RecycleBinPage'
import AlbumPage from './pages/AlbumPage'
import AlbumDetailPage from './pages/AlbumDetailPage'
import MusicPage from './pages/MusicPage'
import PlaylistPage from './pages/PlaylistPage'
import PlaylistDetailPage from './pages/PlaylistDetailPage'
import ServiceStatusPage from './pages/ServiceStatusPage'
import ErrorBoundary from './components/ErrorBoundary'
import { colors } from './theme/tokens'

const { darkAlgorithm, defaultAlgorithm } = antdTheme

export default function App() {
  const isDark = useThemeStore((s) => s.isDark)

  const theme = {
    algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
    token: {
      colorPrimary: colors.primary,
      colorPrimaryBg: colors.primaryLight,
      colorPrimaryBorder: isDark ? 'rgba(129,236,254,0.25)' : '#f5d5b0',
      colorPrimaryHover: isDark ? '#a0f0ff' : colors.primaryDark,
      colorPrimaryActive: colors.primaryDark,
      colorBgLayout: colors.bg,
      colorBgContainer: colors.bgCard,
      colorBgElevated: isDark ? 'rgba(255,255,255,0.06)' : colors.bgCard,
      colorBgSpotlight: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.02)',
      colorBorder: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)',
      colorBorderSecondary: isDark ? 'rgba(255,255,255,0.04)' : '#f0eeeb',
      colorText: colors.text,
      colorTextSecondary: colors.textSecondary,
      colorTextTertiary: isDark ? '#666666' : '#999999',
      borderRadius: isDark ? 12 : 10,
      borderRadiusLG: 16,
      borderRadiusSM: 8,
      borderRadiusXS: 4,
      boxShadow: isDark
        ? '0 0 20px rgba(0,0,0,0.3)'
        : '0 1px 3px rgba(0,0,0,0.04)',
      boxShadowSecondary: isDark
        ? '0 0 40px rgba(0,0,0,0.4)'
        : '0 2px 8px rgba(0,0,0,0.06)',
      controlHeight: isDark ? 40 : 38,
      colorLink: colors.primary,
      fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
      colorSplit: isDark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.06)',
      wireframe: false,
    },
    components: {
      Button: {
        borderRadius: isDark ? 500 : 8,
        controlHeight: isDark ? 40 : 38,
        defaultBg: isDark ? 'rgba(255,255,255,0.04)' : '#ffffff',
        defaultBorderColor: isDark ? 'rgba(255,255,255,0.12)' : '#d9d9d9',
        defaultColor: colors.text,
        defaultHoverBg: isDark ? 'rgba(255,255,255,0.08)' : '#f5f5f5',
        defaultHoverBorderColor: isDark ? 'rgba(255,255,255,0.2)' : colors.primary,
        primaryShadow: isDark ? '0 0 20px rgba(129,236,254,0.3)' : 'none',
        fontWeight: 500,
      },
      Card: {
        borderRadiusLG: isDark ? 16 : 12,
        paddingLG: 24,
      },
      Table: {
        borderRadius: isDark ? 12 : 10,
        headerBg: isDark ? 'rgba(255,255,255,0.03)' : '#fafaf8',
        headerColor: isDark ? '#666666' : '#6b6b6b',
        rowHoverBg: isDark
          ? 'rgba(129,236,254,0.04)'
          : 'rgba(232,150,74,0.04)',
        borderColor: isDark ? 'rgba(255,255,255,0.06)' : '#f0eeeb',
      },
      Input: {
        borderRadius: isDark ? 500 : 8,
        controlHeight: isDark ? 40 : 38,
        activeBorderColor: colors.primary,
        hoverBorderColor: isDark
          ? 'rgba(129,236,254,0.3)'
          : 'rgba(232,150,74,0.3)',
        colorBgContainer: isDark ? 'rgba(255,255,255,0.04)' : '#ffffff',
      },
      Modal: {
        borderRadiusLG: isDark ? 16 : 14,
        headerBg: 'transparent',
        contentBg: isDark ? 'rgba(0,0,0,0.95)' : '#ffffff',
      },
      Menu: {
        itemBg: 'transparent',
        subMenuItemBg: 'transparent',
        darkItemBg: 'transparent',
        itemSelectedBg: isDark
          ? 'rgba(129,236,254,0.1)'
          : 'rgba(232,150,74,0.1)',
        itemSelectedColor: colors.primary,
      },
      Tag: {
        defaultBg: isDark ? 'rgba(255,255,255,0.06)' : '#f5f5f5',
        defaultColor: colors.textSecondary,
      },
      Progress: {
        remainingColor: isDark ? 'rgba(255,255,255,0.04)' : '#f0f0f0',
      },
      Slider: {
        railBg: isDark ? 'rgba(255,255,255,0.1)' : '#f0f0f0',
        trackBg: colors.primary,
        handleColor: colors.primary,
      },
      Tabs: {
        itemSelectedColor: colors.primary,
        inkBarColor: colors.primary,
        colorBgContainer: 'transparent',
      },
      Breadcrumb: {
        lastColor: colors.textSecondary,
        linkColor: colors.textSecondary,
      },
      Switch: {
        colorPrimary: colors.primary,
      },
    },
  }
  return (
    <ConfigProvider locale={zhCN} theme={theme}>
      <BrowserRouter>
        <Routes>
          {/* 公开路由 */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/forbidden" element={<ForbiddenPage />} />
          <Route path="/s/:code" element={<ShareAccessPage />} />

          {/* 普通用户路由 */}
          <Route element={<AuthGuard />}>
            <Route element={<AppLayout />}>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<PageTransition><ErrorBoundary><Dashboard /></ErrorBoundary></PageTransition>} />
              <Route path="/files" element={<PageTransition><ErrorBoundary><FileListPage /></ErrorBoundary></PageTransition>} />
              <Route path="/files/:id/edit" element={<PageTransition><ErrorBoundary><DocumentEditorPage /></ErrorBoundary></PageTransition>} />
              <Route path="/shares" element={<PageTransition><ErrorBoundary><MySharesPage /></ErrorBoundary></PageTransition>} />
              <Route path="/chat" element={<PageTransition><ErrorBoundary><ChatPage /></ErrorBoundary></PageTransition>} />
              <Route path="/friends" element={<PageTransition><ErrorBoundary><FriendPage /></ErrorBoundary></PageTransition>} />
              <Route path="/docker" element={<PageTransition><ErrorBoundary><DockerPage /></ErrorBoundary></PageTransition>} />
              <Route path="/cameras" element={<PageTransition><ErrorBoundary><CameraPage /></ErrorBoundary></PageTransition>} />
              <Route path="/cameras/:id" element={<PageTransition><ErrorBoundary><CameraLiveView /></ErrorBoundary></PageTransition>} />
              <Route path="/documents" element={<PageTransition><ErrorBoundary><DocumentListPage /></ErrorBoundary></PageTransition>} />
              <Route path="/documents/:id" element={<PageTransition><ErrorBoundary><DocumentEditorPage /></ErrorBoundary></PageTransition>} />
              <Route path="/album" element={<PageTransition><ErrorBoundary><AlbumPage /></ErrorBoundary></PageTransition>} />
              <Route path="/album/:id" element={<PageTransition><ErrorBoundary><AlbumDetailPage /></ErrorBoundary></PageTransition>} />
              <Route path="/music" element={<PageTransition><ErrorBoundary><MusicPage /></ErrorBoundary></PageTransition>} />
              <Route path="/playlist" element={<PageTransition><ErrorBoundary><PlaylistPage /></ErrorBoundary></PageTransition>} />
              <Route path="/playlist/:id" element={<PageTransition><ErrorBoundary><PlaylistDetailPage /></ErrorBoundary></PageTransition>} />
              <Route path="/trash" element={<PageTransition><ErrorBoundary><RecycleBinPage /></ErrorBoundary></PageTransition>} />
              <Route path="/settings" element={<PageTransition><ErrorBoundary><UserSettingsPage /></ErrorBoundary></PageTransition>} />
            </Route>
          </Route>

          {/* 管理员路由 */}
          <Route element={<AuthGuard />}>
            <Route element={<AdminGuard />}>
              <Route element={<AppLayout />}>
                <Route path="/admin" element={<PageTransition><ErrorBoundary><AdminPage /></ErrorBoundary></PageTransition>} />
                <Route path="/status" element={<PageTransition><ErrorBoundary><ServiceStatusPage /></ErrorBoundary></PageTransition>} />
              </Route>
            </Route>
          </Route>

          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
