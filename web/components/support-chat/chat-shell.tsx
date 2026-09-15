"use client"

import {
  HeadphonesIcon,
  Maximize2Icon,
  Minimize2Icon,
  MoreHorizontalIcon,
  MinusIcon,
  RotateCwIcon,
  XIcon,
} from "lucide-react"
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ComponentProps,
  type CSSProperties,
} from "react"
import { useShallow } from "zustand/react/shallow"

import { SupportChatConnectionStatus } from "@/components/support-chat/connection-status"
import { getStandaloneClosedUrl } from "@/components/support-chat/close-navigation"
import { CustomerMessageEditor } from "@/components/support-chat/customer-message-editor"
import {
  SupportChatMessageList,
  type SupportChatMessageListHandle,
} from "@/components/support-chat/message-list"
import {
  bindSupportHostBridge,
  requestSupportHostClose,
  requestSupportHostMinimize,
  requestSupportHostToggleMaximize,
} from "@/lib/support-host-bridge"
import { useSupportChatStore } from "@/lib/stores/support-chat"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cn } from "@/lib/utils"
import { useI18n } from "@/i18n/provider"

const windowActionButtonClass =
  "size-8 rounded-full border-0 bg-transparent text-muted-foreground shadow-none hover:bg-foreground/[0.06] hover:text-foreground focus-visible:ring-primary/20 dark:hover:bg-white/10"

function WindowActionButton({
  className,
  ...props
}: ComponentProps<typeof Button>) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(windowActionButtonClass, className)}
      {...props}
    />
  )
}

function getStatusDotClass(status: string) {
  if (status === "connected") {
    return "bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.14)]"
  }
  if (status === "connecting") {
    return "bg-amber-500 shadow-[0_0_0_3px_rgba(245,158,11,0.16)]"
  }
  return "bg-muted-foreground shadow-[0_0_0_3px_rgba(148,163,184,0.14)]"
}

function useSupportChatSystemTheme() {
  useLayoutEffect(() => {
    if (typeof window === "undefined") {
      return
    }
    const root = document.documentElement
    const query = window.matchMedia("(prefers-color-scheme: dark)")
    const previousDarkClass = root.classList.contains("dark")
    const previousColorScheme = root.style.colorScheme

    const syncTheme = () => {
      const isDark = query.matches
      root.classList.toggle("dark", isDark)
      root.style.colorScheme = isDark ? "dark" : "light"
    }

    syncTheme()
    query.addEventListener("change", syncTheme)

    return () => {
      query.removeEventListener("change", syncTheme)
      root.classList.toggle("dark", previousDarkClass)
      root.style.colorScheme = previousColorScheme
    }
  }, [])
}

function isEmbeddedInHost() {
  if (typeof window === "undefined") {
    return false
  }

  try {
    return window.parent !== window
  } catch {
    return false
  }
}

