import { useEffect, useState, useCallback } from 'react'
import { Card, Form, Input, Button, message, Typography, Avatar, Switch, List, Popconfirm, Tag } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, SaveOutlined, SafetyOutlined, TabletOutlined } from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import api from '../services/api'

const { Title, Text } = Typography

interface Privacy {
  allow_search: boolean
  allow_add_friend: boolean
  show_online: boolean
}

interface Session {
  id: string
  jti: string
  user_agent: string
  ip_address: string
  login_at: string
  last_active_at: string
  is_current: boolean
}

export default function UserSettingsPage() {
  const { user, fetchProfile } = useAuthStore()
  const [profileLoading, setProfileLoading] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [avatar, setAvatar] = useState(user?.avatar || '')
  const [privacy, setPrivacy] = useState<Privacy>({ allow_search: true, allow_add_friend: true, show_online: true })
  const [privacyLoading, setPrivacyLoading] = useState(false)
  const [sessions, setSessions] = useState<Session[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(false)

  useEffect(() => {
    if (!user) fetchProfile()
  }, [])

  useEffect(() => {
    setAvatar(user?.avatar || '')
  }, [user])

  const fetchPrivacy = useCallback(async () => {
    try {
      const res = await api.get('/user/privacy')
      setPrivacy(res.data.data)
    } catch { /* ignore */ }
  }, [])

  const fetchSessions = useCallback(async () => {
    setSessionsLoading(true)
    try {
      const res = await api.get('/user/sessions')
      setSessions(res.data.data?.sessions || [])
    } catch { /* ignore */ }
    finally { setSessionsLoading(false) }
  }, [])

  useEffect(() => {
    fetchPrivacy()
    fetchSessions()
  }, [fetchPrivacy, fetchSessions])

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

  const handlePrivacyChange = async (key: keyof Privacy, value: boolean) => {
    const next = { ...privacy, [key]: value }
    setPrivacy(next)
    setPrivacyLoading(true)
    try {
      await api.put('/user/privacy', next)
      message.success('隐私设置已更新')
    } catch (err: any) {
      setPrivacy(privacy)
      message.error(err.response?.data?.message || '更新失败')
    } finally {
      setPrivacyLoading(false)
    }
  }

  const handleRevokeSession = async (jti: string) => {
    try {
      await api.delete(`/user/sessions/${jti}`)
      message.success('已强制下线')
      fetchSessions()
    } catch (err: any) {
      message.error(err.response?.data?.message || '操作失败')
    }
  }

  const handleRevokeAllSessions = async () => {
    try {
      await api.delete('/user/sessions')
      message.success('其他设备已全部下线')
      fetchSessions()
    } catch (err: any) {
      message.error(err.response?.data?.message || '操作失败')
    }
  }

  const formatUA = (ua: string) => {
    if (!ua) return '未知设备'
    if (ua.includes('Windows')) return ua.split(' ').slice(0, 3).join(' ')
    if (ua.includes('Mac')) return 'Mac'
    if (ua.includes('Linux')) return 'Linux'
    return ua.substring(0, 40)
  }

  return (
    <div style={{ maxWidth: 640, margin: '0 auto' }}>
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <Avatar size={64} icon={<UserOutlined />} src={avatar}
            style={{ backgroundColor: '#81ecfe' }} />
          <div>
            <Title level={4} style={{ margin: 0 }}>{user?.username}</Title>
            <Text type="secondary">{user?.email}</Text>
            {user?.is_admin && <Text type="secondary" style={{ marginLeft: 8, color: '#81ecfe' }}>(管理员)</Text>}
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

      <Card title={<span><SafetyOutlined /> 隐私设置</span>} style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <Text strong>允许被搜索</Text>
              <br /><Text type="secondary" style={{ fontSize: 12 }}>关闭后其他用户无法通过搜索找到你</Text>
            </div>
            <Switch checked={privacy.allow_search} loading={privacyLoading} onChange={(v) => handlePrivacyChange('allow_search', v)} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <Text strong>允许添加好友</Text>
              <br /><Text type="secondary" style={{ fontSize: 12 }}>关闭后其他人无法向你发送好友申请</Text>
            </div>
            <Switch checked={privacy.allow_add_friend} loading={privacyLoading} onChange={(v) => handlePrivacyChange('allow_add_friend', v)} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <Text strong>显示在线状态</Text>
              <br /><Text type="secondary" style={{ fontSize: 12 }}>关闭后好友无法看到你的在线状态</Text>
            </div>
            <Switch checked={privacy.show_online} loading={privacyLoading} onChange={(v) => handlePrivacyChange('show_online', v)} />
          </div>
        </div>
      </Card>

      <Card title={<span><TabletOutlined /> 会话管理</span>} style={{ marginBottom: 16 }}>
        <List
          loading={sessionsLoading}
          dataSource={sessions}
          locale={{ emptyText: '暂无活跃会话' }}
          renderItem={(s) => (
            <List.Item
              extra={
                !s.is_current && (
                  <Popconfirm title="强制下线此设备？" onConfirm={() => handleRevokeSession(s.jti)} okText="确认" cancelText="取消">
                    <Button size="small" danger>下线</Button>
                  </Popconfirm>
                )
              }
            >
              <List.Item.Meta
                avatar={<Tag color={s.is_current ? 'green' : 'default'}>{s.is_current ? '当前' : '其他'}</Tag>}
                title={formatUA(s.user_agent)}
                description={`IP: ${s.ip_address || '-'} · ${s.login_at ? new Date(s.login_at).toLocaleString() : '-'}`}
              />
            </List.Item>
          )}
        />
        {sessions.filter((s) => !s.is_current).length > 0 && (
          <div style={{ marginTop: 12 }}>
            <Popconfirm title="将所有其他设备强制下线？" onConfirm={handleRevokeAllSessions} okText="确认" cancelText="取消">
              <Button danger block>下线所有其他设备</Button>
            </Popconfirm>
          </div>
        )}
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
              { min: 8, message: '密码至少8位' },
            ]}>
            <Input.Password prefix={<LockOutlined />} placeholder="请输入新密码（至少8位）" />
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
