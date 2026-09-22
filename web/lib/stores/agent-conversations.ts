"use client"

import { create } from "zustand"

import {
  fetchAgentConversations,
  fetchAgentMessages,
  markAgentMessageRead,
  recallAgentMessage,
  sendAgentMessage,
  uploadAgentConversationAttachment,
  uploadAgentConversationImage,
  type AgentAsset,
  type AgentConversation,
  type AgentMessage,
} from "@/lib/api/agent"
import type { RealtimeConnectionStatusValue } from "@/components/realtime-connection-status"
import {
  cursorFromLoadedImMessages,
  hasMoreAfterLatestImMessageMerge,
  mergeImMessagesByIdAsc,
  parseImMessageCursorId,
} from "@/lib/im-message-merge"
import {
  markMessagesReadToMessageId,
  patchConversationList,
  patchConversationListWithMessage,
  type RealtimeConversationPatch,
} from "@/lib/im-realtime-state"
import { summarizeIMMessage } from "@/lib/im-message"
import { generateUUID } from "@/lib/utils"

export const agentConversationFilterOptions = [
  { value: "active", labelKey: "conversation.filterActive" },
  { value: "pending", labelKey: "conversation.filterPending" },
  { value: "ai_serving", labelKey: "conversation.filterAiServing" },
  { value: "closed", labelKey: "conversation.filterClosed" },
] as const

export type AgentConversationFilterKey =
  (typeof agentConversationFilterOptions)[number]["value"]

function buildConversationQuery(filter: AgentConversationFilterKey, keyword: string) {
  const query: Record<string, string | number | undefined> = {
    filter,
    keyword: keyword.trim() || undefined,
    limit: 100,
  }

  return query
}

type LoadMessagesOptions = {
  forceLoading?: boolean
  reset?: boolean
}

function ensureArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

type AgentConversationsStore = {
  searchKeyword: string
  conversationFilter: AgentConversationFilterKey
  conversations: AgentConversation[]
  conversationsLoading: boolean
  conversationsLoaded: boolean
  selectedConversationId: number | null
  messages: AgentMessage[]
  messagesLoading: boolean
  messagesLoadingMore: boolean
  messagesCursor: string
  messagesHasMore: boolean
  messagesLoadedConversationId: number | null
  sending: boolean
  uploadingAsset: boolean
  recallingMessageId: number
  readingMessageId: number
  realtimeStatus: RealtimeConnectionStatusValue
  setSearchKeyword: (keyword: string) => void
  setConversationFilter: (filter: AgentConversationFilterKey) => void
  setRealtimeStatus: (status: RealtimeConnectionStatusValue) => void
  setConversationTags: (
    conversationId: number,
    tags: AgentConversation["tags"]
  ) => void
  loadConversations: () => Promise<void>
  selectConversation: (conversationId: number) => Promise<void>
  loadMessages: (conversationId: number, options?: LoadMessagesOptions) => Promise<void>
  loadOlderMessages: () => Promise<void>
  syncLatestMessages: (conversationId: number) => Promise<void>
  markSelectedConversationRead: () => Promise<void>
  sendMessage: (html: string, messageType?: "html" | "note") => Promise<AgentMessage | null>
  uploadImage: (file: File) => Promise<AgentAsset | null>
  sendAttachment: (file: File) => Promise<AgentMessage | null>
  recallMessage: (messageId: number) => Promise<AgentMessage | null>
  applyRealtimeMessageCreated: (message: AgentMessage) => void
  applyRealtimeConversationChanged: (patch: RealtimeConversationPatch) => void
  applyRealtimeMessageRecalled: (messageId: number, patch: Partial<AgentMessage>) => void
  resyncRealtimeData: (conversationId?: number) => Promise<void>
}

let conversationsRequestSeq = 0
let messagesRequestSeq = 0

