import { useEffect, useState, useRef } from 'react'
import {
  Input, Button, Typography, Avatar,
  message, Modal, Space, Checkbox, Divider,
  List,
} from 'antd'
import {
  PlusOutlined, UserOutlined,
  TeamOutlined, PictureOutlined,
} from '@ant-design/icons'
import { useChatStore } from '../stores/chatStore'
import { useAuthStore } from '../stores/authStore'
import { useFriendStore } from '../stores/friendStore'
import { useWebSocket } from '../hooks/useWebSocket'
import { useNavigate } from 'react-router-dom'
import type { FriendRequest } from '../services/chat'
import FilePickerModal from '../components/FilePickerModal'
import { getDownloadUrl, getPreviewUrl, uploadFile, getFileList, createDirectory } from '../services/file'
import { fetchLinkPreview, exportConversation, importConversation } from '../services/chat'
import { getAlbums, addFilesToAlbum } from '../services/album'
import { usePlayerStore } from '../stores/playerStore'
import type { Track } from '../services/music'
import type { FileItem } from '../services/file'
import type { Album } from '../services/album'
import ConversationList from '../components/chat/ConversationList'
import MessageArea from '../components/chat/MessageArea'
import MemberPanel from '../components/chat/MemberPanel'
import type { LinkPreview } from '../components/chat/MessageArea'

const { Text } = Typography

function getFriendUserId(f: FriendRequest, myId?: string): string {
  if (!myId) return ''
  return f.user_id === myId ? f.friend_id : f.user_id
}

