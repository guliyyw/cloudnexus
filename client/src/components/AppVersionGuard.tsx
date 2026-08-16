import { useEffect } from 'react'
import { Modal } from 'antd'
import { useLocation } from 'react-router-dom'

let updatePromptOpen = false

export default function AppVersionGuard() {
  const location = useLocation()

  useEffect(() => {
    const checkVersion = async () => {
      try {
        const response = await fetch(`/version.json?t=${Date.now()}`, { cache: 'no-store' })
        if (!response.ok) return
        const data = await response.json() as { buildId?: string }
        if (!data.buildId || data.buildId === __APP_BUILD_ID__ || updatePromptOpen) return
        updatePromptOpen = true
        Modal.confirm({
          title: '系统已更新',
          content: '检测到新的前端版本，刷新后继续使用，避免页面样式和功能不一致。',
          okText: '立即刷新',
          cancelText: '稍后',
          onOk: () => window.location.reload(),
          afterClose: () => { updatePromptOpen = false },
        })
      } catch {
        // A transient version-check failure must not interrupt normal navigation.
      }
    }

    void checkVersion()
    const handleVisibility = () => {
      if (document.visibilityState === 'visible') void checkVersion()
    }
    document.addEventListener('visibilitychange', handleVisibility)
    return () => document.removeEventListener('visibilitychange', handleVisibility)
  }, [location.pathname])

  return null
}
