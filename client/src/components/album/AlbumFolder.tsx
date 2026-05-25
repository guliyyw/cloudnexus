import { useMemo } from 'react'
import { Typography } from 'antd'
import { FolderOutlined } from '@ant-design/icons'
import { colors } from '../../theme/tokens'
import type { FileItem } from '../../services/file'

const { Text } = Typography

interface Props {
  files: FileItem[]
  onPreview: (file: FileItem, index: number) => void
}

function getDownloadUrl(file: FileItem): string {
  const token = localStorage.getItem('access_token')
  return `/api/v1/file/download/${file.id}?token=${token}&inline=true`
}

export default function AlbumFolder({ files, onPreview }: Props) {
  const groups = useMemo(() => {
    const map = new Map<string, FileItem[]>()
    files.forEach((f) => {
      // parent_id 是 ID，无法还原路径，统一归入"我的文件"
      const dir = '我的文件'
      if (!map.has(dir)) map.set(dir, [])
      map.get(dir)!.push(f)
    })
    return Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b))
  }, [files])

  if (!groups.length) {
    return <Text type="secondary">暂无照片</Text>
  }

  return (
    <div>
      {groups.map(([dir, items]) => (
        <div key={dir} style={{ marginBottom: 24 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <FolderOutlined style={{ color: colors.primary, fontSize: 16 }} />
            <Text strong style={{ fontSize: 15, color: colors.text }}>
              {dir}
            </Text>
            <Text type="secondary" style={{ fontSize: 12 }}>
              ({items.length} 张)
            </Text>
          </div>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(120px, 1fr))',
              gap: 8,
            }}
          >
            {items.map((file) => (
              <div
                key={file.id}
                onClick={() => {
                  const globalIdx = files.findIndex((f) => f.id === file.id)
                  onPreview(file, globalIdx)
                }}
                style={{
                  cursor: 'pointer',
                  borderRadius: 6,
                  overflow: 'hidden',
                  aspectRatio: '1',
                  background: 'rgba(255,255,255,0.04)',
                }}
              >
                {file.mime_type?.startsWith('image/') ? (
                  <img
                    src={getDownloadUrl(file)}
                    alt={file.name}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                    loading="lazy"
                  />
                ) : (
                  <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'rgba(255,255,255,0.2)' }}>
                    {file.name}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
