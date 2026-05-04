import { create } from 'zustand'
import * as chatApi from '../services/chat'
import type { Conversation, Message, GroupMember } from '../services/chat'

interface ChatState {
  conversations: Conversation[]
  currentConvId: string | null
  messages: Message[]
  members: GroupMember[]
  loading: boolean

  fetchConversations: () => Promise<void>
  createConv: (targetUserId: string) => Promise<void>
  createGroup: (name: string, memberIds: string[]) => Promise<void>
  selectConv: (id: string) => void
  fetchMessages: (convId: string, before?: string) => Promise<void>
  addMessage: (msg: Message) => void
  deleteConversation: (id: string) => Promise<void>
  fetchMembers: (convId: string) => Promise<void>
  addMember: (convId: string, userId: string) => Promise<void>
  removeMember: (convId: string, userId: string) => Promise<void>
  leaveGroup: (convId: string) => Promise<void>
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  currentConvId: null,
  messages: [],
  members: [],
  loading: false,

  fetchConversations: async () => {
    set({ loading: true })
    try {
      const convs = await chatApi.getConversations()
      set({ conversations: convs })
    } finally {
      set({ loading: false })
    }
  },

  createConv: async (targetUserId: string) => {
    const conv = await chatApi.createConversation('private', [targetUserId])
    await get().fetchConversations()
    set({ currentConvId: conv.id })
    get().fetchMessages(conv.id)
  },

  createGroup: async (name: string, memberIds: string[]) => {
    const conv = await chatApi.createConversation('group', memberIds, name)
    await get().fetchConversations()
    set({ currentConvId: conv.id })
    get().fetchMessages(conv.id)
    get().fetchMembers(conv.id)
  },

  selectConv: (id: string) => {
    set({ currentConvId: id, messages: [], members: [] })
    get().fetchMessages(id)
    const conv = get().conversations.find((c) => c.id === id)
    if (conv?.type === 'group') {
      get().fetchMembers(id)
    }
  },

  fetchMessages: async (convId: string, before?: string) => {
    const msgs = await chatApi.getMessages(convId, before, 50)
    set((state) => ({
      messages: before ? [...msgs, ...state.messages] : msgs,
    }))
  },

  addMessage: (msg: Message) => {
    set((state) => {
      const exists = state.messages.some((m) => m.id === msg.id)
      if (exists) return state
      return { messages: [...state.messages, msg] }
    })
  },

  deleteConversation: async (id: string) => {
    await chatApi.deleteConversation(id)
    const state = get()
    if (state.currentConvId === id) {
      set({ currentConvId: null, messages: [], members: [] })
    }
    await get().fetchConversations()
  },

  fetchMembers: async (convId: string) => {
    const members = await chatApi.getGroupMembers(convId)
    set({ members })
  },

  addMember: async (convId: string, userId: string) => {
    await chatApi.addGroupMember(convId, userId)
    get().fetchMembers(convId)
  },

  removeMember: async (convId: string, userId: string) => {
    await chatApi.removeGroupMember(convId, userId)
    get().fetchMembers(convId)
  },

  leaveGroup: async (convId: string) => {
    await chatApi.leaveGroup(convId)
    set({ currentConvId: null, messages: [], members: [] })
    await get().fetchConversations()
  },
}))
