import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
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
import MusicPage from './pages/MusicPage'
import ServiceStatusPage from './pages/ServiceStatusPage'
import ErrorBoundary from './components/ErrorBoundary'
import { colors } from './theme/tokens'

const theme = {
  token: {
    colorPrimary: colors.primary,
    colorPrimaryBg: colors.primaryLight,
    colorPrimaryBorder: '#f5d5b0',
    colorBgLayout: colors.bg,
    colorBgContainer: colors.bgCard,
    colorBgElevated: colors.bgCard,
    colorBorderSecondary: '#f0eeeb',
    colorText: colors.text,
    colorTextSecondary: colors.textSecondary,
    borderRadius: 10,
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    boxShadowSecondary: '0 2px 8px rgba(0,0,0,0.06)',
    controlHeight: 38,
    colorLink: colors.primary,
  },
  components: {
    Button: {
      borderRadius: 8,
      controlHeight: 38,
    },
    Card: {
      borderRadiusLG: 12,
      paddingLG: 24,
    },
    Table: {
      borderRadius: 10,
      headerBg: '#fafaf8',
      headerColor: '#6b6b6b',
    },
    Input: {
      borderRadius: 8,
      controlHeight: 38,
    },
    Modal: {
      borderRadiusLG: 14,
    },
    Menu: {
      itemBg: 'transparent',
      subMenuItemBg: 'transparent',
    },
  },
}

export default function App() {
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
              <Route path="/music" element={<PageTransition><ErrorBoundary><MusicPage /></ErrorBoundary></PageTransition>} />
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
