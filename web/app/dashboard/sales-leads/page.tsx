"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import {
  BellRingIcon,
  CalendarClockIcon,
  DownloadIcon,
  ExternalLinkIcon,
  SearchIcon,
  UploadCloudIcon,
} from "lucide-react"
import { toast } from "sonner"

import { DashboardListPage, type DashboardListColumn } from "@/components/dashboard/list"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { fetchUsersAll, type AdminUser } from "@/lib/api/admin"
import {
  claimUnassignedSalesLeads,
  exportSalesLeads,
  fetchSalesLeadAppointmentSummary,
  fetchSalesLeads,
  fetchSalesLeadFollowUpReminderSummary,
  sendSalesLeadAppointmentReminder,
  sendSalesLeadFollowUpReminder,
  syncSalesLeadToCRM,
  type AdminSalesLead,
  type SalesLeadAppointmentSummary,
  type SalesLeadFollowUpReminderSummary,
  type SalesLeadIntent,
  type SalesLeadStage,
  type SalesLeadStatus,
  updateSalesLeadStatus,
} from "@/lib/api/sales-lead"
import { SalesLeadDetailDialog } from "./_components/sales-lead-detail-dialog"

const statusOptions = [
  { value: "all", label: "全部状态" },
  { value: "new", label: "新线索" },
  { value: "following", label: "跟进中" },
  { value: "visited", label: "已到店" },
  { value: "converted", label: "已转化" },
  { value: "invalid", label: "无效" },
  { value: "closed", label: "已关闭" },
]

const intentOptions = [
  { value: "all", label: "全部意向" },
  { value: "high", label: "高意向" },
  { value: "medium", label: "中意向" },
  { value: "low", label: "低意向" },
  { value: "unknown", label: "未知" },
]

const followUpOptions = [
  { value: "all", label: "全部跟进" },
  { value: "overdue", label: "已逾期" },
  { value: "today", label: "今日待跟进" },
  { value: "scheduled", label: "未来已安排" },
  { value: "none", label: "未设置" },
]

const appointmentOptions = [
  { value: "none", label: "全部预约" },
  { value: "overdue", label: "逾期未到店" },
  { value: "today", label: "今日预约" },
  { value: "upcoming", label: "未来到店" },
  { value: "unscheduled", label: "未定时间" },
]

const taskViewOptions = [
  { value: "all", label: "全部线索" },
  { value: "today", label: "今日任务" },
  { value: "overdue", label: "逾期" },
  { value: "high_intent", label: "高意向" },
  { value: "appointment", label: "预约" },
  { value: "after_sales", label: "售后风险" },
]

function statusLabel(status: SalesLeadStatus) {
  const map: Record<SalesLeadStatus, string> = {
    new: "新线索",
    following: "跟进中",
    visited: "已到店",
    converted: "已转化",
    invalid: "无效",
    closed: "已关闭",
  }
  return map[status] ?? status
}

function intentLabel(intent: SalesLeadIntent) {
  const map: Record<SalesLeadIntent, string> = {
    high: "高意向",
    medium: "中意向",
    low: "低意向",
    unknown: "未知",
  }
  return map[intent] ?? intent
}

function stageLabel(stage: SalesLeadStage) {
  const map: Record<SalesLeadStage, string> = {
    appointment: "预约到店",
    ready_to_buy: "准备购买",
    comparing: "对比决策",
    consulting: "咨询了解",
    after_sales: "售后问题",
    unknown: "未知",
  }
  return map[stage] ?? stage
}

function mergeKeyLabel(value?: string) {
  const map: Record<string, string> = {
    new: "新建",
    conversation: "同会话",
    phone: "同手机号",
    wechat: "同微信",
    customer: "同客户",
  }
  return map[value || ""] ?? value ?? "-"
}

function budgetText(item: AdminSalesLead) {
  if (item.budgetMin > 0 && item.budgetMax > 0) {
    return `¥${item.budgetMin.toLocaleString()} - ¥${item.budgetMax.toLocaleString()}`
  }
  if (item.budgetMax > 0) {
    return `¥${item.budgetMax.toLocaleString()} 左右`
  }
  return "-"
}

function appointmentText(item: AdminSalesLead) {
  const parts = [
    item.appointmentAt,
    item.appointmentTimeText,
    item.appointmentStore,
    item.appointmentPeople > 0 ? `${item.appointmentPeople}人` : "",
  ].filter(Boolean)
  return parts.length > 0 ? parts.join(" / ") : "-"
}

