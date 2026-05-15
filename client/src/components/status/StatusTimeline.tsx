import { Timeline, Tag, Typography } from 'antd'
import { ClockCircleOutlined } from '@ant-design/icons'
import { colors } from '../../theme/tokens'
import type { ParsedSnapshot } from '../../stores/statusStore'

const { Text } = Typography

const statusColorMap: Record<string, string> = {
  healthy: colors.statusGreen,
  warning: colors.statusYellow,
  error: colors.statusRed,
  unresponsive: colors.statusYellow,
  offline: colors.statusRed,
  green: colors.statusGreen,
  yellow: colors.statusYellow,
  red: colors.statusRed,
}

interface Props {
  snapshots: ParsedSnapshot[]
}

export default function StatusTimeline({ snapshots }: Props) {
  if (!snapshots.length) {
    return <Text type="secondary">暂无历史数据</Text>
  }

  return (
    <Timeline
      items={snapshots.map((snap) => ({
        dot: <ClockCircleOutlined style={{ fontSize: 14, color: colors.primary }} />,
        children: (
          <div>
            <Text strong style={{ fontSize: 12 }}>
              {new Date(snap.timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
            </Text>
            <div style={{ marginTop: 4, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
              {snap.modules.length === 0 ? (
                <Text type="secondary" style={{ fontSize: 12 }}>无模块数据</Text>
              ) : (
                snap.modules.map((mod) => (
                  <Tag
                    key={mod.name}
                    color={statusColorMap[mod.status] || '#d9d9d9'}
                    style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}
                  >
                    {mod.name}
                  </Tag>
                ))
              )}
            </div>
          </div>
        ),
      }))}
    />
  )
}
