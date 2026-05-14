import { useState } from 'react'
import { Tabs } from 'antd'
import { VideoCameraOutlined, SmileOutlined, ClockCircleOutlined } from '@ant-design/icons'
import CameraListPage from './CameraListPage'
import FaceLibraryPage from './FaceLibraryPage'
import FaceAttendancePage from './FaceAttendancePage'

export default function CameraPage() {
  const [activeTab, setActiveTab] = useState('cameras')

  const tabStyle = (key: string): React.CSSProperties => ({
    display: activeTab === key ? 'block' : 'none',
  })

  return (
    <Tabs
      activeKey={activeTab}
      onChange={setActiveTab}
      style={{ marginTop: -8 }}
      items={[
        {
          key: 'cameras',
          label: (
            <span>
              <VideoCameraOutlined style={{ marginRight: 6 }} />
              摄像头
            </span>
          ),
          children: <div style={tabStyle('cameras')}><CameraListPage /></div>,
        },
        {
          key: 'faces',
          label: (
            <span>
              <SmileOutlined style={{ marginRight: 6 }} />
              人脸库
            </span>
          ),
          children: <div style={tabStyle('faces')}><FaceLibraryPage /></div>,
        },
        {
          key: 'attendance',
          label: (
            <span>
              <ClockCircleOutlined style={{ marginRight: 6 }} />
              考勤
            </span>
          ),
          children: <div style={tabStyle('attendance')}><FaceAttendancePage /></div>,
        },
      ]}
    />
  )
}