export const useAgentConversationsStore = create<AgentConversationsStore>((set, get) => ({
  searchKeyword: "",
  conversationFilter: "active",
  conversations: [],
  conversationsLoading: false,
  conversationsLoaded: false,
  selectedConversationId: null,
  messages: [],
  messagesLoading: false,
  messagesLoadingMore: false,
  messagesCursor: "",
  messagesHasMore: false,
  messagesLoadedConversationId: null,
  sending: false,
  uploadingAsset: false,
  recallingMessageId: 0,
  readingMessageId: 0,
  realtimeStatus: "connecting",

  setSearchKeyword: (keyword) => {
    set({ searchKeyword: keyword })
  },

  setConversationFilter: (filter) => {
    set({ conversationFilter: filter })
  },

  setRealtimeStatus: (status) => {
    set({ realtimeStatus: status })
  },

  setConversationTags: (conversationId, tags) => {
    set((state) => ({
      conversations: state.conversations.map((item) =>
        item.id === conversationId
          ? {
              ...item,
              tags: tags && tags.length > 0 ? tags : [],
            }
          : item
      ),
    }))
  },

  loadConversations: async () => {
    const requestSeq = ++conversationsRequestSeq
    const store = get()

    if (!store.conversationsLoaded) {
      set({ conversationsLoading: true })
    }

    try {
      const data = await fetchAgentConversations(
        buildConversationQuery(store.conversationFilter, store.searchKeyword)
      )
      const conversations = ensureArray(data.results)

      if (requestSeq !== conversationsRequestSeq) {
        return
      }

      const currentSelectedId = get().selectedConversationId
      const hasCurrentSelection =
        currentSelectedId !== null && conversations.some((item) => item.id === currentSelectedId)
      const nextSelectedId = hasCurrentSelection ? currentSelectedId : (conversations[0]?.id ?? null)
      const selectionChanged = nextSelectedId !== currentSelectedId

      set({
        conversations,
        conversationsLoaded: true,
        conversationsLoading: false,
        selectedConversationId: nextSelectedId,
      })

      if (nextSelectedId === null) {
        set({
          messages: [],
          messagesLoading: false,
          messagesLoadingMore: false,
          messagesCursor: "",
          messagesHasMore: false,
          messagesLoadedConversationId: null,
        })
        return
      }

      if (selectionChanged || get().messagesLoadedConversationId === null) {
        await get().loadMessages(nextSelectedId, {
          forceLoading: true,
          reset: true,
        })
      }
    } catch (error) {
      if (requestSeq === conversationsRequestSeq) {
        set({ conversationsLoading: false })
      }
      throw error
    }
  },

  selectConversation: async (conversationId) => {
    if (get().selectedConversationId === conversationId) {
      return
    }

    set({
      selectedConversationId: conversationId,
      messages: [],
      messagesLoading: true,
      messagesLoadingMore: false,
      messagesCursor: "",
      messagesHasMore: false,
      messagesLoadedConversationId: null,
    })

    await get().loadMessages(conversationId, {
      forceLoading: true,
      reset: true,
    })
  },

  loadMessages: async (conversationId, options = {}) => {
    const requestSeq = ++messagesRequestSeq
    const store = get()
    const shouldShowLoading =
      options.forceLoading || store.messagesLoadedConversationId !== conversationId

    if (shouldShowLoading) {
      set({
        messagesLoading: true,
        ...(options.reset
          ? {
              messages: [],
              messagesCursor: "",
              messagesHasMore: false,
            }
          : {}),
      })
    }

    try {
      const data = await fetchAgentMessages({
        conversationId,
        limit: 50,
      })

      if (requestSeq !== messagesRequestSeq) {
        return
      }

      if (get().selectedConversationId !== conversationId) {
        return
      }

      const list = ensureArray(data.results)
      set({
        messages: list,
        messagesLoading: false,
        messagesLoadedConversationId: conversationId,
        messagesCursor:
          cursorFromLoadedImMessages(list) || (data.cursor ?? ""),
        messagesHasMore: Boolean(data.hasMore),
      })
    } catch (error) {
      if (requestSeq === messagesRequestSeq) {
        set({ messagesLoading: false })
      }
      throw error
    }
  },

  loadOlderMessages: async () => {
    const conversationId = get().selectedConversationId
    if (!conversationId || get().messagesLoadingMore || !get().messagesHasMore) {
      return
    }
    const cursorId = parseImMessageCursorId(get().messagesCursor)
    if (cursorId <= 0) {
      return
    }

    set({ messagesLoadingMore: true })
    try {
      const data = await fetchAgentMessages({
        conversationId,
        cursor: cursorId,
        limit: 50,
      })
      if (get().selectedConversationId !== conversationId) {
        return
      }
      const incoming = ensureArray(data.results)
      set((state) => {
        const merged = mergeImMessagesByIdAsc(state.messages, incoming)
        return {
          messages: merged,
          messagesCursor:
            cursorFromLoadedImMessages(merged) ||
            (data.cursor ?? state.messagesCursor),
          messagesHasMore: Boolean(data.hasMore),
          messagesLoadingMore: false,
        }
      })
    } catch (error) {
      set({ messagesLoadingMore: false })
      throw error
    }
  },

  syncLatestMessages: async (conversationId) => {
    if (conversationId <= 0) {
      return
    }
    try {
      const data = await fetchAgentMessages({
        conversationId,
        limit: 50,
      })
      if (get().selectedConversationId !== conversationId) {
        return
      }
      const batch = ensureArray(data.results)
      if (batch.length === 0) {
        return
      }
      set((state) => {
        const merged = mergeImMessagesByIdAsc(state.messages, batch)
        return {
          messages: merged,
          messagesCursor:
            cursorFromLoadedImMessages(merged) ||
            (data.cursor ?? state.messagesCursor),
          messagesHasMore: hasMoreAfterLatestImMessageMerge({
            previousMessages: state.messages,
            previousHasMore: state.messagesHasMore,
            merged,
            apiHasMore: Boolean(data.hasMore),
          }),
        }
      })
    } catch {
      // Keep realtime callback errors contained in the store.
    }
  },

  markSelectedConversationRead: async () => {
    const store = get()
    const conversationId = store.selectedConversationId
    const conversation = store.conversations.find((item) => item.id === conversationId)
    const lastMessage = store.messages.at(-1)
    if (!conversationId || !conversation || !lastMessage) {
      return
    }
    if (
      conversation.agentUnreadCount <= 0 &&
      (conversation.agentLastReadMessageId ?? 0) >= lastMessage.id
    ) {
      return
    }
    if (store.readingMessageId === lastMessage.id) {
      return
    }

    set({ readingMessageId: lastMessage.id })
    try {
      await markAgentMessageRead(conversationId, lastMessage.id)
      set((current) => {
        if (current.selectedConversationId !== conversationId) {
          return { readingMessageId: 0 }
        }
        return {
          readingMessageId: 0,
          messages: current.messages.map((item) => {
            if (item.id > lastMessage.id) {
              return item
            }
            return item.agentRead ? item : { ...item, agentRead: true }
          }),
          conversations: current.conversations.map((item) =>
            item.id === conversationId
              ? {
                  ...item,
                  agentUnreadCount: 0,
                  agentLastReadMessageId: lastMessage.id,
                }
              : item
          ),
        }
      })
    } catch (error) {
      set({ readingMessageId: 0 })
      throw error
    }
  },

  applyRealtimeMessageCreated: (message) => {
    set((state) => {
      const isSelected = state.selectedConversationId === message.conversationId
      const nextMessages = isSelected
        ? mergeImMessagesByIdAsc(state.messages, [message])
        : state.messages
      return {
        messages: nextMessages,
        conversations: patchConversationListWithMessage(
          state.conversations,
          message
        ),
      }
    })
  },

  applyRealtimeConversationChanged: (patch) => {
    set((state) => {
      const conversationId = patch.id ?? patch.conversationId ?? 0
      let nextMessages = state.messages
      if (
        conversationId > 0 &&
        state.selectedConversationId === conversationId
      ) {
        if ((patch.agentLastReadMessageId ?? 0) > 0) {
          nextMessages = markMessagesReadToMessageId(
            nextMessages,
            patch.agentLastReadMessageId ?? 0,
            "agent",
            patch.agentLastReadAt
          )
        }
        if ((patch.customerLastReadMessageId ?? 0) > 0) {
          nextMessages = markMessagesReadToMessageId(
            nextMessages,
            patch.customerLastReadMessageId ?? 0,
            "customer",
            patch.customerLastReadAt
          )
        }
      }
      return {
        messages: nextMessages,
        conversations: patchConversationList(state.conversations, patch),
      }
    })
  },

  applyRealtimeMessageRecalled: (messageId, patch) => {
    if (messageId <= 0) {
      return
    }
    set((state) => ({
      messages: state.messages.map((item) =>
        item.id === messageId ? { ...item, ...patch, id: item.id } : item
      ),
    }))
  },

  resyncRealtimeData: async (conversationId) => {
    await get().loadConversations()
    const selectedConversationId = get().selectedConversationId
    const targetConversationId = conversationId ?? selectedConversationId
    if (targetConversationId && selectedConversationId === targetConversationId) {
      await get().loadMessages(targetConversationId, {
        forceLoading: false,
        reset: false,
      })
    }
  },

  sendMessage: async (html, messageType = "html") => {
    const trimmedContent = html.trim()
    const { selectedConversationId, sending } = get()
    if (!selectedConversationId || !trimmedContent || sending) {
      return null
    }

    set({ sending: true })
    try {
      const message = await sendAgentMessage({
        conversationId: selectedConversationId,
        messageType: messageType,
        content: trimmedContent,
        clientMsgId: `agent_${generateUUID()}`,
      })

      if (get().selectedConversationId === selectedConversationId) {
        set((current) => ({
          messages: current.messages.some((m) => m.id === message.id)
            ? current.messages.map((m) => (m.id === message.id ? message : m))
            : [...current.messages, message],
          conversations: patchConversationList(
            patchConversationListWithMessage(current.conversations, message),
            {
              conversationId: selectedConversationId,
              agentUnreadCount: 0,
              customerUnreadCount:
                messageType === "note"
                  ? (current.conversations.find((item) => item.id === selectedConversationId)?.customerUnreadCount ?? 0)
                  : (current.conversations.find((item) => item.id === selectedConversationId)?.customerUnreadCount ?? 0) + 1,
              agentLastReadMessageId: message.id,
            }
          ),
        }))
      }

      return message
    } finally {
      set({ sending: false })
    }
  },

  uploadImage: async (file) => {
    const { selectedConversationId, sending, uploadingAsset } = get()
    if (!selectedConversationId || sending || uploadingAsset) {
      return null
    }

    set({ uploadingAsset: true })
    try {
      return await uploadAgentConversationImage(selectedConversationId, file)
    } finally {
      set({ uploadingAsset: false })
    }
  },

  sendAttachment: async (file) => {
    const { selectedConversationId, sending, uploadingAsset } = get()
    if (!selectedConversationId || sending || uploadingAsset) {
      return null
    }

    set({ uploadingAsset: true })
    try {
      const asset = await uploadAgentConversationAttachment(selectedConversationId, file)
      const message = await sendAgentMessage({
        conversationId: selectedConversationId,
        messageType: "attachment",
        content: asset.filename,
        payload: JSON.stringify({ assetId: asset.assetId }),
        clientMsgId: `agent_attachment_${generateUUID()}`,
      })

      if (get().selectedConversationId === selectedConversationId) {
        set((current) => ({
          messages: current.messages.some((m) => m.id === message.id)
            ? current.messages.map((m) => (m.id === message.id ? message : m))
            : [...current.messages, message],
          conversations: patchConversationList(
            patchConversationListWithMessage(current.conversations, message),
            {
              conversationId: selectedConversationId,
              agentUnreadCount: 0,
              customerUnreadCount:
                (current.conversations.find((item) => item.id === selectedConversationId)
                  ?.customerUnreadCount ?? 0) + 1,
              agentLastReadMessageId: message.id,
            }
          ),
        }))
      }

      return message
    } finally {
      set({ uploadingAsset: false })
    }
  },

  recallMessage: async (messageId) => {
    const { selectedConversationId, recallingMessageId } = get()
    if (!selectedConversationId || messageId <= 0 || recallingMessageId === messageId) {
      return null
    }

    set({ recallingMessageId: messageId })
    try {
      const message = await recallAgentMessage(messageId)
      if (get().selectedConversationId === selectedConversationId) {
        set((current) => {
          const nextMessages = current.messages.map((item) =>
            item.id === message.id ? message : item
          )
          const lastActiveMessage = [...nextMessages]
            .reverse()
            .find((item) => !item.recalledAt && item.sendStatus !== 6)
          return {
            recallingMessageId: 0,
            messages: nextMessages,
            conversations: current.conversations.map((item) =>
              item.id === selectedConversationId
                ? {
                    ...item,
                    lastMessageId: lastActiveMessage?.id ?? 0,
                    lastMessageAt: lastActiveMessage?.sentAt ?? "",
                    lastMessageSummary: lastActiveMessage
                      ? summarizeIMMessage(lastActiveMessage)
                      : "",
                  }
                : item
            ),
          }
        })
      } else {
        set({ recallingMessageId: 0 })
      }
      return message
    } catch (error) {
      set({ recallingMessageId: 0 })
      throw error
    }
  },
}))

export const agentConversationSelectors = {
  selectedConversation: (state: AgentConversationsStore) =>
    state.conversations.find((item) => item.id === state.selectedConversationId) ?? null,
}
