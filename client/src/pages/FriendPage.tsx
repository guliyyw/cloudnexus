import { useEffect, useState } from 'react'
import {
  Card, List, Button, Input, Modal, Space, Typography, Avatar,
  Badge, Tabs, Popconfirm, message, Tag, Tooltip, Input as AntInput,
} from 'antd'
import {
  UserOutlined, UserAddOutlined, SearchOutlined,
  CheckOutlined, CloseOutlined, DeleteOutlined,
  MessageOutlined, ReloadOutlined, TeamOutlined,
  UserSwitchOutlined, StopOutlined, UndoOutlined,
  EditOutlined,
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
    friends, pendingRequests, blocklist, onlineStatus, loading,
    fetchFriends, fetchPendingRequests, fetchBlocklist,
    sendRequest, acceptRequest, rejectRequest, removeFriend,
    blockUser, unblockUser, setRemark,
  } = useFriendStore()
  const { createConv } = useChatStore()
  const { user } = useAuthStore()

  const [addFriendVisible, setAddFriendVisible] = useState(false)
  const [addFriendName, setAddFriendName] = useState('')
  const [addFriendMsg, setAddFriendMsg] = useState('')
  const [adding, setAdding] = useState(false)
  const [activeTab, setActiveTab] = useState('friends')
  const [editingRemark, setEditingRemark] = useState<string | null>(null)
  const [remarkValue, setRemarkValue] = useState('')

  useEffect(() => {
    fetchFriends()
    fetchPendingRequests()
    fetchBlocklist()
  }, [])

  // Refresh online status every 30s
  useEffect(() => {
    const timer = setInterval(() => {
      const store = useFriendStore.getState()
      if (store.friends.length > 0) store.fetchOnlineStatus()
    }, 30000)
    return () => clearInterval(timer)
  }, [])

  const handleAddFriend = async () => {
    if (!addFriendName.trim()) return
    setAdding(true)
    try {
      await sendRequest(addFriendName.trim(), addFriendMsg, 0)
      message.success('好友请求已发送')
      setAddFriendName('')
      setAddFriendMsg('')
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

  const handleBlock = async (friendId: string) => {
    await blockUser(friendId)
    message.success('已拉黑')
  }

  const handleUnblock = async (friendId: string) => {
    await unblockUser(friendId)
    message.success('已取消拉黑')
  }

  const handleSetRemark = async (friendId: string) => {
    await setRemark(friendId, remarkValue)
    setEditingRemark(null)
    message.success('备注已设置')
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
        const isOnline = onlineStatus[friendId] === true
        const displayName = f.remark || f.friend_nickname || f.friend_username || `用户 ${friendId}`
        return (
          <List.Item
            actions={[
              <Tooltip title="发起会话" key="chat">
                <Button type="link" icon={<MessageOutlined />}
                  onClick={() => handleStartChat(f)} />
              </Tooltip>,
              <Tooltip title="设置备注" key="remark">
                <Button type="link" icon={<EditOutlined />}
                  onClick={() => { setEditingRemark(friendId); setRemarkValue(f.remark || '') }} />
              </Tooltip>,
              <Popconfirm
                key="block"
                title="拉黑此好友？"
                description="拉黑后将无法收到对方消息"
                onConfirm={() => handleBlock(friendId)}
              >
                <Button type="link" icon={<StopOutlined />} style={{ color: '#faad14' }} />
              </Popconfirm>,
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
                <Badge status={isOnline ? 'success' : 'default'}
                  offset={[-4, 32]}
                  title={isOnline ? '在线' : '离线'}>
                  <Avatar icon={<UserOutlined />} src={f.friend_avatar} size="large" />
                </Badge>
              }
              title={
                <Space>
                  <Text strong>{displayName}</Text>
                  {f.remark && <Tag color="blue" style={{ fontSize: 11 }}>备注</Tag>}
                  {f.online !== undefined && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {f.online ? '在线' : '离线'}
                    </Text>
                  )}
                </Space>
              }
              description={
                <Space size="small">
                  <Tag color="green">好友</Tag>
                  {f.friend_username && f.remark && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      用户名: {f.friend_username}
                    </Text>
                  )}
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
              <Space size="small" direction="vertical" style={{ gap: 0 }}>
                <Space size="small">
                  <Tag color="orange">等待确认</Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {new Date(req.created_at).toLocaleDateString()}
                  </Text>
                </Space>
                {req.message && (
                  <Text type="secondary" style={{ fontSize: 12, fontStyle: 'italic' }}>
                    "{req.message}"
                  </Text>
                )}
              </Space>
            }
          />
        </List.Item>
      )}
    />
  )

  const blocklistContent = (
    <List
      dataSource={blocklist}
      loading={loading}
      locale={{ emptyText: '黑名单为空' }}
      renderItem={(entry) => (
        <List.Item
          actions={[
            <Button key="unblock" size="small" icon={<UndoOutlined />}
              onClick={() => handleUnblock(entry.blocked_user_id)}>
              解除拉黑
            </Button>,
          ]}
        >
          <List.Item.Meta
            avatar={<Avatar icon={<StopOutlined />} size="large" style={{ backgroundColor: '#ff4d4f' }} />}
            title={
              <Space>
                <Text strong>{entry.blocked_username || `用户 ${entry.blocked_user_id}`}</Text>
                <Tag color="red">已拉黑</Tag>
              </Space>
            }
            description={
              <Space size="small">
                {entry.reason && <Text type="secondary">原因: {entry.reason}</Text>}
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {new Date(entry.created_at).toLocaleDateString()}
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
    {
      key: 'blocklist',
      label: (
        <Space>
          <StopOutlined />
          <span>黑名单</span>
          <Text type="secondary">({blocklist.length})</Text>
        </Space>
      ),
      children: blocklistContent,
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
          <Button icon={<ReloadOutlined />} onClick={() => { fetchFriends(); fetchPendingRequests(); fetchBlocklist() }}>
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
              {blocklist.length > 0 && `，${blocklist.length} 条拉黑`}
            </Text>
          }
        />
      </Card>

      {/* Add Friend Modal */}
      <Modal
        title={<span><UserAddOutlined style={{ marginRight: 8 }} />添加好友</span>}
        open={addFriendVisible}
        onOk={handleAddFriend}
        onCancel={() => { setAddFriendVisible(false); setAddFriendName(''); setAddFriendMsg('') }}
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
            style={{ marginBottom: 12 }}
          />
          <AntInput.TextArea
            placeholder="验证信息（选填）"
            value={addFriendMsg}
            onChange={(e) => setAddFriendMsg(e.target.value)}
            maxLength={200}
            rows={2}
          />
        </div>
      </Modal>

      {/* Edit Remark Modal */}
      <Modal
        title="设置备注"
        open={editingRemark !== null}
        onOk={() => handleSetRemark(editingRemark!)}
        onCancel={() => setEditingRemark(null)}
        okText="保存"
        cancelText="取消"
      >
        <Input
          placeholder="输入备注名称"
          value={remarkValue}
          onChange={(e) => setRemarkValue(e.target.value)}
          maxLength={50}
        />
      </Modal>
    </div>
  )
}
