import { useEffect, useState, useCallback, useRef } from 'react'
import { Button, Typography, Spin, Card, Row, Col } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import * as adminApi from '../../services/admin'
import type { MetricSnapshot } from '../../services/admin'
import { colors, chart as chartTokens } from '../../theme/tokens'

const { Text } = Typography

export default function HistoricalMetrics() {
  const [snapshots, setSnapshots] = useState<MetricSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const intRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchHistory = useCallback(async () => {
    try {
      const res = await adminApi.getMetricsHistory(60)
      setSnapshots(res.snapshots || [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    setLoading(true)
    fetchHistory()
    intRef.current = setInterval(fetchHistory, 10000)
    return () => { if (intRef.current) clearInterval(intRef.current) }
  }, [fetchHistory])

  const chartData = snapshots.map((s) => ({
    ...s,
    time: new Date(s.timestamp).toLocaleTimeString(),
  }))

  return (
    <Spin spinning={loading}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 16 }}>历史指标 (最近 10 分钟)</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchHistory}>刷新</Button>
      </div>

      <Card title="CPU 使用率 (%)" size="small" style={{ marginBottom: 16 }}>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke={chartTokens.gridStroke} />
            <XAxis dataKey="time" fontSize={11} tick={{ fill: chartTokens.tickFill }} />
            <YAxis domain={[0, 100]} fontSize={11} tick={{ fill: chartTokens.tickFill }} />
            <Tooltip contentStyle={{ fontSize: chartTokens.tooltip.fontSize, borderRadius: chartTokens.tooltip.borderRadius, background: chartTokens.tooltip.background, border: chartTokens.tooltip.border, color: chartTokens.tooltip.color }} />
            <Line type="monotone" dataKey="cpu_percent" stroke={colors.primary} dot={false} strokeWidth={2} />
          </LineChart>
        </ResponsiveContainer>
      </Card>

      <Row gutter={16}>
        <Col span={12}>
          <Card title="内存使用率 (%)" size="small">
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartTokens.gridStroke} />
                <XAxis dataKey="time" fontSize={11} tick={{ fill: chartTokens.tickFill }} />
                <YAxis domain={[0, 100]} fontSize={11} tick={{ fill: chartTokens.tickFill }} />
                <Tooltip contentStyle={{ fontSize: chartTokens.tooltip.fontSize, borderRadius: chartTokens.tooltip.borderRadius, background: chartTokens.tooltip.background, border: chartTokens.tooltip.border, color: chartTokens.tooltip.color }} />
                <Line type="monotone" dataKey="mem_percent" stroke="#5ac8d8" dot={false} strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="Goroutines / 堆内存 (MB)" size="small">
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartTokens.gridStroke} />
                <XAxis dataKey="time" fontSize={11} tick={{ fill: chartTokens.tickFill }} />
                <YAxis yAxisId="left" fontSize={11} tick={{ fill: chartTokens.tickFill }} />
                <YAxis yAxisId="right" orientation="right" fontSize={11} tick={{ fill: chartTokens.tickFill }} />
                <Tooltip contentStyle={{ fontSize: chartTokens.tooltip.fontSize, borderRadius: chartTokens.tooltip.borderRadius, background: chartTokens.tooltip.background, border: chartTokens.tooltip.border, color: chartTokens.tooltip.color }} />
                <Legend />
                <Line yAxisId="left" type="monotone" dataKey="goroutines" stroke="#52c41a" dot={false} strokeWidth={2} name="Goroutines" />
                <Line yAxisId="right" type="monotone" dataKey="heap_alloc_mb" stroke="#faad14" dot={false} strokeWidth={2} name="堆内存(MB)" />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </Spin>
  )
}
