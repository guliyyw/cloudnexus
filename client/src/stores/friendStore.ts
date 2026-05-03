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
  acceptRequest: (id: number) => Promise<void>
  rejectRequest: (id: number) => Promise<void>
  removeFriend: (friendId: number) => Promise<void>
}

export const useFriendStore = create<FriendState>((set, get) => ({
  friends: [],
  pendingRequests: [],
  loading: false,

  fetchFriends: async () => {
    set({ loading: true })
    try {
      const friends = await chatApi.listFriends()
      set({ friends })
    } finally {
      set({ loading: false })
    }
  },

  fetchPendingRequests: async () => {
    const requests = await chatApi.listFriendRequests()
    set({ pendingRequests: requests })
  },

  sendRequest: async (friendName: string) => {
    await chatApi.sendFriendRequest(friendName)
  },

  acceptRequest: async (id: number) => {
    await chatApi.acceptRequest(id)
    await get().fetchPendingRequests()
    await get().fetchFriends()
  },

  rejectRequest: async (id: number) => {
    await chatApi.rejectRequest(id)
    await get().fetchPendingRequests()
  },

  removeFriend: async (friendId: number) => {
    await chatApi.removeFriend(friendId)
    await get().fetchFriends()
  },
}))
