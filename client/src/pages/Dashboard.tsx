import { useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Col, Progress, Row, Skeleton, Statistic, Typography } from 'antd'
import {
  CloudOutlined,
  ClusterOutlined,
  ContainerOutlined,
  CustomerServiceOutlined,
  DeleteOutlined,
  FileTextOutlined,
  MessageOutlined,
  PictureOutlined,
  ShareAltOutlined,
  TeamOutlined,
  UnorderedListOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons'
import ModuleCard from '../components/dashboard/ModuleCard'
import { useDashboardStore } from '../stores/dashboardStore'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Paragraph, Text, Title } = Typography

const serviceIconMap: Record<string, React.ReactNode> = {
  files: <CloudOutlined />,
  im: <MessageOutlined />,
  docker: <ContainerOutlined />,
  camera: <VideoCameraOutlined />,
  collab: <FileTextOutlined />,
  infra: <ClusterOutlined />,
}

const featureModules = [
  { key: 'files', name: '文件工作台', icon: <CloudOutlined />, path: '/files', detail: '统一处理目录、上传、共享、协作与版本操作。', eyebrow: 'Workspace' },
  { key: 'documents', name: '在线文档', icon: <FileTextOutlined />, path: '/documents', detail: '查看文档列表并进入实时协作编辑。', eyebrow: 'Docs' },
  { key: 'chat', name: '即时通讯', icon: <MessageOutlined />, path: '/chat', detail: '围绕当前会话进行消息、文件与成员协作。', eyebrow: 'Realtime' },
  { key: 'album', name: '相册', icon: <PictureOutlined />, path: '/album', detail: '按媒体视图、时间线与目录维度浏览内容。', eyebrow: 'Media' },
  { key: 'music', name: '音乐', icon: <CustomerServiceOutlined />, path: '/music', detail: '浏览曲库、管理播放状态并配合全局播放器。', eyebrow: 'Audio' },
  { key: 'playlist', name: '播放列表', icon: <UnorderedListOutlined />, path: '/playlist', detail: '组织歌单、排序条目并执行导入导出。', eyebrow: 'Queue' },
  { key: 'friends', name: '好友', icon: <TeamOutlined />, path: '/friends', detail: '维护联系人网络并快速打开私聊入口。', eyebrow: 'Network' },
  { key: 'shares', name: '我的分享', icon: <ShareAltOutlined />, path: '/shares', detail: '查看分享记录、访问策略和有效期。', eyebrow: 'Public Links' },
  { key: 'trash', name: '回收站', icon: <DeleteOutlined />, path: '/trash', detail: '检查误删内容并执行恢复或清理。', eyebrow: 'Recovery' },
]

