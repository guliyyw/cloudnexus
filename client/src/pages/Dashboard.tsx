import { useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Col, Progress, Row, Statistic, Typography } from 'antd'
import {
  BgColorsOutlined,
  CloudOutlined,
  CustomerServiceOutlined,
  DeleteOutlined,
  FileTextOutlined,
  MessageOutlined,
  PictureOutlined,
  PlaySquareOutlined,
  SettingOutlined,
  ShareAltOutlined,
  TeamOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons'
import ModuleCard from '../components/dashboard/ModuleCard'
import { useAccess } from '../hooks/useAccess'
import { useDashboardStore } from '../stores/dashboardStore'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Text, Title } = Typography

const featureModules = [
  { key: 'files', name: '文件', icon: <CloudOutlined />, path: '/files', detail: '上传、预览、分享和管理云盘文件。', eyebrow: 'Workspace' },
  { key: 'drama', name: '短剧', icon: <PlaySquareOutlined />, path: '/drama', detail: '管理剧本、分镜、角色资产和生成任务。', eyebrow: 'AI Video' },
  { key: 'image_generation', name: '图片生成', icon: <BgColorsOutlined />, path: '/image-generation', detail: '使用提示词和参考图片创作图片。', eyebrow: 'AI Image' },
  { key: 'documents', name: '文档', icon: <FileTextOutlined />, path: '/documents', detail: '创建和编辑在线协作文档。', eyebrow: 'Docs' },
  { key: 'chat', name: '聊天', icon: <MessageOutlined />, path: '/chat', detail: '发送消息、文件和实时会话。', eyebrow: 'Realtime' },
  { key: 'cameras', name: '视频监控', icon: <VideoCameraOutlined />, path: '/cameras', detail: '查看摄像头、人脸库和考勤签到。', eyebrow: 'Monitoring' },
  { key: 'album', name: '相册', icon: <PictureOutlined />, path: '/album', detail: '整理图片和视频内容。', eyebrow: 'Media' },
  { key: 'music', name: '音乐', icon: <CustomerServiceOutlined />, path: '/music', detail: '播放公共音乐和云盘音频。', eyebrow: 'Audio' },
  { key: 'friends', name: '好友', icon: <TeamOutlined />, path: '/friends', detail: '管理联系人和私聊入口。', eyebrow: 'Network' },
  { key: 'shares', name: '分享', icon: <ShareAltOutlined />, path: '/shares', detail: '查看和管理对外分享链接。', eyebrow: 'Links' },
  { key: 'trash', name: '回收站', icon: <DeleteOutlined />, path: '/trash', detail: '恢复或清理已删除文件。', eyebrow: 'Recovery' },
]

export default function Dashboard() {
  const navigate = useNavigate()
  const { isAdmin, hasPermission } = useAccess()
  const { modules, summary, fetchStatus } = useDashboardStore()

  const permittedFeatureModules = featureModules.filter((mod) => hasPermission(`module:${mod.key}`))
  const visibleFeatureModules = isAdmin
    ? [
        ...permittedFeatureModules,
        {
          key: 'admin',
          name: '管理后台',
          icon: <SettingOutlined />,
          path: '/admin',
          detail: '管理用户、权限、配额、日志和系统配置。',
          eyebrow: 'Administration',
        },
      ]
    : permittedFeatureModules

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
      { label: '告警', value: summary.warning, suffix: ' 个' },
      { label: '异常', value: summary.error, suffix: ' 个' },
    ]
  }, [summary])

  const moduleStatusMap = useMemo(() => {
    const map = new Map<string, { status: 'green' | 'yellow' | 'red'; detail: string }>()
    modules.forEach((mod) => map.set(mod.key, { status: mod.status, detail: mod.detail }))
    return map
  }, [modules])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.xl }}>
      <section style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1.8fr) minmax(320px, 1fr)', gap: spacing.lg }}>
        <div style={{ borderRadius: radius.lg, padding: '28px 28px 24px', background: colors.surfaceRaised, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card }}>
          <Text style={{ color: colors.primary, fontWeight: 700, letterSpacing: 0.5 }}>DASHBOARD</Text>
          <Title level={2} style={{ marginTop: 10, marginBottom: 0, color: colors.text }}>
            工作台
          </Title>
        </div>

        <div style={{ borderRadius: radius.lg, padding: '24px 24px 20px', background: colors.surface, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card }}>
          <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>健康摘要</Text>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 12, marginBottom: 16 }}>
            <Title level={3} style={{ margin: 0, color: colors.text }}>系统状态</Title>
            <Text style={{ color: colors.primary, fontWeight: 700 }}>{healthRate}%</Text>
          </div>
          <Progress percent={healthRate} showInfo={false} strokeColor={colors.primary} trailColor={colors.surfaceMuted} strokeLinecap="round" />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: spacing.sm, marginTop: 20 }}>
            {serviceMetrics.map((metric) => (
              <div key={metric.label} style={{ padding: '14px 12px', borderRadius: radius.md, background: colors.surfaceMuted, border: `1px solid ${colors.borderSubtle}` }}>
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

      <section style={{ display: 'flex', flexDirection: 'column', gap: spacing.md }}>
        <div style={{ display: 'flex', alignItems: 'end', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
          <div>
            <Text style={{ color: colors.textSecondary, fontSize: 12, fontWeight: 600 }}>常用入口</Text>
            <Title level={4} style={{ margin: '6px 0 0', color: colors.text }}>功能模块</Title>
          </div>
        </div>
        <Row gutter={[18, 18]}>
          {visibleFeatureModules.map((mod) => (
            <Col xs={24} md={12} xl={8} key={mod.key}>
              <ModuleCard
                icon={mod.icon}
                name={mod.name}
                status={moduleStatusMap.get(mod.key)?.status || 'green'}
                detail={moduleStatusMap.get(mod.key)?.detail || mod.detail}
                eyebrow={mod.eyebrow}
                onClick={() => navigate(mod.path)}
              />
            </Col>
          ))}
        </Row>
      </section>
    </div>
  )
}
