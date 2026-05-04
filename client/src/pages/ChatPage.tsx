import { useEffect, useState, useRef } from 'react'
import {
  List, Input, Button, Card, Typography, Avatar,
  message, Modal, Popconfirm, Space, Checkbox, Divider, Badge, Tag,
} from 'antd'
import {
  SendOutlined, PlusOutlined, UserOutlined, DeleteOutlined,
  TeamOutlined, UsergroupAddOutlined, CrownOutlined,
  UserAddOutlined, UserDeleteOutlined, LogoutOutlined,
} from '@ant-design/icons'
import { useChatStore } from '../stores/chatStore'
import { useAuthStore } from '../stores/authStore'
import { useFriendStore } from '../stores/friendStore'
import { useWebSocket } from '../hooks/useWebSocket'
import { useNavigate } from 'react-router-dom'
import type { Message, GroupMember } from '../services/chat'
import type { FriendRequest } from '../services/chat'

const { Text } = Typography

function getFriendUserId(f: FriendRequest, myId: string): string {
  return f.user_id === myId ? f.friend_id : f.user_id
}

export default function ChatPage() {
  const navigate = useNavigate()
  const {
    conversations, currentConvId, messages, members, loading,
    fetchConversations, createConv, createGroup, selectConv, addMessage, deleteConversation,
    addMember, removeMember, leaveGroup,
  } = useChatStore()
  const { user } = useAuthStore()
  const { friends, fetchFriends } = useFriendStore()

  const [inputText, setInputText] = useState('')
  const [friendModalVisible, setFriendModalVisible] = useState(false)
  const [groupModalVisible, setGroupModalVisible] = useState(false)
  const [groupName, setGroupName] = useState('')
  const [selectedFriends, setSelectedFriends] = useState<string[]>([])
  const [memberModalVisible, setMemberModalVisible] = useState(false)
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
    } else if (wsMsg.type === 'read_receipt' && wsMsg.conversation_id === currentConvId) {
    }
  })

  useEffect(() => {
    if (!currentConvId || messages.length === 0) return
    const lastMsg = messages[messages.length - 1]
    if (lastMsg.seq > 0) {
      sendMessage({
        type: 'read_receipt',
        conversation_id: currentConvId,
        last_read_msg_id: String(lastMsg.seq),
      })
    }
  }, [currentConvId, messages.length])

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

  const handleCreateGroup = async () => {
    if (!groupName.trim()) {
      message.warning('请输入群名称')
      return
    }
    if (selectedFriends.length === 0) {
      message.warning('请至少选择一位好友')
      return
    }
    await createGroup(groupName.trim(), selectedFriends)
    setGroupModalVisible(false)
    setGroupName('')
    setSelectedFriends([])
    message.success('群聊已创建')
  }

  const handleAddMember = async (friendId: string) => {
    if (!currentConvId) return
    await addMember(currentConvId, friendId)
    setMemberModalVisible(false)
    message.success('已添加成员')
  }

  const handleRemoveMember = async (userId: string) => {
    if (!currentConvId) return
    await removeMember(currentConvId, userId)
    message.success('已移除成员')
  }

  const handleLeaveGroup = async () => {
    if (!currentConvId) return
    Modal.confirm({
      title: '退出群聊',
      content: '确定要退出该群聊吗？',
      onOk: async () => {
        await leaveGroup(currentConvId)
        message.success('已退出群聊')
      },
    })
  }

  const currentConv = conversations.find((c) => c.id === currentConvId)
  const isGroup = currentConv?.type === 'group'
  const myMember = members.find((m) => m.user_id === user?.id)
  const isOwner = myMember?.role === 'owner'

  // Friends not already in group
  const memberIds = new Set(members.map((m) => m.user_id))
  const addableFriends = friends.filter((f) => {
    const fid = getFriendUserId(f, user!.id)
    return !memberIds.has(fid)
  })

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 200px)', gap: 16 }}>
      {/* Conversation List */}
      <Card
        title="会话"
        style={{ width: 280, display: 'flex', flexDirection: 'column' }}
        extra={
          <Space size={4}>
            <Button type="text" icon={<UsergroupAddOutlined />} title="创建群聊"
              onClick={() => setGroupModalVisible(true)} />
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
                avatar={
                  <Badge count={conv.unread} size="small" offset={[-4, 4]}>
                    <Avatar icon={conv.type === 'group' ? <TeamOutlined /> : <UserOutlined />} />
                  </Badge>
                }
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
                  {msg.msg_type === 'system' ? (
                    <div style={{ textAlign: 'center', width: '100%', marginBottom: 8 }}>
                      <Text type="secondary" style={{ fontSize: 12, background: '#f5f5f5', padding: '2px 12px', borderRadius: 8 }}>
                        {msg.content}
                      </Text>
                    </div>
                  ) : (
                    <>
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
                    </>
                  )}
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

      {/* Member Panel for Group Chat */}
      {isGroup && (
        <Card
          title={<span><TeamOutlined /> 成员 ({members.length})</span>}
          style={{ width: 220, display: 'flex', flexDirection: 'column' }}
          styles={{ body: { flex: 1, overflow: 'auto', padding: 0 } }}
          extra={
            <Button type="text" size="small" icon={<UserAddOutlined />}
              onClick={() => setMemberModalVisible(true)} />
          }
        >
          <List
            dataSource={members}
            size="small"
            renderItem={(m: GroupMember) => (
              <List.Item
                style={{ padding: '6px 12px' }}
                actions={isOwner && m.user_id !== user?.id ? [
                  <Button key="remove" type="text" size="small" danger icon={<UserDeleteOutlined />}
                    onClick={() => handleRemoveMember(m.user_id)} />
                ] : []}
              >
                <Space>
                  <Avatar icon={<UserOutlined />} size="small" />
                  <span>{m.user_id === user?.id ? '我' : `用户 ${m.user_id}`}</span>
                  {m.role === 'owner' && <Tag color="gold" style={{ margin: 0, fontSize: 10 }}><CrownOutlined /></Tag>}
                </Space>
              </List.Item>
            )}
          />
          <div style={{ padding: 8, borderTop: '1px solid #f0f0f0' }}>
            <Button type="text" danger icon={<LogoutOutlined />} block onClick={handleLeaveGroup}>
              退出群聊
            </Button>
          </div>
        </Card>
      )}

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

      {/* Group Creation Modal */}
      <Modal
        title="创建群聊"
        open={groupModalVisible}
        onCancel={() => { setGroupModalVisible(false); setGroupName(''); setSelectedFriends([]) }}
        onOk={handleCreateGroup}
        okText="创建"
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Input
            placeholder="群名称"
            value={groupName}
            onChange={(e) => setGroupName(e.target.value)}
          />
          <Divider style={{ margin: 0 }} />
          <Text strong>选择成员</Text>
          {friends.length === 0 ? (
            <Text type="secondary">暂无好友，请先添加好友</Text>
          ) : (
            <Checkbox.Group
              style={{ width: '100%' }}
              value={selectedFriends}
              onChange={(values) => setSelectedFriends(values as string[])}
            >
              <List
                dataSource={friends}
                renderItem={(f) => {
                  const fid = getFriendUserId(f, user!.id)
                  return (
                    <List.Item style={{ padding: '4px 0' }}>
                      <Checkbox value={fid}>
                        <Space>
                          <Avatar icon={<UserOutlined />} size="small" />
                          {f.friend_username || fid}
                        </Space>
                      </Checkbox>
                    </List.Item>
                  )
                }}
              />
            </Checkbox.Group>
          )}
        </Space>
      </Modal>

      {/* Add Member Modal */}
      <Modal
        title="添加成员"
        open={memberModalVisible}
        onCancel={() => setMemberModalVisible(false)}
        footer={null}
      >
        {addableFriends.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Text type="secondary">所有好友已在群中</Text>
          </div>
        ) : (
          <List
            dataSource={addableFriends}
            renderItem={(f) => {
              const fid = getFriendUserId(f, user!.id)
              return (
                <List.Item
                  style={{ cursor: 'pointer', padding: '8px 12px', borderRadius: 6 }}
                  onClick={() => handleAddMember(fid)}
                >
                  <List.Item.Meta
                    avatar={<Avatar icon={<UserOutlined />} />}
                    title={f.friend_username || fid}
                  />
                  <Button type="primary" size="small" icon={<PlusOutlined />}>添加</Button>
                </List.Item>
              )
            }}
          />
        )}
      </Modal>
    </div>
  )
}
