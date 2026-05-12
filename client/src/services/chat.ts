import api from './api'

export interface Conversation {
  id: string
  type: string
  name: string
  creator_id: string
  last_msg_seq: number
  unread: number
  last_message: string
  last_msg_type: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  msg_type: string
  seq: number
  created_at: string
}

export async function getConversations(): Promise<Conversation[]> {
  const res = await api.get('/im/conversations')
  return res.data.data
}

export interface GroupMember {
  id: string
  conversation_id: string
  user_id: string
  role: string
  last_read_seq: number
  joined_at: string
}

export async function createConversation(type: string, memberIds: string[], name?: string): Promise<Conversation> {
  const res = await api.post('/im/conversations', { type, member_ids: memberIds, name })
  return res.data.data
}

export async function getGroupMembers(convId: string): Promise<GroupMember[]> {
  const res = await api.get(`/im/conversations/${convId}/members`)
  return res.data.data
}

export async function addGroupMember(convId: string, userId: string): Promise<void> {
  await api.post(`/im/conversations/${convId}/members`, { user_id: userId })
}

export async function removeGroupMember(convId: string, userId: string): Promise<void> {
  await api.delete(`/im/conversations/${convId}/members/${userId}`)
}

export async function leaveGroup(convId: string): Promise<void> {
  await api.post(`/im/conversations/${convId}/leave`)
}

export async function deleteConversation(id: string): Promise<void> {
  await api.delete(`/im/conversations/${id}`)
}

export async function getMessages(convId: string, before?: string, limit = 50): Promise<Message[]> {
  const res = await api.get(`/im/conversations/${convId}/messages`, {
    params: { before: before || undefined, limit },
  })
  return res.data.data
}

// --- Friend APIs ---

export interface FriendRequest {
  id: string
  user_id: string
  friend_id: string
  friend_username: string
  friend_nickname?: string
  friend_avatar?: string
  online?: boolean
  remark?: string
  message?: string
  status: 'pending' | 'accepted' | 'blocked'
  created_at: string
  updated_at: string
}

export interface BlocklistEntry {
  id: string
  user_id: string
  blocked_user_id: string
  blocked_username?: string
  reason: string
  created_at: string
}

export async function sendFriendRequest(friendName: string, message?: string, expiresIn?: number): Promise<FriendRequest> {
  const res = await api.post('/im/friends/requests', { friend_name: friendName, message, expires_in: expiresIn })
  return res.data.data
}

export async function listFriendRequests(): Promise<FriendRequest[]> {
  const res = await api.get('/im/friends/requests')
  return res.data.data
}

export async function acceptRequest(id: string): Promise<Conversation> {
  const res = await api.put(`/im/friends/requests/${id}/accept`)
  return res.data.data
}

export async function rejectRequest(id: string): Promise<void> {
  await api.put(`/im/friends/requests/${id}/reject`)
}

export async function listFriends(): Promise<FriendRequest[]> {
  const res = await api.get('/im/friends')
  return res.data.data
}

export async function removeFriend(friendId: string): Promise<void> {
  await api.delete(`/im/friends/${friendId}`)
}

export async function blockUser(friendId: string, reason?: string): Promise<void> {
  await api.post(`/im/friends/${friendId}/block`, { reason })
}

export async function unblockUser(friendId: string): Promise<void> {
  await api.delete(`/im/friends/${friendId}/block`)
}

export async function getBlocklist(): Promise<BlocklistEntry[]> {
  const res = await api.get('/im/blocklist')
  return res.data.data
}

export async function setFriendRemark(friendId: string, remark: string): Promise<void> {
  await api.put(`/im/friends/${friendId}/remark`, { remark })
}

export async function getOnlineStatus(userIds: string[]): Promise<Record<string, boolean>> {
  const res = await api.get('/im/friends/online', { params: { ids: userIds } })
  return res.data.data
}

export interface LinkPreview {
  url: string
  title: string
  description: string
  image: string
  site_name: string
}

export async function fetchLinkPreview(url: string): Promise<LinkPreview> {
  const res = await api.post('/im/link-preview', { url })
  return res.data.data
}

// --- Chat Export / Import ---

export interface ExportMessage {
  id: string
  conversation_id: string
  sender_id: string
  sender_name: string
  content: string
  msg_type: string
  seq: number
  created_at: string
}

export interface ChatExport {
  version: string
  conversation_id: string
  conversation_type: string
  conversation_name: string
  participants: string[]
  exported_at: string
  exported_by: string
  message_count: number
  last_message_seq: number
  checksum: string
  messages: ExportMessage[]
}

export interface ImportSummary {
  inserted: number
  skipped: number
  total: number
  last_seq: number
}

export async function exportConversation(id: string): Promise<ChatExport> {
  const res = await api.get(`/im/conversations/${id}/export`)
  return res.data
}

export async function importConversation(file: File): Promise<ImportSummary> {
  const form = new FormData()
  form.append('file', file)
  const res = await api.post('/im/conversations/import', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data
}
