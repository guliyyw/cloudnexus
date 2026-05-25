import { useEffect, useRef, useCallback } from 'react'

interface WSMessage {
  type: string
  id?: string
  conversation_id?: string
  sender_id?: string
  content?: string
  msg_type?: string
  status?: string
  user_id?: string
  created_at?: string
  msg_id?: string
  last_read_msg_id?: string
}

type MessageHandler = (msg: WSMessage) => void

export function useWebSocket(handler: MessageHandler) {
  const wsRef = useRef<WebSocket | null>(null)
  const pingRef = useRef<number>(0)
  const handlerRef = useRef<MessageHandler>(handler)
  const tokenRef = useRef<string | null>(null)

  // 更新 handler ref，避免 stale closure
  handlerRef.current = handler

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    
    // 如果 token 未变化且 WebSocket 已连接，不重连
    if (token === tokenRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }
    
    // token 变化或首次连接，更新 ref
    tokenRef.current = token
    
    if (!token) {
      // 无 token 时关闭连接
      if (wsRef.current) {
        clearInterval(pingRef.current)
        wsRef.current.close()
        wsRef.current = null
      }
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws?token=${token}`

    // 关闭旧连接
    if (wsRef.current) {
      clearInterval(pingRef.current)
      wsRef.current.close()
    }

    wsRef.current = new WebSocket(url)

    wsRef.current.onopen = () => {
      pingRef.current = window.setInterval(() => {
        wsRef.current?.send(JSON.stringify({ type: 'ping' }))
      }, 30000)
    }

    wsRef.current.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        if (msg.type === 'pong') return
        // 使用 ref 调用最新的 handler，避免 stale closure
        handlerRef.current(msg)
      } catch {}
    }

    wsRef.current.onclose = () => {
      clearInterval(pingRef.current)
    }

    return () => {
      clearInterval(pingRef.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, []) // 空依赖数组，通过 tokenRef 手动检测变化

  const sendMessage = useCallback((msg: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  return { sendMessage }
}