function intentVariant(intent: SalesLeadIntent) {
  if (intent === "high") return "default"
  if (intent === "medium") return "secondary"
  return "outline"
}

function statusVariant(status: SalesLeadStatus) {
  if (status === "new") return "default"
  if (status === "following") return "secondary"
  if (status === "visited") return "secondary"
  return "outline"
}

function parseLocalDateTime(value?: string) {
  if (!value) return null
  const date = new Date(value.replace(" ", "T"))
  return Number.isNaN(date.getTime()) ? null : date
}

function followUpState(item: AdminSalesLead) {
  const date = parseLocalDateTime(item.nextFollowUpAt)
  if (!date) {
    return { label: "未设置", variant: "outline" as const }
  }
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const tomorrowStart = new Date(todayStart)
  tomorrowStart.setDate(todayStart.getDate() + 1)
  if (date < todayStart) {
    return { label: "已逾期", variant: "destructive" as const }
  }
  if (date < tomorrowStart) {
    return { label: "今日跟进", variant: "default" as const }
  }
  return { label: "已安排", variant: "secondary" as const }
}

function appointmentStateLabel(state: string) {
  const map: Record<string, string> = {
    overdue: "逾期未到店",
    today: "今日预约",
    upcoming: "即将到店",
    unscheduled: "未定时间",
  }
  return map[state] ?? state
}

function appointmentStateVariant(state: string) {
  if (state === "overdue") return "destructive" as const
  if (state === "today") return "default" as const
  if (state === "upcoming") return "secondary" as const
  return "outline" as const
}

