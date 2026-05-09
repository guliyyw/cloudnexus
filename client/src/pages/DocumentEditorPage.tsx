import { useEffect, useMemo, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Button, Spin, message } from 'antd'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import { getFileMeta } from '../services/file'
import { useAuthStore } from '../stores/authStore'

export default function DocumentEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const [title, setTitle] = useState('')
  const [loading, setLoading] = useState(true)

  // 立即创建 Y.Doc（useEditor 在首次渲染时需要）
  const ydoc = useMemo(() => new Y.Doc(), [])

  // 加载文档信息
  useEffect(() => {
    if (!id) return
    getFileMeta(id).then((f) => {
      setTitle(f.name)
      setLoading(false)
    }).catch(() => {
      message.error('文档不存在')
      navigate('/files')
    })
  }, [id])

  // 建立 Yjs WebSocket 连接
  useEffect(() => {
    if (!id || loading) return

    const token = localStorage.getItem('access_token')
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const serverUrl = `${protocol}//${location.host}`
    const roomName = `ws/collab/${id}?token=${token}`

    const provider = new WebsocketProvider(serverUrl, roomName, ydoc, {
      connect: true,
    })

    return () => {
      provider.disconnect()
    }
  }, [id, loading])

  const editor = useEditor({
    extensions: [
      StarterKit,
      Collaboration.configure({
        document: ydoc,
      }),
    ],
    editorProps: {
      attributes: {
        style: 'min-height: 400px; padding: 0 8px; outline: none;',
      },
    },
  })

  if (loading) {
    return <div style={{ textAlign: 'center', paddingTop: 100 }}><Spin size="large" /></div>
  }

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/documents')}>返回</Button>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>{title}</h2>
        <div style={{ flex: 1 }} />
        <span style={{ fontSize: 12, color: '#8c8c8c' }}>
          {user?.username || 'anonymous'}
        </span>
      </div>

      <div style={{
        background: '#fff',
        border: '1px solid #f0eeeb',
        borderRadius: 12,
        padding: 24,
        minHeight: 500,
      }}>
        {editor ? <EditorContent editor={editor} /> : <Spin />}
      </div>
    </div>
  )
}