export function SupportChatShell() {
  const t = useI18n()
  useSupportChatSystemTheme()

  const messageListRef = useRef<SupportChatMessageListHandle | null>(null)
  const [isEmbedded, setIsEmbedded] = useState(false)
  const [isMaximized, setIsMaximized] = useState(false)
  const [isCloseDialogOpen, setIsCloseDialogOpen] = useState(false)
  const [isClosingConversation, setIsClosingConversation] = useState(false)

  const {
    title,
    subtitle,
    themeColor,
    conversation,
    messages,
    messagesHasMore,
    messagesLoadingMore,
    loadOlderMessages,
    status,
    error,
    isOpen,
    isVisible,
    setIsOpen,
    setIsVisible,
    bootstrap,
    handleSendMessage,
    uploadMessageImage,
    sendAttachment,
    retry,
    disconnectSocket,
    markConversationRead,
    closeConversation,
  } = useSupportChatStore(
    useShallow((state) => ({
      title: state.title,
      subtitle: state.subtitle,
      themeColor: state.themeColor,
      conversation: state.conversation,
      messages: state.messages,
      messagesHasMore: state.messagesHasMore,
      messagesLoadingMore: state.messagesLoadingMore,
      loadOlderMessages: state.loadOlderMessages,
      status: state.status,
      error: state.error,
      isOpen: state.isOpen,
      isVisible: state.isVisible,
      setIsOpen: state.setIsOpen,
      setIsVisible: state.setIsVisible,
      bootstrap: state.bootstrap,
      handleSendMessage: state.handleSendMessage,
      uploadMessageImage: state.uploadMessageImage,
      sendAttachment: state.sendAttachment,
      retry: state.retry,
      disconnectSocket: state.disconnectSocket,
      markConversationRead: state.markConversationRead,
      closeConversation: state.closeConversation,
    }))
  )
  const safeMessages = Array.isArray(messages) ? messages : []

  useEffect(() => {
    setIsEmbedded(isEmbeddedInHost())
  }, [])

  const maybeMarkConversationRead = useCallback(() => {
    if (!isVisible || !conversation || typeof document === "undefined") {
      return
    }
    if (document.visibilityState !== "visible") {
      return
    }
    void markConversationRead().catch((readError) => {
      console.error("Failed to mark support chat conversation read", readError)
    })
  }, [conversation?.id, isVisible, markConversationRead])

  useEffect(() => {
    return bindSupportHostBridge({
      onOpen: () => {
        setIsOpen(true)
        setIsVisible(true)
      },
      onMinimize: () => {
        setIsVisible(false)
      },
      onMaximizedChange: (nextIsMaximized) => {
        setIsMaximized(nextIsMaximized)
      },
    })
  }, [setIsOpen, setIsVisible])

  useEffect(() => {
    bootstrap()

    return () => {
      if (!isOpen) {
        disconnectSocket()
      }
    }
  }, [isOpen, bootstrap, disconnectSocket])

  useEffect(() => {
    maybeMarkConversationRead()
  }, [maybeMarkConversationRead, safeMessages.length])

  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        maybeMarkConversationRead()
      }
    }
    const handleFocus = () => {
      maybeMarkConversationRead()
    }

    document.addEventListener("visibilitychange", handleVisibilityChange)
    window.addEventListener("focus", handleFocus)
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange)
      window.removeEventListener("focus", handleFocus)
    }
  }, [maybeMarkConversationRead])

  async function handleSend(content: string) {
    await handleSendMessage(content)
    messageListRef.current?.scrollToBottom()
  }

  function handleMinimize() {
    setIsVisible(false)
    requestSupportHostMinimize()
  }

  function handleToggleMaximize() {
    requestSupportHostToggleMaximize()
  }

  async function confirmCloseConversation() {
    if (isClosingConversation) {
      return
    }
    setIsClosingConversation(true)
    try {
      if (conversation?.id) {
        await closeConversation()
      }
      setIsCloseDialogOpen(false)
      if (isEmbedded) {
        requestSupportHostClose()
      } else {
        window.location.replace(getStandaloneClosedUrl())
      }
    } catch (closeError) {
      window.alert(closeError instanceof Error ? closeError.message : t("supportChat.closeConversationFailed"))
    } finally {
      setIsClosingConversation(false)
    }
  }

  useEffect(() => {
    if (!isCloseDialogOpen) {
      return
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !isClosingConversation) {
        setIsCloseDialogOpen(false)
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [isCloseDialogOpen, isClosingConversation])

  return (
    <main
      className="relative flex h-[100dvh] min-h-[100dvh] overflow-hidden bg-muted text-foreground supports-not-[height:100dvh]:h-screen supports-not-[height:100dvh]:min-h-screen"
      style={{ "--primary": themeColor } as CSSProperties}
    >
      <section className="flex h-full w-full flex-col overflow-hidden bg-card text-card-foreground">
        <header className="shrink-0 border-b border-border/70 bg-card/95 px-3 py-2 shadow-[0_1px_0_rgba(15,23,42,0.02)] backdrop-blur dark:border-border/60 dark:bg-card/90 sm:px-4">
          <div className="flex min-w-0 items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2.5">
              <div className="relative flex size-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary ring-1 ring-primary/15 dark:bg-primary/15 dark:ring-primary/25">
                <HeadphonesIcon className="size-4" />
                <span
                  className={cn(
                    "absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full ring-2 ring-card",
                    getStatusDotClass(status)
                  )}
                  aria-hidden="true"
                />
              </div>
              <div className="min-w-0 space-y-0.5">
                <div className="truncate text-sm font-semibold leading-5 text-foreground">
                  {title}
                </div>
                <div className="truncate text-[11px] leading-4 text-muted-foreground">
                  {subtitle}
                </div>
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-0.5 sm:hidden">
              {!isEmbedded && status !== "connected" ? (
                <WindowActionButton
                  onClick={retry}
                  aria-label={t("supportChat.retry")}
                  title={t("supportChat.retry")}
                >
                  <RotateCwIcon className="size-4" />
                </WindowActionButton>
              ) : null}
              {isEmbedded ? (
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={<WindowActionButton aria-label={t("supportChat.moreActions")} title={t("supportChat.moreActions")} />}
                  >
                    <MoreHorizontalIcon className="size-4" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-36">
                    <DropdownMenuItem onClick={retry}>
                      <RotateCwIcon className="size-4" />
                      {t("supportChat.retry")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={handleMinimize}>
                      <MinusIcon className="size-4" />
                      {t("supportChat.minimize")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={handleToggleMaximize}>
                      {isMaximized ? (
                        <Minimize2Icon className="size-4" />
                      ) : (
                        <Maximize2Icon className="size-4" />
                      )}
                      {isMaximized ? t("supportChat.restoreWindow") : t("supportChat.maximize")}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      variant="destructive"
                      onClick={() => setIsCloseDialogOpen(true)}
                    >
                      <XIcon className="size-4" />
                      {t("supportChat.closeWindow")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : <WindowActionButton
                onClick={() => setIsCloseDialogOpen(true)}
                aria-label={t("supportChat.closeWindow")}
                title={t("supportChat.closeWindow")}
                className="hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/45 dark:hover:text-rose-300"
              >
                <XIcon className="size-4" />
              </WindowActionButton>
              }
              <WindowActionButton
                onClick={handleMinimize}
                aria-label={t("supportChat.minimize")}
                title={t("supportChat.minimize")}
                className="hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/45 dark:hover:text-rose-300"
              >
                <XIcon className="size-4" />
              </WindowActionButton>
            </div>
            <div className="hidden shrink-0 items-center gap-1 sm:flex">
              {status !== "connected" ? (
                <SupportChatConnectionStatus status={status} />
              ) : null}
              <WindowActionButton
                onClick={retry}
                aria-label={t("supportChat.retry")}
                title={t("supportChat.retry")}
              >
                <RotateCwIcon className="size-4" />
              </WindowActionButton>
              {isEmbedded ? (
                <>
                  <WindowActionButton
                    onClick={handleMinimize}
                    aria-label={t("supportChat.minimize")}
                    title={t("supportChat.minimize")}
                  >
                    <MinusIcon className="size-4" />
                  </WindowActionButton>
                  <WindowActionButton
                    onClick={handleToggleMaximize}
                    aria-label={isMaximized ? t("supportChat.restoreWindow") : t("supportChat.maximizeWindow")}
                    title={isMaximized ? t("supportChat.restoreWindow") : t("supportChat.maximizeWindow")}
                  >
                    {isMaximized ? (
                      <Minimize2Icon className="size-4" />
                    ) : (
                      <Maximize2Icon className="size-4" />
                    )}
                  </WindowActionButton>
                </>
              ) : null}
              <WindowActionButton
                onClick={() => setIsCloseDialogOpen(true)}
                aria-label={t("supportChat.closeChatWindow")}
                title={t("supportChat.closeChatWindow")}
                className="hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/45 dark:hover:text-rose-300"
              >
                <XIcon className="size-4" />
              </WindowActionButton>
            </div>
          </div>
        </header>

        <div className="grid min-h-0 flex-1 grid-rows-[minmax(0,1fr)_auto] overflow-hidden bg-muted/60 dark:bg-muted/30">
          <SupportChatMessageList
            ref={messageListRef}
            messages={safeMessages}
            onNearBottomVisible={maybeMarkConversationRead}
            hasMoreOlder={messagesHasMore}
            loadingOlder={messagesLoadingMore}
            onLoadOlder={loadOlderMessages}
          />
          <div className="shrink-0 border-t border-border/80 bg-card/95 pb-[env(safe-area-inset-bottom)] shadow-[0_-10px_24px_rgba(15,23,42,0.05)] dark:bg-card/90 dark:shadow-none">
            <CustomerMessageEditor
              disabled={!conversation}
              onSend={handleSend}
              onUploadImage={uploadMessageImage}
              onSendAttachment={sendAttachment}
            />
          </div>
        </div>

        {error ? (
          <div className="border-t border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {error}
          </div>
        ) : null}
      </section>

      <Dialog
        open={isCloseDialogOpen}
        onOpenChange={(open) => {
          if (!isClosingConversation) {
            setIsCloseDialogOpen(open)
          }
        }}
      >
        <DialogContent className="max-w-[320px]" showCloseButton={!isClosingConversation}>
          <DialogHeader>
            <DialogTitle>{t("supportChat.closeDialogTitle")}</DialogTitle>
            <DialogDescription className="text-xs leading-5">
              {t("supportChat.closeDialogDescription")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={isClosingConversation}
              onClick={() => setIsCloseDialogOpen(false)}
            >
              {t("supportChat.continueConversation")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={isClosingConversation}
              onClick={() => void confirmCloseConversation()}
            >
              {isClosingConversation ? t("supportChat.closing") : t("supportChat.confirmClose")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  )
}
