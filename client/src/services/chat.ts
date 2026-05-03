import api from './api'

export interface Conversation {
  id: number
  type: string
  name: string
  creator_id: number
  last_msg_seq: number
  created_at: string
  updated_at: string
}

export interface Message {
  id: number
  conversation_id: number
  sender_id: number
  content: string
  msg_type: string
  seq: number
  created_at: string
}

export async function getConversations(): Promise<Conversation[]> {
  const res = await api.get('/im/conversations')
  return res.data.data
}

export async function createConversation(type: string, memberIds: number[]): Promise<Conversation> {
  const res = await api.post('/im/conversations', { type, member_ids: memberIds })
  return res.data.data
}

export async function deleteConversation(id: number): Promise<void> {
  await api.delete(`/im/conversations/${id}`)
}

export async function getMessages(convId: number, before?: number, limit = 50): Promise<Message[]> {
  const res = await api.get(`/im/conversations/${convId}/messages`, {
    params: { before: before || undefined, limit },
  })
  return res.data.data
}

// --- Friend APIs ---

export interface FriendRequest {
  id: number
  user_id: number
  friend_id: number
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

export async function acceptRequest(id: number): Promise<Conversation> {
  const res = await api.put(`/im/friends/requests/${id}/accept`)
  return res.data.data
}

export async function rejectRequest(id: number): Promise<void> {
  await api.put(`/im/friends/requests/${id}/reject`)
}

export async function listFriends(): Promise<FriendRequest[]> {
  const res = await api.get('/im/friends')
  return res.data.data
}

export async function removeFriend(friendId: number): Promise<void> {
  await api.delete(`/im/friends/${friendId}`)
}
