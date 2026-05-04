import { create } from 'zustand'
import * as chatApi from '../services/chat'
import type { Conversation, Message } from '../services/chat'

interface ChatState {
  conversations: Conversation[]
  currentConvId: number | null
  messages: Message[]
  loading: boolean

  fetchConversations: () => Promise<void>
  createConv: (targetUserId: number) => Promise<void>
  selectConv: (id: number) => void
  fetchMessages: (convId: number, before?: number) => Promise<void>
  addMessage: (msg: Message) => void
  deleteConversation: (id: number) => Promise<void>
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  currentConvId: null,
  messages: [],
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

  createConv: async (targetUserId: number) => {
    const conv = await chatApi.createConversation('private', [targetUserId])
    await get().fetchConversations()
    set({ currentConvId: conv.id })
    get().fetchMessages(conv.id)
  },

  selectConv: (id: number) => {
    set({ currentConvId: id, messages: [] })
    get().fetchMessages(id)
  },

  fetchMessages: async (convId: number, before?: number) => {
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

  deleteConversation: async (id: number) => {
    await chatApi.deleteConversation(id)
    const state = get()
    if (state.currentConvId === id) {
      set({ currentConvId: null, messages: [] })
    }
    await get().fetchConversations()
  },
}))
