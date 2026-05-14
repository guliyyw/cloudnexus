import { Typography } from 'antd'
import { colors, radius, shadow } from '../../theme/tokens'

const { Text } = Typography

interface Props {
  icon: React.ReactNode
  name: string
  status: 'green' | 'yellow' | 'red'
  detail: string
  onClick: () => void
}

const statusColor: Record<string, string> = {
  green: colors.statusGreen,
  yellow: colors.statusYellow,
  red: colors.statusRed,
}

export default function ModuleCard({ icon, name, status, detail, onClick }: Props) {
  return (
    <div
      onClick={onClick}
      style={{
        background: colors.bgCard,
        borderRadius: radius.lg,
        padding: '20px 24px',
        border: '1px solid #f0eeeb',
        cursor: 'pointer',
        transition: 'box-shadow 0.25s ease, transform 0.15s ease',
        boxShadow: shadow.card,
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        userSelect: 'none',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = shadow.hover
        e.currentTarget.style.transform = 'translateY(-2px)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = shadow.card
        e.currentTarget.style.transform = 'translateY(0)'
      }}
    >
      <div style={{ fontSize: 32, color: colors.primary, flexShrink: 0 }}>{icon}</div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 2, color: colors.text }}>{name}</div>
        <Text type="secondary" style={{ fontSize: 12 }} ellipsis>{detail}</Text>
      </div>
      <div style={{ flexShrink: 0 }}>
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: '50%',
            backgroundColor: statusColor[status],
            animation: status !== 'green' ? 'pulse 1.5s ease-in-out infinite' : undefined,
          }}
        />
      </div>
      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
      `}</style>
    </div>
  )
}