export default function DashboardSalesLeadsPage() {
  const searchParams = useSearchParams()
  const [users, setUsers] = useState<AdminUser[]>([])
  const [claiming, setClaiming] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [updatingStatusKey, setUpdatingStatusKey] = useState<string | null>(null)
  const [syncingCRMLeadId, setSyncingCRMLeadId] = useState<number | null>(null)
  const [appointmentLoading, setAppointmentLoading] = useState(false)
  const [appointmentSending, setAppointmentSending] = useState(false)
  const [appointmentSummary, setAppointmentSummary] =
    useState<SalesLeadAppointmentSummary | null>(null)
  const [reminderLoading, setReminderLoading] = useState(false)
  const [reminderSending, setReminderSending] = useState(false)
  const [reminderSummary, setReminderSummary] =
    useState<SalesLeadFollowUpReminderSummary | null>(null)
  const [selectedLeadId, setSelectedLeadId] = useState<number | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    const leadId = Number(searchParams.get("leadId") || 0)
    if (leadId > 0) {
      setSelectedLeadId(leadId)
    }
  }, [searchParams])

  useEffect(() => {
    void loadReminderSummary()
    void loadAppointmentSummary()
    void loadUsers()
  }, [])

  async function loadUsers() {
    try {
      const result = await fetchUsersAll()
      setUsers(result.filter((user) => user.status === 0))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载负责人失败")
    }
  }

  async function loadAppointmentSummary() {
    setAppointmentLoading(true)
    try {
      const result = await fetchSalesLeadAppointmentSummary({ days: 7, limit: 5 })
      setAppointmentSummary(result)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载预约看板失败")
    } finally {
      setAppointmentLoading(false)
    }
  }

  async function handleSendAppointmentReminder() {
    setAppointmentSending(true)
    try {
      const result = await sendSalesLeadAppointmentReminder({ days: 7, limit: 5 })
      setAppointmentSummary(result)
      if (result.notificationSent) {
        toast.success("预约提醒已发送")
      } else {
        toast.success("当前没有需要提醒的预约事项")
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "发送预约提醒失败")
    } finally {
      setAppointmentSending(false)
    }
  }

  async function loadReminderSummary() {
    setReminderLoading(true)
    try {
      const result = await fetchSalesLeadFollowUpReminderSummary({ limit: 5 })
      setReminderSummary(result)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载跟进提醒失败")
    } finally {
      setReminderLoading(false)
    }
  }

  async function handleSendReminder() {
    setReminderSending(true)
    try {
      const result = await sendSalesLeadFollowUpReminder({ limit: 5 })
      setReminderSummary(result)
      if (result.notificationSent) {
        toast.success("跟进提醒已发送")
      } else {
        toast.success("当前没有需要提醒的跟进事项")
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "发送跟进提醒失败")
    } finally {
      setReminderSending(false)
    }
  }

  async function handleClaimUnassigned(
    query: Record<string, string | number | undefined>,
    reload: () => Promise<void>
  ) {
    setClaiming(true)
    try {
      const result = await claimUnassignedSalesLeads({
        keyword: typeof query.keyword === "string" ? query.keyword : undefined,
        status: typeof query.status === "string" ? query.status : undefined,
        intent: typeof query.intent === "string" ? query.intent : undefined,
        taskView: typeof query.taskView === "string" ? query.taskView : undefined,
        followUpStatus:
          typeof query.followUpStatus === "string" ? query.followUpStatus : undefined,
        appointmentStatus:
          typeof query.appointmentStatus === "string"
            ? query.appointmentStatus
            : undefined,
        limit: 100,
      })
      toast.success(result.message || `已领取 ${result.claimedCount} 条线索`)
      await reload()
      void loadReminderSummary()
      void loadAppointmentSummary()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "领取未分配线索失败")
    } finally {
      setClaiming(false)
    }
  }

  async function handleExport(query: Record<string, string | number | undefined>) {
    setExporting(true)
    try {
      await exportSalesLeads({
        keyword: typeof query.keyword === "string" ? query.keyword : undefined,
        status: typeof query.status === "string" ? query.status : undefined,
        intent: typeof query.intent === "string" ? query.intent : undefined,
        taskView: typeof query.taskView === "string" ? query.taskView : undefined,
        ownerUserId:
          typeof query.ownerUserId === "number" ? query.ownerUserId : undefined,
        followUpStatus:
          typeof query.followUpStatus === "string" ? query.followUpStatus : undefined,
        appointmentStatus:
          typeof query.appointmentStatus === "string"
            ? query.appointmentStatus
            : undefined,
      })
      toast.success("销售线索已导出")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "导出销售线索失败")
    } finally {
      setExporting(false)
    }
  }

  async function handleQuickStatus(
    item: AdminSalesLead,
    status: SalesLeadStatus,
    reload: () => Promise<void>
  ) {
    const key = `${item.id}:${status}`
    setUpdatingStatusKey(key)
    try {
      await updateSalesLeadStatus({
        id: item.id,
        status,
        remark:
          status === "converted"
            ? "列表快捷标记：已转化"
            : status === "visited"
              ? "列表快捷标记：客户已到店"
            : status === "invalid"
              ? "列表快捷标记：无效线索"
              : "",
      })
      toast.success(
        status === "converted"
          ? "已标记成交"
          : status === "visited"
            ? "已标记到店"
            : "已标记无效"
      )
      await reload()
      void loadReminderSummary()
      void loadAppointmentSummary()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "更新线索状态失败")
    } finally {
      setUpdatingStatusKey(null)
    }
  }

  async function handleSyncCRM(item: AdminSalesLead) {
    setSyncingCRMLeadId(item.id)
    try {
      const result = await syncSalesLeadToCRM(item.id, "销售线索列表手动同步")
      if (result.sent) {
        toast.success(result.message || "线索已同步到 CRM")
      } else {
        toast.info(result.message || "CRM Webhook 未启用")
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "同步 CRM 失败")
    } finally {
      setSyncingCRMLeadId(null)
    }
  }

  const columns: DashboardListColumn<AdminSalesLead>[] = [
    {
      key: "customer",
      label: "客户",
      render: (item) => (
        <div className="min-w-36">
          <div className="font-medium">{item.customerName || "未命名客户"}</div>
          <div className="text-xs text-muted-foreground">
            {item.phone || item.wechat || "-"}
          </div>
        </div>
      ),
    },
    {
      key: "intent",
      label: "意向",
      className: "w-24",
      render: (item) => (
        <Badge variant={intentVariant(item.intentLevel)}>
          {intentLabel(item.intentLevel)}
        </Badge>
      ),
    },
    {
      key: "stage",
      label: "阶段",
      className: "w-28",
      render: (item) => (
        <span className="text-muted-foreground">
          {stageLabel(item.buyingStage)}
        </span>
      ),
    },
    {
      key: "autoTags",
      label: "自动标签",
      className: "w-56",
      render: (item) => (
        <div className="flex max-w-[220px] flex-wrap gap-1">
          {(item.autoTags ?? []).length > 0 ? (
            item.autoTags.slice(0, 4).map((tag) => (
              <Badge key={tag} variant="outline" className="max-w-full truncate">
                {tag}
              </Badge>
            ))
          ) : (
            <span className="text-xs text-muted-foreground">-</span>
          )}
        </div>
      ),
    },
    {
      key: "mergeReason",
      label: "归并依据",
      className: "w-56",
      render: (item) => (
        <div className="max-w-[220px] space-y-1">
          <Badge variant={item.mergeKey === "new" ? "outline" : "secondary"}>
            {mergeKeyLabel(item.mergeKey)}
          </Badge>
          <div className="truncate text-xs text-muted-foreground">
            {item.mergeReason || "-"}
          </div>
        </div>
      ),
    },
    {
      key: "budget",
      label: "预算",
      className: "w-40",
      render: (item) => (
        <span className="text-muted-foreground">{budgetText(item)}</span>
      ),
    },
    {
      key: "appointment",
      label: "预约",
      className: "w-56",
      render: (item) => (
        <div className="max-w-[220px] truncate text-muted-foreground">
          {appointmentText(item)}
        </div>
      ),
    },
    {
      key: "products",
      label: "意向产品",
      render: (item) => (
        <div className="max-w-[220px] truncate text-muted-foreground">
          {item.interestedProducts || "-"}
        </div>
      ),
    },
    {
      key: "summary",
      label: "需求摘要",
      render: (item) => (
        <div className="line-clamp-2 max-w-[360px] text-muted-foreground">
          {item.demandSummary || "-"}
        </div>
      ),
    },
    {
      key: "recentMessage",
      label: "最近消息",
      render: (item) => (
        <div className="max-w-[320px] space-y-1">
          {item.lastCustomerMessage ? (
            <div className="line-clamp-2 text-muted-foreground">
              客户：{item.lastCustomerMessage}
            </div>
          ) : null}
          {item.lastMessageSummary ? (
            <div className="line-clamp-1 text-xs text-muted-foreground">
              摘要：{item.lastMessageSummary}
            </div>
          ) : null}
          {!item.lastCustomerMessage && !item.lastMessageSummary ? (
            <span className="text-xs text-muted-foreground">-</span>
          ) : null}
        </div>
      ),
    },
    {
      key: "status",
      label: "状态",
      className: "w-24",
      render: (item) => (
        <Badge variant={statusVariant(item.status)}>
          {statusLabel(item.status)}
        </Badge>
      ),
    },
    {
      key: "nextFollowUpAt",
      label: "下次跟进",
      className: "w-40",
      render: (item) => {
        const state = followUpState(item)
        return (
          <div className="space-y-1">
            <Badge variant={state.variant}>{state.label}</Badge>
            <div className="text-xs text-muted-foreground">
              {item.nextFollowUpAt || "-"}
            </div>
          </div>
        )
      },
    },
    {
      key: "createdAt",
      label: "创建时间",
      className: "w-40",
      render: (item) => (
        <span className="text-muted-foreground">{item.createdAt || "-"}</span>
      ),
    },
    {
      key: "actions",
      label: "",
      className: "w-96 text-right",
      render: (item, context) => (
        <div className="flex justify-end gap-2">
          {item.status !== "visited" && item.status !== "converted" && item.status !== "invalid" && item.status !== "closed" ? (
            <Button
              variant="outline"
              size="sm"
              disabled={updatingStatusKey !== null}
              onClick={(event) => {
                event.stopPropagation()
                void handleQuickStatus(item, "visited", context.reload)
              }}
            >
              {updatingStatusKey === `${item.id}:visited` ? "处理中" : "到店"}
            </Button>
          ) : null}
          {item.status !== "converted" && item.status !== "closed" ? (
            <Button
              variant="outline"
              size="sm"
              disabled={updatingStatusKey !== null}
              onClick={(event) => {
                event.stopPropagation()
                void handleQuickStatus(item, "converted", context.reload)
              }}
            >
              {updatingStatusKey === `${item.id}:converted` ? "处理中" : "成交"}
            </Button>
          ) : null}
          {item.status !== "invalid" && item.status !== "converted" && item.status !== "closed" ? (
            <Button
              variant="outline"
              size="sm"
              disabled={updatingStatusKey !== null}
              onClick={(event) => {
                event.stopPropagation()
                void handleQuickStatus(item, "invalid", context.reload)
              }}
            >
              {updatingStatusKey === `${item.id}:invalid` ? "处理中" : "无效"}
            </Button>
          ) : null}
          <Button
            variant="outline"
            size="sm"
            disabled={syncingCRMLeadId !== null}
            onClick={(event) => {
              event.stopPropagation()
              void handleSyncCRM(item)
            }}
          >
            <UploadCloudIcon className={syncingCRMLeadId === item.id ? "animate-pulse" : undefined} />
            {syncingCRMLeadId === item.id ? "同步中" : "CRM"}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={(event) => {
              event.stopPropagation()
              setSelectedLeadId(item.id)
            }}
          >
            详情
          </Button>
          <Link
            className={buttonVariants({ variant: "outline", size: "sm" })}
            href={`/dashboard/conversations?conversationId=${item.conversationId}`}
            onClick={(event) => event.stopPropagation()}
          >
            <ExternalLinkIcon />
            会话
          </Link>
        </div>
      ),
    },
  ]

  return (
    <>
      <div className="mb-4 rounded-md border bg-background p-4">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="min-w-0 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <CalendarClockIcon className="size-4 text-muted-foreground" />
              <div className="font-medium">预约到店看板</div>
              {appointmentLoading ? (
                <Badge variant="outline">加载中</Badge>
              ) : (
                <Badge variant="outline">
                  未来 {appointmentSummary?.days ?? 7} 天
                </Badge>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleSendAppointmentReminder()}
                disabled={appointmentSending || appointmentLoading}
              >
                <BellRingIcon
                  className={appointmentSending ? "animate-pulse" : undefined}
                />
                {appointmentSending ? "发送中" : "发送预约提醒"}
              </Button>
            </div>
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
              <div className="rounded-md border px-3 py-2">
                <div className="text-xs text-muted-foreground">今日预约</div>
                <div className="text-lg font-semibold">
                  {appointmentSummary?.todayCount ?? 0}
                </div>
              </div>
              <div className="rounded-md border px-3 py-2">
                <div className="text-xs text-muted-foreground">未来到店</div>
                <div className="text-lg font-semibold">
                  {appointmentSummary?.upcomingCount ?? 0}
                </div>
              </div>
              <div className="rounded-md border px-3 py-2">
                <div className="text-xs text-muted-foreground">逾期未到店</div>
                <div className="text-lg font-semibold text-destructive">
                  {appointmentSummary?.overdueCount ?? 0}
                </div>
              </div>
              <div className="rounded-md border px-3 py-2">
                <div className="text-xs text-muted-foreground">未定时间</div>
                <div className="text-lg font-semibold">
                  {appointmentSummary?.unscheduledCount ?? 0}
                </div>
              </div>
              <div className="rounded-md border px-3 py-2">
                <div className="text-xs text-muted-foreground">未分配</div>
                <div className="text-lg font-semibold">
                  {appointmentSummary?.unassignedCount ?? 0}
                </div>
              </div>
            </div>
          </div>
          <div className="w-full space-y-2 xl:max-w-xl">
            <div className="text-sm font-medium">重点预约</div>
            {appointmentSummary?.previewAppointments?.length ? (
              <div className="space-y-2">
                {appointmentSummary.previewAppointments.map((item) => {
                  const timeText =
                    item.appointmentAt ||
                    item.appointmentTimeText ||
                    "未定时间"
                  return (
                    <button
                      key={item.id}
                      type="button"
                      className="w-full rounded-md border px-3 py-2 text-left transition hover:bg-muted"
                      onClick={() => setSelectedLeadId(item.id)}
                    >
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">
                          {item.customerName || `线索 #${item.id}`}
                        </span>
                        <Badge variant={appointmentStateVariant(item.appointmentState)}>
                          {appointmentStateLabel(item.appointmentState)}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {item.phone || item.wechat || "暂无联系方式"}
                        </span>
                      </div>
                      <div className="mt-1 line-clamp-1 text-xs text-muted-foreground">
                        {timeText}
                        {item.appointmentStore ? ` / ${item.appointmentStore}` : ""}
                        {item.appointmentPeople > 0
                          ? ` / ${item.appointmentPeople}人`
                          : ""}
                        {item.demandSummary ? ` / ${item.demandSummary}` : ""}
                      </div>
                    </button>
                  )
                })}
              </div>
            ) : (
              <div className="rounded-md border border-dashed px-3 py-5 text-sm text-muted-foreground">
                暂无预约线索
              </div>
            )}
          </div>
        </div>
      </div>
      <DashboardListPage<AdminSalesLead>
        filters={[
          {
            name: "taskView",
            label: "任务视图",
            type: "segment",
            defaultValue: "all",
            allValue: "all",
            options: taskViewOptions,
            className: "flex w-full flex-wrap gap-2",
          },
          {
            name: "keyword",
            label: "关键词",
            placeholder: "姓名、电话、需求、产品",
            defaultValue: "",
            trim: true,
            className: "w-full sm:w-72",
            icon: <SearchIcon />,
            inputClassName: "pl-9",
          },
          {
            name: "status",
            label: "状态",
            type: "select",
            defaultValue: "all",
            allValue: "all",
            options: statusOptions,
            className: "w-full sm:w-36",
          },
          {
            name: "intent",
            label: "意向",
            type: "select",
            defaultValue: "all",
            allValue: "all",
            options: intentOptions,
            className: "w-full sm:w-36",
          },
          {
            name: "followUpStatus",
            label: "跟进",
            type: "select",
            defaultValue: "all",
            allValue: "all",
            options: followUpOptions,
            className: "w-full sm:w-40",
          },
          {
            name: "appointmentStatus",
            label: "预约",
            type: "select",
            defaultValue: "none",
            allValue: "none",
            options: appointmentOptions,
            className: "w-full sm:w-40",
          },
          {
            name: "ownerUserId",
            label: "负责人",
            type: "select",
            defaultValue: 0,
            allValue: 0,
            valueType: "number",
            placeholder: "全部负责人",
            options: [
              { value: "0", label: "全部负责人" },
              { value: "-1", label: "未分配" },
              ...users.map((user) => ({
                value: String(user.id),
                label: user.nickname || user.username,
              })),
            ],
            className: "w-full sm:w-44",
          },
        ]}
        columns={columns}
        fetchList={(query) =>
          fetchSalesLeads({
            keyword: typeof query.keyword === "string" ? query.keyword : undefined,
            status: typeof query.status === "string" ? query.status : undefined,
            intent: typeof query.intent === "string" ? query.intent : undefined,
            taskView:
              typeof query.taskView === "string" ? query.taskView : undefined,
            followUpStatus:
              typeof query.followUpStatus === "string"
                ? query.followUpStatus
                : undefined,
            appointmentStatus:
              typeof query.appointmentStatus === "string"
                ? query.appointmentStatus
                : undefined,
            ownerUserId:
              typeof query.ownerUserId === "number" ? query.ownerUserId : undefined,
            page: Number(query.page),
            limit: Number(query.limit),
          })
        }
        getItemId={(item) => item.id}
        getRowClassName={() => "cursor-pointer"}
        onRowClick={(item) => setSelectedLeadId(item.id)}
        reloadKey={reloadKey}
        renderToolbarActions={({ query, reload }) => (
          <div className="flex flex-wrap items-center justify-end gap-2">
            <div className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
              <Badge variant={reminderSummary?.overdueCount ? "destructive" : "outline"}>
                逾期 {reminderSummary?.overdueCount ?? 0}
              </Badge>
              <Badge variant={reminderSummary?.todayCount ? "default" : "outline"}>
                今日 {reminderSummary?.todayCount ?? 0}
              </Badge>
              <Badge variant={reminderSummary?.unassignedDueCount ? "secondary" : "outline"}>
                未分配 {reminderSummary?.unassignedDueCount ?? 0}
              </Badge>
              <Badge variant={reminderSummary?.missingScheduleCount ? "secondary" : "outline"}>
                未设置 {reminderSummary?.missingScheduleCount ?? 0}
              </Badge>
            </div>
            {query.ownerUserId === -1 ? (
              <Button
                variant="outline"
                onClick={() => void handleClaimUnassigned(query, reload)}
                disabled={claiming}
              >
                {claiming ? "领取中" : "领取未分配"}
              </Button>
            ) : null}
            <Button
              variant="outline"
              onClick={() => void handleSendReminder()}
              disabled={reminderSending || reminderLoading}
            >
              <BellRingIcon className={reminderSending ? "animate-pulse" : undefined} />
              {reminderSending ? "发送中" : "发送跟进提醒"}
            </Button>
            <Button
              variant="outline"
              onClick={() => void handleExport(query)}
              disabled={exporting}
            >
              <DownloadIcon className={exporting ? "animate-pulse" : undefined} />
              {exporting ? "导出中" : "导出"}
            </Button>
          </div>
        )}
        labels={{
          refresh: "刷新",
          query: "查询",
          loading: "加载销售线索中",
          empty: "暂无销售线索",
          loadFailed: "加载销售线索失败",
        }}
      />
      <SalesLeadDetailDialog
        leadId={selectedLeadId}
        open={selectedLeadId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedLeadId(null)
          }
        }}
        onChanged={() => {
          setReloadKey((value) => value + 1)
          void loadReminderSummary()
          void loadAppointmentSummary()
        }}
      />
    </>
  )
}
