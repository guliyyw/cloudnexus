import { useEffect, useState } from 'react'
import { Table, Button, Tag, Space, message, Popconfirm, Card, Statistic, Row, Col, Typography } from 'antd'
import { UserOutlined, CheckCircleOutlined, StopOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as adminApi from '../services/admin'
import type { AdminUser } from '../services/admin'

const { Text } = Typography

export default function AdminPage() {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const res = await adminApi.getUsers(page, pageSize)
      setUsers(res.items)
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchUsers() }, [page])

  const handleToggleAdmin = async (id: string) => {
    const updated = await adminApi.toggleAdmin(id)
    setUsers((prev) => prev.map((u) => (u.id === id ? { ...u, is_admin: updated.is_admin } : u)))
    message.success(`已${updated.is_admin ? '设为' : '取消'}管理员`)
  }

  const handleToggleStatus = async (id: string) => {
    const updated = await adminApi.toggleStatus(id)
    setUsers((prev) => prev.map((u) => (u.id === id ? { ...u, status: updated.status } : u)))
    message.success(`已${updated.status === 1 ? '启用' : '禁用'}用户`)
  }

  const activeUsers = users.filter((u) => u.status === 1).length
  const adminUsers = users.filter((u) => u.is_admin).length

  const columns: ColumnsType<AdminUser> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 200, render: (v: string) => <Text copyable={{ text: v }}>{v.slice(-8)}</Text> },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 150 },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 200, ellipsis: true },
    {
      title: '角色', dataIndex: 'is_admin', key: 'is_admin', width: 100,
      render: (v: boolean) => v ? <Tag color="red">管理员</Tag> : <Tag>用户</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (v: number) => v === 1 ? <Tag color="green" icon={<CheckCircleOutlined />}>正常</Tag> : <Tag color="red" icon={<StopOutlined />}>禁用</Tag>,
    },
    {
      title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 180,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '操作', key: 'actions', width: 200,
      render: (_: any, record: AdminUser) => (
        <Space>
          <Popconfirm title={record.is_admin ? '取消管理员？' : '设为管理员？'} onConfirm={() => handleToggleAdmin(record.id)}>
            <Button size="small">{record.is_admin ? '取消管理员' : '设为管理员'}</Button>
          </Popconfirm>
          <Popconfirm
            title={record.status === 1 ? '禁用此用户？' : '启用此用户？'}
            onConfirm={() => handleToggleStatus(record.id)}
          >
            <Button size="small" danger={record.status === 1}>{record.status === 1 ? '禁用' : '启用'}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card><Statistic title="用户总数" value={total} prefix={<UserOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="活跃用户" value={activeUsers} valueStyle={{ color: '#3f8600' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="管理员" value={adminUsers} valueStyle={{ color: '#cf1322' }} prefix={<UserOutlined />} /></Card>
        </Col>
      </Row>

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 16 }}>用户管理</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchUsers}>刷新</Button>
      </div>

      <Table
        columns={columns}
        dataSource={users}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showTotal: (t) => `共 ${t} 用户`,
        }}
        size="middle"
      />
    </div>
  )
}
