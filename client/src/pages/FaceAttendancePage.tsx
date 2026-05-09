import { useState, useEffect, useCallback } from 'react'
import { Card, Table, DatePicker, Space, Button, message, Tag, Modal } from 'antd'
import { ReloadOutlined, ClockCircleOutlined, UserOutlined } from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import type { DailyAttendanceItem, AttendanceSession } from '../services/camera'
import { getDailyAttendance, getAttendanceByFace } from '../services/camera'

export default function FaceAttendancePage() {
  const [date, setDate] = useState<Dayjs>(dayjs())
  const [items, setItems] = useState<DailyAttendanceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [detailName, setDetailName] = useState('')
  const [detailSessions, setDetailSessions] = useState<AttendanceSession[]>([])
  const [detailLoading, setDetailLoading] = useState(false)

  const fetch = useCallback(async () => {
    setLoading(true)
    try {
      const list = await getDailyAttendance(date.format('YYYY-MM-DD'))
      setItems(list)
    } catch {
      message.error('查询考勤失败')
    } finally {
      setLoading(false)
    }
  }, [date])

  useEffect(() => { fetch() }, [fetch])

  const showDetail = async (faceId: string, name: string) => {
    setDetailName(name)
    setDetailVisible(true)
    setDetailLoading(true)
    try {
      const sessions = await getAttendanceByFace(faceId, date.format('YYYY-MM-DD'), date.format('YYYY-MM-DD'))
      setDetailSessions(sessions)
    } catch {
      message.error('查询详情失败')
    } finally {
      setDetailLoading(false)
    }
  }

  const columns = [
    { title: '姓名', dataIndex: 'face_name', key: 'face_name', width: 120 },
    {
      title: '签到', dataIndex: 'check_in', key: 'check_in', width: 180,
      render: (t: string) => <Tag color="green">{new Date(t).toLocaleTimeString()}</Tag>,
    },
    {
      title: '签退', dataIndex: 'check_out', key: 'check_out', width: 180,
      render: (t: string) => <Tag color="orange">{new Date(t).toLocaleTimeString()}</Tag>,
    },
    {
      title: '出现次数', dataIndex: 'session_count', key: 'session_count', width: 80,
      render: (c: number) => `${c} 次`,
    },
    {
      title: '签到时长', key: 'duration', width: 100,
      render: (_: any, r: DailyAttendanceItem) => {
        const dur = new Date(r.check_out).getTime() - new Date(r.check_in).getTime()
        const mins = Math.round(dur / 60000)
        if (mins < 60) return `${mins} 分钟`
        const h = Math.floor(mins / 60)
        const m = mins % 60
        return `${h}h ${m}m`
      },
    },
    {
      title: '操作', key: 'actions', width: 80,
      render: (_: any, r: DailyAttendanceItem) => (
        <Button type="link" size="small" onClick={() => showDetail(r.face_id, r.face_name)}>
          详情
        </Button>
      ),
    },
  ]

  const detailColumns = [
    { title: '开始', dataIndex: 'start_time', key: 'start_time', width: 180,
      render: (t: string) => new Date(t).toLocaleTimeString() },
    { title: '结束', dataIndex: 'end_time', key: 'end_time', width: 180,
      render: (t: string) => new Date(t).toLocaleTimeString() },
    {
      title: '持续', key: 'dur', width: 80,
      render: (_: any, r: AttendanceSession) => {
        const dur = new Date(r.end_time).getTime() - new Date(r.start_time).getTime()
        const mins = Math.round(dur / 60000)
        return mins < 1 ? '<1分钟' : `${mins} 分钟`
      },
    },
  ]

  return (
    <div>
      <Card
        title={<span><ClockCircleOutlined style={{ marginRight: 8 }} />人脸考勤记录</span>}
        extra={
          <Space>
            <DatePicker value={date} onChange={(d) => d && setDate(d)} allowClear={false} />
            <Button icon={<ReloadOutlined />} onClick={fetch}>刷新</Button>
          </Space>
        }
      >
        <Table
          dataSource={items}
          columns={columns}
          rowKey="face_id"
          loading={loading}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 人签到` }}
          locale={{ emptyText: `暂无考勤记录。请在摄像头实时画面中开启人脸识别。` }}
          summary={() => (
            <Table.Summary.Row>
              <Table.Summary.Cell index={0}>
                <Tag color="blue"><UserOutlined /> {items.length} 人签到</Tag>
              </Table.Summary.Cell>
              <Table.Summary.Cell index={1} colSpan={5} />
            </Table.Summary.Row>
          )}
        />
      </Card>

      <Modal
        title={`${detailName} — ${date.format('YYYY-MM-DD')} 签到详情`}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={560}
      >
        <Table
          dataSource={detailSessions}
          columns={detailColumns}
          rowKey="id"
          size="small"
          loading={detailLoading}
          pagination={false}
          locale={{ emptyText: '暂无记录' }}
        />
      </Modal>
    </div>
  )
}
