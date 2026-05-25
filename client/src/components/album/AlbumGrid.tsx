import { useState } from 'react'
import { Card, Checkbox, Typography } from 'antd'
import { FileOutlined } from '@ant-design/icons'
import { colors, radius, shadow } from '../../theme/tokens'
import type { FileItem } from '../../services/file'

const { Text } = Typography

interface Props {
  files: FileItem[]
  selectedIds: string[]
  onSelect: (id: string, checked: boolean) => void
  onPreview: (file: FileItem, index: number) => void
}

function getDownloadUrl(file: FileItem): string {
  const token = localStorage.getItem('access_token')
  return `/api/v1/file/download/${file.id}?token=${token}&inline=true`
}

export default function AlbumGrid({ files, selectedIds, onSelect, onPreview }: Props) {
  const [hoveredId, setHoveredId] = useState<string | null>(null)

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))',
        gap: 12,
      }}
    >
      {files.map((file, index) => {
        const isImage = file.mime_type?.startsWith('image/')
        return (
          <div
            key={file.id}
            style={{ position: 'relative' }}
            onMouseEnter={() => setHoveredId(file.id)}
            onMouseLeave={() => setHoveredId(null)}
          >
            <Card
              hoverable
              size="small"
              style={{
                borderRadius: radius.md,
                overflow: 'hidden',
                boxShadow: shadow.card,
                height: '100%',
              }}
              bodyStyle={{ padding: 0 }}
              onClick={() => onPreview(file, index)}
            >
              <div
                style={{
                  width: '100%',
                  paddingTop: '100%',
                  position: 'relative',
                  background: 'rgba(255,255,255,0.04)',
                }}
              >
                {isImage ? (
                  <img
                    src={getDownloadUrl(file)}
                    alt={file.name}
                    style={{
                      position: 'absolute',
                      top: 0,
                      left: 0,
                      width: '100%',
                      height: '100%',
                      objectFit: 'cover',
                    }}
                  />
                ) : (
                  <div
                    style={{
                      position: 'absolute',
                      top: 0,
                      left: 0,
                      width: '100%',
                      height: '100%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 48,
                      color: 'rgba(255,255,255,0.2)',
                    }}
                  >
                    <FileOutlined />
                  </div>
                )}
              </div>
            </Card>
            {(hoveredId === file.id || selectedIds.includes(file.id)) && (
              <Checkbox
                checked={selectedIds.includes(file.id)}
                onChange={(e) => onSelect(file.id, e.target.checked)}
                onClick={(e) => e.stopPropagation()}
                style={{
                  position: 'absolute',
                  top: 4,
                  left: 4,
                  zIndex: 2,
                }}
              />
            )}
            <Text
              ellipsis
              style={{
                fontSize: 11,
                display: 'block',
                padding: '2px 4px',
                color: colors.textSecondary,
              }}
            >
              {file.name}
            </Text>
          </div>
        )
      })}
    </div>
  )
}
