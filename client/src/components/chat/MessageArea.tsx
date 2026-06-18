import { useEffect, useRef } from 'react'
import { Card, Typography, Button, Space } from 'antd'
import {
  DownloadOutlined, UploadOutlined, EyeOutlined,
  PictureOutlined, LinkOutlined, CustomerServiceOutlined,
} from '@ant-design/icons'
import type { Message, Conversation } from '../../services/chat'
import { getDownloadUrl, getPreviewUrl } from '../../services/file'
import { isPreviewable } from '../../utils/preview'
import { formatFileSize } from '../../utils/format'
import ChatInput from './ChatInput'

const { Text } = Typography

export interface LinkPreview {
  url: string
  title: string
  description: string
  image: string
  site_name: string
}

interface MessageAreaProps {
  currentConv: Conversation | undefined
  currentConvId: string | null
  messages: Message[]
  userId: string | undefined
  inputText: string
  uploadingImg: boolean
  exporting: boolean
  importing: boolean
  activeMessageId: string | null
  linkPreviews: Record<string, LinkPreview>
  importFileRef: React.RefObject<HTMLInputElement>
  onInputChange: (val: string) => void
  onSend: () => void
  onPaste: (e: React.ClipboardEvent) => void
  onImageUpload: (file: File) => void
  onFilePickerOpen: () => void
  onExport: () => void
  onImportClick: () => void
  onImportFile: (file: File) => void
  onOpenAlbumPicker: (fileId: string) => void
  onPlayInMusic: (fc: { file_id: string; file_name: string; mime_type: string; file_size: number }) => void
  onActiveMessageShown: () => void
}

function parseFileContent(content: string) {
  try {
    return JSON.parse(content) as {
      file_id: string; file_name: string; file_size: number; mime_type: string
      url?: string; download_url?: string
    }
  } catch {
    return null
  }
}

const urlRegex = /https?:\/\/[^\s<]+[^\s<.,;:!?)}\]'"`>]/g

function detectUrls(text: string): string[] {
  const matches = text.match(urlRegex)
  return matches ? [...new Set(matches)] : []
}

