import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Form, Input, Button, Card, message, Typography } from 'antd'
import { MailOutlined } from '@ant-design/icons'
import axios from 'axios'

const { Title, Text } = Typography

export default function ForgotPasswordPage() {
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)

  const onFinish = async (values: { email: string }) => {
    setLoading(true)
    try {
      await axios.post('/api/v1/user/password/forgot', { email: values.email })
      setSent(true)
      message.success('如果该邮箱已注册，重置邮件已发送')
    } catch {
      message.success('如果该邮箱已注册，重置邮件已发送')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#fafaf8' }}>
      <Card style={{ width: 400 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={3}>忘记密码</Title>
          <Text type="secondary">输入注册邮箱，我们将发送重置链接</Text>
        </div>
        {sent ? (
          <div style={{ textAlign: 'center' }}>
            <Text type="success">重置邮件已发送，请检查收件箱。</Text>
            <div style={{ marginTop: 16 }}>
              <Link to="/login">返回登录</Link>
            </div>
          </div>
        ) : (
          <Form onFinish={onFinish} size="large">
            <Form.Item name="email" rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}>
              <Input prefix={<MailOutlined />} placeholder="邮箱" />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} block>
                发送重置邮件
              </Button>
            </Form.Item>
            <div style={{ textAlign: 'center' }}>
              <Link to="/login">返回登录</Link>
            </div>
          </Form>
        )}
      </Card>
    </div>
  )
}
