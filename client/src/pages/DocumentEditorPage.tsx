import { useEffect, useMemo, useRef, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { Alert, Button, Dropdown, Input, Segmented, Select, Space, Spin, Table, Tabs, Tag, Typography, message } from 'antd'
import { ArrowLeftOutlined, DownloadOutlined, FilePdfOutlined, SaveOutlined } from '@ant-design/icons'
import { BubbleMenu } from '@tiptap/react/menus'
import { EditorContent, useEditor } from '@tiptap/react'
import Collaboration from '@tiptap/extension-collaboration'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import StarterKit from '@tiptap/starter-kit'
import TaskItem from '@tiptap/extension-task-item'
import TaskList from '@tiptap/extension-task-list'
import { common, createLowlight } from 'lowlight'
import { marked } from 'marked'
import * as mammoth from 'mammoth'
import { renderAsync as renderDocx } from 'docx-preview'
import * as XLSX from 'xlsx'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import { useAuthStore } from '../stores/authStore'
import { downloadFileBlob, exportWordHtml, getFileMeta, getPreviewUrl, getWordPdfUrl, saveFileContent, saveTextFile, saveWordHtml, type FileItem } from '../services/file'
import { colors, radius, shadow, spacing } from '../theme/tokens'

const { Text, Title } = Typography
const lowlight = createLowlight(common)
const userColors = ['#f58742', '#5b9bd5', '#70ad47', '#ff6384', '#9966ff', '#4bc0c0']
const wordDocumentStyles = `
  .word-preview .docx-wrapper {
    background: #eef0f2;
    padding: 24px;
  }
  .word-preview .docx-wrapper > section.docx {
    margin-bottom: 24px;
    box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);
  }
  .word-document {
    color: #1f2933;
    font-family: "Times New Roman", "Microsoft YaHei", serif;
    font-size: 16px;
    line-height: 1.75;
  }
  .word-document h1,
  .word-document h2,
  .word-document h3 {
    color: #111827;
    line-height: 1.35;
    margin: 18px 0 10px;
  }
  .word-document p {
    margin: 0 0 10px;
  }
  .word-document table {
    border-collapse: collapse;
    width: 100%;
    margin: 14px 0;
  }
  .word-document td,
  .word-document th {
    border: 1px solid #d6d9df;
    padding: 7px 9px;
    vertical-align: top;
  }
  .word-document ul,
  .word-document ol {
    padding-left: 26px;
    margin: 8px 0 12px;
  }
  .word-document img {
    max-width: 100%;
    height: auto;
  }
`

type DocKind = 'collab' | 'markdown' | 'word' | 'excel' | 'pdf' | 'unknown'
type SheetState = { name: string; rows: string[][]; merges?: XLSX.Range[] }
function getUserColor(id?: string): string {
  if (!id) return userColors[0]
  let hash = 0
  for (let i = 0; i < id.length; i++) hash = id.charCodeAt(i) + ((hash << 5) - hash)
  return userColors[Math.abs(hash) % userColors.length]
}

function getDocKind(file?: FileItem): DocKind {
  if (!file) return 'unknown'
  const name = file.name.toLowerCase()
  if (file.collab_type === 'doc' || name.endsWith('.clouddoc')) return 'collab'
  if (name.endsWith('.md') || name.endsWith('.markdown') || file.mime_type.includes('markdown')) return 'markdown'
  if (name.endsWith('.docx') || name.endsWith('.doc')) return 'word'
  if (name.endsWith('.xlsx') || name.endsWith('.xls') || name.endsWith('.csv')) return 'excel'
  if (file.mime_type === 'application/pdf' || name.endsWith('.pdf')) return 'pdf'
  return 'unknown'
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}

function buildWorkbook(sheets: SheetState[]) {
  const wb = XLSX.utils.book_new()
  sheets.forEach((sheet) => {
    const rows = sheet.rows.map((row) => row.map((cell) => (
      typeof cell === 'string' && cell.startsWith('=')
        ? { f: cell.slice(1) }
        : cell
    )))
    const ws = XLSX.utils.aoa_to_sheet(rows)
    if (sheet.merges?.length) ws['!merges'] = sheet.merges
    XLSX.utils.book_append_sheet(wb, ws, sheet.name || 'Sheet')
  })
  return wb
}

export default function DocumentEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const user = useAuthStore((state) => state.user)
  const [file, setFile] = useState<FileItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [connStatus, setConnStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const [saveStatus, setSaveStatus] = useState<'idle' | 'syncing' | 'saved' | 'failed'>('idle')
  const [markdown, setMarkdown] = useState('')
  const [wordHtml, setWordHtml] = useState('')
  const [wordMode, setWordMode] = useState<'preview' | 'edit'>('preview')
  const [wordLoading, setWordLoading] = useState(false)
  const [wordPreviewError, setWordPreviewError] = useState(false)
  const [mdMode, setMdMode] = useState<'split' | 'source' | 'preview'>('split')
  const [sheets, setSheets] = useState<SheetState[]>([])
  const [activeSheet, setActiveSheet] = useState('')
  const [sortColumn, setSortColumn] = useState<number | null>(null)
  const [filterText, setFilterText] = useState('')
  const ydoc = useMemo(() => new Y.Doc(), [])
  const saveTimerRef = useRef<number | null>(null)
  const wordEditorRef = useRef<HTMLDivElement | null>(null)
  const wordPreviewRef = useRef<HTMLDivElement | null>(null)
  const kind = getDocKind(file || undefined)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    getFileMeta(id).then(async (meta) => {
      setFile(meta)
      const nextKind = getDocKind(meta)
      if (nextKind === 'markdown') {
        setMarkdown(await (await downloadFileBlob(meta.id)).text())
      }
      if (nextKind === 'excel') {
        const buffer = await (await downloadFileBlob(meta.id)).arrayBuffer()
        const wb = XLSX.read(buffer, { type: 'array', cellFormula: true, cellStyles: true })
        const parsed = wb.SheetNames.map((name) => {
          const ws = wb.Sheets[name]
          return {
            name,
            rows: XLSX.utils.sheet_to_json<string[]>(ws, { header: 1, defval: '', raw: false }),
            merges: ws['!merges'],
          }
        })
        setSheets(parsed.length ? parsed : [{ name: 'Sheet1', rows: [['']] }])
        setActiveSheet(parsed[0]?.name || 'Sheet1')
      }
    }).catch(() => {
      message.error('文档不存在或无权访问')
      navigate('/files')
    }).finally(() => setLoading(false))
  }, [id, navigate])

  useEffect(() => {
    if (!id || loading || kind !== 'collab') return
    const token = localStorage.getItem('access_token')
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const provider = new WebsocketProvider(`${protocol}//${window.location.host}`, `ws/collab/${id}?token=${token}`, ydoc, { connect: true })
    provider.on('status', ({ status }: { status: string }) => {
      const nextStatus = status as 'connecting' | 'connected' | 'disconnected'
      setConnStatus(nextStatus)
    })
    provider.awareness.setLocalState({ user: { name: user?.username || 'anonymous', color: getUserColor(user?.id) } })
    return () => {
      provider.disconnect()
      provider.destroy()
    }
  }, [id, loading, kind, user?.id, user?.username, ydoc])

  useEffect(() => () => ydoc.destroy(), [ydoc])

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false }),
      CodeBlockLowlight.configure({ lowlight }),
      Collaboration.configure({ document: ydoc }),
      TaskList,
      TaskItem.configure({ nested: true }),
    ],
    editorProps: {
      attributes: { style: 'min-height: 520px; padding: 0 4px; outline: none;' },
      handlePaste: (_, event) => {
        const text = event.clipboardData?.getData('text/plain') || ''
        const looksLikeMarkdown = /(^|\n)(#{1,6}\s|\*\s|-\s|\d+\.\s|```|> |\[.*\]\(.*\)|\*\*|__|~~|`[^`]+`)/m.test(text)
        if (!looksLikeMarkdown) return false
        event.preventDefault()
        editor?.chain().focus().insertContent(marked.parse(text) as string).run()
        return true
      },
    },
  }, [ydoc])

  useEffect(() => {
    if (!editor || kind !== 'collab') return
    const handleUpdate = () => {
      setSaveStatus('syncing')
      if (saveTimerRef.current) window.clearTimeout(saveTimerRef.current)
      saveTimerRef.current = window.setTimeout(async () => {
        if (!file) return
        try {
          const saved = await saveTextFile(file.id, editor.getHTML(), 'Collaborative document online edit')
          setFile(saved)
          setSaveStatus('saved')
        } catch {
          setSaveStatus('failed')
        }
      }, 1200)
    }
    editor.on('update', handleUpdate)
    return () => {
      editor.off('update', handleUpdate)
      if (saveTimerRef.current) window.clearTimeout(saveTimerRef.current)
    }
  }, [editor, file, kind])

  useEffect(() => {
    if (!file || kind !== 'word') return
    let cancelled = false

    const loadWord = async () => {
      setWordLoading(true)
      setWordPreviewError(false)
      try {
        const blob = await downloadFileBlob(file.id)
        const buffer = await blob.arrayBuffer()
        const result = await mammoth.convertToHtml(
          { arrayBuffer: buffer },
          {
            convertImage: mammoth.images.imgElement((image) => (
              image.read('base64').then((imageBuffer) => ({
                src: `data:${image.contentType};base64,${imageBuffer}`,
              }))
            )),
          },
        )
        if (cancelled) return
        setWordHtml(result.value || '<p></p>')
        setSaveStatus('idle')

        if (wordPreviewRef.current) {
          wordPreviewRef.current.innerHTML = ''
          await renderDocx(blob, wordPreviewRef.current, undefined, {
            className: 'docx',
            inWrapper: true,
            ignoreWidth: false,
            ignoreHeight: false,
            ignoreFonts: false,
            breakPages: true,
            renderHeaders: true,
            renderFooters: true,
            renderFootnotes: true,
            useBase64URL: true,
          })
        }
      } catch {
        if (!cancelled) {
          setWordPreviewError(true)
          setWordMode('edit')
          message.warning('Word 原格式预览失败，已切换到在线编辑')
        }
      } finally {
        if (!cancelled) setWordLoading(false)
      }
    }

    void loadWord()
    return () => { cancelled = true }
  }, [file?.id, kind])

  useEffect(() => {
    if (!file || kind !== 'collab' || !editor || file.size === 0) return
    downloadFileBlob(file.id)
      .then((blob) => blob.text())
      .then((content) => {
        if (content.trim().startsWith('<')) {
          editor.commands.setContent(content)
        }
      })
      .catch(() => undefined)
  }, [file?.id, kind, editor])

  const handleBack = () => {
    if (location.pathname.startsWith('/files/')) navigate('/files')
    else navigate('/documents')
  }

  const handleSaveMarkdown = async () => {
    if (!file) return
    setSaving(true)
    setSaveStatus('syncing')
    try {
      const saved = await saveTextFile(file.id, markdown, 'Markdown online edit')
      setFile(saved)
      setSaveStatus('saved')
      message.success('已保存 Markdown')
    } catch {
      setSaveStatus('failed')
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleExportWord = async () => {
    if (!file) return
    const html = kind === 'word' ? (wordEditorRef.current?.innerHTML || wordHtml) : editor?.getHTML()
    if (!html) return
    if (kind === 'word') {
      downloadBlob(await exportWordHtml(file.id, html), file.name.replace(/\.docx?$/i, '.docx'))
      return
    }
    message.info('协作文档导出 Word 正在升级')
  }

  const handleSaveWord = async () => {
    if (!file) return
    setSaving(true)
    setSaveStatus('syncing')
    try {
      const html = wordEditorRef.current?.innerHTML || wordHtml
      const saved = await saveWordHtml(file.id, html, 'Word online edit')
      setFile(saved)
      setSaveStatus('saved')
      message.success('已保存 Word')
    } catch {
      setSaveStatus('failed')
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleExportExcel = () => {
    if (!file) return
    const wb = buildWorkbook(sheets)
    XLSX.writeFile(wb, file.name.replace(/\.(xlsx?|csv)$/i, '.xlsx'))
  }

  const handleSaveExcel = async () => {
    if (!file) return
    setSaving(true)
    setSaveStatus('syncing')
    try {
      const wb = buildWorkbook(sheets)
      const data = XLSX.write(wb, { bookType: 'xlsx', type: 'array' })
      const blob = new Blob([data], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
      const saved = await saveFileContent(file.id, blob, file.name.replace(/\.(xlsx?|csv)$/i, '.xlsx'), 'Excel online edit')
      setFile(saved)
      setSaveStatus('saved')
      message.success('已保存 Excel')
    } catch {
      setSaveStatus('failed')
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  const activeSheetState = sheets.find((sheet) => sheet.name === activeSheet) || sheets[0]
  const excelRows = useMemo(() => {
    const rows = activeSheetState?.rows || []
    const body = rows.map((row, rowIndex) => ({ key: rowIndex, rowIndex, cells: row }))
    let next = filterText.trim()
      ? body.filter((row) => row.cells.some((cell) => String(cell).toLowerCase().includes(filterText.toLowerCase())))
      : body
    if (sortColumn !== null) {
      next = [...next].sort((a, b) => String(a.cells[sortColumn] || '').localeCompare(String(b.cells[sortColumn] || '')))
    }
    return next
  }, [activeSheetState, filterText, sortColumn])

  const updateCell = (rowIndex: number, colIndex: number, value: string) => {
    setSheets((prev) => prev.map((sheet) => {
      if (sheet.name !== activeSheetState?.name) return sheet
      const rows = sheet.rows.map((row) => [...row])
      while (rows.length <= rowIndex) rows.push([])
      while (rows[rowIndex].length <= colIndex) rows[rowIndex].push('')
      rows[rowIndex][colIndex] = value
      return { ...sheet, rows }
    }))
  }

  if (loading) return <div style={{ textAlign: 'center', paddingTop: 120 }}><Spin size="large" /></div>

  const status = connStatus === 'connected' ? { color: 'success', text: '已连接' } : connStatus === 'disconnected' ? { color: 'error', text: '已断开' } : { color: 'processing', text: '连接中' }
  const saveTag = saveStatus === 'saved'
    ? { color: 'success', text: '已保存' }
    : saveStatus === 'failed'
      ? { color: 'error', text: '同步失败' }
      : saveStatus === 'syncing'
        ? { color: 'processing', text: '同步中' }
        : { color: 'default', text: '待编辑' }
  const maxCols = Math.max(8, ...(activeSheetState?.rows || []).map((row) => row.length))

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: spacing.lg, maxWidth: 1280, margin: '0 auto' }}>
      <section style={{ borderRadius: radius.lg, padding: '24px 28px', background: colors.surfaceRaised, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: spacing.md, flexWrap: 'wrap' }}>
          <div>
            <Text style={{ color: colors.primary, fontWeight: 700 }}>在线文档编辑器</Text>
            <Title level={3} style={{ margin: '8px 0', color: colors.text }}>{file?.name}</Title>
            <Space wrap>
              <Tag color={kind === 'pdf' ? 'red' : kind === 'excel' ? 'green' : kind === 'markdown' ? 'blue' : 'orange'}>{kind.toUpperCase()}</Tag>
              {kind === 'collab' && <Tag color={status.color}>{status.text}</Tag>}
              {(kind === 'collab' || kind === 'word' || kind === 'markdown') && <Tag color={saveTag.color}>{saveTag.text}</Tag>}
              <Text style={{ color: colors.textSecondary }}>当前编辑者：{user?.username || 'anonymous'}</Text>
            </Space>
          </div>
          <Space wrap>
            <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>返回</Button>
            {kind === 'markdown' && <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSaveMarkdown}>保存</Button>}
            {kind === 'word' && <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSaveWord}>保存</Button>}
            {kind === 'excel' && <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSaveExcel}>保存</Button>}
            {kind === 'word' && <Button icon={<FilePdfOutlined />} href={id ? getWordPdfUrl(id) : undefined} target="_blank">转 PDF</Button>}
            <Dropdown
              menu={{
                items: [
                  ...(kind === 'markdown' ? [{ key: 'md', label: '导出 Markdown', icon: <DownloadOutlined /> }] : []),
                  ...(kind === 'word' || kind === 'collab' ? [{ key: 'docx', label: '导出 Word', icon: <DownloadOutlined /> }] : []),
                  ...(kind === 'excel' ? [{ key: 'xlsx', label: '导出 Excel', icon: <DownloadOutlined /> }] : []),
                ],
                onClick: ({ key }) => {
                  if (key === 'md') downloadBlob(new Blob([markdown], { type: 'text/markdown' }), file?.name.replace(/\.\w+$/i, '.md') || 'document.md')
                  if (key === 'docx') handleExportWord()
                  if (key === 'xlsx') handleExportExcel()
                },
              }}
            >
              <Button icon={<DownloadOutlined />}>导出</Button>
            </Dropdown>
          </Space>
        </div>
      </section>

      {kind === 'pdf' && (
        <iframe src={id ? getPreviewUrl(id) : ''} title="PDF" style={{ width: '100%', height: '72vh', border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.md, background: '#fff' }} />
      )}

      {kind === 'markdown' && (
        <section style={{ borderRadius: radius.lg, padding: 20, background: colors.surface, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card }}>
          <Tabs activeKey={mdMode} onChange={(key) => setMdMode(key as typeof mdMode)} items={[
            { key: 'split', label: '双栏' },
            { key: 'source', label: '源码' },
            { key: 'preview', label: '预览' },
          ]} />
          <div style={{ display: 'grid', gridTemplateColumns: mdMode === 'split' ? '1fr 1fr' : '1fr', gap: spacing.md }}>
            {mdMode !== 'preview' && <Input.TextArea value={markdown} onChange={(e) => setMarkdown(e.target.value)} autoSize={{ minRows: 24 }} style={{ fontFamily: 'Consolas, monospace' }} />}
            {mdMode !== 'source' && <div style={{ minHeight: 520, padding: 18, border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.md, background: '#fff' }} dangerouslySetInnerHTML={{ __html: marked.parse(markdown) as string }} />}
          </div>
        </section>
      )}

      {kind === 'excel' && activeSheetState && (
        <section style={{ borderRadius: radius.lg, padding: 20, background: colors.surface, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card }}>
          <Space wrap style={{ marginBottom: 14 }}>
            <Select value={activeSheet} onChange={setActiveSheet} options={sheets.map((sheet) => ({ value: sheet.name, label: sheet.name }))} style={{ minWidth: 180 }} />
            <Input.Search placeholder="筛选单元格内容" allowClear value={filterText} onChange={(e) => setFilterText(e.target.value)} style={{ width: 240 }} />
            <Select allowClear placeholder="排序列" value={sortColumn ?? undefined} onChange={(value) => setSortColumn(value ?? null)} options={Array.from({ length: maxCols }).map((_, i) => ({ value: i, label: `列 ${i + 1}` }))} style={{ width: 140 }} />
          </Space>
          <Table
            size="small"
            pagination={false}
            scroll={{ x: maxCols * 160, y: 560 }}
            dataSource={excelRows}
            columns={Array.from({ length: maxCols }).map((_, colIndex) => ({
              title: String.fromCharCode(65 + (colIndex % 26)),
              dataIndex: ['cells', colIndex],
              width: 150,
              render: (_: string, row: { rowIndex: number; cells: string[] }) => (
                <Input value={row.cells[colIndex] || ''} onChange={(e) => updateCell(row.rowIndex, colIndex, e.target.value)} bordered={false} />
              ),
            }))}
          />
        </section>
      )}

      {kind === 'word' && (
        <section style={{ borderRadius: radius.lg, padding: 24, background: colors.surface, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card, minHeight: 620, overflow: 'hidden' }}>
          <style>{wordDocumentStyles}</style>
          <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 20 }}>
            <Segmented
              value={wordMode}
              onChange={(value) => setWordMode(value as 'preview' | 'edit')}
              options={[
                { label: '原格式预览', value: 'preview', disabled: wordPreviewError },
                { label: '在线编辑', value: 'edit' },
              ]}
            />
          </div>
          {wordPreviewError && (
            <Alert type="warning" showIcon message="该文件暂时无法按原格式预览，可继续使用在线编辑。" style={{ marginBottom: 16 }} />
          )}
          <div style={{ display: wordMode === 'preview' ? 'block' : 'none', minHeight: 560, overflow: 'auto' }}>
            {wordLoading && <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>}
            <div ref={wordPreviewRef} className="word-preview" style={{ display: wordLoading ? 'none' : 'block' }} />
          </div>
          <div
            ref={wordEditorRef}
            className="word-document"
            contentEditable
            suppressContentEditableWarning
            onInput={() => setSaveStatus('syncing')}
            dangerouslySetInnerHTML={{ __html: wordHtml || '<p></p>' }}
            style={{
              display: wordMode === 'edit' ? 'block' : 'none',
              maxWidth: 860,
              minHeight: 760,
              margin: '0 auto',
              padding: '72px 82px',
              background: '#fff',
              border: `1px solid ${colors.borderSubtle}`,
              boxShadow: '0 16px 40px rgba(15, 23, 42, 0.08)',
              outline: 'none',
            }}
          />
        </section>
      )}

      {(kind === 'collab' || kind === 'unknown') && (
        <section style={{ borderRadius: radius.lg, padding: 24, background: colors.surface, border: `1px solid ${colors.borderSubtle}`, boxShadow: shadow.card, minHeight: 620 }}>
          {editor ? (
            <>
              <BubbleMenu editor={editor}>
                <Space size={4} style={{ background: colors.panelBg, border: `1px solid ${colors.borderSubtle}`, borderRadius: radius.md, padding: 4, boxShadow: shadow.card }}>
                  <Button size="small" type="text" onClick={() => editor.chain().focus().toggleBold().run()}><strong>B</strong></Button>
                  <Button size="small" type="text" onClick={() => editor.chain().focus().toggleItalic().run()}><em>I</em></Button>
                  <Button size="small" type="text" onClick={() => editor.chain().focus().toggleStrike().run()}><span style={{ textDecoration: 'line-through' }}>S</span></Button>
                  <Button size="small" type="text" onClick={() => editor.chain().focus().toggleCode().run()}>{'</>'}</Button>
                </Space>
              </BubbleMenu>
              <EditorContent editor={editor} />
            </>
          ) : <Spin />}
        </section>
      )}
    </div>
  )
}
