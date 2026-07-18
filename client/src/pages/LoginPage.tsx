import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Input, Button, Card, message, Typography } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import { colors } from '../theme/tokens'

const { Title, Text } = Typography

function getLoginErrorMessage(err: any) {
  const status = err?.response?.status
  const serverMessage = err?.response?.data?.message

  if (status === 429) return '账户已被锁定，请稍后再试'
  if (status === 401) return '用户名或密码错误'
  return serverMessage || '登录失败，请稍后重试'
}

export default function LoginPage() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const login = useAuthStore((s) => s.login)

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      await login(values.username, values.password)
      message.success('登录成功')
      navigate('/dashboard')
    } catch (err: any) {
      message.error(getLoginErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
      <Card style={{ width: 400 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={3} style={{ color: colors.primary }}>CloudNexus</Title>
          <Text type="secondary">自托管协作平台</Text>
        </div>
        <Form onFinish={onFinish} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>
          <div style={{ textAlign: 'center' }}>
            <Text>还没有账号？</Text> <Link to="/register">立即注册</Link>
          </div>
          <div style={{ textAlign: 'center', marginTop: 8 }}>
            <Link to="/forgot-password">忘记密码？</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}
