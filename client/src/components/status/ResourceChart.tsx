import { useMemo } from 'react'
import { Typography, Empty } from 'antd'
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
import { colors } from '../../theme/tokens'
import type { ResourcePoint } from '../../services/status'

const { Text } = Typography

interface Props {
  services: Record<string, ResourcePoint[]>
  serviceFilter: string
}

function formatMemory(mb: number): string {
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${mb.toFixed(0)} MB`
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ts
  }
}

export default function ResourceChart({ services, serviceFilter }: Props) {
  const entries = useMemo(() => Object.entries(services), [services])

  const filteredEntries = useMemo(() => {
    if (serviceFilter === 'all' || !serviceFilter) return entries
    return entries.filter(([name]) => name === serviceFilter)
  }, [entries, serviceFilter])

  if (!filteredEntries.length) {
    return <Empty description="暂无资源数据" />
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {filteredEntries.map(([svcName, points]) => {
        // Downsample to ~200 points max for performance
        const step = Math.max(1, Math.floor(points.length / 200))
        const sampled = points.filter((_, i) => i % step === 0)

        const chartData = sampled.map((p) => ({
          time: formatTime(p.timestamp),
          cpu: Number(p.cpu_percent?.toFixed(1) ?? 0),
          memory: p.memory_used ? Number(((p.memory_used / 1024 / 1024) * 100) / (p.memory_total / 1024 / 1024)).toFixed(1) : 0,
          memoryMB: p.memory_used ? Number(p.memory_used / 1024 / 1024) : 0,
        }))

        return (
          <div key={svcName}>
            <Text strong style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>
              {svcName}
            </Text>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartData} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis
                  yAxisId="left"
                  tick={{ fontSize: 10 }}
                  domain={[0, 100]}
                  unit="%"
                  width={40}
                />
                <YAxis
                  yAxisId="right"
                  orientation="right"
                  tick={{ fontSize: 10 }}
                  domain={[0, 'auto']}
                  tickFormatter={(v: number) => formatMemory(v)}
                  width={60}
                />
                <Tooltip
                  contentStyle={{ fontSize: 12, borderRadius: 8 }}
                  formatter={(value: any, name: any) => {
                    const v = typeof value === 'number' ? value : Number(value || 0)
                    const n = String(name || '')
                    if (n === 'memory') return [`${v}%`, '内存']
                    if (n === 'cpu') return [`${v}%`, 'CPU']
                    return [v, n]
                  }}
                />
                <Legend
                  wrapperStyle={{ fontSize: 12 }}
                  formatter={(value: string) => value === 'cpu' ? 'CPU %' : value === 'memory' ? '内存 %' : value}
                />
                <Line
                  yAxisId="left"
                  type="monotone"
                  dataKey="cpu"
                  stroke={colors.primary}
                  strokeWidth={2}
                  dot={false}
                  name="cpu"
                />
                <Line
                  yAxisId="left"
                  type="monotone"
                  dataKey="memory"
                  stroke="#52c41a"
                  strokeWidth={2}
                  dot={false}
                  name="memory"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )
      })}
    </div>
  )
}
