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

  handlerRef.current = handler

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (!token) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws?token=${token}`

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
      } catch {}
    }

    wsRef.current.onclose = () => {
      clearInterval(pingRef.current)
    }

    return () => {
      clearInterval(pingRef.current)
      wsRef.current?.close()
    }
  }, [])

  const sendMessage = useCallback((msg: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  return { sendMessage }
}
