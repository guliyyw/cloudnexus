import { Tabs } from 'antd'
import {
  BellOutlined,
  ClusterOutlined,
  DashboardOutlined,
  FileTextOutlined,
  HddOutlined,
  SettingOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useSearchParams } from 'react-router-dom'
import UserManagement from '../components/admin/UserManagement'
import LogViewer from '../components/admin/LogViewer'
import ClusterNodes from '../components/admin/ClusterNodes'
import AlertRulesManagement from '../components/admin/AlertRulesManagement'
import SystemConfigPanel from '../components/admin/SystemConfigPanel'
import QuotaTierPanel from '../components/admin/QuotaTierPanel'
import ServiceStatusPage from './ServiceStatusPage'

const ADMIN_TABS = new Set(['users', 'status', 'nodes', 'alerts', 'config', 'quota', 'logs'])

export default function AdminPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const requestedTab = searchParams.get('tab') || 'users'
  const activeTab = ADMIN_TABS.has(requestedTab) ? requestedTab : 'users'

  const tabItems = [
    { key: 'users', label: <span><UserOutlined />用户管理</span>, children: <UserManagement /> },
    { key: 'status', label: <span><DashboardOutlined />系统状态</span>, children: <ServiceStatusPage /> },
    { key: 'nodes', label: <span><ClusterOutlined />集群节点</span>, children: <ClusterNodes /> },
    { key: 'alerts', label: <span><BellOutlined />告警规则</span>, children: <AlertRulesManagement /> },
    { key: 'config', label: <span><SettingOutlined />系统配置</span>, children: <SystemConfigPanel /> },
    { key: 'quota', label: <span><HddOutlined />配额管理</span>, children: <QuotaTierPanel /> },
    { key: 'logs', label: <span><FileTextOutlined />服务器日志</span>, children: <LogViewer /> },
  ]

  return (
    <Tabs
      activeKey={activeTab}
      items={tabItems}
      size="large"
      onChange={(tab) => setSearchParams(tab === 'users' ? {} : { tab })}
    />
  )
}
