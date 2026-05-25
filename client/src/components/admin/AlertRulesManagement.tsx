import { useEffect, useState } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Typography, Modal, Form, Input, InputNumber, Select, Switch, Row, Col } from 'antd'
import { PlusOutlined, BellOutlined, HistoryOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as adminApi from '../../services/admin'
import type { AlertRule, AlertHistoryItem } from '../../services/admin'

const { Text } = Typography

export default function AlertRulesManagement() {
  const [rules, setRules] = useState<AlertRule[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [saving, setSaving] = useState(false)
  const [history, setHistory] = useState<AlertHistoryItem[]>([])
  const [historyTotal, setHistoryTotal] = useState(0)
  const [historyPage, setHistoryPage] = useState(1)
  const [form] = Form.useForm()

  const fetchRules = async () => {
    setLoading(true)
    try {
      setRules(await adminApi.getAlertRules())
    } finally {
      setLoading(false)
    }
  }

  const fetchHistory = async (page = 1) => {
    try {
      const res = await adminApi.getAlertHistory({ page, page_size: 10 })
      setHistory(res.items)
      setHistoryTotal(res.total)
    } catch { /* ignore */ }
  }

  useEffect(() => { fetchRules(); fetchHistory() }, [])

  const handleCreate = () => {
    setEditingRule(null)
    form.resetFields()
    form.setFieldsValue({ node_name: '*', trigger_type: 'status_change', cooldown_seconds: 300, enabled: true })
    setModalOpen(true)
  }

  const handleEdit = (rule: AlertRule) => {
    setEditingRule(rule)
    form.setFieldsValue(rule)
    setModalOpen(true)
  }

  const handleDelete = async (id: string) => {
    await adminApi.deleteAlertRule(id)
    message.success('规则已删除')
    fetchRules()
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try {
      if (editingRule) {
        await adminApi.updateAlertRule(editingRule.id, values as Record<string, unknown>)
        message.success('规则已更新')
      } else {
        await adminApi.createAlertRule(values)
        message.success('规则已创建')
      }
      setModalOpen(false)
      fetchRules()
    } finally {
      setSaving(false)
    }
  }

  const columns: ColumnsType<AlertRule> = [
    { title: '规则名称', dataIndex: 'name', key: 'name', width: 150 },
    { title: '描述', dataIndex: 'description', key: 'description', width: 200, ellipsis: true },
    {
      title: '目标节点', dataIndex: 'node_name', key: 'node_name', width: 100,
      render: (v: string) => v === '*' ? <Tag>全部节点</Tag> : <Tag color="blue">{v}</Tag>,
    },
    {
      title: '触发类型', dataIndex: 'trigger_type', key: 'trigger_type', width: 100,
      render: (v: string) => v === 'status_change' ? <Tag color="orange">状态变更</Tag> : <Tag>阈值</Tag>,
    },
    { title: 'Webhook URL', dataIndex: 'webhook_url', key: 'webhook_url', width: 220, ellipsis: true, render: (v: string) => <Text copyable={{ text: v }}>{v}</Text> },
    {
      title: '冷却(秒)', dataIndex: 'cooldown_seconds', key: 'cooldown_seconds', width: 80,
    },
    {
      title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70,
      render: (v: boolean) => v ? <Tag color="green">启用</Tag> : <Tag color="red">禁用</Tag>,
    },
    {
      title: '操作', key: 'actions', width: 120,
      render: (_: unknown, record: AlertRule) => (
        <Space>
          <Button size="small" onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm title="删除此规则？" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const historyColumns: ColumnsType<AlertHistoryItem> = [
    { title: '时间', dataIndex: 'fired_at', key: 'fired_at', width: 170, render: (v: string) => new Date(v).toLocaleString() },
    { title: '规则', dataIndex: 'rule_name', key: 'rule_name', width: 130 },
    { title: '节点', dataIndex: 'node_name', key: 'node_name', width: 100 },
    {
      title: '类型', dataIndex: 'alert_type', key: 'alert_type', width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { unresponsive: 'orange', offline: 'red', recovery: 'green' }
        const labels: Record<string, string> = { unresponsive: '无响应', offline: '离线', recovery: '恢复' }
        return <Tag color={colors[v] || 'default'}>{labels[v] || v}</Tag>
      },
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 70,
      render: (v: string) => v === 'firing' ? <Tag color="red">告警中</Tag> : <Tag color="green">已恢复</Tag>,
    },
    { title: '消息', dataIndex: 'message', key: 'message', width: 250, ellipsis: true },
    {
      title: '响应码', dataIndex: 'response_code', key: 'response_code', width: 70,
      render: (v: number) => v === 0 ? <Tag>未发送</Tag> : v < 400 ? <Tag color="green">{v}</Tag> : <Tag color="red">{v}</Tag>,
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong style={{ fontSize: 16 }}><BellOutlined /> 告警规则</Text>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新建规则</Button>
      </div>
      <Table columns={columns} dataSource={rules} rowKey="id" loading={loading} size="middle" pagination={false} />

      <div style={{ marginTop: 32 }}>
        <Text strong style={{ fontSize: 16, display: 'block', marginBottom: 8 }}><HistoryOutlined /> 告警历史</Text>
        <Table
          columns={historyColumns}
          dataSource={history}
          rowKey="id"
          size="small"
          pagination={{
            current: historyPage,
            total: historyTotal,
            pageSize: 10,
            onChange: (p) => { setHistoryPage(p); fetchHistory(p) },
          }}
        />
      </div>

      <Modal
        title={editingRule ? '编辑规则' : '新建规则'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input maxLength={128} placeholder="例如：生产节点宕机告警" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} maxLength={512} placeholder="可选描述" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="node_name" label="目标节点">
                <Input placeholder="* 表示全部节点" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="trigger_type" label="触发类型">
                <Select options={[{ label: '状态变更', value: 'status_change' }, { label: '阈值 (预留)', value: 'threshold', disabled: true }]} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="webhook_url" label="Webhook URL" rules={[{ required: true, message: '请输入 Webhook URL' }]}>
            <Input placeholder="https://hooks.example.com/alert" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="cooldown_seconds" label="冷却时间 (秒)">
                <InputNumber min={0} max={86400} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="enabled" label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  )
}
