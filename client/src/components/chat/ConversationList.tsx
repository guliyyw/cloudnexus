import { List, Button, Card, Typography, Avatar, Popconfirm, Space, Badge } from 'antd'
import {
  PlusOutlined, UserOutlined, DeleteOutlined,
  TeamOutlined, UsergroupAddOutlined,
} from '@ant-design/icons'
import type { Conversation } from '../../services/chat'

const { Text } = Typography

interface ConversationListProps {
  conversations: Conversation[]
  currentConvId: string | null
  loading: boolean
  onSelectConv: (id: string) => void
  onDeleteConv: (id: string) => void
  onCreateGroup: () => void
  onAddFriend: () => void
  onNavigateFriends: () => void
}

export default function ConversationList({
  conversations,
  currentConvId,
  loading,
  onSelectConv,
  onDeleteConv,
  onCreateGroup,
  onAddFriend,
  onNavigateFriends,
}: ConversationListProps) {
  return (
    <Card
      title="会话"
      style={{ width: 280, height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { flex: 1, overflow: 'auto', padding: 0 } }}
      extra={
        <Space size={4}>
          <Button type="text" icon={<UsergroupAddOutlined />} title="创建群聊"
            onClick={onCreateGroup} />
          <Button type="text" icon={<TeamOutlined />} title="好友管理"
            onClick={onNavigateFriends} />
          <Button type="text" icon={<PlusOutlined />}
            onClick={onAddFriend} />
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
              background: currentConvId === conv.id ? 'rgba(129,236,254,0.08)' : undefined,
            }}
            onClick={() => onSelectConv(conv.id)}
          >
            <List.Item.Meta
              avatar={
                <Badge count={conv.unread} size="small" offset={[-4, 4]}>
                  <Avatar icon={conv.type === 'group' ? <TeamOutlined /> : <UserOutlined />} />
                </Badge>
              }
              title={conv.name || `会话 ${conv.id}`}
              description={<Text type="secondary" ellipsis>{conv.last_message || (conv.type === 'private' ? '私聊' : '群聊')}</Text>}
            />
            <Popconfirm
              title="确定删除该会话？"
              onConfirm={(e) => {
                e?.stopPropagation()
                onDeleteConv(conv.id)
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
  )
}
