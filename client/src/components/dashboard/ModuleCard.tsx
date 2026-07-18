import { useState } from 'react'
import { Typography } from 'antd'
import { ArrowRightOutlined } from '@ant-design/icons'
import { colors, motion, radius, shadow, spacing } from '../../theme/tokens'

const { Text } = Typography

interface Props {
  icon: React.ReactNode
  name: string
  status: 'green' | 'yellow' | 'red'
  detail: string
  onClick: () => void
  eyebrow?: string
  metric?: string
}

const statusColor: Record<Props['status'], string> = {
  green: colors.statusGreen,
  yellow: colors.statusYellow,
  red: colors.statusRed,
}

const statusLabel: Record<Props['status'], string> = {
  green: '正常',
  yellow: '告警',
  red: '异常',
}

export default function ModuleCard({ icon, name, status, detail, onClick, eyebrow, metric }: Props) {
  const [hovered, setHovered] = useState(false)

  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: '100%',
        padding: 0,
        border: `1px solid ${hovered ? colors.borderStrong : colors.borderSubtle}`,
        borderRadius: radius.lg,
        background: hovered ? colors.surfaceRaised : colors.surface,
        cursor: 'pointer',
        transition: `transform ${motion.fast} ease, box-shadow ${motion.normal} ease, border-color ${motion.normal} ease`,
        boxShadow: hovered ? shadow.hover : shadow.card,
        transform: hovered ? 'translateY(-3px)' : 'translateY(0)',
        userSelect: 'none',
        textAlign: 'left',
        color: colors.text,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'stretch',
          gap: spacing.md,
          padding: '22px 24px',
        }}
      >
        <div
          style={{
            width: 56,
            height: 56,
            borderRadius: radius.md,
            background: colors.primaryLight,
            color: colors.primary,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 28,
            flexShrink: 0,
          }}
        >
          {icon}
        </div>

        <div style={{ flex: 1, minWidth: 0 }}>
          {eyebrow && (
            <div
              style={{
                fontSize: 11,
                fontWeight: 600,
                color: colors.textSecondary,
                marginBottom: 8,
                letterSpacing: 0.5,
                textTransform: 'uppercase',
              }}
            >
              {eyebrow}
            </div>
          )}

          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: spacing.md }}>
            <div style={{ minWidth: 0 }}>
              <div style={{ fontWeight: 700, fontSize: 17, marginBottom: 6, color: colors.text }}>{name}</div>
              <Text style={{ fontSize: 13, lineHeight: 1.6, color: colors.textSecondary }}>
                {detail}
              </Text>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 8, flexShrink: 0 }}>
              <span
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '6px 10px',
                  borderRadius: 999,
                  background: colors.surfaceMuted,
                  color: colors.textSecondary,
                  fontSize: 12,
                  fontWeight: 600,
                }}
              >
                <span
                  style={{
                    width: 10,
                    height: 10,
                    borderRadius: '50%',
                    backgroundColor: statusColor[status],
                    boxShadow: `0 0 0 4px ${colors.surfaceMuted}`,
                  }}
                />
                {statusLabel[status]}
              </span>
              {metric && (
                <span style={{ fontSize: 12, color: colors.textSecondary }}>
                  {metric}
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '0 24px 18px',
          color: colors.textSecondary,
          fontSize: 12,
        }}
      >
        <span>点击进入模块</span>
        <ArrowRightOutlined style={{ color: colors.primary, transform: hovered ? 'translateX(2px)' : 'translateX(0)', transition: `transform ${motion.fast} ease` }} />
      </div>
    </button>
  )
}
