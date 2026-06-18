import { useCallback, useEffect, useRef } from 'react'

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

  // WebSocket 事件监听只会注册一次，所以通过 ref 始终拿到最新回调，避免消息处理闭包过期。
  handlerRef.current = handler

  useEffect(() => {
    const token = localStorage.getItem('access_token')

    // token 不变且连接可用时，保留现有连接，避免切页面时重复握手。
    if (token === tokenRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    tokenRef.current = token

    if (!token) {
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
        handlerRef.current(msg)
      } catch {
        // 后端偶发返回非 JSON 帧时忽略，避免让聊天页因单条异常消息崩掉。
      }
    }

    wsRef.current.onclose = () => {
      clearInterval(pingRef.current)
    }

    return () => {
      clearInterval(pingRef.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [])

  const sendMessage = useCallback((msg: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  return { sendMessage }
}
