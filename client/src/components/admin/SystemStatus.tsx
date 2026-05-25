import { useEffect, useState, useCallback, useRef } from 'react'
import { Button, Typography, Spin, Card, Statistic, Row, Col, Descriptions, Progress } from 'antd'
import { ReloadOutlined, CloudServerOutlined } from '@ant-design/icons'
import * as adminApi from '../../services/admin'
import type { SystemMetrics, ResourceMetrics } from '../../services/admin'

const { Text } = Typography

export default function SystemStatus() {
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null)
  const [resMetrics, setResMetrics] = useState<ResourceMetrics | null>(null)
  const [loading, setLoading] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchMetrics = useCallback(async () => {
    try {
      const [m, rm] = await Promise.all([
        adminApi.getMetrics(),
        adminApi.getResourceMetrics(),
      ])
      setMetrics(m)
      setResMetrics(rm)
    } catch {
      // resource metrics may not be available
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    setLoading(true)
    fetchMetrics()
    intervalRef.current = setInterval(fetchMetrics, 5000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [fetchMetrics])

  const formatUptime = (seconds: number) => {
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = seconds % 60
    const parts = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    if (m > 0) parts.push(`${m}m`)
    parts.push(`${s}s`)
    return parts.join(' ')
  }

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes < 0) return '—'
    if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB/s`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB/s`
    return `${bytes} B/s`
  }

  return (
    <Spin spinning={loading}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 16 }}>系统状态</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchMetrics}>刷新</Button>
      </div>

      {metrics && (
        <>
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}>
              <Card><Statistic title="运行时间" value={formatUptime(metrics.uptime_seconds)} /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="Goroutines" value={metrics.goroutines} /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="堆内存" value={metrics.heap_alloc_mb} suffix="MB" precision={1} /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="GC 次数" value={metrics.num_gc} /></Card>
            </Col>
          </Row>

          <Descriptions bordered size="small" column={2} style={{ marginBottom: 24 }}>
            <Descriptions.Item label="Go 版本">{metrics.go_version}</Descriptions.Item>
            <Descriptions.Item label="CPU 核心">{metrics.num_cpu}</Descriptions.Item>
            <Descriptions.Item label="堆系统内存">{metrics.heap_sys_mb} MB</Descriptions.Item>
            <Descriptions.Item label="栈内存">{metrics.stack_inuse_kb} KB</Descriptions.Item>
          </Descriptions>
        </>
      )}

      {resMetrics && (
        <>
          <div style={{ marginBottom: 12 }}>
            <Text strong style={{ fontSize: 14 }}><CloudServerOutlined /> 服务器资源</Text>
          </div>

          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}>
              <Card size="small">
                <Statistic title="CPU 使用率" value={resMetrics.cpu_percent} suffix="%" precision={1} />
                <Progress percent={resMetrics.cpu_percent} size="small" status={resMetrics.cpu_percent > 80 ? 'exception' : 'normal'} showInfo={false} />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title="内存" value={resMetrics.mem_percent} suffix="%" precision={1} />
                <Progress percent={resMetrics.mem_percent} size="small" status={resMetrics.mem_percent > 80 ? 'exception' : 'normal'} showInfo={false} />
                <Text type="secondary" style={{ fontSize: 11 }}>{resMetrics.mem_used_mb} / {resMetrics.mem_total_mb} MB</Text>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title={`磁盘 ${resMetrics.disk_path}`} value={resMetrics.disk_percent} suffix="%" precision={1} />
                <Progress percent={resMetrics.disk_percent} size="small" status={resMetrics.disk_percent > 80 ? 'exception' : 'normal'} showInfo={false} />
                <Text type="secondary" style={{ fontSize: 11 }}>{resMetrics.disk_used_mb} / {resMetrics.disk_total_mb} MB</Text>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title="网络 接收" value={formatBytes(resMetrics.net_bytes_recv)} />
                <Text type="secondary" style={{ fontSize: 11 }}>发送: {formatBytes(resMetrics.net_bytes_sent)}</Text>
              </Card>
            </Col>
          </Row>
        </>
      )}
    </Spin>
  )
}
