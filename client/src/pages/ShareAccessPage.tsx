import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Descriptions, Input, Button, Spin, Result, Space, Alert, Typography, Image } from 'antd'
import { DownloadOutlined, EyeOutlined } from '@ant-design/icons'
import * as fileApi from '../services/file'
import { isPreviewable } from '../utils/preview'
import { formatFileSize } from '../utils/format'
import type { ShareInfo } from '../services/file'

const { Title, Text } = Typography

export default function ShareAccessPage() {
  const { code } = useParams<{ code: string }>()
  const [share, setShare] = useState<ShareInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [phase, setPhase] = useState<'loading' | 'password' | 'ready' | 'error'>('loading')
  const [password, setPassword] = useState('')
  const [verifying, setVerifying] = useState(false)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [showPreview, setShowPreview] = useState(false)

  useEffect(() => {
    if (!code) return
    fileApi.getShareByCode(code)
      .then((s) => {
        setShare(s)
        setPhase(s.has_password ? 'password' : 'ready')
      })
      .catch((err) => {
        const msg = err?.response?.data?.message || '加载分享失败'
        setError(msg)
        setPhase('error')
      })
  }, [code])

  const handleVerify = async () => {
    if (!code) return
    setVerifying(true)
    setPasswordError(null)
    try {
      await fileApi.verifySharePassword(code, password)
      setPhase('ready')
    } catch (err: any) {
      setPasswordError(err?.response?.data?.message || '验证失败')
    } finally {
      setVerifying(false)
    }
  }

  const renderPreview = () => {
    if (!share) return null
    const url = fileApi.getSharePreviewUrl(share.share_code, password)
    const mime = share.mime_type || ''

    if (mime.startsWith('image/')) {
      return <Image src={url} alt={share.file_name} style={{ maxWidth: '100%', maxHeight: 480 }} />
    }
    if (mime.startsWith('video/')) {
      return <video controls src={url} style={{ maxWidth: '100%', maxHeight: 480 }} />
    }
    if (mime.startsWith('audio/')) {
      return <audio controls src={url} style={{ width: '100%' }} />
    }
    if (mime === 'application/pdf') {
      return <iframe src={url} style={{ width: '100%', height: 560, border: 'none' }} title={share.file_name} />
    }
    return <Text type="secondary">不支持预览此文件类型</Text>
  }

  if (phase === 'loading') {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size="large" tip="加载分享信息..." />
      </div>
    )
  }

  if (phase === 'error') {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Card style={{ width: 480 }}>
          <Result status="error" title="无法访问" subTitle={error} />
        </Card>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f5f5f5' }}>
      <Card style={{ width: 520 }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div style={{ textAlign: 'center' }}>
            <Title level={4} style={{ margin: 0 }}>CloudNexus 文件分享</Title>
          </div>

          {share && (
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="文件名">
                <Text strong ellipsis style={{ maxWidth: 320 }}>{share.file_name}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="大小">{formatFileSize(share.file_size)}</Descriptions.Item>
              <Descriptions.Item label="类型">{share.mime_type || '未知'}</Descriptions.Item>
              {share.expires_at && (
                <Descriptions.Item label="有效期">
                  {new Date(share.expires_at) < new Date()
                    ? <Text type="danger">已过期</Text>
                    : new Date(share.expires_at).toLocaleString()}
                </Descriptions.Item>
              )}
            </Descriptions>
          )}

          {phase === 'password' && (
            <div>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Text>此分享需要密码才能访问：</Text>
                <Input.Password
                  placeholder="请输入访问密码"
                  value={password}
                  onChange={(e) => { setPassword(e.target.value); setPasswordError(null) }}
                  onPressEnter={handleVerify}
                />
                {passwordError && <Alert type="error" message={passwordError} showIcon />}
                <Button type="primary" block onClick={handleVerify} loading={verifying}>
                  验证
                </Button>
              </Space>
            </div>
          )}

          {phase === 'ready' && (
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                {share && isPreviewable(share.mime_type || '') && (
                  <Button icon={<EyeOutlined />} onClick={() => setShowPreview(!showPreview)}>
                    {showPreview ? '收起预览' : '预览'}
                  </Button>
                )}
                <Button type="primary" icon={<DownloadOutlined />}>
                  <a
                    href={fileApi.getShareDownloadUrl(share!.share_code, password)}
                    download={share?.file_name}
                    style={{ color: '#fff', textDecoration: 'none' }}
                  >
                    下载文件
                  </a>
                </Button>
              </Space>
              {showPreview && (
                <Card size="small" style={{ background: '#fafafa' }}>
                  {renderPreview()}
                </Card>
              )}
            </Space>
          )}
        </Space>
      </Card>
    </div>
  )
}
