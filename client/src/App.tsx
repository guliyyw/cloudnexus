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

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
  if (!isLoggedIn) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            element={
              <ProtectedRoute>
                <AppLayout />
              </ProtectedRoute>
            }
          >
            <Route path="/files" element={<FileListPage />} />
            <Route path="/chat" element={<ChatPage />} />
            <Route path="/friends" element={<FriendPage />} />
            <Route path="/docker" element={<DockerPage />} />
            <Route path="/admin" element={<AdminPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/files" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
