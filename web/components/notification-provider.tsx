"use client"

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react"
import { useRouter } from "next/navigation"
import { toast } from "sonner"

import {
  createNotificationWebSocketUrl,
  fetchNotificationUnreadCount,
  markNotificationRead,
  type NotificationItem,
} from "@/lib/api/notification"
import { readSession } from "@/lib/auth"
import {
  createRealtimeConnectionManager,
  type RealtimeConnectionStatus,
} from "@/lib/realtime-connection"
import { useAppLocale, useI18n } from "@/i18n/provider"
import { localizeNotificationItem } from "@/lib/notification-i18n"

type NotificationRealtimeEnvelope = {
  eventId?: string
  type?: string
  data?: {
    notification?: NotificationItem
  }
}

type NotificationContextValue = {
  unreadCount: number
  realtimeStatus: RealtimeConnectionStatus
  refreshUnreadCount: () => Promise<void>
  markReadAndNavigate: (notification: NotificationItem) => Promise<void>
}

const NotificationContext = createContext<NotificationContextValue | null>(null)

export function NotificationProvider({ children }: { children: ReactNode }) {
  const t = useI18n()
  const { locale } = useAppLocale()
  const router = useRouter()
  const [unreadCount, setUnreadCount] = useState(0)
  const [realtimeStatus, setRealtimeStatus] =
    useState<RealtimeConnectionStatus>("disconnected")
  const currentUserIdRef = useRef(readSession()?.user.id ?? 0)

  const refreshUnreadCount = useCallback(async () => {
    const result = await fetchNotificationUnreadCount()
    setUnreadCount(result.unreadCount)
  }, [])

  const markReadAndNavigate = useCallback(
    async (notification: NotificationItem) => {
      if (!notification.readAt) {
        await markNotificationRead(notification.id)
        setUnreadCount((current) => Math.max(0, current - 1))
      }
      if (notification.actionUrl) {
        router.push(notification.actionUrl)
      }
    },
    [router]
  )

  useEffect(() => {
    currentUserIdRef.current = readSession()?.user.id ?? 0
    const refreshTask = window.setTimeout(() => {
      void refreshUnreadCount().catch(() => {
        setUnreadCount(0)
      })
    }, 0)
    return () => window.clearTimeout(refreshTask)
  }, [refreshUnreadCount])

  useEffect(() => {
    const realtime = createRealtimeConnectionManager({
      createSocket: () => new WebSocket(createNotificationWebSocketUrl()),
      canReconnect: () => Boolean(readSession()?.accessToken),
      onStatusChange: setRealtimeStatus,
      onOpen: () => {
        void refreshUnreadCount().catch(() => undefined)
      },
      onMessage: (event, socket) => {
        try {
          const envelope = JSON.parse(event.data) as NotificationRealtimeEnvelope
          const eventType = envelope.type ?? ""
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
          if (eventType !== "notification.created") {
            return
          }
          const notification = envelope.data?.notification
          if (!notification || notification.recipientUserId !== currentUserIdRef.current) {
            return
          }
          const localizedNotification = localizeNotificationItem(notification, locale)
          setUnreadCount((current) => current + 1)
          toast(localizedNotification.title || t("notification.new"), {
            description: localizedNotification.content,
            action: {
              label: t("notification.view"),
              onClick: () => {
                void markReadAndNavigate(localizedNotification).catch((error) => {
                  toast.error(error instanceof Error ? error.message : t("notification.openFailed"))
                })
              },
            },
          })
        } catch {
          // ignore invalid realtime payload
        }
      },
      onConnectError: (error) => {
        toast.error(error instanceof Error ? error.message : t("notification.connectFailed"))
      },
    })

    realtime.connect()
    return () => {
      realtime.disconnect()
    }
  }, [locale, markReadAndNavigate, refreshUnreadCount, t])

  const value = useMemo<NotificationContextValue>(
    () => ({
      unreadCount,
      realtimeStatus,
      refreshUnreadCount,
      markReadAndNavigate,
    }),
    [markReadAndNavigate, realtimeStatus, refreshUnreadCount, unreadCount]
  )

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  )
}

export function useNotifications() {
  const context = useContext(NotificationContext)
  if (!context) {
    throw new Error("useNotifications must be used within NotificationProvider")
  }
  return context
}
