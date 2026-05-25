import { useEffect, useState, useCallback, useRef } from 'react'
import { Button, Typography, Tag, Space, Input, Select } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import * as adminApi from '../../services/admin'
import type { LogEntry } from '../../services/admin'

const { Text } = Typography

export default function LogViewer() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [levelFilter, setLevelFilter] = useState<string>('')
  const [requestIdFilter, setRequestIdFilter] = useState<string>('')
  const [requestIdInput, setRequestIdInput] = useState<string>('')
  const [userIdFilter, setUserIdFilter] = useState<string>('')
  const [serviceFilter, setServiceFilter] = useState<string>('')
  const [services, setServices] = useState<string[]>([])
  const [expandedStacks, setExpandedStacks] = useState<Set<number>>(new Set())
  const intRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await adminApi.getLogs({
        level: levelFilter || undefined,
        requestId: requestIdFilter || undefined,
        userId: userIdFilter || undefined,
        service: serviceFilter || undefined,
      })
      setLogs(res.logs)
    } finally {
      setLoading(false)
    }
  }, [levelFilter, requestIdFilter, userIdFilter, serviceFilter])

  useEffect(() => {
    adminApi.getLogServices().then(setServices).catch(() => {})
  }, [])

  useEffect(() => {
    fetchLogs()
    intRef.current = setInterval(fetchLogs, 3000)
    return () => { if (intRef.current) clearInterval(intRef.current) }
  }, [fetchLogs])

  const levelColor: Record<string, string> = {
    debug: 'default',
    info: 'blue',
    warn: 'orange',
    error: 'red',
  }

  const methodColor: Record<string, string> = {
    GET: 'green',
    POST: 'blue',
    PUT: 'orange',
    DELETE: 'red',
    PATCH: 'purple',
  }

  const logDownloadUrl = adminApi.getLogDownloadUrl(new Date().toISOString().slice(0, 10))

  const handleRequestIdSearch = () => {
    setRequestIdFilter(requestIdInput.trim())
  }

  const handleClickRequestId = (rid: string) => {
    setRequestIdFilter(rid)
    setRequestIdInput(rid)
  }

  const handleClearRequestId = () => {
    setRequestIdFilter('')
    setRequestIdInput('')
  }

  const handleClickUserId = (uid: string) => {
    setUserIdFilter(uid)
  }

  const handleClearUserId = () => {
    setUserIdFilter('')
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8 }}>
        <Text strong style={{ fontSize: 16 }}>服务器日志</Text>
        <Space wrap>
          <Button
            type={levelFilter === '' ? 'primary' : 'default'}
            size="small"
            onClick={() => setLevelFilter('')}
          >全部</Button>
          <Button
            type={levelFilter === 'info' ? 'primary' : 'default'}
            size="small"
            onClick={() => setLevelFilter('info')}
          >INFO</Button>
          <Button
            type={levelFilter === 'warn' ? 'primary' : 'default'}
            size="small"
            onClick={() => setLevelFilter('warn')}
          >WARN</Button>
          <Button
            type={levelFilter === 'error' ? 'primary' : 'default'}
            size="small"
            danger={levelFilter === 'error'}
            onClick={() => setLevelFilter('error')}
          >ERROR</Button>
          {services.length > 0 && (
            <Select
              size="small"
              placeholder="服务"
              value={serviceFilter || undefined}
              onChange={(v) => setServiceFilter(v || '')}
              allowClear
              style={{ width: 130 }}
              options={[{ label: '当前(实时)', value: '' }, ...services.map((s) => ({ label: s, value: s }))]}
            />
          )}
          <Input
            size="small"
            placeholder="Request ID"
            value={requestIdInput}
            onChange={(e) => setRequestIdInput(e.target.value)}
            onPressEnter={handleRequestIdSearch}
            style={{ width: 110 }}
            allowClear
            onClear={handleClearRequestId}
          />
          {requestIdFilter && (
            <Tag color="geekblue" closable onClose={handleClearRequestId}>
              请求: {requestIdFilter.slice(0, 8)}
            </Tag>
          )}
          {userIdFilter && (
            <Tag color="purple" closable onClose={handleClearUserId}>
              uid: {userIdFilter.slice(-8)}
            </Tag>
          )}
          <Button icon={<ReloadOutlined />} onClick={fetchLogs}>刷新</Button>
          <Button icon={<DownloadOutlined />} size="small" href={logDownloadUrl}>下载日志</Button>
        </Space>
      </div>

      <div style={{ background: '#1e1e1e', color: '#d4d4d4', borderRadius: 8, padding: 16, maxHeight: 600, overflow: 'auto', fontFamily: 'monospace', fontSize: 13 }}>
        {logs.length === 0 && !loading && (
          <div style={{ color: '#888' }}>暂无日志</div>
        )}
        {logs.map((entry, i) => {
          const hasStack = !!entry.stack
          const isExpanded = expandedStacks.has(i)
          return (
            <div key={i}>
              <div style={{ lineHeight: 1.8, whiteSpace: 'nowrap', cursor: hasStack ? 'pointer' : 'default' }}
                   onClick={() => {
                     if (hasStack) {
                       setExpandedStacks(prev => {
                         const next = new Set(prev)
                         if (next.has(i)) next.delete(i)
                         else next.add(i)
                         return next
                       })
                     }
                   }}>
                <span style={{ color: '#569cd6' }}>{new Date(entry.timestamp).toLocaleTimeString()}</span>
                {' '}
                <Tag color={levelColor[entry.level] || 'default'} style={{ fontSize: 11, lineHeight: '16px' }}>{entry.level.toUpperCase()}</Tag>
                {' '}
                {entry.service && <Tag style={{ fontSize: 11, lineHeight: '16px' }}>{entry.service}</Tag>}
                {' '}
                {entry.method && <Tag color={methodColor[entry.method] || 'default'} style={{ fontSize: 10, lineHeight: '15px' }}>{entry.method}</Tag>}
                {' '}
                {entry.path && <span style={{ color: '#ce9178', fontSize: 12 }} title={entry.path}>{entry.path.length > 40 ? entry.path.slice(0, 40) + '...' : entry.path}</span>}
                {' '}
                {entry.request_id && (
                  <Tag
                    color="geekblue"
                    style={{ fontSize: 10, lineHeight: '15px', cursor: 'pointer', maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis' }}
                    title={`点击追踪请求 ${entry.request_id}`}
                    onClick={(e) => { e.stopPropagation(); handleClickRequestId(entry.request_id) }}
                  >{entry.request_id.slice(0, 8)}</Tag>
                )}
                {' '}
                {entry.user_id && (
                  <Tag
                    color="purple"
                    style={{ fontSize: 10, lineHeight: '15px', cursor: 'pointer' }}
                    title="点击按用户过滤"
                    onClick={(e) => { e.stopPropagation(); handleClickUserId(entry.user_id) }}
                  >uid:{entry.user_id}</Tag>
                )}
                {' '}
                <span style={{ color: '#888', fontSize: 12 }}>{entry.caller}</span>
                {' '}
                <span>{entry.message}</span>
                {hasStack && <span style={{ color: '#f14c4c', marginLeft: 4, fontSize: 11 }}>{isExpanded ? '[-]' : '[+]'}</span>}
              </div>
              {hasStack && isExpanded && (
                <pre style={{ background: '#2d2d2d', color: '#ce9178', margin: '4px 0 4px 20px', padding: 8, borderRadius: 4, fontSize: 11, whiteSpace: 'pre-wrap', wordBreak: 'break-all', maxHeight: 200, overflow: 'auto' }}>
                  {entry.stack}
                </pre>
              )}
            </div>
          )
        })}
        {loading && <div style={{ color: '#888' }}>加载中...</div>}
      </div>
    </div>
  )
}
