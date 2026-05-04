import { useEffect, useState } from 'react'
import {
  Card, List, Button, Input, Modal, Space, Typography, Avatar,
  Badge, Tabs, Popconfirm, message, Tag, Tooltip,
} from 'antd'
import {
  UserOutlined, UserAddOutlined, SearchOutlined,
  CheckOutlined, CloseOutlined, DeleteOutlined,
  MessageOutlined, ReloadOutlined, TeamOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons'
import { useFriendStore } from '../stores/friendStore'
import { useChatStore } from '../stores/chatStore'
import { useAuthStore } from '../stores/authStore'
import { useNavigate } from 'react-router-dom'
import type { FriendRequest } from '../services/chat'

const { Text, Title } = Typography

export default function FriendPage() {
  const navigate = useNavigate()
  const {
    friends, pendingRequests, loading,
    fetchFriends, fetchPendingRequests,
    sendRequest, acceptRequest, rejectRequest, removeFriend,
  } = useFriendStore()
  const { createConv } = useChatStore()
  const { user } = useAuthStore()

  const [addFriendVisible, setAddFriendVisible] = useState(false)
  const [addFriendName, setAddFriendName] = useState('')
  const [adding, setAdding] = useState(false)
  const [activeTab, setActiveTab] = useState('friends')

  useEffect(() => {
    fetchFriends()
    fetchPendingRequests()
  }, [])

  const handleAddFriend = async () => {
    if (!addFriendName.trim()) return
    setAdding(true)
    try {
      await sendRequest(addFriendName.trim())
      message.success('好友请求已发送')
      setAddFriendName('')
      setAddFriendVisible(false)
      fetchPendingRequests()
    } catch (e: any) {
      message.error(e?.response?.data?.message || '发送失败')
    } finally {
      setAdding(false)
    }
  }

  const handleAccept = async (id: string) => {
    await acceptRequest(id)
    message.success('已接受好友请求')
  }

  const handleReject = async (id: string) => {
    await rejectRequest(id)
    message.success('已拒绝')
  }

  const handleStartChat = async (f: FriendRequest) => {
    const friendId = f.user_id === user!.id ? f.friend_id : f.user_id
    try {
      await createConv(friendId)
      message.success('会话已打开')
      navigate('/chat')
    } catch (e: any) {
      message.error(e?.response?.data?.message || '创建会话失败')
    }
  }

  const handleRemoveFriend = async (friendId: string) => {
    await removeFriend(friendId)
    message.success('已删除好友')
  }

  const getFriendDisplayId = (f: FriendRequest): string => {
    return f.user_id === user!.id ? f.friend_id : f.user_id
  }

  const pendingCount = pendingRequests.length

  const friendListContent = (
    <List
      dataSource={friends}
      loading={loading}
      locale={{ emptyText: '暂无好友，点击"添加好友"开始添加' }}
      renderItem={(f) => {
        const friendId = getFriendDisplayId(f)
        return (
          <List.Item
            actions={[
              <Tooltip title="发起会话" key="chat">
                <Button type="link" icon={<MessageOutlined />}
                  onClick={() => handleStartChat(f)} />
              </Tooltip>,
              <Popconfirm
                key="delete"
                title="确定删除该好友？"
                description="删除后需要重新发送好友请求"
                onConfirm={() => handleRemoveFriend(friendId)}
              >
                <Button type="link" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={
                <Badge status="success" offset={[-4, 32]}>
                  <Avatar icon={<UserOutlined />} size="large" />
                </Badge>
              }
              title={<Text strong>{f.friend_username || `用户 ${friendId}`}</Text>}
              description={
                <Space size="small">
                  <Tag color="green">好友</Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    添加于 {new Date(f.created_at).toLocaleDateString()}
                  </Text>
                </Space>
              }
            />
          </List.Item>
        )
      }}
    />
  )

  const pendingContent = (
    <List
      dataSource={pendingRequests}
      loading={loading}
      locale={{ emptyText: '没有待处理的好友请求' }}
      renderItem={(req) => (
        <List.Item
          actions={[
            <Button key="accept" type="primary" size="small" icon={<CheckOutlined />}
              onClick={() => handleAccept(req.id)}>
              接受
            </Button>,
            <Button key="reject" size="small" danger icon={<CloseOutlined />}
              onClick={() => handleReject(req.id)}>
              拒绝
            </Button>,
          ]}
        >
          <List.Item.Meta
            avatar={<Avatar icon={<UserOutlined />} size="large" />}
            title={<Text strong>{req.friend_username || `用户 ${req.user_id}`}</Text>}
            description={
              <Space size="small">
                <Tag color="orange">等待确认</Tag>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {new Date(req.created_at).toLocaleDateString()}
                </Text>
              </Space>
            }
          />
        </List.Item>
      )}
    />
  )

  const tabItems = [
    {
      key: 'friends',
      label: (
        <Space>
          <TeamOutlined />
          <span>好友列表</span>
          <Text type="secondary">({friends.length})</Text>
        </Space>
      ),
      children: friendListContent,
    },
    {
      key: 'requests',
      label: (
        <Space>
          <UserSwitchOutlined />
          <span>好友请求</span>
          {pendingCount > 0 && <Badge count={pendingCount} size="small" />}
        </Space>
      ),
      children: pendingContent,
    },
  ]

  return (
    <div style={{ maxWidth: 800, margin: '0 auto' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          <TeamOutlined style={{ marginRight: 8 }} />
          好友管理
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => { fetchFriends(); fetchPendingRequests() }}>
            刷新
          </Button>
          <Button type="primary" icon={<UserAddOutlined />}
            onClick={() => setAddFriendVisible(true)}>
            添加好友
          </Button>
        </Space>
      </div>

      <Card style={{ minHeight: 400 }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
          tabBarExtraContent={
            <Text type="secondary" style={{ fontSize: 12 }}>
              共 {friends.length} 位好友
              {pendingCount > 0 && `，${pendingCount} 条待处理请求`}
            </Text>
          }
        />
      </Card>

      {/* Add Friend Modal */}
      <Modal
        title={<span><UserAddOutlined style={{ marginRight: 8 }} />添加好友</span>}
        open={addFriendVisible}
        onOk={handleAddFriend}
        onCancel={() => { setAddFriendVisible(false); setAddFriendName('') }}
        confirmLoading={adding}
        okText="发送请求"
        cancelText="取消"
      >
        <div style={{ padding: '12px 0' }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
            输入对方的用户名，发送好友请求。对方接受后即可开始聊天。
          </Text>
          <Input
            size="large"
            placeholder="输入对方用户名"
            prefix={<SearchOutlined />}
            value={addFriendName}
            onChange={(e) => setAddFriendName(e.target.value)}
            onPressEnter={handleAddFriend}
            autoFocus
          />
        </div>
      </Modal>
    </div>
  )
}
