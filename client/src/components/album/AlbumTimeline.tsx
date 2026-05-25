import { useMemo } from 'react'
import { Typography } from 'antd'
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

export default function AlbumTimeline({ files, onPreview }: Props) {
  const groups = useMemo(() => {
    const map = new Map<string, FileItem[]>()
    files.forEach((f) => {
      const date = f.created_at ? new Date(f.created_at) : new Date()
      const month = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
      if (!map.has(month)) map.set(month, [])
      map.get(month)!.push(f)
    })
    return Array.from(map.entries()).sort(([a], [b]) => b.localeCompare(a))
  }, [files])

  if (!groups.length) {
    return <Text type="secondary">暂无照片</Text>
  }

  return (
    <div>
      {groups.map(([month, items]) => (
        <div key={month} style={{ marginBottom: 24 }}>
          <Text strong style={{ fontSize: 15, color: colors.text, marginBottom: 8, display: 'block' }}>
            {month}
          </Text>
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
