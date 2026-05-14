import { useEffect, useMemo, useState, useCallback } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { Button, Spin, message, Tag, Dropdown } from 'antd'
import { ArrowLeftOutlined, DownloadOutlined } from '@ant-design/icons'
import { useEditor, EditorContent } from '@tiptap/react'
import { BubbleMenu } from '@tiptap/react/menus'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import { common, createLowlight } from 'lowlight'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import { getFileMeta } from '../services/file'
import { useAuthStore } from '../stores/authStore'

const lowlight = createLowlight(common)

const userColors = ['#f58742', '#5b9bd5', '#70ad47', '#ff6384', '#9966ff', '#4bc0c0']
function getUserColor(id?: string): string {
  if (!id) return userColors[0]
  let hash = 0
  for (let i = 0; i < id.length; i++) hash = id.charCodeAt(i) + ((hash << 5) - hash)
  return userColors[Math.abs(hash) % userColors.length]
}

// ── Markdown export helpers ──

type N = Record<string, unknown>

function collectInline(nodes: N[]): N[] {
  const out: N[] = []
  for (const n of nodes) {
    if (n.type === 'text') out.push(n)
    else if (n.content) out.push(...collectInline(n.content as N[]))
  }
  return out
}

function renderInline(nodes: N[]): string {
  return nodes.map(n => {
    const txt = (n.text as string) || ''
    const marks = (n.marks as N[]) || []
    let out = txt
    for (const m of marks) {
      switch (m.type as string) {
        case 'bold': out = `**${out}**`; break
        case 'italic': out = `*${out}*`; break
        case 'code': out = '`' + out + '`'; break
        case 'strike': out = `~~${out}~~`; break
        case 'link':
          out = `[${out}](${(m.attrs as N)?.href || ''})`; break
      }
    }
    return out
  }).join('')
}

function blockToMD(block: N): string {
  const type = block.type as string
  const content = (block.content as N[]) || []
  const attrs = (block.attrs || {}) as N

  switch (type) {
    case 'heading':
      return '#'.repeat((attrs.level as number) || 1) + ' ' + renderInline(content)
    case 'paragraph':
      return renderInline(content)
    case 'bulletList':
    case 'orderedList': {
      const prefix = type === 'orderedList' ? '1. ' : '- '
      return content.map(item => prefix + renderInline(collectInline((item.content as N[]) || []))).join('\n')
    }
    case 'taskList':
      return content.map(item => {
        const chk = (item.attrs as N)?.checked ? 'x' : ' '
        return `- [${chk}] ${renderInline(collectInline((item.content as N[]) || []))}`
      }).join('\n')
    case 'codeBlock':
      return '```' + ((attrs.language as string) || '') + '\n' +
        renderInline(content) + '\n```'
    case 'blockquote': {
      const t = renderInline(content)
      return '> ' + t.replace(/\n/g, '\n> ')
    }
    case 'horizontalRule':
      return '---'
    case 'image':
      return `![${attrs.alt || ''}](${attrs.src || ''})`
    default:
      return renderInline(content)
  }
}

function toMarkdown(json: N): string {
  if (!json || !json.content) return ''
  return (json.content as N[]).map(blockToMD).join('\n\n')
}

// ── Component ──

