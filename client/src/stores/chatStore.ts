import { create } from 'zustand'
import * as chatApi from '../services/chat'
import type { Conversation, Message, GroupMember, MessageSearchResult } from '../services/chat'

interface ChatState {
  conversations: Conversation[]
  currentConvId: string | null
  messages: Message[]
  members: GroupMember[]
  loading: boolean
  searchKeyword: string
  searchResults: MessageSearchResult[]
  searchLoading: boolean
  activeMessageId: string | null

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
  incrementUnread: (convId: string) => void
  updateLastMessage: (convId: string, content: string, msgType: string) => void
  searchMessages: (keyword: string, conversationId?: string) => Promise<void>
  clearSearch: () => void
  jumpToMessage: (conversationId: string, messageId: string) => Promise<void>
  clearActiveMessage: () => void
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  currentConvId: null,
  messages: [],
  members: [],
  loading: false,
  searchKeyword: '',
  searchResults: [],
  searchLoading: false,
  activeMessageId: null,

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
    set((state) => ({
      currentConvId: id,
      messages: [],
      members: [],
      activeMessageId: null,
      conversations: state.conversations.map((c) =>
        c.id === id ? { ...c, unread: 0 } : c
      ),
    }))
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
      if (msg.conversation_id !== state.currentConvId) return state
      const exists = state.messages.some((m) => m.id === msg.id)
      if (exists) return state
      return { messages: [...state.messages, msg] }
    })
  },

  deleteConversation: async (id: string) => {
    await chatApi.deleteConversation(id)
    const state = get()
    if (state.currentConvId === id) {
      set({ currentConvId: null, messages: [], members: [], activeMessageId: null })
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
    set({ currentConvId: null, messages: [], members: [], activeMessageId: null })
    await get().fetchConversations()
  },

  incrementUnread: (convId: string) => {
    set((state) => ({
      conversations: state.conversations.map((c) =>
        c.id === convId ? { ...c, unread: c.unread + 1 } : c
      ),
    }))
  },

  updateLastMessage: (convId: string, content: string, msgType: string) => {
    const preview = msgType === 'text' ? content
      : msgType === 'image' ? '[图片]'
      : msgType === 'video' ? '[视频]'
      : msgType === 'file' ? '[文件]'
      : msgType === 'system' ? content
      : content
    set((state) => ({
      conversations: state.conversations.map((c) =>
        c.id === convId ? { ...c, last_message: preview, last_msg_type: msgType } : c
      ),
    }))
  },

  searchMessages: async (keyword: string, conversationId?: string) => {
    const trimmed = keyword.trim()
    set({ searchKeyword: keyword })
    if (!trimmed) {
      set({ searchResults: [], searchLoading: false, activeMessageId: null })
      return
    }
    set({ searchLoading: true })
    try {
      const res = await chatApi.searchMessages(trimmed, conversationId)
      set({ searchResults: res.items })
    } finally {
      set({ searchLoading: false })
    }
  },

  clearSearch: () => set({ searchKeyword: '', searchResults: [], searchLoading: false, activeMessageId: null }),

  jumpToMessage: async (conversationId: string, messageId: string) => {
    const context = await chatApi.getMessageContext(conversationId, messageId)
    const conv = get().conversations.find((c) => c.id === conversationId)
    set((state) => ({
      currentConvId: conversationId,
      messages: context,
      members: [],
      activeMessageId: messageId,
      conversations: state.conversations.map((c) =>
        c.id === conversationId ? { ...c, unread: 0 } : c
      ),
    }))
    if (conv?.type === 'group') {
      await get().fetchMembers(conversationId)
    }
  },

  clearActiveMessage: () => set({ activeMessageId: null }),
}))
