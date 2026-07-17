import { useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Col,
  Collapse,
  Empty,
  Form,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
  Upload,
  message,
} from 'antd'
import {
  CopyOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  ImportOutlined,
  LeftOutlined,
  PlusOutlined,
  RightOutlined,
  SaveOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import type { UploadProps } from 'antd'
import { useAccess } from '../hooks/useAccess'
import {
  DramaAsset,
  DramaDetail,
  DramaProject,
  DramaSetting,
  DramaTask,
  appendDramaStoryboards,
  batchImportDramaAudio,
  createDramaProject,
  createDramaTask,
  deleteDramaProject,
  exportDramaProject,
  getDramaProject,
  getDramaSetting,
  importDramaAssets,
  importDramaProject,
  listDramaProjects,
  parseDramaScript,
  saveDramaSetting,
  updateDramaAsset,
  updateDramaProject,
  updateDramaStoryboard,
  uploadDramaAssetReference,
  uploadStoryboardAudio,
} from '../services/drama'
import { getPreviewUrl } from '../services/file'

const { Text, Title } = Typography
const { TextArea } = Input

const defaultSuffix = '皮肤毛孔清晰，光影颗粒，新国风写实风格，电影级构图，细节丰富，高清质感'

export default function DramaPage() {
  const { hasPermission, loading: accessLoading } = useAccess()
  const [projects, setProjects] = useState<DramaProject[]>([])
  const [detail, setDetail] = useState<DramaDetail | null>(null)
  const [keyword, setKeyword] = useState('')
  const [sort, setSort] = useState('updated_desc')
  const [loading, setLoading] = useState(false)
  const [script, setScript] = useState('')
  const [currentIndex, setCurrentIndex] = useState(0)
  const [projectModalOpen, setProjectModalOpen] = useState(false)
  const [suffixModalOpen, setSuffixModalOpen] = useState(false)
  const [assetImportOpen, setAssetImportOpen] = useState(false)
  const [suffix, setSuffix] = useState(defaultSuffix)
  const [setting, setSetting] = useState<DramaSetting | null>(null)
  const [aiAssetText, setAiAssetText] = useState('')
  const [activeTab, setActiveTab] = useState('script')
  const [projectForm] = Form.useForm()
  const [settingForm] = Form.useForm()

  const canRead = hasPermission('drama:read')
  const canWrite = hasPermission('drama:write')
  const canGenerate = hasPermission('drama:generate')
  const canAdmin = hasPermission('drama:admin')
  const current = detail?.storyboards[currentIndex]
  const modifiedCount = useMemo(() => detail?.storyboards.filter((item) => item.modified).length || 0, [detail])

  useEffect(() => {
    if (!accessLoading && canRead) {
      loadProjects()
      loadSetting()
    }
  }, [accessLoading, canRead, keyword, sort])

  const loadProjects = async () => {
    setLoading(true)
    try {
      const data = await listDramaProjects({ keyword, sort })
      setProjects(data.items || [])
      if (!detail && data.items?.length) {
        await openProject(data.items[0].id)
      }
    } catch {
      message.error('读取项目失败')
    } finally {
      setLoading(false)
    }
  }

  const loadSetting = async () => {
    try {
      const data = await getDramaSetting()
      setSetting(data)
      settingForm.setFieldsValue(data)
    } catch {
      // Keep the workbench usable if settings are unavailable.
    }
  }

  const openProject = async (id: string) => {
    setLoading(true)
    try {
      const data = await getDramaProject(id)
      setDetail(data)
      setScript(data.project.raw_script || '')
      setCurrentIndex(0)
    } catch {
      message.error('打开项目失败')
    } finally {
      setLoading(false)
    }
  }

  const refreshDetail = async () => {
    if (!detail) return
    setDetail(await getDramaProject(detail.project.id))
  }

  const handleCreate = async () => {
    const values = await projectForm.validateFields()
    const project = await createDramaProject(values)
    setProjectModalOpen(false)
    projectForm.resetFields()
    await loadProjects()
    await openProject(project.id)
  }

  const handleParse = async () => {
    if (!detail || !script.trim()) return
    const data = await parseDramaScript(detail.project.id, script)
    setDetail(data)
    setCurrentIndex(0)
    message.success('剧本已拆分为分镜')
  }

  const handleSaveStoryboard = async () => {
    if (!detail || !current) return
    const storyboard = await updateDramaStoryboard(detail.project.id, current.id, { content: current.content, prompt: current.prompt })
    setDetail({ ...detail, storyboards: detail.storyboards.map((item) => (item.id === storyboard.id ? storyboard : item)) })
    message.success('分镜已保存')
  }

  const handleUploadCurrentAudio = async (file: File) => {
    if (!detail || !current) return Upload.LIST_IGNORE
    const storyboard = await uploadStoryboardAudio(detail.project.id, current.id, file)
    setDetail({ ...detail, storyboards: detail.storyboards.map((item) => (item.id === storyboard.id ? storyboard : item)) })
    message.success('音频已导入，并生成字幕')
    return Upload.LIST_IGNORE
  }

  const updateCurrentContent = (content: string) => {
    if (!detail || !current) return
    setDetail({
      ...detail,
      storyboards: detail.storyboards.map((item, index) => (
        index === currentIndex ? { ...item, content, modified: content !== item.original } : item
      )),
    })
  }

  const handleAppend = async () => {
    if (!detail) return
    const storyboards = await appendDramaStoryboards(detail.project.id, suffix)
    setDetail({ ...detail, storyboards })
    setSuffixModalOpen(false)
    message.success('已追加到全部分镜')
  }

  const copyText = async (text: string) => {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  const copyAll = async () => {
    if (!detail) return
    await copyText([detail.project.preface, ...detail.storyboards.map((item) => item.content)].filter(Boolean).join('\n\n'))
  }

  const handleExport = async () => {
    if (!detail) return
    const blob = await exportDramaProject(detail.project.id)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${detail.project.title}-短剧项目.json`
    a.click()
    URL.revokeObjectURL(url)
    message.success('已导出，并保存到云盘 exports 目录')
  }

  const handleCreateTask = async (type: string, payload?: Record<string, unknown>) => {
    if (!detail) return
    const taskPayload = payload || {
      source: 'storyboards',
      source_label: `全部分镜（${detail.storyboards.length} 镜）`,
      storyboard_count: detail.storyboards.length,
      storyboard_titles: detail.storyboards.map((item) => item.title),
    }
    const task = await createDramaTask(detail.project.id, {
      type,
      storyboard_ids: detail.storyboards.map((item) => item.id),
      payload: JSON.stringify(taskPayload),
    })
    setDetail({ ...detail, tasks: [task, ...detail.tasks] })
    message.success('任务已创建')
  }

  const handleBatchAudioImport = async (files: File[]) => {
    if (!detail || !files.length) return
    const results = await batchImportDramaAudio(detail.project.id, files)
    const matched = results.filter((item) => item.matched)
    await refreshDetail()
    message.success(`已匹配 ${matched.length}/${results.length} 个音频文件`)
  }

  const handleSaveSetting = async () => {
    const values = await settingForm.validateFields()
    const next = await saveDramaSetting(values)
    setSetting(next)
    message.success('设置已保存')
  }

  const buildAssetPrompt = () => {
    if (!detail) return ''
    return `你是短剧视觉资产分析师。请根据下面的短剧前言和分镜内容，提取角色与场景资产，输出严格 JSON，不要输出解释文字。

输出格式：
{
  "characters": [
    {
      "name": "角色名",
      "age": "年龄或年龄段",
      "appearance": "脸型、发型、五官、体态等稳定外貌",
      "clothing": "常见服装、颜色、配饰",
      "personality": "气质与表演状态",
      "voice_suggestion": "适合的中文音色建议",
      "voice_name": "可选：zh-CN-XiaoxiaoNeural 或 zh-CN-YunxiNeural 等",
      "reference_prompt": "可直接发给 ComfyUI/SD 的角色参考图提示词"
    }
  ],
  "scenes": [
    {
      "name": "场景名",
      "environment": "空间结构、陈设、时代、地域",
      "lighting": "光线、天气、时间",
      "style": "视觉风格",
      "reference_prompt": "可直接发给 ComfyUI/SD 的场景参考图提示词"
    }
  ]
}

要求：
1. 角色名称保持短且稳定，避免别名重复。
2. 外貌、服装、场景细节要适合做连续分镜一致性参考。
3. reference_prompt 使用中文，包含写实摄影/影视感/一致性关键词。
4. 只输出 JSON。

前言：
${detail.project.preface || '无'}

分镜：
${detail.storyboards.map((item) => item.content).join('\n\n')}`
  }

  const handleImportAIAssets = async () => {
    if (!detail || !aiAssetText.trim()) return
    const assets = await importDramaAssets(detail.project.id, aiAssetText)
    setDetail({ ...detail, assets })
    setAssetImportOpen(false)
    setAiAssetText('')
    message.success(`已导入 ${assets.length} 个资产`)
  }

  const importProps: UploadProps = {
    showUploadList: false,
    beforeUpload: async (file) => {
      const project = await importDramaProject(file)
      await loadProjects()
      await openProject(project.id)
      message.success('项目已导入')
      return Upload.LIST_IGNORE
    },
  }

  const handleDeleteProject = async (id: string) => {
    await deleteDramaProject(id)
    setDetail(null)
    await loadProjects()
  }

  if (!accessLoading && !canRead) {
    return <Empty description="你还没有短剧工坊权限，请联系管理员授权 drama:* 权限" />
  }

  const showStoryboardNav = activeTab === 'storyboards' && !!detail?.storyboards.length

  const scrollAreaStyle: React.CSSProperties = {
    height: '100%',
    minHeight: 0,
    overflow: 'auto',
    paddingRight: 4,
    paddingBottom: 12,
  }

  return (
    <div style={{ height: '100%', minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <Row gutter={16} style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
        <Col xs={24} lg={7} xl={6} style={{ height: '100%', minHeight: 0 }}>
          <div style={{ width: '100%', height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>
            <Space.Compact style={{ width: '100%' }}>
              <Input.Search placeholder="搜索项目" value={keyword} onChange={(e) => setKeyword(e.target.value)} allowClear />
              <Select
                value={sort}
                onChange={setSort}
                style={{ width: 128 }}
                options={[
                  { value: 'updated_desc', label: '最近更新' },
                  { value: 'created_desc', label: '最新创建' },
                  { value: 'title_asc', label: '标题 A-Z' },
                ]}
              />
            </Space.Compact>
            <Space wrap>
              <Button type="primary" icon={<PlusOutlined />} disabled={!canWrite} onClick={() => setProjectModalOpen(true)}>新建</Button>
              <Upload {...importProps} disabled={!canWrite}>
                <Button icon={<ImportOutlined />} disabled={!canWrite}>导入</Button>
              </Upload>
            </Space>
            <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
              <List
                loading={loading}
                dataSource={projects}
                locale={{ emptyText: '暂无项目' }}
              renderItem={(item) => (
                <List.Item
                  onClick={() => openProject(item.id)}
                  style={{
                    cursor: 'pointer',
                    padding: 12,
                    borderRadius: 8,
                    background: detail?.project.id === item.id ? '#fef3e7' : '#fff',
                    marginBottom: 8,
                    border: '1px solid #f0eeeb',
                  }}
                  actions={[
                    <Popconfirm title="删除项目？" onConfirm={(e) => { e?.stopPropagation(); handleDeleteProject(item.id) }} disabled={!canWrite}>
                      <Button size="small" icon={<DeleteOutlined />} disabled={!canWrite} onClick={(e) => e.stopPropagation()} />
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta title={item.title} description={item.description || '无描述'} />
                </List.Item>
              )}
              />
            </div>
          </div>
        </Col>

        <Col xs={24} lg={17} xl={18} style={{ height: '100%', minHeight: 0 }}>
          {!detail ? (
            <Empty description="选择或新建一个短剧项目" />
          ) : (
            <div style={{ height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column', gap: 16 }}>
              <Card style={{ flexShrink: 0 }}>
                <Row gutter={[12, 12]} align="middle">
                  <Col flex="auto">
                    <Title level={4} style={{ margin: 0 }}>{detail.project.title}</Title>
                    <Text type="secondary">{detail.project.description || '短剧工坊'}</Text>
                  </Col>
                  <Col>
                    <Space wrap>
                      <Tag color="blue">{detail.storyboards.length} 镜</Tag>
                      <Tag color={modifiedCount ? 'orange' : 'default'}>{modifiedCount} 已修改</Tag>
                      <Button icon={<CopyOutlined />} onClick={copyAll}>复制全部</Button>
                      <Button icon={<DownloadOutlined />} onClick={handleExport}>导出</Button>
                    </Space>
                  </Col>
                </Row>
              </Card>

              <Tabs
                className="drama-workbench-tabs"
                activeKey={activeTab}
                onChange={setActiveTab}
                style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}
                tabBarStyle={{ marginBottom: 12 }}
                items={[
                  {
                    key: 'script',
                    label: '剧本解析',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Card>
                          <Space direction="vertical" size={12} style={{ width: '100%' }}>
                          <TextArea
                            value={script}
                            onChange={(e) => setScript(e.target.value)}
                            placeholder="粘贴完整剧本，系统会按【分镜序号】：数字/总数字自动拆分"
                            autoSize={{ minRows: 12, maxRows: 24 }}
                          />
                          <Button type="primary" icon={<EditOutlined />} disabled={!canWrite} onClick={handleParse}>自动分镜解析</Button>
                          </Space>
                        </Card>
                      </div>
                    ),
                  },
                  {
                    key: 'storyboards',
                    label: '分镜编辑',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Row gutter={[16, 16]}>
                        <Col xs={24} xl={7}>
                          <Collapse
                            items={[{
                              key: 'preface',
                              label: '前言 / 人物场景档案',
                              children: <TextArea value={detail.project.preface} autoSize={{ minRows: 8, maxRows: 18 }} onChange={(e) => setDetail({ ...detail, project: { ...detail.project, preface: e.target.value } })} onBlur={() => updateDramaProject(detail.project.id, { preface: detail.project.preface })} />,
                            }]}
                          />
                          <List
                            style={{ marginTop: 12 }}
                            dataSource={detail.storyboards}
                            renderItem={(item, index) => (
                              <List.Item
                                onClick={() => setCurrentIndex(index)}
                                style={{
                                  cursor: 'pointer',
                                  padding: 10,
                                  borderRadius: 8,
                                  background: index === currentIndex ? '#fef3e7' : '#fff',
                                  border: '1px solid #f0eeeb',
                                  marginBottom: 8,
                                }}
                              >
                                <Space>
                                  <Tag color={item.modified ? 'orange' : item.audio_file_id !== '0' ? 'green' : 'default'}>{item.seq}</Tag>
                                  <Text>{item.title}</Text>
                                </Space>
                              </List.Item>
                            )}
                          />
                        </Col>
                        <Col xs={24} xl={17}>
                          {current ? (
                            <Card
                              title={<Space><Text strong>{current.title}</Text>{current.modified && <Tag color="orange">已修改</Tag>}</Space>}
                              extra={
                                <Space>
                                  <Upload showUploadList={false} accept="audio/*" beforeUpload={handleUploadCurrentAudio} disabled={!canWrite}>
                                    <Button icon={<UploadOutlined />} disabled={!canWrite}>导入音频</Button>
                                  </Upload>
                                  <Button icon={<CopyOutlined />} onClick={() => copyText(current.content)}>复制本镜</Button>
                                  <Button icon={<SaveOutlined />} type="primary" disabled={!canWrite} onClick={handleSaveStoryboard}>保存</Button>
                                </Space>
                              }
                            >
                              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                                <Space wrap>
                                  <Tag color={current.audio_file_id !== '0' ? 'green' : 'default'}>
                                    {current.audio_file_id !== '0' ? `已配音 ${Math.round((current.audio_duration_ms || 0) / 1000)}s` : '未配音'}
                                  </Tag>
                                  {current.subtitle_ass && <Button size="small" icon={<CopyOutlined />} onClick={() => copyText(current.subtitle_ass)}>复制字幕 ASS</Button>}
                                </Space>
                                {current.subtitle_ass && (
                                  <Collapse size="small" items={[{ key: 'subtitle', label: '字幕预览', children: <TextArea value={current.subtitle_ass} readOnly autoSize={{ minRows: 5, maxRows: 10 }} /> }]} />
                                )}
                                <TextArea value={current.content} onChange={(e) => updateCurrentContent(e.target.value)} autoSize={{ minRows: 22, maxRows: 38 }} />
                              </Space>
                            </Card>
                          ) : (
                            <Empty description="还没有分镜，请先解析剧本" />
                          )}
                        </Col>
                        </Row>
                      </div>
                    ),
                  },
                  {
                    key: 'assets',
                    label: '角色与场景资产',
                    children: (
                      <div style={{ ...scrollAreaStyle, width: '100%' }}>
                        <Space wrap>
                          <Button icon={<CopyOutlined />} onClick={() => copyText(buildAssetPrompt())}>复制 AI 资产解析提示词</Button>
                          <Button icon={<ImportOutlined />} disabled={!canWrite} onClick={() => setAssetImportOpen(true)}>粘贴 AI 结果导入</Button>
                        </Space>
                        <div style={{ marginTop: 12 }}>
                          <AssetPanel
                            detail={detail}
                            canWrite={canWrite}
                            canGenerate={canGenerate}
                            onChange={setDetail}
                          />
                        </div>
                      </div>
                    ),
                  },
                  {
                    key: 'tasks',
                    label: '生成任务',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Card>
                        <Space direction="vertical" size={16} style={{ width: '100%' }}>
                          <Space wrap>
                            <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => handleCreateTask('tts')}>创建 TTS 任务</Button>
                            <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => handleCreateTask('image')}>创建图片任务</Button>
                            <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => handleCreateTask('video')}>创建视频任务</Button>
                            <Upload
                              multiple
                              showUploadList={false}
                              accept="audio/*"
                              beforeUpload={() => false}
                              customRequest={() => undefined}
                              onChange={({ fileList }) => {
                                const files = fileList.map((item) => item.originFileObj).filter(Boolean) as File[]
                                handleBatchAudioImport(files)
                              }}
                              disabled={!canWrite}
                            >
                              <Button icon={<ImportOutlined />} disabled={!canWrite}>批量导入音频</Button>
                            </Upload>
                          </Space>
                          <List
                            dataSource={detail.tasks}
                            locale={{ emptyText: '暂无生成任务' }}
                            renderItem={(item) => (
                              <List.Item>
                                <List.Item.Meta
                                  title={<Space><Tag>{item.type}</Tag><Tag color={item.status === 'failed' ? 'red' : item.status === 'done' ? 'green' : 'blue'}>{item.status}</Tag></Space>}
                                  description={
                                    <Space direction="vertical" size={2}>
                                      <Text type="secondary">来源：{getTaskSource(item, detail)}</Text>
                                      <Text type="secondary">{item.message || `${item.progress}%`}</Text>
                                    </Space>
                                  }
                                />
                              </List.Item>
                            )}
                          />
                        </Space>
                        </Card>
                      </div>
                    ),
                  },
                  {
                    key: 'settings',
                    label: '系统设置',
                    children: (
                      <div style={scrollAreaStyle}>
                        <Card>
                        <Form form={settingForm} layout="vertical" initialValues={setting || undefined}>
                          <Row gutter={16}>
                            <Col xs={24} md={12}>
                              <Form.Item name="comfyui_url" label="ComfyUI API 地址">
                                <Input placeholder="http://localhost:8188" />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={12}>
                              <Form.Item name="tts_engine" label="TTS 引擎">
                                <Select options={[{ value: 'edge-tts', label: 'edge-tts' }, { value: 'azure', label: 'Azure Speech' }]} />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={12}>
                              <Form.Item name="storage_root" label="云盘根目录">
                                <Input prefix={<SettingOutlined />} />
                              </Form.Item>
                            </Col>
                            <Col xs={24}>
                              <Form.Item name="tts_config" label="TTS 配置 JSON">
                                <TextArea autoSize={{ minRows: 4 }} />
                              </Form.Item>
                            </Col>
                            <Col xs={24}>
                              <Form.Item name="video_settings" label="视频默认参数 JSON">
                                <TextArea autoSize={{ minRows: 5 }} />
                              </Form.Item>
                            </Col>
                          </Row>
                          <Button type="primary" icon={<SaveOutlined />} disabled={!canAdmin} onClick={handleSaveSetting}>保存设置</Button>
                        </Form>
                        </Card>
                      </div>
                    ),
                  },
                ]}
              />
            </div>
          )}
        </Col>
      </Row>

      {showStoryboardNav ? (
        <div style={{ flexShrink: 0, minHeight: 56, background: '#fff', borderTop: '1px solid #f0eeeb', zIndex: 10, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12, marginTop: 12, padding: '8px 12px', flexWrap: 'wrap' }}>
          <Button icon={<LeftOutlined />} disabled={currentIndex <= 0} onClick={() => setCurrentIndex((value) => Math.max(0, value - 1))} />
          <Text>当前分镜</Text>
          <InputNumber min={1} max={detail.storyboards.length} value={currentIndex + 1} onChange={(value) => setCurrentIndex(Math.max(0, Math.min(detail.storyboards.length - 1, Number(value || 1) - 1)))} />
          <Text>/ {detail.storyboards.length}</Text>
          <Button icon={<RightOutlined />} disabled={currentIndex >= detail.storyboards.length - 1} onClick={() => setCurrentIndex((value) => Math.min(detail.storyboards.length - 1, value + 1))} />
          <Button onClick={() => setSuffixModalOpen(true)} disabled={!canWrite}>批量追加</Button>
        </div>
      ) : null}

      <Modal title="新建短剧项目" open={projectModalOpen} onOk={handleCreate} onCancel={() => setProjectModalOpen(false)}>
        <Form form={projectForm} layout="vertical">
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="批量追加内容" open={suffixModalOpen} onOk={handleAppend} onCancel={() => setSuffixModalOpen(false)}>
        <TextArea value={suffix} onChange={(e) => setSuffix(e.target.value)} autoSize={{ minRows: 5 }} />
      </Modal>

      <Modal title="粘贴 AI 资产解析结果" open={assetImportOpen} onOk={handleImportAIAssets} onCancel={() => setAssetImportOpen(false)} width={760}>
        <TextArea value={aiAssetText} onChange={(e) => setAiAssetText(e.target.value)} placeholder="粘贴 AI 输出的 JSON" autoSize={{ minRows: 14, maxRows: 24 }} />
      </Modal>
    </div>
  )
}

function getTaskSource(task: DramaTask, detail: DramaDetail) {
  try {
    const payload = JSON.parse(task.payload || '{}') as {
      source_label?: string
      asset_type?: string
      name?: string
      storyboard_count?: number
    }
    if (payload.source_label) return payload.source_label
    if (payload.name && payload.asset_type) {
      return `${payload.asset_type === 'character' ? '角色' : '场景'}：${payload.name}`
    }
    if (payload.storyboard_count) return `全部分镜（${payload.storyboard_count} 镜）`
  } catch {
    // Older tasks may contain an empty or non-JSON payload.
  }
  if (task.type === 'asset_reference') return '角色或场景参考图'
  return detail.storyboards.length ? `全部分镜（${detail.storyboards.length} 镜）` : '当前项目'
}

function AssetPanel({
  detail,
  canWrite,
  canGenerate,
  onChange,
}: {
  detail: DramaDetail
  canWrite: boolean
  canGenerate: boolean
  onChange: (detail: DramaDetail) => void
}) {
  const saveAsset = async (asset: DramaAsset) => {
    const next = await updateDramaAsset(detail.project.id, asset.id, { name: asset.name, description: asset.description, voice_name: asset.voice_name })
    onChange({ ...detail, assets: detail.assets.map((item) => (item.id === next.id ? next : item)) })
    message.success('资产已保存')
  }

  const uploadReference = async (asset: DramaAsset, file: File) => {
    const next = await uploadDramaAssetReference(detail.project.id, asset.id, file)
    onChange({ ...detail, assets: detail.assets.map((item) => (item.id === next.id ? next : item)) })
    message.success('参考图已上传，并进入云盘 assets 目录')
  }

  const handlePasteReference = (asset: DramaAsset, event: React.ClipboardEvent<HTMLDivElement>) => {
    const file = Array.from(event.clipboardData.files).find((item) => item.type.startsWith('image/'))
    if (file && canWrite) {
      uploadReference(asset, file)
    }
  }

  const createAssetImageTask = async (asset: DramaAsset) => {
    const task = await createDramaTask(detail.project.id, {
      type: 'asset_reference',
      payload: JSON.stringify({
        asset_id: asset.id,
        asset_type: asset.type,
        name: asset.name,
        source_label: `${asset.type === 'character' ? '角色' : '场景'}：${asset.name}`,
        prompt: asset.description,
      }),
    })
    onChange({ ...detail, tasks: [task, ...detail.tasks] })
    message.success('已创建 ComfyUI 参考图任务')
  }

  return (
    <Row gutter={[16, 16]}>
      {detail.assets.map((asset) => (
        <Col xs={24} md={12} xl={8} key={asset.id}>
          <Card
            title={<Space><Tag color={asset.type === 'character' ? 'purple' : 'cyan'}>{asset.type === 'character' ? '角色' : '场景'}</Tag>{asset.name}</Space>}
            extra={asset.reference_file_id !== '0' ? <Tag color="green">已传参考图</Tag> : <Tag>无参考图</Tag>}
          >
            <Space direction="vertical" style={{ width: '100%' }} onPaste={(event) => handlePasteReference(asset, event)}>
              {asset.reference_file_id !== '0' && (
                <img
                  src={getPreviewUrl(asset.reference_file_id)}
                  alt={asset.name}
                  style={{ width: '100%', height: 180, objectFit: 'cover', borderRadius: 8, border: '1px solid #f0eeeb' }}
                />
              )}
              <Input value={asset.name} disabled={!canWrite} onChange={(e) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, name: e.target.value } : item)) })} />
              {asset.type === 'character' && (
                <Select
                  value={asset.voice_name || undefined}
                  disabled={!canWrite}
                  placeholder="选择角色音色"
                  options={[
                    { value: 'zh-CN-XiaoxiaoNeural', label: 'Xiaoxiao 女声' },
                    { value: 'zh-CN-YunxiNeural', label: 'Yunxi 男声' },
                    { value: 'zh-CN-XiaoyiNeural', label: 'Xiaoyi 女声' },
                    { value: 'zh-CN-YunjianNeural', label: 'Yunjian 男声' },
                  ]}
                  onChange={(value) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, voice_name: value } : item)) })}
                />
              )}
              <TextArea value={asset.description} disabled={!canWrite} autoSize={{ minRows: 4 }} onChange={(e) => onChange({ ...detail, assets: detail.assets.map((item) => (item.id === asset.id ? { ...item, description: e.target.value } : item)) })} />
              <Space wrap>
                <Button icon={<SaveOutlined />} disabled={!canWrite} onClick={() => saveAsset(asset)}>保存</Button>
                <Upload showUploadList={false} disabled={!canWrite} beforeUpload={(file) => { uploadReference(asset, file); return Upload.LIST_IGNORE }}>
                  <Button icon={<UploadOutlined />} disabled={!canWrite}>上传/粘贴参考图</Button>
                </Upload>
                <Button icon={<ThunderboltOutlined />} disabled={!canGenerate} onClick={() => createAssetImageTask(asset)}>ComfyUI 生成</Button>
              </Space>
            </Space>
          </Card>
        </Col>
      ))}
      {!detail.assets.length && <Col span={24}><Empty description="复制 AI 提示词解析剧本，再粘贴结果导入角色与场景资产" /></Col>}
    </Row>
  )
}
