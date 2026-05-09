import { useState, useEffect, useCallback } from 'react'
import { Card, Table, DatePicker, Space, Button, message, Tag, Modal, Tabs, Popconfirm } from 'antd'
import { ReloadOutlined, ClockCircleOutlined, TeamOutlined, DeleteOutlined } from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import type { DailyAttendanceItem, AttendanceSession, AttendanceStatusItem } from '../services/camera'
import {
  getDailyAttendance,
  getAttendanceByFace,
  getAttendanceStatus,
  deleteAttendanceSession,
  clearAttendanceByFaceDate,
} from '../services/camera'

export default function FaceAttendancePage() {
  const [date, setDate] = useState<Dayjs>(dayjs())
  const [items, setItems] = useState<DailyAttendanceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [detailName, setDetailName] = useState('')
  const [detailFaceId, setDetailFaceId] = useState('')
  const [detailSessions, setDetailSessions] = useState<AttendanceSession[]>([])
  const [detailLoading, setDetailLoading] = useState(false)
  const [statusItems, setStatusItems] = useState<AttendanceStatusItem[]>([])
  const [statusLoading, setStatusLoading] = useState(false)
  const [statusSummary, setStatusSummary] = useState({ total: 0, signed: 0, unsigned: 0 })

  const fetchDaily = useCallback(async () => {
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

  const fetchStatus = useCallback(async () => {
    setStatusLoading(true)
    try {
      const res = await getAttendanceStatus(date.format('YYYY-MM-DD'))
      setStatusItems(res.items)
      setStatusSummary({ total: res.total, signed: res.signed_count, unsigned: res.unsigned_count })
    } catch {
      message.error('查询人员状态失败')
    } finally {
      setStatusLoading(false)
    }
  }, [date])

  useEffect(() => { fetchDaily(); fetchStatus() }, [fetchDaily, fetchStatus])

  const showDetail = async (faceId: string, name: string) => {
    setDetailName(name)
    setDetailFaceId(faceId)
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

  const handleDeleteSession = async (id: string) => {
    try {
      await deleteAttendanceSession(id)
      message.success('已删除')
      // Refresh detail and daily
      setDetailSessions(prev => prev.filter(s => s.id !== id))
      fetchDaily()
      fetchStatus()
    } catch {
      message.error('删除失败')
    }
  }

  const handleClearPerson = async (faceId: string) => {
    try {
      await clearAttendanceByFaceDate(faceId, date.format('YYYY-MM-DD'))
      message.success('已清除该人员当日考勤')
      fetchDaily()
      fetchStatus()
      setDetailVisible(false)
    } catch {
      message.error('清除失败')
    }
  }

  const dailyColumns = [
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
      title: '操作', key: 'actions', width: 140,
      render: (_: any, r: DailyAttendanceItem) => (
        <Space size={0}>
          <Button type="link" size="small" onClick={() => showDetail(r.face_id, r.face_name)}>
            详情
          </Button>
          <Popconfirm title={`确定清除 ${r.face_name} ${date.format('YYYY-MM-DD')} 全部考勤？`} onConfirm={() => handleClearPerson(r.face_id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const detailColumns = [
    { title: '开始', dataIndex: 'start_time', key: 'start_time', width: 160,
      render: (t: string) => new Date(t).toLocaleTimeString() },
    { title: '结束', dataIndex: 'end_time', key: 'end_time', width: 160,
      render: (t: string) => new Date(t).toLocaleTimeString() },
    {
      title: '持续', key: 'dur', width: 80,
      render: (_: any, r: AttendanceSession) => {
        const dur = new Date(r.end_time).getTime() - new Date(r.start_time).getTime()
        const mins = Math.round(dur / 60000)
        return mins < 1 ? '<1分钟' : `${mins} 分钟`
      },
    },
    {
      title: '操作', key: 'actions', width: 60,
      render: (_: any, r: AttendanceSession) => (
        <Popconfirm title="确定删除？" onConfirm={() => handleDeleteSession(r.id)}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ]

  const statusColumns = [
    { title: '姓名', dataIndex: 'face_name', key: 'face_name', width: 120 },
    {
      title: '签到状态', dataIndex: 'signed_in', key: 'signed_in', width: 100,
      render: (v: boolean) => v ? <Tag color="green">已签到</Tag> : <Tag color="red">未签到</Tag>,
    },
    {
      title: '签到时间', dataIndex: 'check_in', key: 'check_in', width: 180,
      render: (t: string | null) => t ? <Tag color="green">{new Date(t).toLocaleTimeString()}</Tag> : <span style={{ color: '#999' }}>—</span>,
    },
    {
      title: '签退时间', dataIndex: 'check_out', key: 'check_out', width: 180,
      render: (t: string | null) => t ? <Tag color="orange">{new Date(t).toLocaleTimeString()}</Tag> : <span style={{ color: '#999' }}>—</span>,
    },
    {
      title: '出现次数', dataIndex: 'session_count', key: 'session_count', width: 80,
      render: (c: number) => c > 0 ? `${c} 次` : '—',
    },
    {
      title: '签到时长', key: 'duration', width: 100,
      render: (_: any, r: AttendanceStatusItem) => {
        if (!r.check_in || !r.check_out) return '—'
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
      render: (_: any, r: AttendanceStatusItem) => (
        r.signed_in ? (
          <Button type="link" size="small" onClick={() => showDetail(r.face_id, r.face_name)}>
            详情
          </Button>
        ) : null
      ),
    },
  ]

  return (
    <div>
      <Card
        title={<span><ClockCircleOutlined style={{ marginRight: 8 }} />人脸考勤</span>}
        extra={
          <Space>
            <DatePicker value={date} onChange={(d) => d && setDate(d)} allowClear={false} />
            <Button icon={<ReloadOutlined />} onClick={() => { fetchDaily(); fetchStatus() }}>刷新</Button>
          </Space>
        }
      >
        <Tabs
          defaultActiveKey="daily"
          items={[
            {
              key: 'daily',
              label: `签到记录 (${items.length})`,
              children: (
                <Table
                  dataSource={items}
                  columns={dailyColumns}
                  rowKey="face_id"
                  loading={loading}
                  pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 人签到` }}
                  locale={{ emptyText: '暂无考勤记录。请在摄像头实时画面中开启人脸识别。' }}
                />
              ),
            },
            {
              key: 'personnel',
              label: (
                <span>
                  <TeamOutlined style={{ marginRight: 4 }} />
                  人员管理
                  <Tag color="green" style={{ marginLeft: 8 }}>{statusSummary.signed} 已签到</Tag>
                  {statusSummary.unsigned > 0 && (
                    <Tag color="red" style={{ marginLeft: 4 }}>{statusSummary.unsigned} 未签到</Tag>
                  )}
                </span>
              ),
              children: (
                <Table
                  dataSource={statusItems}
                  columns={statusColumns}
                  rowKey="face_id"
                  loading={statusLoading}
                  pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 人` }}
                  locale={{ emptyText: '人脸库为空。请先在"人脸库"页面注册人脸。' }}
                />
              ),
            },
          ]}
        />
      </Card>

      <Modal
        title={`${detailName} — ${date.format('YYYY-MM-DD')} 签到详情`}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={
          <Popconfirm title={`确定清除 ${detailName} ${date.format('YYYY-MM-DD')} 全部考勤？`} onConfirm={() => handleClearPerson(detailFaceId)}>
            <Button danger icon={<DeleteOutlined />}>清除当日考勤</Button>
          </Popconfirm>
        }
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
