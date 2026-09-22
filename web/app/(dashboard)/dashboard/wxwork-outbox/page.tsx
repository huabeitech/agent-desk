"use client"

import { useState } from "react"
import { BanIcon, RotateCcwIcon } from "lucide-react"
import { toast } from "sonner"

import {
  DashboardListPage,
  type DashboardListRenderContext,
} from "@/components/dashboard/list"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  fetchWxWorkOutboxFailures,
  ignoreWxWorkOutbox,
  retryWxWorkOutbox,
  type ChannelMessageOutbox,
} from "@/lib/api/admin"
import { useI18n } from "@/i18n/provider"
import { formatDateTime } from "@/lib/utils"

function statusVariant(status: string) {
  if (status === "failed") return "destructive" as const
  if (status === "ignored") return "outline" as const
  return "secondary" as const
}

function formatOptionalTime(value: string) {
  return value ? formatDateTime(value) : "-"
}

function OutboxActions({
  item,
  reload,
}: {
  item: ChannelMessageOutbox
  reload: DashboardListRenderContext<ChannelMessageOutbox>["reload"]
}) {
  const t = useI18n()
  const [runningAction, setRunningAction] = useState<"retry" | "ignore" | null>(null)

  async function runAction(action: "retry" | "ignore") {
    setRunningAction(action)
    try {
      if (action === "retry") {
        await retryWxWorkOutbox(item.id)
        toast.success(t("wxworkOutbox.retrySuccess"))
      } else {
        await ignoreWxWorkOutbox(item.id)
        toast.success(t("wxworkOutbox.ignoreSuccess"))
      }
      await reload()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("wxworkOutbox.actionFailed"))
    } finally {
      setRunningAction(null)
    }
  }

  return (
    <div className="flex items-center justify-end gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={runningAction !== null}
        onClick={() => void runAction("retry")}
      >
        <RotateCcwIcon />
        {t("wxworkOutbox.retry")}
      </Button>
      {item.sendStatus === "failed" ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={runningAction !== null}
          onClick={() => void runAction("ignore")}
        >
          <BanIcon />
          {t("wxworkOutbox.ignore")}
        </Button>
      ) : null}
    </div>
  )
}

export default function DashboardWxWorkOutboxPage() {
  const t = useI18n()

  const statusOptions = [
    { value: "failed", label: t("wxworkOutbox.statusFailed") },
    { value: "ignored", label: t("wxworkOutbox.statusIgnored") },
    { value: "all", label: t("wxworkOutbox.statusAll") },
  ]

  const statusLabel = (status: string) => {
    if (status === "failed") return t("wxworkOutbox.statusFailed")
    if (status === "ignored") return t("wxworkOutbox.statusIgnored")
    return status || "-"
  }

  return (
    <DashboardListPage<ChannelMessageOutbox>
      filters={[
        {
          name: "sendStatus",
          label: t("wxworkOutbox.columnStatus"),
          defaultValue: "failed",
          type: "segment",
          options: statusOptions,
        },
        {
          name: "conversationId",
          label: t("wxworkOutbox.conversationId"),
          placeholder: t("wxworkOutbox.conversationId"),
          defaultValue: "",
          valueType: "number",
          className: "w-full sm:w-40",
        },
        {
          name: "messageId",
          label: t("wxworkOutbox.messageId"),
          placeholder: t("wxworkOutbox.messageId"),
          defaultValue: "",
          valueType: "number",
          className: "w-full sm:w-40",
        },
      ]}
      fetchList={fetchWxWorkOutboxFailures}
      getItemId={(item) => item.id}
      columns={[
        {
          key: "id",
          label: "Outbox",
          className: "w-28 text-xs text-muted-foreground",
          render: (item) => `#${item.id}`,
        },
        {
          key: "message",
          label: t("wxworkOutbox.columnMessage"),
          className: "w-48",
          render: (item) => (
            <div className="space-y-1 text-xs">
              <div>{t("wxworkOutbox.conversationLine", { id: String(item.conversationId || "-") })}</div>
              <div className="text-muted-foreground">{t("wxworkOutbox.messageLine", { id: String(item.messageId || "-") })}</div>
            </div>
          ),
        },
        {
          key: "status",
          label: t("wxworkOutbox.columnStatus"),
          className: "w-28",
          render: (item) => (
            <Badge variant={statusVariant(item.sendStatus)}>
              {statusLabel(item.sendStatus)}
            </Badge>
          ),
        },
        {
          key: "retry",
          label: t("wxworkOutbox.columnRetry"),
          className: "w-44 text-xs",
          render: (item) => (
            <div className="space-y-1">
              <div>{t("wxworkOutbox.retriesCount", { count: String(item.retryCount) })}</div>
              <div className="text-muted-foreground">
                {t("wxworkOutbox.nextRetry", { time: formatOptionalTime(item.nextRetryAt) })}
              </div>
            </div>
          ),
        },
        {
          key: "error",
          label: t("wxworkOutbox.columnError"),
          className: "min-w-72 max-w-[32rem]",
          render: (item) =>
            item.lastError ? (
              <span className="block truncate text-xs text-destructive" title={item.lastError}>
                {item.lastError}
              </span>
            ) : (
              "-"
            ),
        },
        {
          key: "updatedAt",
          label: t("wxworkOutbox.columnUpdatedAt"),
          className: "w-44 text-xs text-muted-foreground",
          render: (item) => formatOptionalTime(item.updatedAt),
        },
        {
          key: "actions",
          label: <span className="block text-right">{t("wxworkOutbox.columnActions")}</span>,
          className: "w-44",
          render: (item, context) => (
            <OutboxActions item={item} reload={context.reload} />
          ),
        },
      ]}
      labels={{
        refresh: t("wxworkOutbox.refresh"),
        query: t("wxworkOutbox.query"),
        loading: t("wxworkOutbox.loading"),
        empty: t("wxworkOutbox.empty"),
        loadFailed: t("wxworkOutbox.loadFailed"),
      }}
    />
  )
}
