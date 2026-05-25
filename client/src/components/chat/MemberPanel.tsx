import { Card, List, Button, Space, Avatar, Tag } from 'antd'
import {
  TeamOutlined, UserOutlined, UserAddOutlined,
  UserDeleteOutlined, LogoutOutlined, CrownOutlined,
} from '@ant-design/icons'
import type { GroupMember } from '../../services/chat'

interface MemberPanelProps {
  members: GroupMember[]
  userId: string | undefined
  isOwner: boolean
  onRemoveMember: (userId: string) => void
  onAddMemberClick: () => void
  onLeaveGroup: () => void
}

export default function MemberPanel({
  members,
  userId,
  isOwner,
  onRemoveMember,
  onAddMemberClick,
  onLeaveGroup,
}: MemberPanelProps) {
  return (
    <Card
      title={<span><TeamOutlined /> 成员 ({members.length})</span>}
      style={{ width: 220, display: 'flex', flexDirection: 'column' }}
      styles={{ body: { flex: 1, overflow: 'auto', padding: 0 } }}
      extra={
        <Button type="text" size="small" icon={<UserAddOutlined />}
          onClick={onAddMemberClick} />
      }
    >
      <List
        dataSource={members}
        size="small"
        renderItem={(m: GroupMember) => (
          <List.Item
            style={{ padding: '6px 12px' }}
            actions={isOwner && m.user_id !== userId ? [
              <Button key="remove" type="text" size="small" danger icon={<UserDeleteOutlined />}
                onClick={() => onRemoveMember(m.user_id)} />
            ] : []}
          >
            <Space>
              <Avatar icon={<UserOutlined />} size="small" />
              <span>{m.user_id === userId ? '我' : `用户 ${m.user_id}`}</span>
              {m.role === 'owner' && <Tag color="gold" style={{ margin: 0, fontSize: 10 }}><CrownOutlined /></Tag>}
            </Space>
          </List.Item>
        )}
      />
      <div style={{ padding: 8, borderTop: '1px solid rgba(255,255,255,0.06)' }}>
        <Button type="text" danger icon={<LogoutOutlined />} block onClick={onLeaveGroup}>
          退出群聊
        </Button>
      </div>
    </Card>
  )
}
