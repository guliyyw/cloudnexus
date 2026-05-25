import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Row, Col, Typography, Spin } from 'antd'
import {
  CloudOutlined,
  MessageOutlined,
  ContainerOutlined,
  VideoCameraOutlined,
  FileTextOutlined,
  ClusterOutlined,
  PictureOutlined,
  CustomerServiceOutlined,
  UnorderedListOutlined,
  TeamOutlined,
  ShareAltOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import ModuleCard from '../components/dashboard/ModuleCard'
import { useDashboardStore } from '../stores/dashboardStore'

const { Title } = Typography

// 服务健康状态模块图标
const serviceIconMap: Record<string, React.ReactNode> = {
  files: <CloudOutlined />,
  im: <MessageOutlined />,
  docker: <ContainerOutlined />,
  camera: <VideoCameraOutlined />,
  collab: <FileTextOutlined />,
  infra: <ClusterOutlined />,
}

// 功能模块定义
const featureModules = [
  { key: 'files', name: '文件管理', icon: <CloudOutlined />, path: '/files' },
  { key: 'album', name: '相册', icon: <PictureOutlined />, path: '/album' },
  { key: 'music', name: '音乐', icon: <CustomerServiceOutlined />, path: '/music' },
  { key: 'playlist', name: '播放列表', icon: <UnorderedListOutlined />, path: '/playlist' },
  { key: 'chat', name: '即时通讯', icon: <MessageOutlined />, path: '/chat' },
  { key: 'friends', name: '好友', icon: <TeamOutlined />, path: '/friends' },
  { key: 'shares', name: '我的分享', icon: <ShareAltOutlined />, path: '/shares' },
  { key: 'trash', name: '回收站', icon: <DeleteOutlined />, path: '/trash' },
]

export default function Dashboard() {
  const navigate = useNavigate()
  const { modules, summary, loading, fetchStatus } = useDashboardStore()

  useEffect(() => {
    fetchStatus()
  }, [])

  const handleServiceCardClick = (key: string) => {
    const routes: Record<string, string> = {
      files: '/files',
      im: '/chat',
      docker: '/docker',
      camera: '/cameras',
      collab: '/documents',
      infra: '/admin',
    }
    navigate(routes[key] || '/files')
  }

  const handleFeatureCardClick = (path: string) => {
    navigate(path)
  }

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>系统概览</Title>

      {/* 功能模块卡片 */}
      <Title level={5} style={{ marginBottom: 16 }}>功能模块</Title>
      <Row gutter={[16, 16]} style={{ marginBottom: 32 }}>
        {featureModules.map((mod) => (
          <Col xs={24} sm={12} lg={6} key={mod.key}>
            <ModuleCard
              icon={mod.icon}
              name={mod.name}
              status="green"
              detail="点击进入"
              onClick={() => handleFeatureCardClick(mod.path)}
            />
          </Col>
        ))}
      </Row>

      {/* 服务健康状态 */}
      <Title level={5} style={{ marginBottom: 16 }}>服务状态</Title>
      {loading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : (
        <Row gutter={[16, 16]}>
          {modules.map((mod) => (
            <Col xs={24} sm={12} lg={8} key={mod.key}>
              <ModuleCard
                icon={serviceIconMap[mod.key]}
                name={mod.name}
                status={mod.status}
                detail={mod.detail}
                onClick={() => handleServiceCardClick(mod.key)}
              />
            </Col>
          ))}
        </Row>
      )}
      {summary && (
        <div style={{ marginTop: 24, color: '#888', fontSize: 12, textAlign: 'center' }}>
          共 {summary.total} 个服务，{summary.healthy} 个正常
          {summary.warning > 0 && `，${summary.warning} 个警告`}
          {summary.error > 0 && `，${summary.error} 个异常`}
        </div>
      )}
    </div>
  )
}
