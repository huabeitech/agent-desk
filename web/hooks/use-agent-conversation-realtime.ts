"use client"

import { useEffect, useRef } from "react"
import { toast } from "sonner"

import { createAdminWebSocketUrl } from "@/lib/api/admin"
import { type AgentMessage } from "@/lib/api/agent"
import { shouldReloadConversationListForRealtimePatch } from "@/lib/agent-conversation-realtime"
import { readSession } from "@/lib/auth"
import {
  normalizeRealtimeMessage,
  type RealtimeConversationPatch,
  type RealtimeMessageCreatedPayload,
} from "@/lib/im-realtime-state"
import { createRealtimeConnectionManager } from "@/lib/realtime-connection"
import { getNotificationBody, showNotification } from "@/lib/services/notification"
import { useAgentConversationsStore } from "@/lib/stores/agent-conversations"
import { useI18n } from "@/i18n/provider"

type AgentRealtimeConnection = ReturnType<typeof createRealtimeConnectionManager>
type AgentRealtimeEnvelope = {
  eventId?: string
  type?: string
  data?: RealtimeMessageCreatedPayload<AgentMessage> &
    RealtimeConversationPatch & {
      messageId?: number
      recalledAt?: string
      sendStatus?: number
    }
}

export function useAgentConversationRealtime() {
  const t = useI18n()
  const selectedConversationId = useAgentConversationsStore(
    (state) => state.selectedConversationId
  )
  const setRealtimeStatus = useAgentConversationsStore(
    (state) => state.setRealtimeStatus
  )
  const realtimeRef = useRef<AgentRealtimeConnection | null>(null)
  const subscribedConversationIdRef = useRef<number | null>(null)
  const selectedConversationIdRef = useRef<number | null>(selectedConversationId)
  const currentUserIdRef = useRef<number>(readSession()?.user.id ?? 0)

  useEffect(() => {
    selectedConversationIdRef.current = selectedConversationId
  }, [selectedConversationId])

  useEffect(() => {
    const realtime = createRealtimeConnectionManager({
      createSocket: () => new WebSocket(createAdminWebSocketUrl()),
      onStatusChange: setRealtimeStatus,
      onOpen: (socket) => {
        console.info("[agent-realtime] websocket connected", {
          url: socket.url,
        })

        const conversationId = selectedConversationIdRef.current
        if (conversationId) {
          socket.send(
            JSON.stringify({
              type: "subscribe",
              topics: [`conversation:${conversationId}`],
            })
          )
          subscribedConversationIdRef.current = conversationId
        } else {
          subscribedConversationIdRef.current = null
        }
      },
      onMessage: (event, socket) => {
        try {
          const envelope = JSON.parse(event.data) as AgentRealtimeEnvelope
          const eventType = envelope.type ?? ""
          const payload = envelope.data
          const conversationId = payload?.conversationId ?? 0
          const eventId = envelope.eventId?.trim() ?? ""

          if (
            eventType === "" ||
            eventType === "connected" ||
            eventType === "pong" ||
            eventType === "subscribed" ||
            eventType === "unsubscribed"
          ) {
            return
          }

          if (eventId && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: "ack", eventId }))
          }

          const store = useAgentConversationsStore.getState()
          if (eventType === "resyncRequired") {
            void store.resyncRealtimeData(conversationId).catch((error) => {
              toast.error(error instanceof Error ? error.message : t("conversation.syncConversationDataFailed"))
            })
            return
          }

          // message.updated 为 AI 占位回复被正式回复替换，合并语义一致（按 ID 覆盖）
          if (eventType === "message.created" || eventType === "message.updated") {
            const message = normalizeRealtimeMessage<AgentMessage>(payload)
            if (!message) {
              void store.resyncRealtimeData(conversationId).catch((error) => {
                toast.error(error instanceof Error ? error.message : t("conversation.syncMessagesFailed"))
              })
              return
            }
            store.applyRealtimeMessageCreated(message)

            const shouldNotify =
              message.senderType === "customer" &&
              payload?.status === 2 &&
              (payload.currentAssigneeId ?? 0) > 0 &&
              payload.currentAssigneeId === currentUserIdRef.current &&
              typeof document !== "undefined" &&
              document.visibilityState !== "visible"

            if (shouldNotify) {
              showNotification(t("conversation.newMessage"), getNotificationBody(message), () => {
                void store.selectConversation(message.conversationId)
              })
            }
            return
          }

          if (eventType === "message.recalled" && payload?.messageId) {
            store.applyRealtimeMessageRecalled(payload.messageId, {
              sendStatus: payload.sendStatus,
              recalledAt: payload.recalledAt,
            })
            return
          }

          if (eventType.startsWith("conversation.") && payload) {
            store.applyRealtimeConversationChanged(payload)
            if (shouldReloadConversationListForRealtimePatch(payload)) {
              void store.resyncRealtimeData(conversationId).catch((error) => {
                toast.error(error instanceof Error ? error.message : t("conversation.syncConversationListFailed"))
              })
            }
          }
        } catch {
          // ignore invalid ws payload
        }
      },
      onClose: (event, socket) => {
        console.log("[agent-realtime] websocket closed", {
          url: socket.url,
          readyState: socket.readyState,
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean,
        })
        subscribedConversationIdRef.current = null
      },
      onError: (_event, socket) => {
        console.log("[agent-realtime] websocket error", {
          url: socket.url,
          readyState: socket.readyState,
        })
      },
      onConnectError: (error) => {
        toast.error(error instanceof Error ? error.message : t("conversation.realtimeConnectFailed"))
      },
    })

    realtimeRef.current = realtime
    realtime.connect()

    return () => {
      realtimeRef.current = null
      realtime.disconnect()
      subscribedConversationIdRef.current = null
    }
  }, [setRealtimeStatus, t])

  useEffect(() => {
    const socket = realtimeRef.current?.getSocket()
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return
    }

    const previousConversationId = subscribedConversationIdRef.current
    const nextConversationId = selectedConversationId ?? null

    if (previousConversationId && previousConversationId !== nextConversationId) {
      socket.send(
        JSON.stringify({
          type: "unsubscribe",
          topics: [`conversation:${previousConversationId}`],
        })
      )
    }

    if (nextConversationId && nextConversationId !== previousConversationId) {
      socket.send(
        JSON.stringify({
          type: "subscribe",
          topics: [`conversation:${nextConversationId}`],
        })
      )
      subscribedConversationIdRef.current = nextConversationId
      return
    }

    if (!nextConversationId) {
      subscribedConversationIdRef.current = null
    }
  }, [selectedConversationId])
}
