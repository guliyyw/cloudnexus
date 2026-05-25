import { useEffect, useState } from 'react'
import { Table, Button, Space, message, Popconfirm, Typography, Modal, Form, Input } from 'antd'
import { PlusOutlined, HddOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as adminApi from '../../services/admin'
import type { QuotaTier } from '../../services/admin'
import { formatFileSize } from '../../utils/format'

const { Text } = Typography

export default function QuotaTierPanel() {
  const [tiers, setTiers] = useState<QuotaTier[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingTier, setEditingTier] = useState<QuotaTier | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const fetchTiers = async () => {
    setLoading(true)
    try {
      setTiers(await adminApi.getQuotaTiers())
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchTiers() }, [])

  const parseStorageInput = (v: string): number => {
    const s = v.trim().toLowerCase()
    const num = parseFloat(s)
    if (isNaN(num)) return 0
    if (s.endsWith('tb')) return Math.round(num * 1099511627776)
    if (s.endsWith('gb')) return Math.round(num * 1073741824)
    if (s.endsWith('mb')) return Math.round(num * 1048576)
    if (s.endsWith('kb')) return Math.round(num * 1024)
    return Math.round(num)
  }

  const handleCreate = () => {
    setEditingTier(null)
    form.resetFields()
    form.setFieldsValue({ storage_limit: '' })
    setModalOpen(true)
  }

  const handleEdit = (tier: QuotaTier) => {
    setEditingTier(tier)
    // Convert bytes to readable format for editing
    const gb = tier.storage_limit / 1073741824
    const mb = tier.storage_limit / 1048576
    if (gb >= 1 && gb % 1 === 0) {
      form.setFieldsValue({ ...tier, storage_limit: `${gb}GB` })
    } else if (mb >= 1) {
      form.setFieldsValue({ ...tier, storage_limit: `${mb}MB` })
    } else {
      form.setFieldsValue({ ...tier, storage_limit: `${tier.storage_limit}B` })
    }
    setModalOpen(true)
  }

  const handleDelete = async (id: string) => {
    try {
      await adminApi.deleteQuotaTier(id)
      message.success('已删除')
      fetchTiers()
    } catch (e: any) {
      message.error(e.response?.data?.message || '删除失败')
    }
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    const raw = values.storage_limit
    let storageLimit: number
    if (typeof raw === 'string') {
      storageLimit = parseStorageInput(raw)
      if (storageLimit <= 0) {
        message.error('请输入有效的存储上限，如 500MB, 2GB, 1TB')
        return
      }
    } else {
      storageLimit = raw
    }
    const payload = { ...values, storage_limit: storageLimit }
    setSaving(true)
    try {
      if (editingTier) {
        await adminApi.updateQuotaTier(editingTier.id, payload)
        message.success('已更新')
      } else {
        await adminApi.createQuotaTier(payload)
        message.success('已创建')
      }
      setModalOpen(false)
      fetchTiers()
    } catch (e: any) {
      message.error(e.response?.data?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const columns: ColumnsType<QuotaTier> = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
    {
      title: '存储上限', dataIndex: 'storage_limit', key: 'storage_limit', width: 150,
      render: (v: number) => formatFileSize(v),
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '操作', key: 'actions', width: 150,
      render: (_: unknown, record: QuotaTier) => (
        <Space>
          <Button size="small" onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm title="删除此等级？" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong style={{ fontSize: 16 }}><HddOutlined /> 配额等级管理</Text>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新建等级</Button>
      </div>
      <Table columns={columns} dataSource={tiers} rowKey="id" loading={loading} size="middle" pagination={false} />

      <Modal
        title={editingTier ? '编辑配额等级' : '新建配额等级'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入等级名称' }]}>
            <Input placeholder="例如: free, premium, enterprise" />
          </Form.Item>
          <Form.Item name="storage_limit" label="存储上限" rules={[{ required: true, message: '请输入存储上限' }]}>
            <Input placeholder="例如: 500MB, 2GB, 1TB" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
