import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useAuthStore } from './stores/authStore'
import AppLayout from './components/Layout'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import FileListPage from './pages/FileListPage'
import ChatPage from './pages/ChatPage'
import FriendPage from './pages/FriendPage'
import DockerPage from './pages/DockerPage'
import AdminPage from './pages/AdminPage'
import MySharesPage from './pages/MySharesPage'
import ShareAccessPage from './pages/ShareAccessPage'
import UserSettingsPage from './pages/UserSettingsPage'
import CameraListPage from './pages/CameraListPage'
import CameraLiveView from './pages/CameraLiveView'
import FaceLibraryPage from './pages/FaceLibraryPage'
import FaceAttendancePage from './pages/FaceAttendancePage'
import DocumentListPage from './pages/DocumentListPage'
import DocumentEditorPage from './pages/DocumentEditorPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import ForbiddenPage from './pages/ForbiddenPage'
import ErrorBoundary from './components/ErrorBoundary'
import { useAccess } from './hooks/useAccess'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  if (!isLoggedIn) return <Navigate to="/login" replace />
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { hasRole } = useAccess()
  if (!hasRole('admin') && !hasRole('super_admin')) return <Navigate to="/forbidden" replace />
  return <>{children}</>
}

const theme = {
  token: {
    colorPrimary: '#e8964a',
    colorPrimaryBg: '#fef3e7',
    colorPrimaryBorder: '#f5d5b0',
    colorBgLayout: '#fafaf8',
    colorBgContainer: '#ffffff',
    colorBgElevated: '#ffffff',
    colorBorderSecondary: '#f0eeeb',
    colorText: '#2c2c2c',
    colorTextSecondary: '#8c8c8c',
    borderRadius: 10,
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    boxShadowSecondary: '0 2px 8px rgba(0,0,0,0.06)',
    controlHeight: 38,
    colorLink: '#e8964a',
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
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/forbidden" element={<ForbiddenPage />} />
          <Route path="/s/:code" element={<ShareAccessPage />} />
          <Route
            element={
              <ProtectedRoute>
                <AppLayout />
              </ProtectedRoute>
            }
          >
            <Route path="/files" element={<ErrorBoundary><FileListPage /></ErrorBoundary>} />
            <Route path="/files/:id/edit" element={<ErrorBoundary><DocumentEditorPage /></ErrorBoundary>} />
            <Route path="/shares" element={<ErrorBoundary><MySharesPage /></ErrorBoundary>} />
            <Route path="/chat" element={<ErrorBoundary><ChatPage /></ErrorBoundary>} />
            <Route path="/friends" element={<ErrorBoundary><FriendPage /></ErrorBoundary>} />
            <Route path="/docker" element={<ErrorBoundary><DockerPage /></ErrorBoundary>} />
            <Route path="/admin" element={<AdminRoute><ErrorBoundary><AdminPage /></ErrorBoundary></AdminRoute>} />
            <Route path="/cameras" element={<ErrorBoundary><CameraListPage /></ErrorBoundary>} />
            <Route path="/cameras/:id" element={<ErrorBoundary><CameraLiveView /></ErrorBoundary>} />
            <Route path="/faces" element={<ErrorBoundary><FaceLibraryPage /></ErrorBoundary>} />
            <Route path="/attendance" element={<ErrorBoundary><FaceAttendancePage /></ErrorBoundary>} />
            <Route path="/documents" element={<ErrorBoundary><DocumentListPage /></ErrorBoundary>} />
            <Route path="/documents/:id" element={<ErrorBoundary><DocumentEditorPage /></ErrorBoundary>} />
            <Route path="/settings" element={<ErrorBoundary><UserSettingsPage /></ErrorBoundary>} />
          </Route>
          <Route path="*" element={<Navigate to="/files" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
