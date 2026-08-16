import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

window.addEventListener('pageshow', (event) => {
  if (event.persisted) window.location.reload()
})

// Prevent browsers from restoring an in-memory SPA snapshot after a deployment.
window.addEventListener('unload', () => undefined)

async function removeLegacyFrontendCaches() {
  if ('caches' in window) {
    const keys = await caches.keys()
    await Promise.all(keys.map((key) => caches.delete(key)))
  }

  if (!('serviceWorker' in navigator)) return false
  const registrations = await navigator.serviceWorker.getRegistrations()
  await Promise.all(registrations.map((registration) => registration.unregister()))
  return Boolean(navigator.serviceWorker.controller)
}

async function bootstrap() {
  try {
    const wasControlled = await removeLegacyFrontendCaches()
    const reloadKey = `frontend-cache-cleaned:${__APP_BUILD_ID__}`
    if (wasControlled && sessionStorage.getItem(reloadKey) !== '1') {
      sessionStorage.setItem(reloadKey, '1')
      window.location.reload()
      return
    }
  } catch {
    // Cache cleanup is best-effort and must never prevent the app from starting.
  }

  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>,
  )
}

void bootstrap()
