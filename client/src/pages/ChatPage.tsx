import { useEffect, useState, useRef } from 'react'
import {
  List, Input, Button, Card, Typography, Avatar,
  message, Modal, InputNumber,
} from 'antd'
import { SendOutlined, PlusOutlined, UserOutlined } from '@ant-design/icons'
import { useChatStore } from '../stores/chatStore'
import { useWebSocket } from '../hooks/useWebSocket'
import type { Message } from '../services/chat'

const { Text } = Typography

export default function ChatPage() {
  const {
    conversations, currentConvId, messages, loading,
    fetchConversations, createConv, selectConv, addMessage,
  } = useChatStore()

  const [inputText, setInputText] = useState('')
  const [newConvVisible, setNewConvVisible] = useState(false)
  const [targetUserId, setTargetUserId] = useState<number | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => { fetchConversations() }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const { sendMessage } = useWebSocket((wsMsg) => {
    if (wsMsg.type === 'message') {
      addMessage({
        id: wsMsg.id!,
        conversation_id: wsMsg.conversation_id!,
        sender_id: wsMsg.sender_id!,
        content: wsMsg.content!,
        msg_type: wsMsg.msg_type || 'text',
        seq: 0,
        created_at: wsMsg.created_at || new Date().toISOString(),
      })
    }
  })

  const handleSend = () => {
    if (!inputText.trim() || !currentConvId) return
    sendMessage({
      type: 'message',
      conversation_id: currentConvId,
      content: inputText.trim(),
      msg_type: 'text',
    })
    setInputText('')
  }

  const currentConv = conversations.find((c) => c.id === currentConvId)

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 200px)', gap: 16 }}>
      {/* Conversation List */}
      <Card title="会话" style={{ width: 280, display: 'flex', flexDirection: 'column' }}
        extra={<Button type="text" icon={<PlusOutlined />} onClick={() => setNewConvVisible(true)} />}>
        <List
          dataSource={conversations}
          loading={loading}
          renderItem={(conv) => (
            <List.Item
              style={{
                cursor: 'pointer',
                padding: '8px 12px',
                borderRadius: 6,
                background: currentConvId === conv.id ? '#e6f4ff' : undefined,
              }}
              onClick={() => selectConv(conv.id)}
            >
              <List.Item.Meta
                avatar={<Avatar icon={<UserOutlined />} />}
                title={conv.name || `会话 ${conv.id}`}
                description={<Text type="secondary" ellipsis>{conv.type === 'private' ? '私聊' : '群聊'}</Text>}
              />
            </List.Item>
          )}
        />
      </Card>

      {/* Chat Area */}
      <Card
        title={currentConv ? (currentConv.name || `会话 ${currentConv.id}`) : '选择一个会话'}
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', padding: 0 }}
      >
        {currentConvId ? (
          <>
            <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
              {messages.map((msg: Message) => (
                <div key={msg.id} style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: msg.sender_id === 0 ? 'flex-end' : 'flex-start' }}>
                  <Text type="secondary" style={{ fontSize: 12, marginBottom: 4 }}>
                    {msg.sender_id} · {new Date(msg.created_at).toLocaleTimeString()}
                  </Text>
                  <div style={{
                    maxWidth: '70%', padding: '8px 14px', borderRadius: 12,
                    background: '#e6f4ff',
                    wordBreak: 'break-word',
                  }}>
                    {msg.content}
                  </div>
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>
            <div style={{ padding: '12px 16px', borderTop: '1px solid #f0f0f0', display: 'flex', gap: 8 }}>
              <Input.TextArea
                value={inputText}
                onChange={(e) => setInputText(e.target.value)}
                onPressEnter={(e) => { e.preventDefault(); handleSend() }}
                placeholder="输入消息..."
                autoSize={{ minRows: 1, maxRows: 4 }}
                style={{ flex: 1 }}
              />
              <Button type="primary" icon={<SendOutlined />} onClick={handleSend}>发送</Button>
            </div>
          </>
        ) : (
          <div style={{ flex: 1, display: 'flex', justifyContent: 'center', alignItems: 'center', color: '#999' }}>
            选择或创建一个会话开始聊天
          </div>
        )}
      </Card>

      <Modal
        title="创建私聊"
        open={newConvVisible}
        onOk={async () => {
          if (targetUserId) {
            await createConv(targetUserId)
            setNewConvVisible(false)
            setTargetUserId(null)
            message.success('会话已创建')
          }
        }}
        onCancel={() => setNewConvVisible(false)}
      >
        <InputNumber
          placeholder="目标用户 ID"
          value={targetUserId}
          onChange={(v) => setTargetUserId(v)}
          style={{ width: '100%' }}
          min={1}
        />
      </Modal>
    </div>
  )
}
