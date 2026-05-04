import api from './api'

export interface Conversation {
  id: string
  type: string
  name: string
  creator_id: string
  last_msg_seq: number
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
  status: 'pending' | 'accepted' | 'blocked'
  created_at: string
  updated_at: string
}

export async function sendFriendRequest(friendName: string): Promise<FriendRequest> {
  const res = await api.post('/im/friends/requests', { friend_name: friendName })
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
