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
} from '@ant-design/icons'
import ModuleCard from '../components/dashboard/ModuleCard'
import { useDashboardStore } from '../stores/dashboardStore'

const { Title } = Typography

const iconMap: Record<string, React.ReactNode> = {
  files: <CloudOutlined />,
  im: <MessageOutlined />,
  docker: <ContainerOutlined />,
  camera: <VideoCameraOutlined />,
  collab: <FileTextOutlined />,
  infra: <ClusterOutlined />,
}

export default function Dashboard() {
  const navigate = useNavigate()
  const { modules, summary, loading, fetchStatus } = useDashboardStore()

  useEffect(() => {
    fetchStatus()
  }, [])

  const handleCardClick = (key: string) => {
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

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>系统概览</Title>
      {loading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : (
        <Row gutter={[16, 16]}>
          {modules.map((mod) => (
            <Col xs={24} sm={12} lg={8} key={mod.key}>
              <ModuleCard
                icon={iconMap[mod.key]}
                name={mod.name}
                status={mod.status}
                detail={mod.detail}
                onClick={() => handleCardClick(mod.key)}
              />
            </Col>
          ))}
        </Row>
      )}
      {summary && (
        <div style={{ marginTop: 24, color: '#8c8c8c', fontSize: 12, textAlign: 'center' }}>
          共 {summary.total} 个模块，{summary.healthy} 个正常
          {summary.warning > 0 && `，${summary.warning} 个警告`}
          {summary.error > 0 && `，${summary.error} 个异常`}
        </div>
      )}
    </div>
  )
}
