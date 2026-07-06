"use client"

import { useCallback, useEffect, useState } from "react"
import { ClipboardIcon, RefreshCwIcon, SendIcon } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent } from "@/components/ui/card"
import { useI18n } from "@/i18n/provider"
import {
  createKnowledgeFAQDraftFromRetrieveLog,
  type KnowledgeFAQ,
} from "@/lib/api/admin"
import {
  fetchABTestReport,
  fetchAIQualityReport,
  fetchBusinessTrendReport,
  fetchDailyBusinessReport,
  fetchDashboardOverview,
  fetchSalesFunnelReport,
  sendDailyBusinessReport,
  type DashboardABTestReport,
  type DashboardAIQualityReport,
  type DashboardBusinessTrendReport,
  type DashboardDailyBusinessReport,
  type DashboardOverview,
  type DashboardRange,
  type DashboardSalesFunnelReport,
} from "@/lib/api/dashboard"
import { SummaryCards } from "./summary-cards"
import { TrendPanel } from "./trend-panel"
import { TeamLoadPanel } from "./team-load-panel"
import { AlertList } from "./alert-list"

function LoadingCards() {
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
      {Array.from({ length: 6 }).map((_, index) => (
        <Card key={index} className="rounded-md shadow-none">
          <CardContent className="space-y-3 p-4">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-8 w-20" />
            <Skeleton className="h-4 w-full" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function DigitalStorePanel({ stats }: { stats: DashboardOverview["digitalStoreStats"] }) {
  const t = useI18n()
  const metrics = [
    {
      label: t("dashboardHome.digitalStoreTodayConsultations"),
      value: stats.todayConsultations,
    },
    {
      label: t("dashboardHome.digitalStoreTodayLeads"),
      value: stats.todayLeads,
    },
    {
      label: t("dashboardHome.digitalStoreLeadConversionRate"),
      value: `${stats.leadConversionRate.toFixed(1)}%`,
    },
    {
      label: t("dashboardHome.digitalStoreHighIntent"),
      value: stats.todayHighIntentLeads,
    },
    {
      label: t("dashboardHome.digitalStoreAppointments"),
      value: stats.todayAppointmentLeads,
    },
    {
      label: t("dashboardHome.digitalStoreConverted"),
      value: stats.todayConvertedLeads,
    },
    {
      label: t("dashboardHome.digitalStorePendingFollowUp"),
      value: stats.pendingFollowUpLeads,
    },
    {
      label: t("dashboardHome.digitalStoreActiveProducts"),
      value: stats.activeProducts,
    },
    {
      label: t("dashboardHome.digitalStoreActivePromotions"),
      value: stats.activePromotions,
    },
    {
      label: t("dashboardHome.digitalStoreHandoffs"),
      value: stats.todayHandoffs,
    },
  ]

  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">{t("dashboardHome.digitalStoreTitle")}</div>
            <div className="mt-1 text-sm text-muted-foreground">{stats.summary}</div>
          </div>
        </div>
        <div className="grid gap-3 lg:grid-cols-[1fr_280px]">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {metrics.map((item) => (
              <div key={item.label} className="min-w-0 rounded-md border bg-background px-3 py-2.5">
                <div className="truncate text-sm text-muted-foreground">{item.label}</div>
                <div className="mt-1 text-2xl font-semibold">{item.value}</div>
              </div>
            ))}
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.digitalStoreTopProducts")}</div>
            <div className="mt-3 space-y-2">
              {stats.topLeadProducts.length > 0 ? (
                stats.topLeadProducts.map((item) => (
                  <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                    <span className="shrink-0 font-medium">{item.count}</span>
                  </div>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">{t("dashboardHome.digitalStoreNoTopProducts")}</div>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function buildDailyReportText(report: DashboardDailyBusinessReport) {
  const sections = [
    report.summary,
    "",
    "经营亮点",
    ...report.highlights.map((item) => `- ${item}`),
    "",
    "成交结果",
    `- 今日成交线索：${report.convertedCount}`,
    "",
    "热门咨询问题",
    ...(report.topQuestions.length > 0
      ? report.topQuestions.map((item) => `- ${item.name}（${item.count}次）`)
      : ["- 暂无"]),
    "",
    "未解决问题",
    ...(report.unansweredQuestions.length > 0
      ? report.unansweredQuestions.map((item) => `- ${item.name}（${item.count}次）`)
      : ["- 暂无"]),
    "",
    "跟进风险",
    `- 逾期未跟进：${report.overdueFollowUpCount}`,
    `- 今日待跟进：${report.todayFollowUpCount}`,
    `- 未排计划高意向/预约：${report.unscheduledHotLeads}`,
    `- 未分配重点线索：${report.unassignedPriorityLeadCount}`,
    "",
    "预约风险",
    `- 逾期未到店：${report.overdueAppointmentCount}`,
    `- 今日预约：${report.todayAppointmentCount}`,
    `- 未定时间：${report.unscheduledAppointmentCount}`,
    "",
    "售后/投诉风险",
    `- 未处理工单：${report.pendingAfterSalesTicketCount}`,
    `- 今日新增：${report.todayAfterSalesTicketCount}`,
    `- 今日已处理：${report.todayHandledAfterSalesTicketCount}`,
    ...(report.afterSalesTickets.length > 0
      ? report.afterSalesTickets.map((ticket) => {
          const owner = ticket.currentAssigneeName || "未分配"
          const progress = ticket.latestProgress
            ? `｜最近进展：${ticket.latestProgress}${ticket.latestProgressAt ? `（${ticket.latestProgressAt}）` : ""}`
            : ""
          return `- ${ticket.ticketNo || `#${ticket.id}`}｜${ticketStatusText(ticket.status)}｜负责人：${owner}｜${ticket.title}｜${ticket.description || "暂无描述"}${progress}`
        })
      : ["- 暂无"]),
    "",
    "AI 质量反馈",
    `- 今日反馈：${report.aiFeedbackCount}`,
    `- 点赞：${report.aiFeedbackLikeCount}`,
    `- 负反馈：${report.aiFeedbackNegativeCount}`,
    `- 负反馈率：${report.aiFeedbackNegativeRate.toFixed(1)}%`,
    ...(report.topAiFeedbackReasons.length > 0
      ? report.topAiFeedbackReasons.map((item) => `- ${item.name}（${item.count}次）`)
      : ["- 暂无负反馈原因"]),
    ...(report.recentNegativeAiFeedbacks.length > 0
      ? [
          "- 最近负反馈明细：",
          ...report.recentNegativeAiFeedbacks.map((item) => {
            const reason = item.feedbackReason || item.feedbackTypeName || "未填写原因"
            const question = item.question || `检索日志 #${item.retrieveLogId}`
            return `  - ${item.createdAt}｜${item.feedbackTypeName}｜${question}｜${reason}｜${knowledgeRetrieveLogHref(item.retrieveLogId, item.knowledgeBaseId)}`
          }),
        ]
      : []),
    "",
    "待确认 FAQ 草稿",
    `- 待确认数量：${report.pendingFaqDraftCount}`,
    ...(report.pendingFaqDrafts.length > 0
      ? report.pendingFaqDrafts.map((item) => {
          return `- ${item.createdAt}｜${item.question || `FAQ #${item.id}`}｜${knowledgeFaqHref(item.id, item.knowledgeBaseId)}`
        })
      : ["- 暂无"]),
    "",
    "跟进建议",
    ...report.followUpSuggestions.map((item) => `- ${item}`),
    "",
    "优先跟进名单",
    ...(report.priorityFollowUps.length > 0
      ? report.priorityFollowUps.map((lead) => {
          const contact = lead.phone || lead.wechat || "-"
          const owner = lead.ownerUserName || "未分配"
          return `- ${lead.customerName || "未命名客户"}｜${contact}｜${followUpStateText(lead.followUpState)}｜${lead.nextFollowUpAt || "未设置"}｜负责人：${owner}｜${lead.demandSummary || "暂无需求摘要"}`
        })
      : ["- 暂无"]),
    "",
    "知识库建议",
    ...report.knowledgeSuggestions.map((item) => `- ${item}`),
  ]
  if (report.highIntentLeads.length > 0) {
    sections.push(
      "",
      "高意向线索",
      ...report.highIntentLeads.map((lead) => {
        const contact = lead.phone || lead.wechat || "-"
        const appointment = [
          lead.appointmentAt,
          lead.appointmentTimeText,
          lead.appointmentStore,
          lead.appointmentPeople > 0 ? `${lead.appointmentPeople}人` : "",
        ].filter(Boolean).join(" / ")
        return `- ${lead.customerName || "未命名客户"}｜${contact}｜${lead.interestedProducts || "未填写产品"}｜${appointment || "未填写预约"}｜${lead.demandSummary || "暂无需求摘要"}`
      })
    )
  }
  return sections.join("\n")
}

function knowledgeRetrieveLogHref(retrieveLogId: number, knowledgeBaseId?: number) {
  const params = new URLSearchParams({
    tab: "retrieveLogs",
    retrieveLogId: String(retrieveLogId),
  })
  if (knowledgeBaseId) {
    params.set("knowledgeBaseId", String(knowledgeBaseId))
  }
  return `/dashboard/knowledge?${params.toString()}`
}

function knowledgeFaqHref(faqId: number, knowledgeBaseId?: number) {
  const params = new URLSearchParams({
    tab: "documents",
    faqId: String(faqId),
  })
  if (knowledgeBaseId) {
    params.set("knowledgeBaseId", String(knowledgeBaseId))
  }
  return `/dashboard/knowledge?${params.toString()}`
}

function followUpStateText(value?: string) {
  if (value === "overdue") return "已逾期"
  if (value === "today") return "今日跟进"
  if (value === "scheduled") return "已安排"
  if (value === "unscheduled") return "未设置"
  return value || "-"
}

function ticketStatusText(value?: string) {
  if (value === "pending") return "待处理"
  if (value === "in_progress") return "处理中"
  if (value === "done") return "已处理"
  return value || "-"
}

function ticketStatusVariant(value?: string) {
  if (value === "pending") return "destructive" as const
  if (value === "in_progress") return "default" as const
  return "outline" as const
}

function followUpStateVariant(value?: string) {
  if (value === "overdue") return "destructive" as const
  if (value === "today") return "default" as const
  if (value === "unscheduled") return "secondary" as const
  return "outline" as const
}

function qualityTodoVariant(level?: string) {
  if (level === "error") return "destructive" as const
  if (level === "warning") return "secondary" as const
  return "outline" as const
}

function AIQualityPanel({ report }: { report: DashboardAIQualityReport }) {
  const [creatingDraftId, setCreatingDraftId] = useState<number | null>(null)
  const [createdDrafts, setCreatedDrafts] = useState<Record<number, KnowledgeFAQ>>({})
  const handleCreateFAQDraft = async (retrieveLogId: number) => {
    if (!retrieveLogId || creatingDraftId) return
    setCreatingDraftId(retrieveLogId)
    try {
      const draft = await createKnowledgeFAQDraftFromRetrieveLog({
        retrieveLogId,
        remark: "由首页 AI 质检待办生成的待确认 FAQ 草稿",
      })
      setCreatedDrafts((prev) => ({ ...prev, [retrieveLogId]: draft }))
      toast.success(draft.status === 0 ? "已复用现有 FAQ" : "FAQ 草稿已生成，确认答案后启用")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "生成 FAQ 草稿失败")
    } finally {
      setCreatingDraftId(null)
    }
  }
  const metrics = [
    { label: "检索次数", value: report.retrieveTotal },
    { label: "命中率", value: `${report.retrieveHitRate.toFixed(1)}%` },
    { label: "风险回复", value: report.riskAnswerCount },
    { label: "负反馈率", value: `${report.negativeFeedbackRate.toFixed(1)}%` },
    { label: "待确认 FAQ", value: report.pendingFaqDraftCount },
  ]
  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">AI 质检待办</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {report.startDate} 至 {report.endDate}，集中处理未命中、兜底、风控和负反馈。
            </div>
          </div>
          <Badge variant={report.todoTotal ? "secondary" : "default"}>
            待办 {report.todoTotal}
          </Badge>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          {metrics.map((item) => (
            <div key={item.label} className="rounded-md border bg-background px-3 py-2.5">
              <div className="truncate text-sm text-muted-foreground">{item.label}</div>
              <div className="mt-1 text-2xl font-semibold">{item.value}</div>
            </div>
          ))}
        </div>
        <div className="grid gap-3 lg:grid-cols-[1.2fr_0.8fr]">
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">待处理事项</div>
            <div className="mt-3 grid gap-2 md:grid-cols-2">
              {report.todos.length > 0 ? (
                report.todos.map((todo) => (
                  <a
                    key={todo.key}
                    href={todo.actionHref || "/dashboard/knowledge"}
                    className="min-w-0 rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="truncate font-medium">{todo.title}</span>
                      <Badge variant={qualityTodoVariant(todo.level)}>{todo.count}</Badge>
                    </div>
                    <div className="mt-1 line-clamp-2 text-muted-foreground">
                      {todo.description}
                    </div>
                    <div className="mt-2 text-xs text-primary">
                      {todo.actionLabel || "去处理"}
                    </div>
                  </a>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">当前周期暂无 AI 质检待办。</div>
              )}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">知识修正建议</div>
            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
              {report.knowledgeSuggestions.map((item) => (
                <div key={item}>{item}</div>
              ))}
            </div>
          </div>
        </div>
        <div className="rounded-md border bg-background p-3">
          <div className="flex items-center justify-between gap-3">
            <div className="text-sm font-medium">高频待处理问题</div>
            <Badge variant={report.pendingQuestionGroups.length ? "secondary" : "outline"}>
              {report.pendingQuestionGroups.length}
            </Badge>
          </div>
          <div className="mt-3 grid gap-2 lg:grid-cols-2">
            {report.pendingQuestionGroups.length > 0 ? (
              report.pendingQuestionGroups.slice(0, 6).map((item) => {
	                const createdDraft = createdDrafts[item.latestRetrieveLogId]
	                return (
	                  <div
	                    key={`${item.question}-${item.latestRetrieveLogId}`}
	                    className="min-w-0 rounded-md border px-3 py-2 text-sm"
	                  >
	                    <div className="flex items-center justify-between gap-2">
	                      <span className="min-w-0 truncate font-medium">{item.question}</span>
	                      <Badge variant="secondary">{item.count}</Badge>
	                    </div>
	                    <div className="mt-2 flex flex-wrap gap-1 text-xs text-muted-foreground">
	                      {item.noAnswerCount > 0 ? <span>无答案 {item.noAnswerCount}</span> : null}
	                      {item.fallbackCount > 0 ? <span>兜底 {item.fallbackCount}</span> : null}
	                      {item.blockedCount > 0 ? <span>风控 {item.blockedCount}</span> : null}
	                      {item.negativeFeedbackCount > 0 ? <span>负反馈 {item.negativeFeedbackCount}</span> : null}
	                    </div>
	                    <div className="mt-3 flex flex-wrap gap-2">
	                      <a
	                        href={item.actionHref || "/dashboard/knowledge?tab=retrieveLogs"}
	                        className="inline-flex h-8 items-center justify-center rounded-md border bg-background px-3 text-xs font-medium transition-colors hover:bg-muted"
	                      >
	                        {item.actionLabel || "查看日志"}
	                      </a>
	                      <Button
	                        variant="outline"
	                        size="sm"
	                        onClick={() => void handleCreateFAQDraft(item.latestRetrieveLogId)}
	                        disabled={!item.latestRetrieveLogId || creatingDraftId === item.latestRetrieveLogId}
	                      >
	                        {creatingDraftId === item.latestRetrieveLogId
	                          ? "生成中"
	                          : createdDraft
	                            ? "已生成草稿"
	                            : "生成 FAQ 草稿"}
	                      </Button>
	                    </div>
	                  </div>
	                )
	              })
            ) : (
              <div className="text-sm text-muted-foreground">当前周期暂无高频待处理问题。</div>
            )}
          </div>
        </div>
        <div className="grid gap-3 lg:grid-cols-3">
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">未解决问题</div>
            <div className="mt-3 space-y-2">
              {report.unansweredQuestions.length > 0 ? (
                report.unansweredQuestions.slice(0, 5).map((item) => (
                  <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                    <span className="shrink-0 font-medium">{item.count}</span>
                  </div>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">暂无未解决问题</div>
              )}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">负反馈原因</div>
            <div className="mt-3 space-y-2">
              {report.topNegativeReasons.length > 0 ? (
                report.topNegativeReasons.slice(0, 5).map((item) => (
                  <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                    <span className="shrink-0 font-medium">{item.count}</span>
                  </div>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">暂无负反馈原因</div>
              )}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">风险回复样本</div>
            <div className="mt-3 space-y-2">
              {report.recentRiskAnswerSamples.length > 0 ? (
                report.recentRiskAnswerSamples.slice(0, 5).map((item) => (
                  <a
                    key={item.id}
                    href={item.actionHref}
                    className="block rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="min-w-0 truncate font-medium">
                        {item.question || `检索日志 #${item.id}`}
                      </span>
                      <Badge variant="secondary">{item.answerStatusName}</Badge>
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      命中 {item.hitCount} / 分数 {item.topScore}
                    </div>
                  </a>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">暂无风险回复样本</div>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function formatMinutes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return "-"
  if (value < 60) return `${Math.round(value)} 分钟`
  return `${(value / 60).toFixed(1)} 小时`
}

function SalesFunnelPanel({ report }: { report: DashboardSalesFunnelReport }) {
  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">线索转化漏斗</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {report.startDate} 至 {report.endDate}，咨询到留资 {report.leadConversionRate.toFixed(1)}%，留资到成交 {report.closedConversionRate.toFixed(1)}%。
            </div>
          </div>
          <div className="flex flex-wrap gap-1">
            <Badge variant={report.unassignedTotal ? "secondary" : "outline"}>
              未分配 {report.unassignedTotal}
            </Badge>
            <Badge variant={report.overdueFollowUpTotal ? "destructive" : "outline"}>
              逾期 {report.overdueFollowUpTotal}
            </Badge>
          </div>
        </div>
        <div className="grid gap-2 md:grid-cols-3 xl:grid-cols-7">
          {report.steps.map((step) => (
            <a
              key={step.key}
              href={step.actionHref || "/dashboard/sales-leads"}
              className="rounded-md border bg-background px-3 py-2.5 transition-colors hover:bg-muted/50"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="truncate text-sm text-muted-foreground">{step.label}</span>
                {step.dropOffCount > 0 ? (
                  <Badge variant="outline">流失 {step.dropOffCount}</Badge>
                ) : null}
              </div>
              <div className="mt-1 text-2xl font-semibold">{step.count}</div>
              <div className="mt-1 text-xs text-muted-foreground">
                总体 {step.rate.toFixed(1)}% / 环节流失 {step.dropOffRate.toFixed(1)}%
              </div>
            </a>
          ))}
        </div>
        <div className="grid gap-3 lg:grid-cols-[1fr_320px]">
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">顾问效率</div>
            <div className="mt-3 overflow-x-auto">
              <table className="w-full min-w-[860px] text-sm">
                <thead className="text-left text-muted-foreground">
                  <tr className="border-b">
                    <th className="py-2 pr-3 font-medium">顾问</th>
                    <th className="py-2 pr-3 font-medium">线索</th>
                    <th className="py-2 pr-3 font-medium">跟进</th>
                    <th className="py-2 pr-3 font-medium">逾期</th>
                    <th className="py-2 pr-3 font-medium">成交</th>
                    <th className="py-2 pr-3 font-medium">无效</th>
                    <th className="py-2 pr-3 font-medium">无效原因</th>
                    <th className="py-2 pr-3 font-medium">转化率</th>
                    <th className="py-2 font-medium">首跟进</th>
                  </tr>
                </thead>
                <tbody>
                  {report.advisorStats.length > 0 ? (
                    report.advisorStats.map((item) => (
                      <tr key={item.ownerUserId || "unassigned"} className="border-b last:border-0">
                        <td className="py-2 pr-3 font-medium">{item.ownerUserName || "未分配"}</td>
                        <td className="py-2 pr-3">{item.assignedLeadCount}</td>
                        <td className="py-2 pr-3">{item.followUpCount}</td>
                        <td className="py-2 pr-3">
                          <Badge variant={item.overdueFollowUpCount ? "destructive" : "outline"}>
                            {item.overdueFollowUpCount}
                          </Badge>
                        </td>
                        <td className="py-2 pr-3">{item.convertedLeadCount}</td>
                        <td className="py-2 pr-3">{item.invalidLeadCount}</td>
                        <td className="py-2 pr-3">
                          <div className="max-w-[180px] truncate text-muted-foreground">
                            {item.invalidReasons.length > 0
                              ? item.invalidReasons.map((reason) => `${reason.name} ${reason.count}`).join(" / ")
                              : "-"}
                          </div>
                        </td>
                        <td className="py-2 pr-3">{item.conversionRate.toFixed(1)}%</td>
                        <td className="py-2">{formatMinutes(item.averageFirstFollowUpMinutes)}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td className="py-3 text-muted-foreground" colSpan={9}>
                        当前周期暂无线索数据
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">漏斗建议</div>
            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
              {report.suggestions.map((item) => (
                <div key={item}>{item}</div>
              ))}
            </div>
            <div className="mt-4 border-t pt-3">
              <div className="text-sm font-medium">无效原因 Top</div>
              <div className="mt-2 space-y-2">
                {report.invalidReasons.length > 0 ? (
                  report.invalidReasons.map((item) => (
                    <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                      <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                      <Badge variant="outline">{item.count}</Badge>
                    </div>
                  ))
                ) : (
                  <div className="text-sm text-muted-foreground">当前周期暂无无效原因</div>
                )}
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function BusinessTrendPanel({ report }: { report: DashboardBusinessTrendReport }) {
  const latestSeries = report.series.slice(-7)
  const maxConversation = Math.max(1, ...latestSeries.map((item) => item.conversationCount))
  const handleCopyReport = async () => {
    try {
      await navigator.clipboard.writeText(report.reportMarkdown)
      toast.success("经营趋势复盘已复制")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "复制经营趋势复盘失败")
    }
  }
  const metrics = [
    { label: "咨询", value: report.conversationTotal },
    { label: "留资", value: report.leadTotal },
    { label: "留资率", value: `${report.leadConversionRate.toFixed(1)}%` },
    { label: "高意向", value: report.highIntentTotal },
    { label: "预约", value: report.appointmentTotal },
    { label: "到店", value: report.visitedTotal },
    { label: "成交", value: report.convertedTotal },
    { label: "转人工", value: report.handoffTotal },
    { label: "负反馈", value: report.negativeFeedbackTotal },
  ]
  const rankingGroups = [
    { title: "热门产品", items: report.topProducts },
    { title: "来源渠道", items: report.topChannels },
    { title: "高频问题", items: report.topQuestions },
    { title: "未解决问题", items: report.topUnansweredQuestions },
  ]

  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">经营趋势复盘</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {report.startDate} 至 {report.endDate}，按产品、渠道、问题和顾问汇总数字店长经营趋势。
            </div>
          </div>
          <div className="flex flex-wrap gap-1">
            <Button variant="outline" size="sm" onClick={handleCopyReport} disabled={!report.reportMarkdown}>
              <ClipboardIcon className="mr-2 size-4" />
              复制复盘
            </Button>
            <Badge variant={report.pendingFaqDraftCount ? "secondary" : "outline"}>
              FAQ 草稿 {report.pendingFaqDraftCount}
            </Badge>
            <Badge variant={report.negativeFeedbackTotal ? "destructive" : "outline"}>
              负反馈 {report.negativeFeedbackTotal}
            </Badge>
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-9">
          {metrics.map((item) => (
            <div key={item.label} className="rounded-md border bg-background px-3 py-2.5">
              <div className="truncate text-sm text-muted-foreground">{item.label}</div>
              <div className="mt-1 text-2xl font-semibold">{item.value}</div>
            </div>
          ))}
        </div>
        <div className="grid gap-3 xl:grid-cols-[1fr_340px]">
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">最近 7 天趋势</div>
            <div className="mt-3 grid gap-2 md:grid-cols-7">
              {latestSeries.map((item) => {
                const height = Math.max(8, Math.round((item.conversationCount / maxConversation) * 56))
                return (
                  <div key={item.date} className="rounded-md border px-2 py-2 text-xs">
                    <div className="truncate text-muted-foreground">{item.date.slice(5)}</div>
                    <div className="mt-2 flex h-16 items-end gap-1">
                      <div
                        className="w-3 rounded-sm bg-primary"
                        style={{ height }}
                        title={`咨询 ${item.conversationCount}`}
                      />
                      <div
                        className="w-3 rounded-sm bg-emerald-500"
                        style={{ height: Math.max(6, Math.round((item.leadCount / maxConversation) * 56)) }}
                        title={`留资 ${item.leadCount}`}
                      />
                      <div
                        className="w-3 rounded-sm bg-amber-500"
                        style={{ height: Math.max(6, Math.round((item.visitedCount / maxConversation) * 56)) }}
                        title={`到店 ${item.visitedCount}`}
                      />
                      <div
                        className="w-3 rounded-sm bg-rose-500"
                        style={{ height: Math.max(6, Math.round((item.convertedCount / maxConversation) * 56)) }}
                        title={`成交 ${item.convertedCount}`}
                      />
                    </div>
                    <div className="mt-2 space-y-0.5 text-muted-foreground">
                      <div>咨询 {item.conversationCount}</div>
                      <div>留资 {item.leadCount}</div>
                      <div>到店 {item.visitedCount}</div>
                      <div>成交 {item.convertedCount}</div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">经营建议</div>
            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
              {report.suggestions.map((item) => (
                <div key={item}>{item}</div>
              ))}
            </div>
          </div>
        </div>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          {rankingGroups.map((group) => (
            <div key={group.title} className="rounded-md border bg-background p-3">
              <div className="text-sm font-medium">{group.title}</div>
              <div className="mt-3 space-y-2">
                {group.items.length > 0 ? (
                  group.items.slice(0, 5).map((item) => (
                    <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                      <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                      <span className="shrink-0 font-medium">{item.count}</span>
                    </div>
                  ))
                ) : (
                  <div className="text-sm text-muted-foreground">暂无数据</div>
                )}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function abQualityRiskLabel(level: string) {
  switch (level) {
    case "high":
      return "高风险"
    case "medium":
      return "中风险"
    case "low":
      return "低风险"
    case "sample_low":
      return "样本少"
    default:
      return "观察中"
  }
}

function ABTestPanel({ report }: { report: DashboardABTestReport }) {
  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">A/B 话术效果</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {report.startDate} 至 {report.endDate}，按 sourceChannel 对比不同入口、开场白或预约引导版本。
            </div>
          </div>
          <div className="flex flex-wrap gap-1">
            <Badge variant="outline">版本 {report.variantTotal}</Badge>
            <Badge variant="outline">线索 {report.leadTotal}</Badge>
            {report.feedbackTotal > 0 ? (
              <Badge variant={report.negativeFeedbackTotal > 0 ? "secondary" : "outline"}>
                AI负反馈 {report.negativeFeedbackRate.toFixed(1)}%
              </Badge>
            ) : null}
          </div>
        </div>
        <div className="overflow-x-auto rounded-md border bg-background">
          <table className="w-full min-w-[1080px] text-sm">
            <thead className="text-left text-muted-foreground">
              <tr className="border-b">
                <th className="px-3 py-2 font-medium">版本</th>
                <th className="px-3 py-2 font-medium">线索</th>
                <th className="px-3 py-2 font-medium">高意向率</th>
                <th className="px-3 py-2 font-medium">预约率</th>
                <th className="px-3 py-2 font-medium">到店率</th>
                <th className="px-3 py-2 font-medium">成交率</th>
                <th className="px-3 py-2 font-medium">无效率</th>
                <th className="px-3 py-2 font-medium">质量风险</th>
                <th className="px-3 py-2 font-medium">主产品</th>
                <th className="px-3 py-2 font-medium">建议</th>
              </tr>
            </thead>
            <tbody>
              {report.variants.length > 0 ? (
                report.variants.map((item) => (
                  <tr key={item.variantCode} className="border-b last:border-0">
                    <td className="px-3 py-2">
                      <div className="font-medium">{item.variantName || item.variantCode}</div>
                      <div className="text-xs text-muted-foreground">{item.variantCode}</div>
                    </td>
                    <td className="px-3 py-2">{item.leadCount}</td>
                    <td className="px-3 py-2">{item.highIntentRate.toFixed(1)}%</td>
                    <td className="px-3 py-2">{item.appointmentRate.toFixed(1)}%</td>
                    <td className="px-3 py-2">{item.visitRate.toFixed(1)}%</td>
                    <td className="px-3 py-2">{item.conversionRate.toFixed(1)}%</td>
                    <td className="px-3 py-2">{item.invalidRate.toFixed(1)}%</td>
                    <td className="px-3 py-2">
                      <div className="space-y-1">
                        <Badge
                          variant={
                            item.qualityRiskLevel === "high"
                              ? "destructive"
                              : item.qualityRiskLevel === "medium"
                                ? "secondary"
                                : "outline"
                          }
                        >
                          {abQualityRiskLabel(item.qualityRiskLevel)}
                        </Badge>
                        <div className="max-w-[220px] text-xs text-muted-foreground">
                          {item.qualityRiskReason || "暂无明显风险"}
                        </div>
                      </div>
                    </td>
                    <td className="px-3 py-2">{item.topProduct || "-"}</td>
                    <td className="px-3 py-2 text-muted-foreground">
                      {item.recommendedAction}
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td className="px-3 py-4 text-muted-foreground" colSpan={10}>
                    当前周期暂无可对比线索。
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
          {report.suggestions.map((item) => (
            <div key={item} className="rounded-md border bg-background px-3 py-2">
              {item}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function ChannelSourcePanel({ report }: { report: DashboardABTestReport }) {
  const topChannels = report.variants.slice(0, 6)
  const bestChannel = topChannels[0]

  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">渠道来源统计</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {report.startDate} 至 {report.endDate}，按 sourceChannel 统计不同入口带来的线索质量。
            </div>
          </div>
          <div className="flex flex-wrap gap-1">
            <Badge variant="outline">来源 {report.variantTotal}</Badge>
            <Badge variant="outline">线索 {report.leadTotal}</Badge>
          </div>
        </div>

        {bestChannel ? (
          <div className="rounded-md border bg-background p-3">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <div className="text-sm font-medium">当前主力来源：{bestChannel.variantName}</div>
                <div className="mt-1 text-xs text-muted-foreground">
                  主产品 {bestChannel.topProduct || "-"} / 留资 {bestChannel.leadCount}
                </div>
              </div>
              <div className="grid grid-cols-4 gap-2 text-center text-xs sm:min-w-[360px]">
                <div className="rounded-md bg-muted/50 px-2 py-1">
                  <div className="font-medium text-foreground">{bestChannel.highIntentRate.toFixed(1)}%</div>
                  <div className="text-muted-foreground">高意向</div>
                </div>
                <div className="rounded-md bg-muted/50 px-2 py-1">
                  <div className="font-medium text-foreground">{bestChannel.appointmentRate.toFixed(1)}%</div>
                  <div className="text-muted-foreground">预约</div>
                </div>
                <div className="rounded-md bg-muted/50 px-2 py-1">
                  <div className="font-medium text-foreground">{bestChannel.visitRate.toFixed(1)}%</div>
                  <div className="text-muted-foreground">到店</div>
                </div>
                <div className="rounded-md bg-muted/50 px-2 py-1">
                  <div className="font-medium text-foreground">{bestChannel.conversionRate.toFixed(1)}%</div>
                  <div className="text-muted-foreground">成交</div>
                </div>
              </div>
            </div>
          </div>
        ) : null}

        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {topChannels.length > 0 ? (
            topChannels.map((item) => {
              const share = report.leadTotal > 0 ? (item.leadCount / report.leadTotal) * 100 : 0
              return (
                <div key={item.variantCode} className="rounded-md border bg-background p-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">{item.variantName}</div>
                      <div className="mt-1 truncate text-xs text-muted-foreground">{item.variantCode}</div>
                    </div>
                    <Badge variant={item.invalidRate >= 40 ? "destructive" : "secondary"}>
                      {share.toFixed(1)}%
                    </Badge>
                  </div>
                  <div className="mt-3 h-2 overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{ width: `${Math.min(100, Math.max(4, share))}%` }}
                    />
                  </div>
                  <div className="mt-3 grid grid-cols-5 gap-2 text-center text-xs">
                    <div>
                      <div className="font-medium">{item.leadCount}</div>
                      <div className="text-muted-foreground">线索</div>
                    </div>
                    <div>
                      <div className="font-medium">{item.highIntentCount}</div>
                      <div className="text-muted-foreground">高意向</div>
                    </div>
                    <div>
                      <div className="font-medium">{item.appointmentCount}</div>
                      <div className="text-muted-foreground">预约</div>
                    </div>
                    <div>
                      <div className="font-medium">{item.visitedCount}</div>
                      <div className="text-muted-foreground">到店</div>
                    </div>
                    <div>
                      <div className="font-medium">{item.convertedCount}</div>
                      <div className="text-muted-foreground">成交</div>
                    </div>
                  </div>
                  {item.invalidCount > 0 ? (
                    <div className="mt-2 text-xs text-muted-foreground">
                      无效 {item.invalidCount}，无效率 {item.invalidRate.toFixed(1)}%
                    </div>
                  ) : null}
                </div>
              )
            })
          ) : (
            <div className="rounded-md border bg-background p-3 text-sm text-muted-foreground">
              暂无来源数据。建议官网、企微、广告落地页传入不同 sourceChannel。
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function DailyBusinessReportPanel({ report }: { report: DashboardDailyBusinessReport }) {
  const t = useI18n()
  const [sending, setSending] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(buildDailyReportText(report))
      toast.success(t("dashboardHome.dailyReportCopied"))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("dashboardHome.dailyReportCopyFailed"))
    }
  }

  const handleSend = async () => {
    if (sending) return
    setSending(true)
    try {
      const result = await sendDailyBusinessReport(report.reportDate)
      if (result.sent) {
        toast.success(result.message || "日报已发送")
      } else {
        toast.info(result.message || "日报未发送")
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "发送日报失败")
    } finally {
      setSending(false)
    }
  }

  return (
    <Card className="rounded-md shadow-none">
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="text-base font-semibold">{t("dashboardHome.dailyReportTitle")}</div>
            <div className="mt-1 text-sm text-muted-foreground">{report.summary}</div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" onClick={handleCopy}>
              <ClipboardIcon className="mr-2 size-4" />
              {t("dashboardHome.dailyReportCopy")}
            </Button>
            <Button variant="outline" size="sm" onClick={handleSend} disabled={sending}>
              <SendIcon className="mr-2 size-4" />
              {sending ? "发送中" : "发送日报"}
            </Button>
          </div>
        </div>
        <div className="grid gap-3 lg:grid-cols-3">
          <div className="rounded-md border bg-background p-3 lg:col-span-3">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="text-sm font-medium">{t("dashboardHome.dailyReportPriorityFollowUps")}</div>
              <div className="flex flex-wrap items-center gap-1 text-xs">
                <Badge variant={report.overdueFollowUpCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportOverdue")} {report.overdueFollowUpCount}
                </Badge>
                <Badge variant={report.todayFollowUpCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportDueToday")} {report.todayFollowUpCount}
                </Badge>
                <Badge variant={report.unscheduledHotLeads ? "secondary" : "outline"}>
                  {t("dashboardHome.dailyReportUnscheduledHot")} {report.unscheduledHotLeads}
                </Badge>
                <Badge variant={report.unassignedPriorityLeadCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportUnassignedPriorityLeads")} {report.unassignedPriorityLeadCount}
                </Badge>
                <Badge variant={report.overdueAppointmentCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportOverdueAppointments")} {report.overdueAppointmentCount}
                </Badge>
                <Badge variant={report.todayAppointmentCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportTodayAppointments")} {report.todayAppointmentCount}
                </Badge>
                <Badge variant={report.unscheduledAppointmentCount ? "secondary" : "outline"}>
                  {t("dashboardHome.dailyReportUnscheduledAppointments")} {report.unscheduledAppointmentCount}
                </Badge>
                <Badge variant={report.pendingAfterSalesTicketCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportPendingAfterSalesTickets")} {report.pendingAfterSalesTicketCount}
                </Badge>
                <Badge variant={report.todayAfterSalesTicketCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportTodayAfterSalesTickets")} {report.todayAfterSalesTicketCount}
                </Badge>
                <Badge variant={report.todayHandledAfterSalesTicketCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportHandledAfterSalesTickets")} {report.todayHandledAfterSalesTicketCount}
                </Badge>
                <Badge variant={report.aiFeedbackNegativeCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportNegativeAIFeedback")} {report.aiFeedbackNegativeCount}
                </Badge>
                <Badge variant={report.pendingFaqDraftCount ? "secondary" : "outline"}>
                  {t("dashboardHome.dailyReportPendingFAQDrafts")} {report.pendingFaqDraftCount}
                </Badge>
              </div>
            </div>
            <div className="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {report.priorityFollowUps.length > 0 ? (
                report.priorityFollowUps.slice(0, 6).map((lead) => (
                  <a
                    key={lead.id}
                    href={`/dashboard/sales-leads?leadId=${lead.id}`}
                    className="min-w-0 rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="truncate font-medium">
                        {lead.customerName || t("dashboardHome.dailyReportUnknownCustomer")}
                      </div>
                      <Badge variant={followUpStateVariant(lead.followUpState)}>
                        {followUpStateText(lead.followUpState)}
                      </Badge>
                    </div>
                    <div className="mt-1 truncate text-muted-foreground">
                      {lead.phone || lead.wechat || t("dashboardHome.dailyReportNoContact")}
                    </div>
                    <div className="mt-1 truncate text-muted-foreground">
                      {lead.nextFollowUpAt || t("dashboardHome.dailyReportNoNextFollowUp")}
                    </div>
                    <div className="mt-1 truncate text-muted-foreground">
                      {lead.ownerUserName || t("dashboardHome.dailyReportUnassigned")}
                    </div>
                  </a>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">
                  {t("dashboardHome.dailyReportNoPriorityFollowUps")}
                </div>
              )}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3 lg:col-span-3">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="text-sm font-medium">{t("dashboardHome.dailyReportAfterSalesTickets")}</div>
              <div className="flex flex-wrap items-center gap-1 text-xs">
                <Badge variant={report.pendingAfterSalesTicketCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportPendingAfterSalesTickets")} {report.pendingAfterSalesTicketCount}
                </Badge>
                <Badge variant={report.todayAfterSalesTicketCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportTodayAfterSalesTickets")} {report.todayAfterSalesTicketCount}
                </Badge>
                <Badge variant={report.todayHandledAfterSalesTicketCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportHandledAfterSalesTickets")} {report.todayHandledAfterSalesTicketCount}
                </Badge>
              </div>
            </div>
            <div className="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {report.afterSalesTickets.length > 0 ? (
                report.afterSalesTickets.slice(0, 6).map((ticket) => (
                  <a
                    key={ticket.id}
                    href={`/dashboard/tickets?ticketId=${ticket.id}`}
                    className="min-w-0 rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="truncate font-medium">
                        {ticket.ticketNo || `#${ticket.id}`}
                      </div>
                      <Badge variant={ticketStatusVariant(ticket.status)}>
                        {ticketStatusText(ticket.status)}
                      </Badge>
                    </div>
                    <div className="mt-1 truncate text-muted-foreground">{ticket.title}</div>
                    <div className="mt-1 truncate text-muted-foreground">
                      {ticket.currentAssigneeName || t("dashboardHome.dailyReportUnassigned")}
                    </div>
                    {ticket.latestProgress ? (
                      <div className="mt-1 line-clamp-2 text-muted-foreground">
                        {ticket.latestProgress}
                      </div>
                    ) : null}
                    <div className="mt-1 truncate text-muted-foreground">
                      {ticket.handledAt || ticket.latestProgressAt || ticket.updatedAt}
                    </div>
                  </a>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">
                  {t("dashboardHome.dailyReportNoAfterSalesTickets")}
                </div>
              )}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3 lg:col-span-3">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="text-sm font-medium">{t("dashboardHome.dailyReportAIFeedback")}</div>
              <div className="flex flex-wrap items-center gap-1 text-xs">
                <Badge variant={report.aiFeedbackCount ? "outline" : "secondary"}>
                  {t("dashboardHome.dailyReportTotalAIFeedback")} {report.aiFeedbackCount}
                </Badge>
                <Badge variant={report.aiFeedbackLikeCount ? "default" : "outline"}>
                  {t("dashboardHome.dailyReportPositiveAIFeedback")} {report.aiFeedbackLikeCount}
                </Badge>
                <Badge variant={report.aiFeedbackNegativeCount ? "destructive" : "outline"}>
                  {t("dashboardHome.dailyReportNegativeAIFeedback")} {report.aiFeedbackNegativeCount}
                </Badge>
                <Badge variant={report.aiFeedbackNegativeCount ? "destructive" : "outline"}>
                  {report.aiFeedbackNegativeRate.toFixed(1)}%
                </Badge>
              </div>
            </div>
            <div className="mt-3 space-y-2">
              {report.topAiFeedbackReasons.length > 0 ? (
                report.topAiFeedbackReasons.map((item) => (
                  <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                    <span className="shrink-0 font-medium">{item.count}</span>
                  </div>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">
                  {t("dashboardHome.dailyReportNoAIFeedbackReasons")}
                </div>
              )}
            </div>
            <div className="mt-3 space-y-2">
              {report.recentNegativeAiFeedbacks.length > 0 ? (
                report.recentNegativeAiFeedbacks.map((item) => (
                  <a
                    key={item.id}
                    href={knowledgeRetrieveLogHref(item.retrieveLogId, item.knowledgeBaseId)}
                    className="block rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="min-w-0 truncate font-medium">
                        {item.question || `#${item.retrieveLogId}`}
                      </div>
                      <Badge variant="destructive">{item.feedbackTypeName}</Badge>
                    </div>
                    <div className="mt-1 truncate text-muted-foreground">
                      {item.feedbackReason || t("dashboardHome.dailyReportNoAIFeedbackReason")}
                    </div>
                    <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span>{item.createdAt}</span>
                      <span>{item.answerStatusName || "-"}</span>
                      <span>{item.modelName || "-"}</span>
                    </div>
                  </a>
                ))
              ) : null}
            </div>
            <div className="mt-3 space-y-2">
              {report.pendingFaqDrafts.length > 0 ? (
                report.pendingFaqDrafts.map((item) => (
                  <a
                    key={item.id}
                    href={knowledgeFaqHref(item.id, item.knowledgeBaseId)}
                    className="block rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="min-w-0 truncate font-medium">
                        {item.question || `FAQ #${item.id}`}
                      </div>
                      <Badge variant="secondary">{t("dashboardHome.dailyReportFAQDraft")}</Badge>
                    </div>
                    <div className="mt-1 truncate text-muted-foreground">
                      {item.answer || t("dashboardHome.dailyReportFAQDraftNoAnswer")}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">{item.createdAt}</div>
                  </a>
                ))
              ) : null}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.dailyReportHighlights")}</div>
            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
              {report.highlights.map((item) => (
                <div key={item}>{item}</div>
              ))}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.dailyReportFollowUps")}</div>
            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
              {report.followUpSuggestions.map((item) => (
                <div key={item}>{item}</div>
              ))}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.dailyReportKnowledge")}</div>
            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
              {report.knowledgeSuggestions.map((item) => (
                <div key={item}>{item}</div>
              ))}
            </div>
          </div>
        </div>
        <div className="grid gap-3 lg:grid-cols-2">
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.dailyReportTopQuestions")}</div>
            <div className="mt-3 space-y-2">
              {report.topQuestions.length > 0 ? (
                report.topQuestions.map((item) => (
                  <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                    <span className="shrink-0 font-medium">{item.count}</span>
                  </div>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">{t("dashboardHome.dailyReportNoQuestions")}</div>
              )}
            </div>
          </div>
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.dailyReportUnansweredQuestions")}</div>
            <div className="mt-3 space-y-2">
              {report.unansweredQuestions.length > 0 ? (
                report.unansweredQuestions.map((item) => (
                  <div key={item.name} className="flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-muted-foreground">{item.name}</span>
                    <span className="shrink-0 font-medium">{item.count}</span>
                  </div>
                ))
              ) : (
                <div className="text-sm text-muted-foreground">{t("dashboardHome.dailyReportNoUnanswered")}</div>
              )}
            </div>
          </div>
        </div>
        {report.highIntentLeads.length > 0 ? (
          <div className="rounded-md border bg-background p-3">
            <div className="text-sm font-medium">{t("dashboardHome.dailyReportHighIntentLeads")}</div>
            <div className="mt-3 grid gap-2 md:grid-cols-2">
              {report.highIntentLeads.slice(0, 4).map((lead) => (
                <div key={lead.id} className="min-w-0 rounded-md border px-3 py-2 text-sm">
                  <div className="truncate font-medium">
                    {lead.customerName || t("dashboardHome.dailyReportUnknownCustomer")}
                  </div>
                  <div className="mt-1 truncate text-muted-foreground">
                    {lead.phone || lead.wechat || t("dashboardHome.dailyReportNoContact")}
                  </div>
                  <div className="mt-1 truncate text-muted-foreground">
                    {lead.interestedProducts || t("dashboardHome.dailyReportNoProduct")}
                  </div>
                  <div className="mt-1 truncate text-muted-foreground">
                    {[lead.appointmentAt, lead.appointmentTimeText, lead.appointmentStore].filter(Boolean).join(" / ") || "-"}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

export function DashboardHome() {
  const t = useI18n()
  const [range, setRange] = useState<DashboardRange>("7d")
  const [data, setData] = useState<DashboardOverview | null>(null)
  const [dailyReport, setDailyReport] = useState<DashboardDailyBusinessReport | null>(null)
  const [aiQualityReport, setAIQualityReport] = useState<DashboardAIQualityReport | null>(null)
  const [salesFunnelReport, setSalesFunnelReport] = useState<DashboardSalesFunnelReport | null>(null)
  const [businessTrendReport, setBusinessTrendReport] = useState<DashboardBusinessTrendReport | null>(null)
  const [abTestReport, setABTestReport] = useState<DashboardABTestReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const loadData = useCallback(
    async (nextRange: DashboardRange, showRefreshing = false) => {
      if (showRefreshing) {
        setRefreshing(true)
      } else {
        setLoading(true)
      }
      try {
        const [overviewResult, reportResult, aiQualityResult, salesFunnelResult, businessTrendResult, abTestResult] =
          await Promise.allSettled([
            fetchDashboardOverview(nextRange),
            fetchDailyBusinessReport(),
            fetchAIQualityReport(nextRange),
            fetchSalesFunnelReport(nextRange),
            fetchBusinessTrendReport(nextRange),
            fetchABTestReport(nextRange),
          ])
        if (overviewResult.status === "fulfilled") {
          setData(overviewResult.value)
        } else {
          setData(null)
          toast.error(overviewResult.reason instanceof Error ? overviewResult.reason.message : t("dashboardHome.loadFailed"))
        }
        setDailyReport(reportResult.status === "fulfilled" ? reportResult.value : null)
        setAIQualityReport(aiQualityResult.status === "fulfilled" ? aiQualityResult.value : null)
        setSalesFunnelReport(salesFunnelResult.status === "fulfilled" ? salesFunnelResult.value : null)
        setBusinessTrendReport(businessTrendResult.status === "fulfilled" ? businessTrendResult.value : null)
        setABTestReport(abTestResult.status === "fulfilled" ? abTestResult.value : null)
        const optionalResults = [reportResult, aiQualityResult, salesFunnelResult, businessTrendResult, abTestResult]
        optionalResults.forEach((result) => {
          if (result.status === "rejected") {
            console.warn("[dashboard] optional report failed", result.reason)
          }
        })
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t("dashboardHome.loadFailed"))
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [t]
  )

  useEffect(() => {
    void loadData(range)
  }, [loadData, range])

  const rangeOptions: Array<{ value: DashboardRange; label: string }> = [
    { value: "today", label: t("dashboardHome.rangeToday") },
    { value: "7d", label: t("dashboardHome.range7d") },
    { value: "30d", label: t("dashboardHome.range30d") },
  ]

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 lg:p-5">
      <div className="flex justify-end">
        <div className="flex flex-wrap items-center gap-2">
          <div className="rounded-md border bg-background p-1">
            {rangeOptions.map((item) => (
              <Button
                key={item.value}
                variant={range === item.value ? "secondary" : "ghost"}
                size="sm"
                onClick={() => setRange(item.value)}
              >
                {item.label}
              </Button>
            ))}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void loadData(range, true)}
            disabled={loading || refreshing}
          >
            <RefreshCwIcon className={refreshing ? "mr-2 size-4 animate-spin" : "mr-2 size-4"} />
            {t("dashboardHome.refresh")}
          </Button>
        </div>
      </div>

      {loading && !data ? (
        <LoadingCards />
      ) : data ? (
        <>
          <SummaryCards summary={data.summary} />

          <DigitalStorePanel stats={data.digitalStoreStats} />

          {salesFunnelReport ? <SalesFunnelPanel report={salesFunnelReport} /> : null}

          {businessTrendReport ? <BusinessTrendPanel report={businessTrendReport} /> : null}

          {abTestReport ? <ChannelSourcePanel report={abTestReport} /> : null}

          {abTestReport ? <ABTestPanel report={abTestReport} /> : null}

          {aiQualityReport ? <AIQualityPanel report={aiQualityReport} /> : null}

          {dailyReport ? <DailyBusinessReportPanel report={dailyReport} /> : null}

          <TrendPanel
            title={t("dashboardHome.conversationTrend")}
            description={t("dashboardHome.conversationTrendDescription")}
            trend={data.conversationStats.trend}
            distribution={data.conversationStats.statusDistribution}
          />

          <div className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
            <TeamLoadPanel agentStats={data.agentStats} />

            <Card className="rounded-md shadow-none">
              <CardContent className="grid gap-3 p-4 sm:grid-cols-2">
                <div className="rounded-md border bg-background px-3 py-2.5">
                  <div className="text-sm text-muted-foreground">{t("dashboardHome.enabledAiAgents")}</div>
                  <div className="mt-1 text-2xl font-semibold">{data.aiStats.enabledAiAgents}</div>
                </div>
                <div className="rounded-md border bg-background px-3 py-2.5">
                  <div className="text-sm text-muted-foreground">{t("dashboardHome.enabledChannels")}</div>
                  <div className="mt-1 text-2xl font-semibold">{data.aiStats.enabledChannels}</div>
                </div>
                <div className="rounded-md border bg-background px-3 py-2.5">
                  <div className="text-sm text-muted-foreground">{t("dashboardHome.todayKnowledgeRetrieves")}</div>
                  <div className="mt-1 text-2xl font-semibold">
                    {data.aiStats.todayKnowledgeRetrieves}
                  </div>
                </div>
                <div className="rounded-md border bg-background px-3 py-2.5">
                  <div className="text-sm text-muted-foreground">{t("dashboardHome.todayKnowledgeRetrieveFailRate")}</div>
                  <div className="mt-1 text-2xl font-semibold">
                    {data.aiStats.todayKnowledgeRetrieveFailRate.toFixed(1)}%
                  </div>
                </div>
                <div className="rounded-md border bg-background px-3 py-2.5">
                  <div className="text-sm text-muted-foreground">{t("dashboardHome.todaySkillRunFailCount")}</div>
                  <div className="mt-1 text-2xl font-semibold">
                    {data.aiStats.todaySkillRunFailCount}
                  </div>
                </div>
                <div className="rounded-md border bg-background px-3 py-2.5">
                  <div className="text-sm text-muted-foreground">{t("dashboardHome.todayAiHandoffCount")}</div>
                  <div className="mt-1 text-2xl font-semibold">
                    {data.aiStats.todayAiHandoffCount}
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <AlertList alerts={data.alerts} />
        </>
      ) : (
        <Card className="rounded-md shadow-none">
          <CardContent className="flex min-h-60 items-center justify-center p-6 text-sm text-muted-foreground">
            {t("dashboardHome.empty")}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