export default function MessageArea({
  currentConv,
  currentConvId,
  messages,
  userId,
  inputText,
  uploadingImg,
  exporting,
  importing,
  activeMessageId,
  linkPreviews,
  importFileRef,
  onInputChange,
  onSend,
  onPaste,
  onImageUpload,
  onFilePickerOpen,
  onExport,
  onImportClick,
  onImportFile,
  onOpenAlbumPicker,
  onPlayInMusic,
  onActiveMessageShown,
}: MessageAreaProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const activeMessageRef = useRef<HTMLDivElement | null>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    if (activeMessageId) {
      const timer = setTimeout(() => {
        activeMessageRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
        onActiveMessageShown()
      }, 200)
      return () => clearTimeout(timer)
    }
    scrollToBottom()
    const timer = setTimeout(scrollToBottom, 400)
    return () => clearTimeout(timer)
  }, [messages.length, activeMessageId, onActiveMessageShown])

  return (
    <Card
      title={currentConv ? (currentConv.name || `会话 ${currentConv.id}`) : '选择一个会话'}
      style={{ flex: 1, height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', padding: 0, overflow: 'hidden' } }}
      extra={
        currentConvId ? (
          <Space size={4}>
            <Button type="text" size="small" icon={<DownloadOutlined />}
              title="导出聊天记录" loading={exporting}
              onClick={onExport} />
            <Button type="text" size="small" icon={<UploadOutlined />}
              title="导入聊天记录" loading={importing}
              onClick={onImportClick} />
          </Space>
        ) : undefined
      }
    >
      {currentConvId ? (
        <>
          <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
            {messages.map((msg: Message) => {
              const isMe = msg.sender_id === userId
              const alignStyle = { alignItems: isMe ? 'flex-end' as const : 'flex-start' as const }
              const senderLabel = isMe ? '我' : (currentConv?.name || `用户${msg.sender_id}`)
              const timeStr = new Date(msg.created_at).toLocaleTimeString()
              const urls = msg.msg_type === 'text' ? detectUrls(msg.content) : []
              const linkPrev = linkPreviews[msg.id]

              const isActive = activeMessageId === msg.id

              return (
              <div
                key={msg.id}
                ref={isActive ? activeMessageRef : null}
                style={{
                  marginBottom: 16,
                  display: 'flex',
                  flexDirection: 'column',
                  ...alignStyle,
                  padding: isActive ? '8px' : 0,
                  borderRadius: 12,
                  background: isActive ? 'rgba(129,236,254,0.12)' : undefined,
                  boxShadow: isActive ? '0 0 0 1px rgba(129,236,254,0.25)' : undefined,
                }}
              >
                {msg.msg_type === 'system' ? (
                  <div style={{ textAlign: 'center', width: '100%', marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12, background: 'rgba(255,255,255,0.05)', padding: '2px 12px', borderRadius: 8 }}>
                      {msg.content}
                    </Text>
                  </div>
                ) : msg.msg_type === 'image' ? (
                  (() => {
                    const fc = parseFileContent(msg.content)
                    const src = fc?.url || getPreviewUrl(fc?.file_id || '')
                    return (
                      <div style={{ maxWidth: '70%' }}>
                        <Text type="secondary" style={{ fontSize: 12, marginBottom: 4, display: 'block' }}>
                          {senderLabel} · {timeStr}
                        </Text>
                        <img
                          src={src}
                          alt={fc?.file_name || '图片'}
                          style={{ maxWidth: 320, maxHeight: 320, borderRadius: 12, cursor: 'pointer', objectFit: 'cover' }}
                          onClick={() => window.open(src, '_blank')}
                        />
                      </div>
                    )
                  })()
                ) : msg.msg_type === 'video' ? (
                  (() => {
                    const fc = parseFileContent(msg.content)
                    const src = fc?.url || getPreviewUrl(fc?.file_id || '')
                    return (
                      <div style={{ maxWidth: '70%' }}>
                        <Text type="secondary" style={{ fontSize: 12, marginBottom: 4, display: 'block' }}>
                          {senderLabel} · {timeStr}
                        </Text>
                        <video
                          src={src}
                          controls
                          preload="metadata"
                          style={{ maxWidth: 320, maxHeight: 320, borderRadius: 12 }}
                        />
                      </div>
                    )
                  })()
                ) : msg.msg_type === 'file' ? (
                  (() => {
                    const fc = parseFileContent(msg.content)
                    if (!fc) return (
                      <div style={{
                        maxWidth: '70%', padding: '8px 14px', borderRadius: 12,
                        background: 'rgba(129,236,254,0.06)', wordBreak: 'break-word',
                      }}>
                        {msg.content}
                      </div>
                    )
                    return (
                      <div style={{ maxWidth: '70%' }}>
                        <Text type="secondary" style={{ fontSize: 12, marginBottom: 4, display: 'block' }}>
                          {senderLabel} · {timeStr}
                        </Text>
                        <Card
                          size="small"
                          style={{ borderRadius: 12, background: 'rgba(129,236,254,0.06)' }}
                          styles={{ body: { padding: '10px 14px' } }}
                        >
                          <Space direction="vertical" size={4}>
                            <Text strong style={{ fontSize: 13 }}>{fc.file_name}</Text>
                            <Text type="secondary" style={{ fontSize: 11 }}>
                              {fc.mime_type || '未知类型'} · {formatFileSize(fc.file_size)}
                            </Text>
                            <Space size={4}>
                              {isPreviewable(fc.mime_type) && (
                                <Button type="link" size="small" icon={<EyeOutlined />}
                                  href={getPreviewUrl(fc.file_id)} target="_blank">
                                  预览
                                </Button>
                              )}
                              {fc.mime_type?.startsWith('image/') && (
                                <Button type="link" size="small" icon={<PictureOutlined />}
                                  onClick={() => onOpenAlbumPicker(fc.file_id)}>
                                  相册
                                </Button>
                              )}
                              {fc.mime_type?.startsWith('audio/') && (
                                <Button type="link" size="small" icon={<CustomerServiceOutlined />}
                                  onClick={() => onPlayInMusic(fc)}>
                                  播放
                                </Button>
                              )}
                              <Button type="link" size="small" icon={<DownloadOutlined />}
                                href={getDownloadUrl(fc.file_id)}>
                                下载
                              </Button>
                            </Space>
                          </Space>
                        </Card>
                      </div>
                    )
                  })()
                ) : (
                  <div style={{ maxWidth: '70%' }}>
                    <Text type="secondary" style={{ fontSize: 12, marginBottom: 4, display: 'block' }}>
                      {senderLabel} · {timeStr}
                    </Text>
                    <div style={{
                      padding: '8px 14px', borderRadius: 12,
                      background: 'rgba(129,236,254,0.06)', wordBreak: 'break-word',
                    }}>
                      {msg.content}
                    </div>
                    {/* Link Card */}
                    {linkPrev && (linkPrev.title || linkPrev.image) ? (
                      <a
                        href={linkPrev.url || urls[0]}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ textDecoration: 'none' }}
                      >
                        <Card
                          size="small"
                          style={{ marginTop: 6, borderRadius: 10, background: 'rgba(255,255,255,0.04)' }}
                          styles={{ body: { padding: '10px 12px' } }}
                        >
                          <div style={{ display: 'flex', gap: 10 }}>
                            {linkPrev.image && (
                              <img
                                src={linkPrev.image}
                                alt=""
                                style={{ width: 60, height: 60, borderRadius: 6, objectFit: 'cover', flexShrink: 0 }}
                              />
                            )}
                            <div style={{ flex: 1, minWidth: 0 }}>
                              <Text strong style={{ fontSize: 13, display: 'block' }} ellipsis>
                                <LinkOutlined style={{ marginRight: 4 }} />
                                {linkPrev.title || urls[0]}
                              </Text>
                              {linkPrev.description && (
                                <Text type="secondary" style={{ fontSize: 11, display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' } as React.CSSProperties}>
                                  {linkPrev.description}
                                </Text>
                              )}
                              {linkPrev.site_name && (
                                <Text type="secondary" style={{ fontSize: 10 }}>{linkPrev.site_name}</Text>
                              )}
                            </div>
                          </div>
                        </Card>
                      </a>
                    ) : null}
                    {/* Simple link fallback when no preview or empty preview */}
                    {(!linkPrev || !(linkPrev.title || linkPrev.image)) && urls.length > 0 && (
                      <div style={{ marginTop: 4 }}>
                        {urls.map((u, i) => (
                          <a key={i} href={u} target="_blank" rel="noopener noreferrer" style={{ fontSize: 12, display: 'block' }}>
                            <LinkOutlined /> {u.length > 50 ? u.slice(0, 50) + '...' : u}
                          </a>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )})}
            <div ref={messagesEndRef} />
          </div>
          <ChatInput
            value={inputText}
            onChange={onInputChange}
            onSend={onSend}
            onPaste={onPaste}
            onImageUpload={onImageUpload}
            onFilePickerOpen={onFilePickerOpen}
            uploadingImg={uploadingImg}
          />
          {/* Hidden file input for import */}
          <input
            ref={importFileRef}
            type="file"
            accept=".json"
            style={{ display: 'none' }}
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) onImportFile(file)
              e.target.value = ''
            }}
          />
        </>
      ) : (
        <div style={{ flex: 1, display: 'flex', justifyContent: 'center', alignItems: 'center', color: '#888' }}>
          选择或创建一个会话开始聊天
        </div>
      )}
    </Card>
  )
}
