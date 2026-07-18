import { useCallback, useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { Button, Dropdown, Space, Spin, Tag, Typography, message } from 'antd'
import { ArrowLeftOutlined, DownloadOutlined } from '@ant-design/icons'
import { BubbleMenu } from '@tiptap/react/menus'
import { EditorContent, useEditor } from '@tiptap/react'
import Collaboration from '@tiptap/extension-collaboration'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import StarterKit from '@tiptap/starter-kit'
import TaskItem from '@tiptap/extension-task-item'
import TaskList from '@tiptap/extension-task-list'
import { common, createLowlight } from 'lowlight'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import { useAuthStore } from '../stores/authStore'
import { getFileMeta } from '../services/file'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Paragraph, Text, Title } = Typography
const lowlight = createLowlight(common)

const userColors = ['#f58742', '#5b9bd5', '#70ad47', '#ff6384', '#9966ff', '#4bc0c0']

function getUserColor(id?: string): string {
  if (!id) return userColors[0]
  let hash = 0
  for (let i = 0; i < id.length; i++) hash = id.charCodeAt(i) + ((hash << 5) - hash)
  return userColors[Math.abs(hash) % userColors.length]
}

type RichNode = Record<string, unknown>

function collectInline(nodes: RichNode[]): RichNode[] {
  const out: RichNode[] = []
  for (const node of nodes) {
    if (node.type === 'text') out.push(node)
    else if (node.content) out.push(...collectInline(node.content as RichNode[]))
  }
  return out
}

function renderInline(nodes: RichNode[]): string {
  return nodes.map((node) => {
    const text = (node.text as string) || ''
    const marks = (node.marks as RichNode[]) || []
    let output = text
    for (const mark of marks) {
      switch (mark.type as string) {
        case 'bold':
          output = `**${output}**`
          break
        case 'italic':
          output = `*${output}*`
          break
        case 'code':
          output = `\`${output}\``
          break
        case 'strike':
          output = `~~${output}~~`
          break
        case 'link':
          output = `[${output}](${(mark.attrs as RichNode)?.href || ''})`
          break
      }
    }
    return output
  }).join('')
}

function blockToMarkdown(block: RichNode): string {
  const type = block.type as string
  const content = (block.content as RichNode[]) || []
  const attrs = (block.attrs || {}) as RichNode

  switch (type) {
    case 'heading':
      return '#'.repeat((attrs.level as number) || 1) + ' ' + renderInline(content)
    case 'paragraph':
      return renderInline(content)
    case 'bulletList':
    case 'orderedList': {
      const prefix = type === 'orderedList' ? '1. ' : '- '
      return content.map((item) => prefix + renderInline(collectInline((item.content as RichNode[]) || []))).join('\n')
    }
    case 'taskList':
      return content.map((item) => {
        const checked = (item.attrs as RichNode)?.checked ? 'x' : ' '
        return `- [${checked}] ${renderInline(collectInline((item.content as RichNode[]) || []))}`
      }).join('\n')
    case 'codeBlock':
      return '```' + ((attrs.language as string) || '') + '\n' + renderInline(content) + '\n```'
    case 'blockquote': {
      const text = renderInline(content)
      return '> ' + text.replace(/\n/g, '\n> ')
    }
    case 'horizontalRule':
      return '---'
    case 'image':
      return `![${attrs.alt || ''}](${attrs.src || ''})`
    default:
      return renderInline(content)
  }
}

function toMarkdown(json: RichNode): string {
  if (!json || !json.content) return ''
  return (json.content as RichNode[]).map(blockToMarkdown).join('\n\n')
}

