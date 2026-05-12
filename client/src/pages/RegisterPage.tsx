import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Input, Button, Card, message, Typography, Progress, Space } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, SafetyOutlined } from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import { getCaptcha } from '../services/captcha'

const { Title, Text } = Typography

function getPasswordStrength(pw: string): { percent: number; status: 'exception' | 'active' | 'success'; text: string } {
  if (!pw) return { percent: 0, status: 'exception', text: '' }
  let score = 0
  if (pw.length >= 8) score += 25
  if (/[A-Z]/.test(pw)) score += 25
  if (/[a-z]/.test(pw)) score += 15
  if (/\d/.test(pw)) score += 15
  if (/[^A-Za-z0-9]/.test(pw)) score += 20
  if (score < 50) return { percent: score, status: 'exception', text: '弱' }
  if (score < 80) return { percent: score, status: 'active', text: '中等' }
  return { percent: score, status: 'success', text: '强' }
}

export default function RegisterPage() {
  const [loading, setLoading] = useState(false)
  const [captchaId, setCaptchaId] = useState('')
  const [captchaImg, setCaptchaImg] = useState('')
  const [password, setPassword] = useState('')
  const navigate = useNavigate()
  const register = useAuthStore((s) => s.register)

  const fetchCaptcha = async () => {
    try {
      const data = await getCaptcha()
      setCaptchaId(data.captcha_id)
      setCaptchaImg(data.image_base64)
    } catch {
      // captcha service may be unavailable
    }
  }

  useEffect(() => {
    fetchCaptcha()
  }, [])

  const onFinish = async (values: any) => {
    setLoading(true)
    try {
      await register(values.username, values.email, values.password, captchaId, values.captcha_code)
      message.success('注册成功，请登录')
      navigate('/login')
    } catch (err: any) {
      fetchCaptcha()
      message.error(err.response?.data?.message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  const strength = getPasswordStrength(password)

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#fafaf8' }}>
      <Card style={{ width: 400 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={3}>CloudNexus</Title>
          <Text type="secondary">创建新账号</Text>
        </div>
        <Form onFinish={onFinish} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名', min: 3 }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>
          <Form.Item name="email" rules={[{ required: true, message: '请输入邮箱', type: 'email' }]}>
            <Input prefix={<MailOutlined />} placeholder="邮箱" />
          </Form.Item>
          <Form.Item
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少8位' },
              { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z0-9]).{8,}$/, message: '需包含大小写字母、数字和特殊字符' },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="密码 (至少8位，含大小写+数字+特殊字符)"
              onChange={(e) => setPassword(e.target.value)}
            />
          </Form.Item>
          {password && (
            <Form.Item>
              <Progress percent={strength.percent} status={strength.status} showInfo={false} strokeColor={strength.status === 'success' ? '#52c41a' : strength.status === 'active' ? '#faad14' : '#ff4d4f'} />
              <Text type="secondary" style={{ fontSize: 12 }}>密码强度: {strength.text}</Text>
            </Form.Item>
          )}
          <Form.Item style={{ marginBottom: 8 }}>
            <Space align="start">
              {captchaImg && <img src={captchaImg} alt="验证码" style={{ height: 40, cursor: 'pointer', border: '1px solid #d9d9d9', borderRadius: 4 }} onClick={fetchCaptcha} />}
              <Button type="link" size="small" onClick={fetchCaptcha} style={{ padding: 0 }}>换一张</Button>
            </Space>
          </Form.Item>
          <Form.Item name="captcha_code" rules={[{ required: true, message: '请输入验证码' }]}>
            <Input prefix={<SafetyOutlined />} placeholder="验证码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              注册
            </Button>
          </Form.Item>
          <div style={{ textAlign: 'center' }}>
            <Text>已有账号？</Text> <Link to="/login">返回登录</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}
