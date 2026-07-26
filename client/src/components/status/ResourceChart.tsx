import { useMemo } from 'react'
import { Empty, Progress, Statistic, Typography } from 'antd'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { colors, chart as chartTokens, radius, spacing } from '../../theme/tokens'
import type { ResourcePoint } from '../../services/status'

const { Text } = Typography

interface Props {
  services: Record<string, ResourcePoint[]>
  serviceFilter: string
}

function formatMemoryFromBytes(bytes: number): string {
  const mb = bytes / 1024 / 1024
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${mb.toFixed(0)} MB`
}

function formatMemoryMB(mb: number): string {
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${mb.toFixed(0)} MB`
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return ts
  }
}

function memoryPercent(point?: ResourcePoint): number {
  if (!point?.memory_used || !point.memory_total) return 0
  return Number(((point.memory_used / point.memory_total) * 100).toFixed(1))
}

export default function ResourceChart({ services, serviceFilter }: Props) {
  const entries = useMemo(() => Object.entries(services), [services])

  const filteredEntries = useMemo(() => {
    if (serviceFilter === 'all' || !serviceFilter) return entries
    return entries.filter(([name]) => name === serviceFilter)
  }, [entries, serviceFilter])

  const latestStats = useMemo(() => (
    filteredEntries
      .map(([name, points]) => ({ name, latest: points[points.length - 1] }))
      .filter((item) => item.latest)
  ), [filteredEntries])

  if (!filteredEntries.length) {
    return <Empty description="暂无资源数据" />
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.lg }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: spacing.md }}>
        {latestStats.map(({ name, latest }) => {
          const cpu = Number(latest.cpu_percent?.toFixed(1) ?? 0)
          const memPct = memoryPercent(latest)
          return (
            <div
              key={name}
              style={{
                border: `1px solid ${colors.borderSubtle}`,
                borderRadius: radius.md,
                background: colors.surface,
                padding: spacing.md,
              }}
            >
              <Text strong style={{ display: 'block', marginBottom: 12 }}>{name}</Text>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: spacing.sm }}>
                <div>
                  <Statistic title="CPU" value={cpu} suffix="%" precision={1} valueStyle={{ fontSize: 22, color: colors.text }} />
                  <Progress percent={Math.min(cpu, 100)} size="small" showInfo={false} status={cpu > 80 ? 'exception' : 'normal'} />
                </div>
                <div>
                  <Statistic title="内存" value={memPct} suffix="%" precision={1} valueStyle={{ fontSize: 22, color: colors.text }} />
                  <Progress percent={Math.min(memPct, 100)} size="small" showInfo={false} status={memPct > 80 ? 'exception' : 'normal'} />
                </div>
              </div>
              <Text type="secondary" style={{ display: 'block', marginTop: 10, fontSize: 12 }}>
                {formatMemoryFromBytes(latest.memory_used)} / {formatMemoryFromBytes(latest.memory_total)} · {formatTime(latest.timestamp)}
              </Text>
            </div>
          )
        })}
      </div>

      {filteredEntries.map(([svcName, points]) => {
        const step = Math.max(1, Math.floor(points.length / 200))
        const sampled = points.filter((_, i) => i % step === 0)

        const chartData = sampled.map((p) => ({
          time: formatTime(p.timestamp),
          cpu: Number(p.cpu_percent?.toFixed(1) ?? 0),
          memory: memoryPercent(p),
          memoryMB: p.memory_used ? Number((p.memory_used / 1024 / 1024).toFixed(1)) : 0,
        }))

        return (
          <div key={svcName}>
            <Text strong style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>
              {svcName}
            </Text>
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={chartData} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartTokens.gridStroke} />
                <XAxis dataKey="time" tick={{ fontSize: 10, fill: chartTokens.tickFill }} interval="preserveStartEnd" />
                <YAxis yAxisId="left" tick={{ fontSize: 10, fill: chartTokens.tickFill }} domain={[0, 100]} unit="%" width={40} />
                <YAxis
                  yAxisId="right"
                  orientation="right"
                  tick={{ fontSize: 10, fill: chartTokens.tickFill }}
                  domain={[0, 'auto']}
                  tickFormatter={(v: number) => formatMemoryMB(v)}
                  width={60}
                />
                <Tooltip
                  contentStyle={{ fontSize: chartTokens.tooltip.fontSize, borderRadius: chartTokens.tooltip.borderRadius, background: chartTokens.tooltip.background, border: chartTokens.tooltip.border, color: chartTokens.tooltip.color }}
                  formatter={(value: any, name: any) => {
                    const v = typeof value === 'number' ? value : Number(value || 0)
                    const n = String(name || '')
                    if (n === 'memory') return [`${v}%`, '内存']
                    if (n === 'cpu') return [`${v}%`, 'CPU']
                    if (n === 'memoryMB') return [formatMemoryMB(v), '内存占用']
                    return [v, n]
                  }}
                />
                <Legend
                  wrapperStyle={{ fontSize: 12 }}
                  formatter={(value: string) => value === 'cpu' ? 'CPU %' : value === 'memory' ? '内存 %' : value === 'memoryMB' ? '内存占用' : value}
                />
                <Line yAxisId="left" type="monotone" dataKey="cpu" stroke={colors.primary} strokeWidth={2} dot={false} name="cpu" />
                <Line yAxisId="left" type="monotone" dataKey="memory" stroke="#52c41a" strokeWidth={2} dot={false} name="memory" />
                <Line yAxisId="right" type="monotone" dataKey="memoryMB" stroke="#1677ff" strokeWidth={1.5} dot={false} name="memoryMB" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )
      })}
    </div>
  )
}
