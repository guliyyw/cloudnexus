import { create } from 'zustand'
import * as chatApi from '../services/chat'
import type { FriendRequest } from '../services/chat'

interface FriendState {
  friends: FriendRequest[]
  pendingRequests: FriendRequest[]
  loading: boolean

  fetchFriends: () => Promise<void>
  fetchPendingRequests: () => Promise<void>
  sendRequest: (friendName: string) => Promise<void>
  acceptRequest: (id: string) => Promise<void>
  rejectRequest: (id: string) => Promise<void>
  removeFriend: (friendId: string) => Promise<void>
}

export const useFriendStore = create<FriendState>((set, get) => ({
  friends: [],
  pendingRequests: [],
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

  sendRequest: async (friendName: string) => {
    await chatApi.sendFriendRequest(friendName)
  },

  acceptRequest: async (id: string) => {
    await chatApi.acceptRequest(id)
    await get().fetchPendingRequests()
    await get().fetchFriends()
  },

  rejectRequest: async (id: string) => {
    await chatApi.rejectRequest(id)
    await get().fetchPendingRequests()
  },

  removeFriend: async (friendId: string) => {
    await chatApi.removeFriend(friendId)
    await get().fetchFriends()
  },
}))
