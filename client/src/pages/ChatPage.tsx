import { useEffect, useState, useRef } from 'react'
import {
  List, Input, Button, Card, Typography, Avatar,
  message, Modal, Popconfirm, Space,
} from 'antd'
import {
  SendOutlined, PlusOutlined, UserOutlined, DeleteOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import { useChatStore } from '../stores/chatStore'
import { useAuthStore } from '../stores/authStore'
import { useFriendStore } from '../stores/friendStore'
import { useWebSocket } from '../hooks/useWebSocket'
import { useNavigate } from 'react-router-dom'
import type { Message } from '../services/chat'
import type { FriendRequest } from '../services/chat'

const { Text } = Typography

function getFriendUserId(f: FriendRequest, myId: string): string {
  return f.user_id === myId ? f.friend_id : f.user_id
}

export default function ChatPage() {
  const navigate = useNavigate()
  const {
    conversations, currentConvId, messages, loading,
    fetchConversations, createConv, selectConv, addMessage, deleteConversation,
  } = useChatStore()
  const { user } = useAuthStore()
  const { friends, fetchFriends } = useFriendStore()

  const [inputText, setInputText] = useState('')
  const [friendModalVisible, setFriendModalVisible] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchConversations()
    if (user) {
      fetchFriends()
    }
  }, [user])

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

  const handleStartChat = async (friendId: string) => {
    await createConv(friendId)
    setFriendModalVisible(false)
    message.success('会话已打开')
  }

  const currentConv = conversations.find((c) => c.id === currentConvId)

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 200px)', gap: 16 }}>
      {/* Conversation List */}
      <Card
        title="会话"
        style={{ width: 280, display: 'flex', flexDirection: 'column' }}
        extra={
          <Space size={4}>
            <Button type="text" icon={<TeamOutlined />} title="好友管理"
              onClick={() => navigate('/friends')} />
            <Button type="text" icon={<PlusOutlined />}
              onClick={() => setFriendModalVisible(true)} />
          </Space>
        }
      >
        <List
          dataSource={conversations}
          loading={loading}
          locale={{ emptyText: '暂无会话，请先添加好友再开始聊天' }}
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
              <Popconfirm
                title="确定删除该会话？"
                onConfirm={(e) => {
                  e?.stopPropagation()
                  deleteConversation(conv.id)
                }}
                onCancel={(e) => e?.stopPropagation()}
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={(e) => e.stopPropagation()}
                />
              </Popconfirm>
            </List.Item>
          )}
        />
      </Card>

      {/* Chat Area */}
      <Card
        title={currentConv ? (currentConv.name || `会话 ${currentConv.id}`) : '选择一个会话'}
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', padding: 0 } }}
      >
        {currentConvId ? (
          <>
            <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
              {messages.map((msg: Message) => (
                <div key={msg.id} style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: msg.sender_id === user?.id ? 'flex-end' : 'flex-start' }}>
                  <Text type="secondary" style={{ fontSize: 12, marginBottom: 4 }}>
                    {msg.sender_id === user?.id ? '我' : (currentConv?.name || `用户${msg.sender_id}`)} · {new Date(msg.created_at).toLocaleTimeString()}
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

      {/* Friend Selection Modal */}
      <Modal
        title="选择好友"
        open={friendModalVisible}
        onCancel={() => setFriendModalVisible(false)}
        footer={null}
      >
        {friends.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Text type="secondary">暂无好友</Text>
            <div style={{ marginTop: 12 }}>
              <Button type="primary" icon={<TeamOutlined />}
                onClick={() => { setFriendModalVisible(false); navigate('/friends') }}>
                前往好友页面添加
              </Button>
            </div>
          </div>
        ) : (
          <List
            dataSource={friends}
            renderItem={(f) => (
              <List.Item
                style={{ cursor: 'pointer', padding: '8px 12px', borderRadius: 6 }}
                onClick={() => handleStartChat(getFriendUserId(f, user!.id))}
              >
                <List.Item.Meta
                  avatar={<Avatar icon={<UserOutlined />} />}
                  title={f.friend_username || `用户 ${getFriendUserId(f, user!.id)}`}
                />
              </List.Item>
            )}
          />
        )}
      </Modal>
    </div>
  )
}
