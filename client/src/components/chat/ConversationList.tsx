import { List, Button, Card, Typography, Avatar, Popconfirm, Space, Badge, Input } from 'antd'
import {
  PlusOutlined, UserOutlined, DeleteOutlined,
  TeamOutlined, UsergroupAddOutlined,
} from '@ant-design/icons'
import type { Conversation, MessageSearchResult } from '../../services/chat'

const { Text } = Typography

interface ConversationListProps {
  conversations: Conversation[]
  currentConvId: string | null
  loading: boolean
  searchKeyword: string
  searchLoading: boolean
  searchResults: MessageSearchResult[]
  onSearchChange: (keyword: string) => void
  onJumpToMessage: (conversationId: string, messageId: string) => void
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
  searchKeyword,
  searchLoading,
  searchResults,
  onSearchChange,
  onJumpToMessage,
  onSelectConv,
  onDeleteConv,
  onCreateGroup,
  onAddFriend,
  onNavigateFriends,
}: ConversationListProps) {
  const showSearchResults = searchKeyword.trim().length > 0

  return (
    <Card
      title="会话"
      style={{ width: 320, height: '100%', display: 'flex', flexDirection: 'column' }}
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
      <div style={{ padding: 12, paddingBottom: 8 }}>
        <Input.Search
          allowClear
          placeholder="搜索消息"
          value={searchKeyword}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>

      {showSearchResults ? (
        <List
          dataSource={searchResults}
          loading={searchLoading}
          locale={{ emptyText: '没有找到相关消息' }}
          renderItem={(item) => (
            <List.Item
              style={{ cursor: 'pointer', padding: '10px 12px', alignItems: 'flex-start' }}
              onClick={() => onJumpToMessage(item.conversation_id, item.id)}
            >
              <List.Item.Meta
                avatar={<Avatar size="small" icon={item.conversation_type === 'group' ? <TeamOutlined /> : <UserOutlined />} />}
                title={
                  <Space direction="vertical" size={0} style={{ width: '100%' }}>
                    <Text strong ellipsis>{item.conversation_name || `会话 ${item.conversation_id}`}</Text>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {item.sender_name || `用户 ${item.sender_id}`} · {new Date(item.created_at).toLocaleString()}
                    </Text>
                  </Space>
                }
                description={
                  <Text type="secondary" style={{ fontSize: 12 }} ellipsis={{ tooltip: item.content }}>
                    {item.content}
                  </Text>
                }
              />
            </List.Item>
          )}
        />
      ) : (
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
      )}
    </Card>
  )
}
