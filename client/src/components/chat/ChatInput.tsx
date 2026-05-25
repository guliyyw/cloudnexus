import { useRef } from 'react'
import { Input, Button } from 'antd'
import {
  SendOutlined, PaperClipOutlined,
  PictureOutlined,
} from '@ant-design/icons'

interface ChatInputProps {
  value: string
  onChange: (val: string) => void
  onSend: () => void
  onPaste: (e: React.ClipboardEvent) => void
  onImageUpload: (file: File) => void
  onFilePickerOpen: () => void
  uploadingImg: boolean
}

export default function ChatInput({
  value,
  onChange,
  onSend,
  onPaste,
  onImageUpload,
  onFilePickerOpen,
  uploadingImg,
}: ChatInputProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)

  return (
    <div style={{ padding: '12px 16px', borderTop: '1px solid rgba(255,255,255,0.06)', display: 'flex', gap: 8 }}>
      <Input.TextArea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onPressEnter={(e) => { e.preventDefault(); onSend() }}
        onPaste={onPaste}
        placeholder="输入消息... (可直接粘贴图片)"
        autoSize={{ minRows: 1, maxRows: 4 }}
        style={{ flex: 1 }}
      />
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*,video/*"
        style={{ display: 'none' }}
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) onImageUpload(file)
          e.target.value = ''
        }}
      />
      <Button type="text" icon={<PictureOutlined />} title="发送图片/视频"
        loading={uploadingImg}
        onClick={() => fileInputRef.current?.click()} />
      <Button type="text" icon={<PaperClipOutlined />} title="发送文件"
        onClick={onFilePickerOpen} />
      <Button type="primary" icon={<SendOutlined />} onClick={onSend}>发送</Button>
    </div>
  )
}
