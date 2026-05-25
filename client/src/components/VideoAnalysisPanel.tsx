import { useState, useRef } from 'react'
import { Upload, Button, Select, Table, Tag, Space, message, Spin, Typography } from 'antd'
import { VideoCameraOutlined, InboxOutlined, DeleteOutlined } from '@ant-design/icons'
import { detectVideo, type VideoDetectResponse, type VideoDetection } from '../services/camera'
import { colors } from '../theme/tokens'

const { Dragger } = Upload
const { Text } = Typography

interface Props {
  open: boolean
}

export default function VideoAnalysisPanel({ open }: Props) {
  const [videoFile, setVideoFile] = useState<File | null>(null)
  const [videoUrl, setVideoUrl] = useState<string>('')
  const [interval, setInterval] = useState(2)
  const [analyzing, setAnalyzing] = useState(false)
  const [result, setResult] = useState<VideoDetectResponse | null>(null)
  const videoRef = useRef<HTMLVideoElement>(null)

  const handleFile = (file: File) => {
    if (videoUrl) URL.revokeObjectURL(videoUrl)
    setVideoFile(file)
    setVideoUrl(URL.createObjectURL(file))
    setResult(null)
    return false
  }

  const handleRemove = () => {
    if (videoUrl) URL.revokeObjectURL(videoUrl)
    setVideoFile(null)
    setVideoUrl('')
    setResult(null)
  }

  const handleAnalyze = async () => {
    if (!videoFile) return
    setAnalyzing(true)
    try {
      const res = await detectVideo(videoFile, interval)
      setResult(res)
      if (res.detections.length === 0) {
        message.info('未检测到物体')
      } else {
        message.success(`分析完成：${res.frames_analyzed} 帧，${res.detections.length} 个检测帧`)
      }
    } catch (e: any) {
      message.error(e?.response?.data?.message || '视频分析失败')
    } finally {
      setAnalyzing(false)
    }
  }

  const handleSeek = (time: number) => {
    if (videoRef.current) {
      videoRef.current.currentTime = time
      videoRef.current.play()
    }
  }

  const detColumns = [
    {
      title: '时间', dataIndex: 'time', key: 'time', width: 100,
      render: (t: number) => (
        <Button type="link" size="small" onClick={() => handleSeek(t)}>
          {t}s
        </Button>
      ),
    },
    {
      title: '检测物体', dataIndex: 'objects', key: 'objects',
      render: (objs: VideoDetection['objects']) => (
        <Space size={4} wrap>
          {objs.map((o, i) => (
            <Tag key={i} color="blue">{o.class} ({Math.round(o.confidence * 100)}%)</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '数量', key: 'count', width: 60,
      render: (_: any, r: VideoDetection) => r.objects.length,
    },
  ]

  if (!open) return null

  return (
    <div style={{ padding: '0 0 24px 0' }}>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
        <VideoCameraOutlined style={{ fontSize: 20 }} />
        <Text strong style={{ fontSize: 16 }}>AI 视频分析</Text>
      </div>

      {!videoUrl ? (
        <div>
          <Dragger
            accept="video/*"
            showUploadList={false}
            beforeUpload={handleFile}
            style={{ padding: 24 }}
          >
            <p className="ant-upload-drag-icon"><InboxOutlined /></p>
            <p style={{ fontWeight: 500 }}>点击或拖拽视频文件到此区域</p>
            <p style={{ color: colors.textSecondary }}>支持 MP4、AVI、MOV、MKV 等常见格式</p>
          </Dragger>
        </div>
      ) : (
        <div style={{ display: 'flex', gap: 24 }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Space>
                <Text>{videoFile?.name}</Text>
                <Button size="small" icon={<DeleteOutlined />} onClick={handleRemove}>移除</Button>
              </Space>
              <Space>
                <span style={{ color: colors.textSecondary }}>采样间隔</span>
                <Select value={interval} onChange={setInterval} size="small" style={{ width: 100 }}>
                  <Select.Option value={0.5}>0.5秒</Select.Option>
                  <Select.Option value={1}>1秒</Select.Option>
                  <Select.Option value={2}>2秒</Select.Option>
                  <Select.Option value={5}>5秒</Select.Option>
                  <Select.Option value={10}>10秒</Select.Option>
                </Select>
                <Button type="primary" onClick={handleAnalyze} loading={analyzing}>
                  开始分析
                </Button>
              </Space>
            </div>

            <video
              ref={videoRef}
              src={videoUrl}
              controls
              style={{ width: '100%', maxHeight: 400, borderRadius: 8, background: '#000' }}
            />

            {result && (
              <div style={{ marginTop: 16 }}>
                <div style={{ marginBottom: 8 }}>
                  <Space>
                    <Tag>时长 {result.video_duration}s</Tag>
                    <Tag>FPS {result.fps}</Tag>
                    <Tag>分析 {result.frames_analyzed} 帧</Tag>
                    <Tag color="blue">{result.detections.length} 个检测帧</Tag>
                  </Space>
                </div>
                <Table
                  dataSource={result.detections}
                  columns={detColumns}
                  rowKey="time"
                  size="small"
                  pagination={{ pageSize: 20, showSizeChanger: false }}
                  locale={{ emptyText: '未检测到物体' }}
                />
              </div>
            )}

            {analyzing && (
              <div style={{ textAlign: 'center', padding: 40 }}>
                <Spin size="large" />
                <div style={{ marginTop: 12, color: colors.textSecondary }}>正在分析视频...</div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
