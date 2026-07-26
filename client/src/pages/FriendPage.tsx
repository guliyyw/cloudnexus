import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Avatar,
  Badge,
  Button,
  Card,
  Input,
  List,
  Modal,
  Popconfirm,
  Space,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import {
  CheckOutlined,
  CloseOutlined,
  DeleteOutlined,
  EditOutlined,
  MessageOutlined,
  ReloadOutlined,
  SearchOutlined,
  StopOutlined,
  TeamOutlined,
  UndoOutlined,
  UserAddOutlined,
  UserOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons'
import { PageHeader, MetricStrip } from '../components/common/PageHeader'
import { useFriendStore } from '../stores/friendStore'
import { useChatStore } from '../stores/chatStore'
import { useAuthStore } from '../stores/authStore'
import { colors, radius, shadow } from '../theme/tokens'
import type { FriendRequest } from '../services/chat'

const { Text } = Typography

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

  const refreshAll = () => {
    fetchFriends()
    fetchPendingRequests()
    fetchBlocklist()
  }

  useEffect(() => {
    refreshAll()
  }, [])

  useEffect(() => {
    const timer = setInterval(() => {
      const store = useFriendStore.getState()
      if (store.friends.length > 0) store.fetchOnlineStatus()
    }, 30000)
    return () => clearInterval(timer)
  }, [])

  const getFriendDisplayId = (friend: FriendRequest): string => (
    friend.user_id === user?.id ? friend.friend_id : friend.user_id
  )

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
    } catch (error: any) {
      message.error(error?.response?.data?.message || '发送失败')
    } finally {
      setAdding(false)
    }
  }

  const handleStartChat = async (friend: FriendRequest) => {
    const friendId = getFriendDisplayId(friend)
    try {
      await createConv(friendId)
      message.success('会话已打开')
      navigate('/chat')
    } catch (error: any) {
      message.error(error?.response?.data?.message || '创建会话失败')
    }
  }

  const handleSetRemark = async (friendId: string) => {
    await setRemark(friendId, remarkValue)
    setEditingRemark(null)
    message.success('备注已保存')
  }

  const friendListContent = (
    <List
      dataSource={friends}
      loading={loading}
      grid={{ gutter: 14, xs: 1, sm: 1, md: 2, xl: 2, xxl: 3 }}
      locale={{ emptyText: '暂无好友，点击“添加好友”开始' }}
      renderItem={(friend) => {
        const friendId = getFriendDisplayId(friend)
        const isOnline = onlineStatus[friendId] === true
        const displayName = friend.remark || friend.friend_nickname || friend.friend_username || `用户 ${friendId}`
        return (
          <List.Item>
            <Card size="small" style={{ borderRadius: radius.md, borderColor: colors.borderSubtle, boxShadow: shadow.card }}>
              <List.Item.Meta
                avatar={
                  <Badge status={isOnline ? 'success' : 'default'} offset={[-4, 32]} title={isOnline ? '在线' : '离线'}>
                    <Avatar icon={<UserOutlined />} src={friend.friend_avatar} size="large" />
                  </Badge>
                }
                title={
                  <Space wrap>
                    <Text strong>{displayName}</Text>
                    {friend.remark && <Tag color="blue">备注</Tag>}
                    <Tag color={isOnline ? 'green' : 'default'}>{isOnline ? '在线' : '离线'}</Tag>
                  </Space>
                }
                description={
                  <Space direction="vertical" size={2}>
                    {friend.friend_username && friend.remark && <Text type="secondary">用户名：{friend.friend_username}</Text>}
                    <Text type="secondary">添加于 {new Date(friend.created_at).toLocaleDateString()}</Text>
                  </Space>
                }
              />
              <Space wrap style={{ marginTop: 14 }}>
                <Tooltip title="打开聊天">
                  <Button size="small" type="primary" icon={<MessageOutlined />} onClick={() => handleStartChat(friend)}>聊天</Button>
                </Tooltip>
                <Tooltip title="设置备注名">
                  <Button size="small" icon={<EditOutlined />} onClick={() => { setEditingRemark(friendId); setRemarkValue(friend.remark || '') }}>备注</Button>
                </Tooltip>
                <Popconfirm title="拉黑此好友？" description="拉黑后将无法收到对方消息" onConfirm={() => blockUser(friendId).then(() => message.success('已拉黑'))}>
                  <Button size="small" icon={<StopOutlined />} style={{ color: colors.warning }}>拉黑</Button>
                </Popconfirm>
                <Popconfirm title="确定删除该好友？" onConfirm={() => removeFriend(friendId).then(() => message.success('已删除好友'))}>
                  <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
                </Popconfirm>
              </Space>
            </Card>
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
      renderItem={(request) => (
        <List.Item
          actions={[
            <Button key="accept" type="primary" size="small" icon={<CheckOutlined />} onClick={() => acceptRequest(request.id).then(() => message.success('已接受好友请求'))}>接受</Button>,
            <Button key="reject" size="small" danger icon={<CloseOutlined />} onClick={() => rejectRequest(request.id).then(() => message.success('已拒绝'))}>拒绝</Button>,
          ]}
        >
          <List.Item.Meta
            avatar={<Avatar icon={<UserOutlined />} size="large" />}
            title={<Text strong>{request.friend_username || `用户 ${request.user_id}`}</Text>}
            description={
              <Space direction="vertical" size={2}>
                <Space>
                  <Tag color="orange">等待确认</Tag>
                  <Text type="secondary">{new Date(request.created_at).toLocaleDateString()}</Text>
                </Space>
                {request.message && <Text type="secondary">“{request.message}”</Text>}
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
            <Button key="unblock" size="small" icon={<UndoOutlined />} onClick={() => unblockUser(entry.blocked_user_id).then(() => message.success('已取消拉黑'))}>解除拉黑</Button>,
          ]}
        >
          <List.Item.Meta
            avatar={<Avatar icon={<StopOutlined />} size="large" style={{ backgroundColor: colors.error }} />}
            title={<Space><Text strong>{entry.blocked_username || `用户 ${entry.blocked_user_id}`}</Text><Tag color="red">已拉黑</Tag></Space>}
            description={<Text type="secondary">{new Date(entry.created_at).toLocaleDateString()}</Text>}
          />
        </List.Item>
      )}
    />
  )

  return (
    <div>
      <PageHeader
        eyebrow="Friends"
        title="好友"
        description="管理联系人、好友请求和黑名单。"
        actions={
          <>
            <Button icon={<ReloadOutlined />} onClick={refreshAll}>刷新</Button>
            <Button type="primary" icon={<UserAddOutlined />} onClick={() => setAddFriendVisible(true)}>添加好友</Button>
          </>
        }
      />

      <MetricStrip
        items={[
          { label: '好友', value: friends.length, tone: 'primary' },
          { label: '待处理', value: pendingRequests.length, tone: pendingRequests.length ? 'warning' : 'default' },
          { label: '黑名单', value: blocklist.length },
        ]}
      />

      <Card style={{ minHeight: 420, borderRadius: radius.lg, borderColor: colors.borderSubtle, boxShadow: shadow.card }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            { key: 'friends', label: <Space><TeamOutlined />好友列表<Text type="secondary">({friends.length})</Text></Space>, children: friendListContent },
            { key: 'requests', label: <Space><UserSwitchOutlined />好友请求{pendingRequests.length > 0 && <Badge count={pendingRequests.length} size="small" />}</Space>, children: pendingContent },
            { key: 'blocklist', label: <Space><StopOutlined />黑名单<Text type="secondary">({blocklist.length})</Text></Space>, children: blocklistContent },
          ]}
        />
      </Card>

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
          <Input
            size="large"
            placeholder="输入对方用户名"
            prefix={<SearchOutlined />}
            value={addFriendName}
            onChange={(event) => setAddFriendName(event.target.value)}
            onPressEnter={handleAddFriend}
            autoFocus
            style={{ marginBottom: 12 }}
          />
          <Input.TextArea
            placeholder="验证信息（选填）"
            value={addFriendMsg}
            onChange={(event) => setAddFriendMsg(event.target.value)}
            maxLength={200}
            rows={2}
          />
        </div>
      </Modal>

      <Modal
        title="设置备注"
        open={editingRemark !== null}
        onOk={() => handleSetRemark(editingRemark!)}
        onCancel={() => setEditingRemark(null)}
        okText="保存"
        cancelText="取消"
      >
        <Input placeholder="输入备注名称" value={remarkValue} onChange={(event) => setRemarkValue(event.target.value)} maxLength={50} />
      </Modal>
    </div>
  )
}