export default function ChatPage() {
  const navigate = useNavigate()
  const {
    conversations, currentConvId, messages, members, loading,
    fetchConversations, createConv, createGroup, selectConv, addMessage, deleteConversation,
    addMember, removeMember, leaveGroup, incrementUnread, updateLastMessage,
  } = useChatStore()
  const { user } = useAuthStore()
  const { friends, fetchFriends } = useFriendStore()

  const [inputText, setInputText] = useState('')
  const [friendModalVisible, setFriendModalVisible] = useState(false)
  const [groupModalVisible, setGroupModalVisible] = useState(false)
  const [groupName, setGroupName] = useState('')
  const [selectedFriends, setSelectedFriends] = useState<string[]>([])
  const [memberModalVisible, setMemberModalVisible] = useState(false)
  const [filePickerVisible, setFilePickerVisible] = useState(false)
  const [uploadingImg, setUploadingImg] = useState(false)
  const [importModalVisible, setImportModalVisible] = useState(false)
  const [importResult, setImportResult] = useState<{ inserted: number; skipped: number; total: number } | null>(null)
  const importFileRef = useRef<HTMLInputElement>(null)
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)
  const [linkPreviews, setLinkPreviews] = useState<Record<string, LinkPreview>>({})
  const fetchedMsgIds = useRef<Set<string>>(new Set())
  const [albumPickerOpen, setAlbumPickerOpen] = useState(false)
  const [albums, setAlbums] = useState<Album[]>([])
  const [addToAlbumFileId, setAddToAlbumFileId] = useState<string>('')
  const play = usePlayerStore((s) => s.play)

  // URL regex for link detection
  const urlRegex = /https?:\/\/[^\s<]+[^\s<.,;:!?)}\]'"`>]/g

  const detectUrls = (text: string): string[] => {
    const matches = text.match(urlRegex)
    return matches ? [...new Set(matches)] : []
  }

  // Fetch link previews in useEffect (not in render) to avoid infinite loop
  useEffect(() => {
    if (messages.length === 0) return
    messages.forEach((msg) => {
      if (msg.msg_type !== 'text') return
      const urls = detectUrls(msg.content)
      if (urls.length === 0 || fetchedMsgIds.current.has(msg.id)) return
      fetchedMsgIds.current.add(msg.id)
      fetchLinkPreview(urls[0]).then((data) => {
        setLinkPreviews((prev) => ({ ...prev, [msg.id]: { url: urls[0], title: data.title, description: data.description, image: data.image, site_name: data.site_name } }))
      }).catch(() => {
        // Cache empty result to avoid re-fetch
        setLinkPreviews((prev) => ({ ...prev, [msg.id]: { url: urls[0], title: '', description: '', image: '', site_name: '' } }))
      })
    })
  }, [messages])

  useEffect(() => {
    fetchConversations()
    if (user) {
      fetchFriends()
    }
  }, [user, fetchConversations, fetchFriends])

  const { sendMessage } = useWebSocket((wsMsg) => {
    if (wsMsg.type === 'message') {
      const msgConvId = wsMsg.conversation_id!
      updateLastMessage(msgConvId, wsMsg.content!, wsMsg.msg_type || 'text')
      if (msgConvId === currentConvId) {
        addMessage({
          id: wsMsg.id!,
          conversation_id: msgConvId,
          sender_id: wsMsg.sender_id!,
          content: wsMsg.content!,
          msg_type: wsMsg.msg_type || 'text',
          seq: 0,
          created_at: wsMsg.created_at || new Date().toISOString(),
        })
      } else {
        incrementUnread(msgConvId)
      }
    }
  })

  useEffect(() => {
    if (!currentConvId || messages.length === 0) return
    const lastMsg = messages[messages.length - 1]
    if (lastMsg.seq > 0) {
      sendMessage({
        type: 'read_receipt',
        conversation_id: currentConvId,
        last_read_msg_id: String(lastMsg.seq),
      })
    }
  }, [currentConvId, messages.length])

  const handleSend = () => {
    if (!inputText.trim() || !currentConvId) return
    sendMessage({
      type: 'message',
      conversation_id: currentConvId,
      content: inputText.trim(),
      msg_type: 'text',
    })
    setInputText('')
  }

  const handleStartChat = async (friendId: string) => {
    await createConv(friendId)
    setFriendModalVisible(false)
    message.success('会话已打开')
  }

  const handleCreateGroup = async () => {
    if (!groupName.trim()) {
      message.warning('请输入群名称')
      return
    }
    if (selectedFriends.length === 0) {
      message.warning('请至少选择一位好友')
      return
    }
    await createGroup(groupName.trim(), selectedFriends)
    setGroupModalVisible(false)
    setGroupName('')
    setSelectedFriends([])
    message.success('群聊已创建')
  }

  const handleAddMember = async (friendId: string) => {
    if (!currentConvId) return
    await addMember(currentConvId, friendId)
    setMemberModalVisible(false)
    message.success('已添加成员')
  }

  const handleRemoveMember = async (userId: string) => {
    if (!currentConvId) return
    await removeMember(currentConvId, userId)
    message.success('已移除成员')
  }

  const handleSendFile = (file: FileItem) => {
    if (!currentConvId) return
    sendMessage({
      type: 'message',
      conversation_id: currentConvId,
      content: JSON.stringify({
        file_id: file.id,
        file_name: file.name,
        file_size: file.size,
        mime_type: file.mime_type,
      }),
      msg_type: 'file',
    })
    setFilePickerVisible(false)
  }

  const handleImgUpload = async (file: File) => {
    if (!currentConvId) return
    setUploadingImg(true)
    try {
      const uploaded = await uploadFile(file, '0')
      const isVideo = file.type.startsWith('video/')
      sendMessage({
        type: 'message',
        conversation_id: currentConvId,
        content: JSON.stringify({
          file_id: uploaded.id,
          file_name: uploaded.name,
          file_size: uploaded.size,
          mime_type: uploaded.mime_type,
          url: getPreviewUrl(uploaded.id),
          download_url: getDownloadUrl(uploaded.id),
        }),
        msg_type: isVideo ? 'video' : 'image',
      })
    } catch {
      message.error('图片上传失败')
    } finally {
      setUploadingImg(false)
    }
  }

  const handlePaste = (e: React.ClipboardEvent) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.type.startsWith('image/')) {
        e.preventDefault()
        const file = item.getAsFile()
        if (file) handleImgUpload(file)
        return
      }
    }
  }

  const ensureChatBackupDir = async (subDirName: string): Promise<string> => {
    const rootList = await getFileList('0', 1, 100)
    let chatDir = rootList.items.find((f) => f.is_dir && f.name === '聊天记录')
    if (!chatDir) {
      chatDir = await createDirectory('聊天记录', '0')
    }
    const subList = await getFileList(chatDir.id, 1, 100)
    let subDir = subList.items.find((f) => f.is_dir && f.name === subDirName)
    if (!subDir) {
      subDir = await createDirectory(subDirName, chatDir.id)
    }
    return subDir.id
  }

  const handleExport = async () => {
    if (!currentConvId || !currentConv) return
    setExporting(true)
    try {
      const data = await exportConversation(currentConvId)
      const jsonStr = JSON.stringify(data, null, 2)
      const blob = new Blob([jsonStr], { type: 'application/json' })
      // Browser download
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      const convName = data.conversation_name || currentConvId
      a.download = `${convName}_${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
      // Upload to cloud
      try {
        const subDir = currentConv.type === 'group' ? '群聊' : '私聊'
        const parentId = await ensureChatBackupDir(subDir)
        const file = new File([blob], a.download, { type: 'application/json' })
        await uploadFile(file, parentId)
        message.success('已导出并保存到云盘')
      } catch {
        message.success('已导出到本地')
      }
    } catch {
      message.error('导出失败')
    } finally {
      setExporting(false)
    }
  }

  const handleImport = async (file: File) => {
    if (!currentConvId) return
    setImporting(true)
    try {
      const summary = await importConversation(file)
      setImportResult(summary)
      setImportModalVisible(true)
    } catch {
      message.error('导入失败，请检查文件格式和校验码')
    } finally {
      setImporting(false)
    }
  }

  const handleOpenAlbumPicker = async (fileId: string) => {
    setAddToAlbumFileId(fileId)
    setAlbumPickerOpen(true)
    try {
      const res = await getAlbums(1, 100)
      setAlbums(res.albums)
    } catch { /* ignore */ }
  }

  const handleAddToAlbum = async (albumId: string) => {
    try {
      await addFilesToAlbum(albumId, [addToAlbumFileId])
      message.success('已添加到相册')
      setAlbumPickerOpen(false)
    } catch {
      message.error('添加失败')
    }
  }

  const handlePlayInMusic = (fc: { file_id: string; file_name: string; mime_type: string; file_size: number }) => {
    const track: Track = {
      id: fc.file_id,
      title: fc.file_name,
      artist: '',
      album: '',
      duration: 0,
      source: 'cloud',
      mime_type: fc.mime_type,
      file_size: fc.file_size,
    }
    play(track)
    message.success('已添加到播放队列')
  }

  const handleLeaveGroup = async () => {
    if (!currentConvId) return
    Modal.confirm({
      title: '退出群聊',
      content: '确定要退出该群聊吗？',
      onOk: async () => {
        await leaveGroup(currentConvId)
        message.success('已退出群聊')
      },
    })
  }

  const currentConv = conversations.find((c) => c.id === currentConvId)
  const isGroup = currentConv?.type === 'group'
  const myMember = members.find((m) => m.user_id === user?.id)
  const isOwner = myMember?.role === 'owner'

  // Friends not already in group
  const memberIds = new Set(members.map((m) => m.user_id))
  const addableFriends = friends.filter((f) => {
    const fid = getFriendUserId(f, user?.id)
    return !memberIds.has(fid)
  })

  return (
    <div style={{ display: 'flex', height: '100%', gap: 16 }}>
      {/* Conversation List */}
      <ConversationList
        conversations={conversations}
        currentConvId={currentConvId}
        loading={loading}
        onSelectConv={selectConv}
        onDeleteConv={deleteConversation}
        onCreateGroup={() => setGroupModalVisible(true)}
        onAddFriend={() => setFriendModalVisible(true)}
        onNavigateFriends={() => navigate('/friends')}
      />

      {/* Chat Area */}
      <MessageArea
        currentConv={currentConv}
        currentConvId={currentConvId}
        messages={messages}
        userId={user?.id}
        inputText={inputText}
        uploadingImg={uploadingImg}
        exporting={exporting}
        importing={importing}
        linkPreviews={linkPreviews}
        importFileRef={importFileRef}
        onInputChange={setInputText}
        onSend={handleSend}
        onPaste={handlePaste}
        onImageUpload={handleImgUpload}
        onFilePickerOpen={() => setFilePickerVisible(true)}
        onExport={handleExport}
        onImportClick={() => importFileRef.current?.click()}
        onImportFile={handleImport}
        onOpenAlbumPicker={handleOpenAlbumPicker}
        onPlayInMusic={handlePlayInMusic}
      />

      {/* Member Panel for Group Chat */}
      {isGroup && (
        <MemberPanel
          members={members}
          userId={user?.id}
          isOwner={!!isOwner}
          onRemoveMember={handleRemoveMember}
          onAddMemberClick={() => setMemberModalVisible(true)}
          onLeaveGroup={handleLeaveGroup}
        />
      )}

      {/* Friend Selection Modal */}
      <Modal
        title="选择好友"
        open={friendModalVisible}
        onCancel={() => setFriendModalVisible(false)}
        footer={null}
      >
        {friends.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Text type="secondary">暂无好友</Text>
            <div style={{ marginTop: 12 }}>
              <Button type="primary" icon={<TeamOutlined />}
                onClick={() => { setFriendModalVisible(false); navigate('/friends') }}>
                前往好友页面添加
              </Button>
            </div>
          </div>
        ) : (
          <List
            dataSource={friends}
            renderItem={(f) => (
              <List.Item
                style={{ cursor: 'pointer', padding: '8px 12px', borderRadius: 6 }}
                onClick={() => handleStartChat(getFriendUserId(f, user?.id))}
              >
                <List.Item.Meta
                  avatar={<Avatar icon={<UserOutlined />} />}
                  title={f.friend_username || `用户 ${getFriendUserId(f, user?.id)}`}
                />
              </List.Item>
            )}
          />
        )}
      </Modal>

      {/* Group Creation Modal */}
      <Modal
        title="创建群聊"
        open={groupModalVisible}
        onCancel={() => { setGroupModalVisible(false); setGroupName(''); setSelectedFriends([]) }}
        onOk={handleCreateGroup}
        okText="创建"
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Input
            placeholder="群名称"
            value={groupName}
            onChange={(e) => setGroupName(e.target.value)}
          />
          <Divider style={{ margin: 0 }} />
          <Text strong>选择成员</Text>
          {friends.length === 0 ? (
            <Text type="secondary">暂无好友，请先添加好友</Text>
          ) : (
            <Checkbox.Group
              style={{ width: '100%' }}
              value={selectedFriends}
              onChange={(values) => setSelectedFriends(values as string[])}
            >
              <List
                dataSource={friends}
                renderItem={(f) => {
                  const fid = getFriendUserId(f, user?.id)
                  return (
                    <List.Item style={{ padding: '4px 0' }}>
                      <Checkbox value={fid}>
                        <Space>
                          <Avatar icon={<UserOutlined />} size="small" />
                          {f.friend_username || fid}
                        </Space>
                      </Checkbox>
                    </List.Item>
                  )
                }}
              />
            </Checkbox.Group>
          )}
        </Space>
      </Modal>

      {/* File Picker Modal */}
      <FilePickerModal
        open={filePickerVisible}
        onOk={handleSendFile}
        onCancel={() => setFilePickerVisible(false)}
      />

      {/* Import Result Modal */}
      <Modal
        title="导入结果"
        open={importModalVisible}
        onCancel={() => { setImportModalVisible(false); setImportResult(null) }}
        footer={<Button type="primary" onClick={() => { setImportModalVisible(false); setImportResult(null) }}>确定</Button>}
      >
        {importResult && (
          <div style={{ textAlign: 'center', padding: '16px 0' }}>
            <p style={{ fontSize: 16 }}>总计 {importResult.total} 条消息</p>
            <p style={{ color: '#52c41a', fontSize: 16 }}>
              成功导入 {importResult.inserted} 条
            </p>
            {importResult.skipped > 0 && (
              <p style={{ color: '#888', fontSize: 14 }}>
                跳过 {importResult.skipped} 条 (已存在)
              </p>
            )}
          </div>
        )}
      </Modal>

      {/* Album Picker Modal */}
      <Modal
        title="选择相册"
        open={albumPickerOpen}
        onCancel={() => setAlbumPickerOpen(false)}
        footer={null}
        width={400}
      >
        <List
          dataSource={albums}
          locale={{ emptyText: '暂无相册，请先创建相册' }}
          renderItem={(album) => (
            <List.Item
              style={{ cursor: 'pointer', padding: '8px 12px', borderRadius: 6 }}
              onClick={() => handleAddToAlbum(album.id)}
            >
              <List.Item.Meta
                avatar={<PictureOutlined style={{ fontSize: 20 }} />}
                title={<Text strong>{album.name}</Text>}
                description={`${album.file_count} 个文件`}
              />
            </List.Item>
          )}
        />
      </Modal>

      {/* Add Member Modal */}
      <Modal
        title="添加成员"
        open={memberModalVisible}
        onCancel={() => setMemberModalVisible(false)}
        footer={null}
      >
        {addableFriends.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Text type="secondary">所有好友已在群中</Text>
          </div>
        ) : (
          <List
            dataSource={addableFriends}
            renderItem={(f) => {
              const fid = getFriendUserId(f, user?.id)
              return (
                <List.Item
                  style={{ cursor: 'pointer', padding: '8px 12px', borderRadius: 6 }}
                  onClick={() => handleAddMember(fid)}
                >
                  <List.Item.Meta
                    avatar={<Avatar icon={<UserOutlined />} />}
                    title={f.friend_username || fid}
                  />
                  <Button type="primary" size="small" icon={<PlusOutlined />}>添加</Button>
                </List.Item>
              )
            }}
          />
        )}
      </Modal>
    </div>
  )
}
