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

export async function getMessages(convId: number, before?: number, limit = 50): Promise<Message[]> {
  const res = await api.get(`/im/conversations/${convId}/messages`, {
    params: { before: before || undefined, limit },
  })
  return res.data.data
}