export default function DocumentEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const user = useAuthStore((state) => state.user)
  const [title, setTitle] = useState('')
  const [loading, setLoading] = useState(true)
  const [connStatus, setConnStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')

  const ydoc = useMemo(() => new Y.Doc(), [])

  useEffect(() => {
    if (!id) return

    getFileMeta(id).then((file) => {
      setTitle(file.name)
      setLoading(false)
    }).catch(() => {
      message.error('文档不存在')
      navigate('/files')
    })
  }, [id, navigate])

  useEffect(() => {
    if (!id || loading) return

    const token = localStorage.getItem('access_token')
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const provider = new WebsocketProvider(
      `${protocol}//${window.location.host}`,
      `ws/collab/${id}?token=${token}`,
      ydoc,
      { connect: true },
    )

    provider.on('status', ({ status }: { status: string }) => {
      setConnStatus(status as 'connecting' | 'connected' | 'disconnected')
    })

    // 文档协作里用户名和颜色都依赖 awareness，同一个 ydoc 生命周期内不要反复重建 provider。
    provider.awareness.setLocalState({
      user: {
        name: user?.username || 'anonymous',
        color: getUserColor(user?.id),
      },
    })

    return () => {
      provider.disconnect()
      provider.destroy()
    }
    // ydoc 通过 useMemo 保持稳定，故意不放进依赖数组，避免协作状态被重建。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, loading, user?.id, user?.username])

  useEffect(() => {
    return () => ydoc.destroy()
  }, [ydoc])

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false }),
      CodeBlockLowlight.configure({ lowlight }),
      Collaboration.configure({ document: ydoc }),
      TaskList,
      TaskItem.configure({ nested: true }),
    ],
    editorProps: {
      attributes: {
        style: 'min-height: 520px; padding: 0 4px; outline: none;',
      },
      handlePaste: (_, event) => {
        const text = event.clipboardData?.getData('text/plain') || ''
        if (!text) return false

        const markdownPattern = /(^|\n)(#{1,6}\s|\*\s|-\s|\d+\.\s|```|> |\[.*\]\(.*\)|\*\*|__|~~|`[^`]+`)/m
        if (!markdownPattern.test(text)) return false

        // 粘贴 Markdown 时转成 HTML 插入，避免用户手工再执行一次格式化动作。
        event.preventDefault()
        import('marked').then(({ marked }) => {
          const html = marked.parse(text) as string
          editor?.chain().focus().insertContent(html).run()
        }).catch(() => undefined)
        return true
      },
    },
  }, [ydoc])

  const handleExportMarkdown = useCallback(() => {
    if (!editor) return
    const json = editor.getJSON() as unknown as RichNode
    const markdown = toMarkdown(json)
    const name = title.replace(/\.clouddoc$/, '')
    const blob = new Blob([markdown], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `${name}.md`
    anchor.click()
    URL.revokeObjectURL(url)
    message.success('已导出 Markdown')
  }, [editor, title])

  const handleBack = () => {
    if (location.pathname.startsWith('/files/')) navigate('/files')
    else navigate('/documents')
  }

  if (loading) {
    return <div style={{ textAlign: 'center', paddingTop: 120 }}><Spin size="large" /></div>
  }

  const statusMap: Record<string, { color: string; text: string }> = {
    connecting: { color: 'processing', text: '连接中' },
    connected: { color: 'success', text: '已连接' },
    disconnected: { color: 'error', text: '已断开' },
  }
  const status = statusMap[connStatus] || { color: 'default', text: connStatus }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.lg, maxWidth: 1200, margin: '0 auto' }}>
      <section
        style={{
          borderRadius: radius.lg,
          padding: '28px 28px 22px',
          background: colors.surfaceRaised,
          border: `1px solid ${colors.borderSubtle}`,
          boxShadow: shadow.card,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
          <div style={{ minWidth: 0 }}>
            <Text style={{ color: colors.primary, fontWeight: 700, letterSpacing: 0.5 }}>COLLAB EDITOR</Text>
            <Title level={3} style={{ margin: '10px 0 8px', color: colors.text }}>{title}</Title>
            <Paragraph style={{ marginBottom: 0, color: colors.textSecondary, fontSize: 14, lineHeight: 1.8 }}>
              编辑区只重构壳层与信息层级，协作底层仍沿用现有 Yjs + WebSocket 架构，避免在大改版中破坏实时同步。
            </Paragraph>
          </div>
          <Space wrap>
            <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>返回</Button>
            <Dropdown
              menu={{
                items: [{ key: 'md', label: '导出 Markdown (.md)', icon: <DownloadOutlined /> }],
                onClick: ({ key }) => {
                  if (key === 'md') handleExportMarkdown()
                },
              }}
            >
              <Button icon={<DownloadOutlined />}>导出</Button>
            </Dropdown>
          </Space>
        </div>
      </section>

      <section
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: spacing.md,
          flexWrap: 'wrap',
          padding: '18px 22px',
          borderRadius: radius.lg,
          background: colors.surface,
          border: `1px solid ${colors.borderSubtle}`,
          boxShadow: shadow.card,
        }}
      >
        <Space size={12} wrap>
          <Tag color={status.color}>{status.text}</Tag>
          <Text style={{ color: colors.textSecondary }}>当前编辑者：{user?.username || 'anonymous'}</Text>
        </Space>
        <Text style={{ color: colors.textSecondary, fontSize: 12 }}>支持 Markdown 粘贴、任务列表和协作同步</Text>
      </section>

      <section
        style={{
          borderRadius: radius.lg,
          padding: 24,
          background: colors.surface,
          border: `1px solid ${colors.borderSubtle}`,
          boxShadow: shadow.card,
          minHeight: 620,
        }}
      >
        {editor ? (
          <>
            <BubbleMenu editor={editor}>
              <div
                style={{
                  display: 'flex',
                  gap: 4,
                  background: colors.panelBg,
                  border: `1px solid ${colors.borderSubtle}`,
                  borderRadius: radius.md,
                  padding: 4,
                  boxShadow: shadow.card,
                }}
              >
                <Button
                  size="small"
                  type="text"
                  onClick={() => editor.chain().focus().toggleBold().run()}
                  style={{ fontWeight: editor.isActive('bold') ? 700 : 400 }}
                >
                  <strong>B</strong>
                </Button>
                <Button
                  size="small"
                  type="text"
                  onClick={() => editor.chain().focus().toggleItalic().run()}
                  style={{ fontStyle: editor.isActive('italic') ? 'italic' : 'normal' }}
                >
                  <em>I</em>
                </Button>
                <Button
                  size="small"
                  type="text"
                  onClick={() => editor.chain().focus().toggleStrike().run()}
                  style={{ textDecoration: editor.isActive('strike') ? 'line-through' : undefined }}
                >
                  <span style={{ textDecoration: 'line-through' }}>S</span>
                </Button>
                <Button
                  size="small"
                  type="text"
                  onClick={() => editor.chain().focus().toggleCode().run()}
                  style={{ fontFamily: 'monospace', background: editor.isActive('code') ? colors.surfaceMuted : undefined }}
                >
                  {'</>'}
                </Button>
                <Button
                  size="small"
                  type="text"
                  onClick={() => {
                    const url = window.prompt('链接地址:')
                    if (url) editor.chain().focus().setLink({ href: url }).run()
                  }}
                >
                  🔗
                </Button>
              </div>
            </BubbleMenu>
            <EditorContent editor={editor} />
          </>
        ) : (
          <Spin />
        )}
      </section>
    </div>
  )
}
