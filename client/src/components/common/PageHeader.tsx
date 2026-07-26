import type { ReactNode } from 'react'
import { Space, Typography } from 'antd'
import { colors, radius, shadow, spacing } from '../../theme/tokens'

const { Text, Title } = Typography

interface PageHeaderProps {
  eyebrow?: string
  title: string
  description?: string
  actions?: ReactNode
}

interface MetricItem {
  label: string
  value: ReactNode
  tone?: 'primary' | 'success' | 'warning' | 'default'
}

interface MetricStripProps {
  items: MetricItem[]
}

const toneColors: Record<NonNullable<MetricItem['tone']>, string> = {
  primary: colors.primary,
  success: colors.success,
  warning: colors.warning,
  default: colors.text,
}

export function PageHeader({ eyebrow, title, description, actions }: PageHeaderProps) {
  return (
    <section
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
        gap: spacing.lg,
        marginBottom: spacing.lg,
        flexWrap: 'wrap',
      }}
    >
      <div style={{ minWidth: 240, maxWidth: 760 }}>
        {eyebrow && (
          <Text style={{ color: colors.primary, fontSize: 12, fontWeight: 700, letterSpacing: 0.5, textTransform: 'uppercase' }}>
            {eyebrow}
          </Text>
        )}
        <Title level={3} style={{ margin: eyebrow ? '6px 0 8px' : '0 0 8px', color: colors.text }}>
          {title}
        </Title>
        {description && <Text style={{ color: colors.textSecondary, lineHeight: 1.7 }}>{description}</Text>}
      </div>
      {actions && (
        <Space wrap size={spacing.sm} style={{ justifyContent: 'flex-end' }}>
          {actions}
        </Space>
      )}
    </section>
  )
}

export function MetricStrip({ items }: MetricStripProps) {
  return (
    <section
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
        gap: spacing.sm,
        marginBottom: spacing.lg,
      }}
    >
      {items.map((item) => (
        <div
          key={item.label}
          style={{
            padding: '14px 16px',
            borderRadius: radius.md,
            background: colors.surface,
            border: `1px solid ${colors.borderSubtle}`,
            boxShadow: shadow.card,
          }}
        >
          <div style={{ color: colors.textSecondary, fontSize: 12, marginBottom: 8 }}>{item.label}</div>
          <div style={{ color: toneColors[item.tone || 'default'], fontWeight: 700, fontSize: 22, lineHeight: 1.2 }}>
            {item.value}
          </div>
        </div>
      ))}
    </section>
  )
}
