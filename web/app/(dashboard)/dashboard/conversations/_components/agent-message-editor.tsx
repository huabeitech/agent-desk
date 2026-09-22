"use client"

import { useEffect, useState } from "react"
import { toast } from "sonner"

import {
  SharedMessageEditor,
  type UploadedMessageEditorImage,
} from "@/components/chat/shared-message-editor"
import { useI18n } from "@/i18n/provider"
import { fetchQuickReplyListAll, type AdminQuickReply } from "@/lib/api/admin"

type AgentMessageEditorProps = {
  disabled?: boolean
  uploadingAsset?: boolean
  onSend: (html: string, messageType?: "html" | "note") => Promise<void>
  onUploadImage: (file: File) => Promise<UploadedMessageEditorImage | null>
  onSendAttachment: (file: File) => Promise<void>
}

export function AgentMessageEditor({
  disabled = false,
  uploadingAsset = false,
  onSend,
  onUploadImage,
  onSendAttachment,
}: AgentMessageEditorProps) {
  const t = useI18n()
  const [quickReplies, setQuickReplies] = useState<AdminQuickReply[]>([])
  const [loadingQuickReplies, setLoadingQuickReplies] = useState(true)
  const [quickReplyPickerOpen, setQuickReplyPickerOpen] = useState(false)

  useEffect(() => {
    let cancelled = false
    void fetchQuickReplyListAll()
      .then((list) => {
        if (!cancelled) {
          setQuickReplies(list)
        }
      })
      .catch((error) => {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("conversation.loadQuickRepliesFailed"))
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingQuickReplies(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [t])

  return (
    <SharedMessageEditor
      variant="agent"
      disabled={disabled}
      uploadingAsset={uploadingAsset}
      quickReplies={{
        open: quickReplyPickerOpen,
        loading: loadingQuickReplies,
        items: quickReplies,
        onOpenChange: setQuickReplyPickerOpen,
      }}
      onSend={onSend}
      onUploadImage={onUploadImage}
      onSendAttachment={onSendAttachment}
    />
  )
}
