import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Input, Button, Card, message, Typography } from 'antd'
import { LockOutlined } from '@ant-design/icons'
import api from '../services/api'

const { Title, Text } = Typography

export default function ResetPasswordPage() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  // token from URL fragment (#token=xxx), not query string, to avoid server log leakage
  const hash = window.location.hash
  const params = new URLSearchParams(hash.startsWith('#') ? hash.slice(1) : hash)
  const token = params.get('token') || ''

  const onFinish = async (values: { new_password: string; confirm_password: string }) => {
    if (values.new_password !== values.confirm_password) {
      message.error('两次密码输入不一致')
      return
    }
    if (!token) {
      message.error('缺少重置令牌')
      return
    }
    setLoading(true)
    try {
      await api.post('/user/password/reset', {
        token,
        new_password: values.new_password,
      })
      message.success('密码已重置，请使用新密码登录')
      navigate('/login')
    } catch (err: any) {
      message.error(err.response?.data?.message || '重置密码失败')
    } finally {
      setLoading(false)
    }
  }

  if (!token) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#fafaf8' }}>
        <Card style={{ width: 400 }}>
          <div style={{ textAlign: 'center' }}>
            <Title level={3}>无效链接</Title>
            <Text type="secondary">重置链接无效或已过期</Text>
            <div style={{ marginTop: 16 }}>
              <Link to="/login">返回登录</Link>
            </div>
          </div>
        </Card>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#fafaf8' }}>
      <Card style={{ width: 400 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={3}>重置密码</Title>
          <Text type="secondary">请输入新密码</Text>
        </div>
        <Form onFinish={onFinish} size="large">
          <Form.Item name="new_password" rules={[
            { required: true, message: '请输入新密码' },
            { min: 8, message: '密码至少 8 位' },
          ]}>
            <Input.Password prefix={<LockOutlined />} placeholder="新密码" />
          </Form.Item>
          <Form.Item name="confirm_password" rules={[
            { required: true, message: '请确认新密码' },
          ]}>
            <Input.Password prefix={<LockOutlined />} placeholder="确认新密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              重置密码
            </Button>
          </Form.Item>
          <div style={{ textAlign: 'center' }}>
            <Link to="/login">返回登录</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}
