import { useEffect, useState } from 'react'
import { Card, Form, Input, Button, message, Typography, Avatar } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, SaveOutlined } from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import api from '../services/api'

const { Title, Text } = Typography

export default function UserSettingsPage() {
  const { user, fetchProfile } = useAuthStore()
  const [profileLoading, setProfileLoading] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [avatar, setAvatar] = useState(user?.avatar || '')

  useEffect(() => {
    if (!user) fetchProfile()
  }, [])

  useEffect(() => {
    setAvatar(user?.avatar || '')
  }, [user])

  const handleUpdateProfile = async (values: { email: string; avatar: string }) => {
    setProfileLoading(true)
    try {
      await api.put('/user/profile', values)
      await fetchProfile()
      message.success('资料已更新')
    } catch (err: any) {
      message.error(err.response?.data?.message || '更新失败')
    } finally {
      setProfileLoading(false)
    }
  }

  const handleChangePassword = async (values: { old_password: string; new_password: string }) => {
    setPasswordLoading(true)
    try {
      await api.put('/user/password', values)
      message.success('密码已修改，请重新登录')
      setTimeout(() => useAuthStore.getState().logout(), 1500)
    } catch (err: any) {
      message.error(err.response?.data?.message || '修改失败')
    } finally {
      setPasswordLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: 640, margin: '0 auto' }}>
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 24 }}>
          <Avatar size={64} icon={<UserOutlined />} src={avatar}
            style={{ backgroundColor: '#e8964a' }} />
          <div>
            <Title level={4} style={{ margin: 0 }}>{user?.username}</Title>
            <Text type="secondary">{user?.email}</Text>
            {user?.is_admin && <Text type="secondary" style={{ marginLeft: 8, color: '#e8964a' }}>(管理员)</Text>}
          </div>
        </div>
      </Card>

      <Card title={<span><UserOutlined /> 修改资料</span>} style={{ marginBottom: 16 }}>
        <Form
          layout="vertical"
          initialValues={{ email: user?.email || '', avatar: user?.avatar || '' }}
          onFinish={handleUpdateProfile}
          key={user?.email}
        >
          <Form.Item name="email" label="邮箱"
            rules={[{ type: 'email', message: '请输入有效邮箱' }]}>
            <Input prefix={<MailOutlined />} placeholder="请输入邮箱" />
          </Form.Item>
          <Form.Item name="avatar" label="头像链接">
            <Input placeholder="输入头像图片 URL" onChange={(e) => setAvatar(e.target.value)} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={profileLoading} icon={<SaveOutlined />}>
            保存修改
          </Button>
        </Form>
      </Card>

      <Card title={<span><LockOutlined /> 修改密码</span>}>
        <Form layout="vertical" onFinish={handleChangePassword}>
          <Form.Item name="old_password" label="当前密码"
            rules={[{ required: true, message: '请输入当前密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="请输入当前密码" />
          </Form.Item>
          <Form.Item name="new_password" label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码至少6位' },
            ]}>
            <Input.Password prefix={<LockOutlined />} placeholder="请输入新密码（至少6位）" />
          </Form.Item>
          <Form.Item name="confirm_password" label="确认新密码"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}>
            <Input.Password prefix={<LockOutlined />} placeholder="请再次输入新密码" />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={passwordLoading} danger>
            修改密码
          </Button>
        </Form>
      </Card>
    </div>
  )
}