export default function Dashboard() {
  const navigate = useNavigate()
  const { modules, summary, loading, fetchStatus } = useDashboardStore()

  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])

  const healthRate = useMemo(() => {
    if (!summary?.total) return 0
    return Math.round((summary.healthy / summary.total) * 100)
  }, [summary])

  const serviceMetrics = useMemo(() => {
    if (!summary) return []
    return [
      { label: '在线服务', value: summary.healthy, suffix: ` / ${summary.total}` },
      { label: '关注中', value: summary.warning, suffix: ' 个' },
      { label: '异常项', value: summary.error, suffix: ' 个' },
    ]
  }, [summary])

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

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.xl }}>
      <section
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(0, 1.8fr) minmax(320px, 1fr)',
          gap: spacing.lg,
        }}
      >
        <div
          style={{
            borderRadius: radius.lg,
            padding: '28px 28px 24px',
            background: colors.surfaceRaised,
            border: `1px solid ${colors.borderSubtle}`,
            boxShadow: shadow.card,
          }}
        >
          <Text style={{ color: colors.primary, fontWeight: 700, letterSpacing: 0.5 }}>DASHBOARD</Text>
          <Title level={2} style={{ marginTop: 10, marginBottom: 12, color: colors.text }}>
            统一入口，先看系统状态，再进入工作流。
          </Title>
          <Paragraph style={{ marginBottom: 0, fontSize: 15, lineHeight: 1.8, color: colors.textSecondary }}>
            这里集中承接 CloudNexus 的核心入口：文件、文档、聊天、媒体和系统模块都通过统一卡片进入，
            同时把当前服务健康概览放在首屏，避免用户在高频协作场景里来回切页面查状态。
          </Paragraph>
        </div>

        <div
          style={{
            borderRadius: radius.lg,
            padding: '24px 24px 20px',
            background: colors.surface,
            border: `1px solid ${colors.borderSubtle}`,
            boxShadow: shadow.card,
          }}
        >
          <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>当前健康摘要</Text>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 12, marginBottom: 16 }}>
            <Title level={3} style={{ margin: 0, color: colors.text }}>系统健康度</Title>
            <Text style={{ color: colors.primary, fontWeight: 700 }}>{healthRate}%</Text>
          </div>
          <Progress
            percent={healthRate}
            showInfo={false}
            strokeColor={colors.primary}
            trailColor={colors.surfaceMuted}
            strokeLinecap="round"
          />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: spacing.sm, marginTop: 20 }}>
            {serviceMetrics.map((metric) => (
              <div
                key={metric.label}
                style={{
                  padding: '14px 12px',
                  borderRadius: radius.md,
                  background: colors.surfaceMuted,
                  border: `1px solid ${colors.borderSubtle}`,
                }}
              >
                <Statistic
                  title={<span style={{ color: colors.textSecondary, fontSize: 12 }}>{metric.label}</span>}
                  value={metric.value}
                  suffix={<span style={{ fontSize: 12, color: colors.textSecondary }}>{metric.suffix}</span>}
                  valueStyle={{ color: colors.text, fontSize: 24, fontWeight: 700 }}
                />
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 首页把“我要做什么”和“系统现在怎么样”拆成两个区域，避免模块入口和健康状态互相抢注意力。 */}
      <section style={{ display: 'flex', flexDirection: 'column', gap: spacing.md }}>
        <div style={{ display: 'flex', alignItems: 'end', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
          <div>
            <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>优先入口</Text>
            <Title level={4} style={{ margin: '6px 0 0', color: colors.text }}>常用工作流</Title>
          </div>
          <Text style={{ color: colors.textSecondary }}>从这里进入文件、文档、聊天和媒体模块。</Text>
        </div>
        <Row gutter={[18, 18]}>
          {featureModules.map((mod) => (
            <Col xs={24} md={12} xl={8} key={mod.key}>
              <ModuleCard
                icon={mod.icon}
                name={mod.name}
                status="green"
                detail={mod.detail}
                eyebrow={mod.eyebrow}
                onClick={() => navigate(mod.path)}
              />
            </Col>
          ))}
        </Row>
      </section>

      <section style={{ display: 'flex', flexDirection: 'column', gap: spacing.md }}>
        <div style={{ display: 'flex', alignItems: 'end', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
          <div>
            <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>服务总览</Text>
            <Title level={4} style={{ margin: '6px 0 0', color: colors.text }}>健康状态与入口联动</Title>
          </div>
          {summary && (
            <Text style={{ color: colors.textSecondary }}>
              共 {summary.total} 个服务，{summary.healthy} 个正常
              {summary.warning > 0 && `，${summary.warning} 个警告`}
              {summary.error > 0 && `，${summary.error} 个异常`}
            </Text>
          )}
        </div>

        {loading ? (
          <div
            style={{
              padding: '24px 0 0',
              borderRadius: radius.lg,
              background: colors.surface,
              border: `1px solid ${colors.borderSubtle}`,
            }}
          >
            <Row gutter={[18, 18]}>
              {Array.from({ length: 6 }).map((_, idx) => (
                <Col xs={24} md={12} xl={8} key={idx}>
                  <div
                    style={{
                      padding: 24,
                      borderRadius: radius.lg,
                      background: colors.surfaceMuted,
                      border: `1px solid ${colors.borderSubtle}`,
                    }}
                  >
                    <Skeleton active paragraph={{ rows: 2 }} title={{ width: '55%' }} />
                  </div>
                </Col>
              ))}
            </Row>
          </div>
        ) : (
          <Row gutter={[18, 18]}>
            {modules.map((mod) => (
              <Col xs={24} md={12} xl={8} key={mod.key}>
                <ModuleCard
                  icon={serviceIconMap[mod.key]}
                  name={mod.name}
                  status={mod.status}
                  detail={mod.detail}
                  metric={mod.status === 'green' ? '状态稳定' : '需要关注'}
                  eyebrow="Service"
                  onClick={() => handleServiceCardClick(mod.key)}
                />
              </Col>
            ))}
          </Row>
        )}
      </section>
    </div>
  )
}
