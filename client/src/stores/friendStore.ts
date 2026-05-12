import { create } from 'zustand'
import * as chatApi from '../services/chat'
import type { FriendRequest, BlocklistEntry } from '../services/chat'

interface FriendState {
  friends: FriendRequest[]
  pendingRequests: FriendRequest[]
  blocklist: BlocklistEntry[]
  onlineStatus: Record<string, boolean>
  loading: boolean

  fetchFriends: () => Promise<void>
  fetchPendingRequests: () => Promise<void>
  fetchBlocklist: () => Promise<void>
  fetchOnlineStatus: () => Promise<void>
  sendRequest: (friendName: string, message?: string, expiresIn?: number) => Promise<void>
  acceptRequest: (id: string) => Promise<void>
  rejectRequest: (id: string) => Promise<void>
  removeFriend: (friendId: string) => Promise<void>
  blockUser: (friendId: string, reason?: string) => Promise<void>
  unblockUser: (friendId: string) => Promise<void>
  setRemark: (friendId: string, remark: string) => Promise<void>
}

export const useFriendStore = create<FriendState>((set, get) => ({
  friends: [],
  pendingRequests: [],
  blocklist: [],
  onlineStatus: {},
  loading: false,

  fetchFriends: async () => {
    set({ loading: true })
    try {
      const friends = await chatApi.listFriends()
      set({ friends: Array.isArray(friends) ? friends : [] })
    } catch {
      set({ friends: [] })
    } finally {
      set({ loading: false })
      get().fetchOnlineStatus()
    }
  },

  fetchPendingRequests: async () => {
    try {
      const requests = await chatApi.listFriendRequests()
      set({ pendingRequests: Array.isArray(requests) ? requests : [] })
    } catch {
      set({ pendingRequests: [] })
    }
  },

  fetchBlocklist: async () => {
    try {
      const list = await chatApi.getBlocklist()
      set({ blocklist: Array.isArray(list) ? list : [] })
    } catch {
      set({ blocklist: [] })
    }
  },

  fetchOnlineStatus: async () => {
    const { friends } = get()
    const ids = friends.map((f) => f.friend_id)
    if (ids.length === 0) return
    try {
      const status = await chatApi.getOnlineStatus(ids)
      set({ onlineStatus: status || {} })
    } catch { /* ignore */ }
  },

  sendRequest: async (friendName, message?, expiresIn?) => {
    await chatApi.sendFriendRequest(friendName, message, expiresIn)
  },

  acceptRequest: async (id) => {
    await chatApi.acceptRequest(id)
    await get().fetchPendingRequests()
    await get().fetchFriends()
  },

  rejectRequest: async (id) => {
    await chatApi.rejectRequest(id)
    await get().fetchPendingRequests()
  },

  removeFriend: async (friendId) => {
    await chatApi.removeFriend(friendId)
    await get().fetchFriends()
  },

  blockUser: async (friendId, reason?) => {
    await chatApi.blockUser(friendId, reason)
    await get().fetchFriends()
    await get().fetchBlocklist()
  },

  unblockUser: async (friendId) => {
    await chatApi.unblockUser(friendId)
    await get().fetchBlocklist()
  },

  setRemark: async (friendId, remark) => {
    await chatApi.setFriendRemark(friendId, remark)
    await get().fetchFriends()
  },
}))
