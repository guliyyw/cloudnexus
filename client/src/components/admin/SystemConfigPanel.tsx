import { useEffect, useState } from 'react'
import { Typography, Spin, Card, Row, Col, Button, Space, InputNumber, Switch, message } from 'antd'
import { SettingOutlined, CloudUploadOutlined } from '@ant-design/icons'
import * as adminApi from '../../services/admin'

const { Text } = Typography

export default function SystemConfigPanel() {
  const [loading, setLoading] = useState(false)
  const [seqMode, setSeqMode] = useState(false)
  const [maxConcurrent, setMaxConcurrent] = useState(3)
  const [saving, setSaving] = useState(false)

  const fetchConfigs = async () => {
    setLoading(true)
    try {
      const list = await adminApi.getSystemConfig()
      const seq = list.find((c) => c.key === 'upload.sequential_mode')
      const max = list.find((c) => c.key === 'upload.max_concurrent_chunks')
      setSeqMode(seq?.value === 'true')
      setMaxConcurrent(parseInt(max?.value || '3', 10))
    } catch { /* ignore */ } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchConfigs() }, [])

  const handleSaveSeq = async (val: boolean) => {
    setSaving(true)
    try {
      await adminApi.updateSystemConfig('upload.sequential_mode', val ? 'true' : 'false')
      setSeqMode(val)
      message.success('已更新')
    } catch { message.error('保存失败') } finally {
      setSaving(false)
    }
  }

  const handleSaveMax = async () => {
    setSaving(true)
    try {
      await adminApi.updateSystemConfig('upload.max_concurrent_chunks', String(maxConcurrent))
      message.success('已更新')
    } catch { message.error('保存失败') } finally {
      setSaving(false)
    }
  }

  return (
    <Spin spinning={loading}>
      <Text strong style={{ fontSize: 16, display: 'block', marginBottom: 24 }}><SettingOutlined /> 系统配置</Text>

      <Card title={<span><CloudUploadOutlined /> 上传配置</span>} size="small" style={{ marginBottom: 16 }}>
        <Row gutter={[16, 16]} align="middle">
          <Col span={12}>
            <Text>顺序上传模式</Text>
            <div><Text type="secondary" style={{ fontSize: 12 }}>启用后分片逐个上传；关闭后并发上传</Text></div>
          </Col>
          <Col span={12}>
            <Switch checked={seqMode} loading={saving} onChange={handleSaveSeq}
              checkedChildren="顺序" unCheckedChildren="并发" />
          </Col>
        </Row>
      </Card>

      <Card title="并发设置" size="small">
        <Row gutter={[16, 16]} align="middle">
          <Col span={12}>
            <Text>最大并发分片数</Text>
            <div><Text type="secondary" style={{ fontSize: 12 }}>仅并发模式生效 (1-10)</Text></div>
          </Col>
          <Col span={12}>
            <Space>
              <InputNumber min={1} max={10} value={maxConcurrent}
                onChange={(v) => setMaxConcurrent(v || 1)} />
              <Button type="primary" loading={saving} onClick={handleSaveMax}>保存</Button>
            </Space>
          </Col>
        </Row>
      </Card>
    </Spin>
  )
}
