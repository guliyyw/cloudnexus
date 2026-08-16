import { useMemo } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider, theme as antdTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useThemeStore } from './stores/themeStore'
import { AuthGuard, AdminGuard, ModuleGuard } from './components/Guard'
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
import DramaPage from './pages/DramaPage'
import ImageGenerationPage from './pages/ImageGenerationPage'
import ErrorBoundary from './components/ErrorBoundary'
import AppVersionGuard from './components/AppVersionGuard'
import { colors, radius, shadow } from './theme/tokens'

const { darkAlgorithm, defaultAlgorithm } = antdTheme

export default function App() {
  const isDark = useThemeStore((s) => s.isDark)

  // 自定义 token 和 Ant Design token 必须同时更新，避免组件库和页面内联样式出现两套主题语言。
  const theme = useMemo(() => ({
    algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
    token: {
      colorPrimary: colors.primary,
      colorPrimaryBg: colors.primaryLight,
      colorPrimaryBorder: colors.borderStrong,
      colorPrimaryHover: isDark ? '#a0f0ff' : colors.primaryDark,
      colorPrimaryActive: colors.primaryDark,
      colorBgLayout: colors.bg,
      colorBgContainer: colors.surfaceRaised,
      colorBgElevated: colors.panelBg,
      colorBgSpotlight: colors.surfaceRaised,
      colorBorder: colors.borderSubtle,
      colorBorderSecondary: colors.surfaceMuted,
      colorText: colors.text,
      colorTextSecondary: colors.textSecondary,
      colorTextTertiary: colors.mutedText,
      borderRadius: radius.md,
      borderRadiusLG: radius.lg,
      borderRadiusSM: radius.sm,
      borderRadiusXS: radius.sm,
      boxShadow: shadow.card,
      boxShadowSecondary: shadow.shell,
      controlHeight: isDark ? 42 : 40,
      colorLink: colors.primary,
      fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
      colorSplit: colors.borderSubtle,
      wireframe: false,
    },
    components: {
      Button: {
        borderRadius: isDark ? radius.xl : radius.md,
        controlHeight: isDark ? 42 : 40,
        defaultBg: colors.surface,
        defaultBorderColor: colors.borderSubtle,
        defaultColor: colors.text,
        defaultHoverBg: colors.hoverBg,
        defaultHoverBorderColor: colors.borderStrong,
        primaryShadow: isDark ? '0 0 20px rgba(129,236,254,0.24)' : '0 12px 24px rgba(232,150,74,0.18)',
        fontWeight: 600,
      },
      Card: {
        borderRadiusLG: radius.lg,
        paddingLG: 24,
        headerBg: 'transparent',
      },
      Table: {
        borderRadius: radius.lg,
        headerBg: colors.surfaceMuted,
        headerColor: colors.textSecondary,
        rowHoverBg: colors.hoverBg,
        borderColor: colors.borderSubtle,
      },
      Input: {
        borderRadius: radius.md,
        controlHeight: isDark ? 42 : 40,
        activeBorderColor: colors.primary,
        hoverBorderColor: colors.borderStrong,
        colorBgContainer: colors.surfaceRaised,
      },
      Modal: {
        borderRadiusLG: radius.lg,
        headerBg: 'transparent',
        contentBg: colors.panelBg,
      },
      Menu: {
        itemBg: 'transparent',
        subMenuItemBg: 'transparent',
        darkItemBg: 'transparent',
        itemSelectedBg: colors.hoverBg,
        itemSelectedColor: colors.primary,
      },
      Tag: {
        defaultBg: colors.surfaceMuted,
        defaultColor: colors.textSecondary,
      },
      Progress: {
        remainingColor: colors.surfaceMuted,
      },
      Slider: {
        railBg: colors.surfaceMuted,
        trackBg: colors.primary,
        handleColor: colors.primary,
      },
      Tabs: {
        itemSelectedColor: colors.primary,
        inkBarColor: colors.primary,
        colorBgContainer: 'transparent',
      },
      Breadcrumb: {
        lastColor: colors.text,
        linkColor: colors.textSecondary,
      },
      Switch: {
        colorPrimary: colors.primary,
      },
      Drawer: {
        colorBgElevated: colors.panelBg,
      },
      Dropdown: {
        colorBgElevated: colors.panelBg,
      },
    },
  }), [isDark])

  return (
    <ConfigProvider locale={zhCN} theme={theme}>
      <BrowserRouter>
        <AppVersionGuard />
        <Routes>
          {/* 公开路由不挂应用壳，避免登录和分享页被业务导航打断。 */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/forbidden" element={<ForbiddenPage />} />
          <Route path="/s/:code" element={<ShareAccessPage />} />

          {/* 普通用户路由统一复用应用壳，让导航、留白和播放器行为保持一致。 */}
          <Route element={<AuthGuard />}>
            <Route element={<AppLayout />}>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<PageTransition><ErrorBoundary><Dashboard /></ErrorBoundary></PageTransition>} />
              <Route path="/files" element={<ModuleGuard permission="module:files"><PageTransition><ErrorBoundary><FileListPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/files/:id/edit" element={<ModuleGuard permission="module:documents"><PageTransition><ErrorBoundary><DocumentEditorPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/shares" element={<ModuleGuard permission="module:shares"><PageTransition><ErrorBoundary><MySharesPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/chat" element={<ModuleGuard permission="module:chat"><PageTransition><ErrorBoundary><ChatPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/friends" element={<ModuleGuard permission="module:friends"><PageTransition><ErrorBoundary><FriendPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/docker" element={<ModuleGuard permission="module:docker"><PageTransition><ErrorBoundary><DockerPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/cameras" element={<ModuleGuard permission="module:cameras"><PageTransition><ErrorBoundary><CameraPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/cameras/:id" element={<ModuleGuard permission="module:cameras"><PageTransition><ErrorBoundary><CameraLiveView /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/documents" element={<ModuleGuard permission="module:documents"><PageTransition><ErrorBoundary><DocumentListPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/documents/:id" element={<ModuleGuard permission="module:documents"><PageTransition><ErrorBoundary><DocumentEditorPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/album" element={<ModuleGuard permission="module:album"><PageTransition><ErrorBoundary><AlbumPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/album/:id" element={<ModuleGuard permission="module:album"><PageTransition><ErrorBoundary><AlbumDetailPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/music" element={<ModuleGuard permission="module:music"><PageTransition><ErrorBoundary><MusicPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/playlist" element={<Navigate to="/music" replace />} />
              <Route path="/playlist/:id" element={<Navigate to="/music" replace />} />
              <Route path="/drama" element={<ModuleGuard permission="module:drama"><ErrorBoundary><DramaPage /></ErrorBoundary></ModuleGuard>} />
              <Route path="/image-generation" element={<ModuleGuard permission="module:image_generation"><PageTransition><ErrorBoundary><ImageGenerationPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/trash" element={<ModuleGuard permission="module:trash"><PageTransition><ErrorBoundary><RecycleBinPage /></ErrorBoundary></PageTransition></ModuleGuard>} />
              <Route path="/settings" element={<PageTransition><ErrorBoundary><UserSettingsPage /></ErrorBoundary></PageTransition>} />
            </Route>
          </Route>

          {/* 管理员路由和普通应用壳共用视觉体系，只在权限上额外加一层守卫。 */}
          <Route element={<AuthGuard />}>
            <Route element={<AdminGuard />}>
              <Route element={<AppLayout />}>
                <Route path="/admin" element={<PageTransition><ErrorBoundary><AdminPage /></ErrorBoundary></PageTransition>} />
                <Route path="/status" element={<Navigate to="/admin?tab=status" replace />} />
              </Route>
            </Route>
          </Route>

          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
