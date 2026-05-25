import { Tabs } from 'antd'
import { UserOutlined, DashboardOutlined, FileTextOutlined, AreaChartOutlined, ClusterOutlined, BellOutlined, SettingOutlined, HddOutlined } from '@ant-design/icons'
import UserManagement from '../components/admin/UserManagement'
import SystemStatus from '../components/admin/SystemStatus'
import HistoricalMetrics from '../components/admin/HistoricalMetrics'
import LogViewer from '../components/admin/LogViewer'
import ClusterNodes from '../components/admin/ClusterNodes'
import AlertRulesManagement from '../components/admin/AlertRulesManagement'
import SystemConfigPanel from '../components/admin/SystemConfigPanel'
import QuotaTierPanel from '../components/admin/QuotaTierPanel'

export default function AdminPage() {
  const tabItems = [
    { key: 'users', label: <span><UserOutlined />用户管理</span>, children: <UserManagement /> },
    { key: 'status', label: <span><DashboardOutlined />系统状态</span>, children: <SystemStatus /> },
    { key: 'nodes', label: <span><ClusterOutlined />集群节点</span>, children: <ClusterNodes /> },
    { key: 'alerts', label: <span><BellOutlined />告警规则</span>, children: <AlertRulesManagement /> },
    { key: 'config', label: <span><SettingOutlined />系统配置</span>, children: <SystemConfigPanel /> },
    { key: 'quota', label: <span><HddOutlined />配额管理</span>, children: <QuotaTierPanel /> },
    { key: 'history', label: <span><AreaChartOutlined />历史指标</span>, children: <HistoricalMetrics /> },
    { key: 'logs', label: <span><FileTextOutlined />服务器日志</span>, children: <LogViewer /> },
  ]

  return <Tabs defaultActiveKey="users" items={tabItems} size="large" />
}
