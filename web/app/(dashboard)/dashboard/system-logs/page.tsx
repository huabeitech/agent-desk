"use client"

import { useState } from "react"
import { ScrollTextIcon } from "lucide-react"
import { toast } from "sonner"

import { DashboardListPage } from "@/components/dashboard/list"
import { JsonTreeViewer } from "@/components/json-tree-viewer"
import { ProjectDialog } from "@/components/project-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { fetchSystemLog, fetchSystemLogs, type SystemLog } from "@/lib/api/admin"
import { formatDateTime } from "@/lib/utils"
import { useI18n } from "@/i18n/provider"

function levelVariant(level: string) {
  if (level === "ERROR") return "destructive" as const
  if (level === "WARN") return "secondary" as const
  return "outline" as const
}

function parseJSON(raw: string): unknown | null {
  try {
    return raw?.trim() ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export default function DashboardSystemLogsPage() {
  const t = useI18n()
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [log, setLog] = useState<SystemLog | null>(null)

  async function openDetail(id: number) {
    setOpen(true)
    setLoading(true)
    try {
      setLog(await fetchSystemLog(id))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("systemLog.loadDetailFailed"))
      setOpen(false)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <DashboardListPage<SystemLog>
        filters={[
          {
            name: "level",
            label: t("systemLog.level"),
            type: "select",
            defaultValue: "",
            placeholder: t("systemLog.levelPlaceholder"),
            className: "w-full sm:w-36",
            options: [
              { value: "INFO", label: t("systemLog.levelInfo") },
              { value: "WARN", label: t("systemLog.levelWarn") },
              { value: "ERROR", label: t("systemLog.levelError") },
            ],
          },
          {
            name: "message",
            label: t("systemLog.message"),
            defaultValue: "",
            placeholder: t("systemLog.messagePlaceholder"),
            className: "w-full sm:w-56",
          },
          {
            name: "source",
            label: t("systemLog.source"),
            defaultValue: "",
            placeholder: t("systemLog.sourcePlaceholder"),
            className: "w-full sm:w-48",
          },
          {
            name: "startTime",
            label: t("systemLog.startTime"),
            defaultValue: "",
            placeholder: "2026-09-01 00:00:00",
            className: "w-full sm:w-48",
          },
          {
            name: "endTime",
            label: t("systemLog.endTime"),
            defaultValue: "",
            placeholder: "2026-09-15 23:59:59",
            className: "w-full sm:w-48",
          },
        ]}
        fetchList={fetchSystemLogs}
        getItemId={(item) => item.id}
        getRowClassName={() => "cursor-pointer"}
        onRowClick={(item) => void openDetail(item.id)}
        columns={[
          {
            key: "createdAt",
            label: t("systemLog.createdAt"),
            className: "w-44 text-xs text-muted-foreground",
            render: (item) => formatDateTime(item.createdAt),
          },
          {
            key: "level",
            label: t("systemLog.level"),
            className: "w-24",
            render: (item) => <Badge variant={levelVariant(item.level)}>{item.levelName || item.level}</Badge>,
          },
          {
            key: "message",
            label: t("systemLog.message"),
            className: "max-w-96",
            render: (item) => (
              <span className="block truncate text-xs" title={item.message}>
                {item.message}
              </span>
            ),
          },
          {
            key: "source",
            label: t("systemLog.source"),
            className: "w-64 max-w-64 text-xs text-muted-foreground",
            render: (item) => (
              <span className="block truncate" title={item.source}>
                {item.source || "-"}
              </span>
            ),
          },
          {
            key: "loggerName",
            label: t("systemLog.logger"),
            className: "w-28 text-xs text-muted-foreground",
            render: (item) => item.loggerName || "-",
          },
        ]}
        labels={{
          refresh: t("systemLog.refresh"),
          query: t("systemLog.query"),
          loading: t("systemLog.loading"),
          empty: t("systemLog.empty"),
          loadFailed: t("systemLog.loadFailed"),
        }}
      />
      <ProjectDialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next)
          if (!next) setLog(null)
        }}
        size="xl"
        title={
          <span className="flex items-center gap-2">
            <ScrollTextIcon className="size-4" />
            {t("systemLog.detailTitle")}
          </span>
        }
        description={log ? `#${log.id} · ${log.level}` : t("systemLog.detailDescription")}
        footer={
          <Button variant="outline" onClick={() => setOpen(false)}>
            {t("systemLog.close")}
          </Button>
        }
      >
        {loading ? (
          <div className="py-10 text-sm text-muted-foreground">{t("systemLog.loadingDetail")}</div>
        ) : log ? (
          <div className="space-y-4">
            <div className="flex flex-wrap gap-2 rounded-md border bg-muted/20 px-3 py-2 text-xs">
              <Meta label={t("systemLog.level")} value={log.levelName || log.level} />
              <Meta label={t("systemLog.logger")} value={log.loggerName || "-"} />
              <Meta label={t("systemLog.createdAt")} value={formatDateTime(log.createdAt)} />
            </div>
            {log.source ? (
              <div>
                <div className="mb-1 text-xs text-muted-foreground">{t("systemLog.source")}</div>
                <code className="block w-full rounded-md border bg-muted/20 px-2 py-1.5 text-xs">{log.source}</code>
              </div>
            ) : null}
            <div>
              <div className="mb-1 text-xs text-muted-foreground">{t("systemLog.message")}</div>
              <pre className="max-h-60 overflow-auto rounded-md border bg-muted/20 p-2 text-xs whitespace-pre-wrap break-all">
                {log.message}
              </pre>
            </div>
            {log.attrs ? (
              <div>
                <div className="mb-1 text-xs text-muted-foreground">{t("systemLog.attrs")}</div>
                {(() => {
                  const parsed = parseJSON(log.attrs)
                  return parsed !== null ? (
                    <JsonTreeViewer value={parsed} collapsed={2} />
                  ) : (
                    <pre className="max-h-60 overflow-auto rounded-md border bg-muted/20 p-2 text-xs whitespace-pre-wrap break-all">
                      {log.attrs}
                    </pre>
                  )
                })()}
              </div>
            ) : null}
          </div>
        ) : (
          <div className="py-10 text-sm text-muted-foreground">{t("systemLog.notFound")}</div>
        )}
      </ProjectDialog>
    </>
  )
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-md border bg-background px-2 py-1">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium">{value || "-"}</span>
    </span>
  )
}