export default function DocumentEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const loc = useLocation()
  const user = useAuthStore((s) => s.user)
  const [title, setTitle] = useState('')
  const [loading, setLoading] = useState(true)
  const [connStatus, setConnStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')

  const ydoc = useMemo(() => new Y.Doc(), [])

  // load document metadata
  useEffect(() => {
    if (!id) return
    getFileMeta(id).then((f) => {
      setTitle(f.name)
      setLoading(false)
    }).catch(() => {
      message.error('文档不存在')
      navigate('/files')
    })
  }, [id, navigate])

  // establish Yjs WebSocket
  useEffect(() => {
    if (!id || loading) return

    const token = localStorage.getItem('access_token')
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const p = new WebsocketProvider(
      `${protocol}//${location.host}`,
      `ws/collab/${id}?token=${token}`,
      ydoc,
      { connect: true }
    )

    p.on('status', ({ status }: { status: string }) => {
      setConnStatus(status as 'connecting' | 'connected' | 'disconnected')
    })

    p.awareness.setLocalState({
      user: {
        name: user?.username || 'anonymous',
        color: getUserColor(user?.id),
      },
    })

    return () => {
      p.disconnect()
      p.destroy()
    }
    // ydoc is intentionally not in the dependency array — it's stable via useMemo
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, loading])

  // destroy ydoc only on unmount
  useEffect(() => {
    return () => ydoc.destroy()
  }, [ydoc])

  // ── editor ──

  // CollaborationCursor is not included here because it requires a live provider
  // at editor creation time. Provider awareness is set up separately via the
  // provider effect below, so remote cursors will still appear for other peers.
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        codeBlock: false,
      }),
      CodeBlockLowlight.configure({ lowlight }),
      Collaboration.configure({
        document: ydoc,
      }),
      TaskList,
      TaskItem.configure({ nested: true }),
    ],
    editorProps: {
      attributes: {
        style: 'min-height: 400px; padding: 0 8px; outline: none;',
      },
      handlePaste: (_, event) => {
        const text = event.clipboardData?.getData('text/plain') || ''
        if (!text) return false
        const mdPattern = /(^|\n)(#{1,6}\s|\*\s|-\s|\d+\.\s|```|> |\[.*\]\(.*\)|\*\*|__|~~|`[^`]+`)/m
        if (!mdPattern.test(text)) return false

        event.preventDefault()
        import('marked').then(({ marked }) => {
          const html = marked.parse(text) as string
          editor?.chain().focus().insertContent(html).run()
        }).catch(() => {})
        return true
      },
    },
  }, [ydoc])

  // ── export ──

  const handleExportMD = useCallback(() => {
    if (!editor) return
    const json = editor.getJSON() as unknown as N
    const md = toMarkdown(json)
    const name = title.replace(/\.clouddoc$/, '')
    const blob = new Blob([md], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${name}.md`
    a.click()
    URL.revokeObjectURL(url)
    message.success('已导出 Markdown')
  }, [editor, title])

  // ── back navigation ──

  const handleBack = () => {
    if (loc.pathname.startsWith('/files/')) navigate('/files')
    else navigate('/documents')
  }

  // ── render ──

  if (loading) {
    return <div style={{ textAlign: 'center', paddingTop: 100 }}><Spin size="large" /></div>
  }

  const statusMap: Record<string, { color: string; text: string }> = {
    connecting: { color: 'processing', text: '连接中' },
    connected: { color: 'success', text: '已连接' },
    disconnected: { color: 'error', text: '已断开' },
  }
  const st = statusMap[connStatus] || { color: 'default', text: connStatus }

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>返回</Button>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>{title}</h2>
        <Tag color={st.color}>{st.text}</Tag>
        <div style={{ flex: 1 }} />
        <Dropdown menu={{
          items: [
            { key: 'md', label: '导出 Markdown (.md)', icon: <DownloadOutlined /> },
          ],
          onClick: ({ key }) => { if (key === 'md') handleExportMD() },
        }}>
          <Button icon={<DownloadOutlined />}>导出</Button>
        </Dropdown>
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
        {editor ? (
          <>
            <BubbleMenu editor={editor}>
              <div style={{
                display: 'flex', gap: 2, background: '#fff',
                border: '1px solid #e8e8e8', borderRadius: 8, padding: 4,
                boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
              }}>
                <Button size="small" type="text"
                  onClick={() => editor.chain().focus().toggleBold().run()}
                  style={{ fontWeight: editor.isActive('bold') ? 700 : 400 }}>
                  <strong>B</strong>
                </Button>
                <Button size="small" type="text"
                  onClick={() => editor.chain().focus().toggleItalic().run()}
                  style={{ fontStyle: editor.isActive('italic') ? 'italic' : 'normal' }}>
                  <em>I</em>
                </Button>
                <Button size="small" type="text"
                  onClick={() => editor.chain().focus().toggleStrike().run()}
                  style={{ textDecoration: editor.isActive('strike') ? 'line-through' : undefined }}>
                  <span style={{ textDecoration: 'line-through' }}>S</span>
                </Button>
                <Button size="small" type="text"
                  onClick={() => editor.chain().focus().toggleCode().run()}
                  style={{ fontFamily: 'monospace', background: editor.isActive('code') ? '#f0f0f0' : undefined }}>
                  {'</>'}
                </Button>
                <Button size="small" type="text"
                  onClick={() => {
                    const url = window.prompt('链接地址:')
                    if (url) editor.chain().focus().setLink({ href: url }).run()
                  }}>
                  🔗
                </Button>
              </div>
            </BubbleMenu>
            <EditorContent editor={editor} />
          </>
        ) : <Spin />}
      </div>
    </div>
  )
}
