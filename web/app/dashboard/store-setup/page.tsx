"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import Link from "next/link"
import {
  BadgeCheckIcon,
  BadgePercentIcon,
  BotMessageSquareIcon,
  BoxesIcon,
  CopyIcon,
  DatabaseZapIcon,
  DownloadIcon,
  ExternalLinkIcon,
  FileTextIcon,
  MessageSquareMoreIcon,
  PackageIcon,
  PrinterIcon,
  RefreshCwIcon,
  RouterIcon,
  ShieldCheckIcon,
  SparklesIcon,
  Trash2Icon,
  UploadIcon,
  UserRoundCheckIcon,
} from "lucide-react"
import { toast } from "sonner"

import { DashboardPage, DashboardToolbar } from "@/components/dashboard-page"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  applyImportedDigitalStoreTemplate,
  applyDigitalStoreTemplate,
  cleanupDigitalStoreDemoData,
  createDigitalStoreDeliveryRecord,
  ensureDigitalStoreRuntime,
  exportDigitalStoreTemplate,
  fetchDigitalStoreDeliveryReport,
  fetchDigitalStoreKnowledgeAssistant,
  fetchDigitalStoreMaintenanceStatus,
  fetchDigitalStoreProfile,
  fetchDigitalStoreSetupStatus,
  fetchDigitalStoreTemplateEffect,
  fetchDigitalStoreTemplates,
  previewDigitalStoreTemplate,
  previewImportedDigitalStoreTemplate,
  syncDigitalStoreKnowledge,
  testDigitalStoreWebhookNotifyScenarios,
  type DigitalStoreDeliveryRecord,
  type DigitalStoreProfile,
  type DigitalStoreDeliveryReport,
  type DigitalStoreKnowledgeAssistant,
  type DigitalStoreMaintenanceStatus,
  type DigitalStoreSetupStatus,
  type DigitalStoreTemplate,
  type DigitalStoreTemplateEffect,
  type DigitalStoreTemplateImportPayload,
  type DigitalStoreTemplatePreview,
  type DigitalStoreWebhookTestResult,
} from "@/lib/api/digital-store"
import { cn } from "@/lib/utils"

type SetupStep = {
  key: string
  title: string
  description: string
  done: boolean
  href: string
  icon: React.ReactNode
}

function securityCheckBadgeVariant(status: string) {
  if (status === "blocking") return "destructive" as const
  if (status === "ok") return "default" as const
  return "secondary" as const
}

function securityCheckStatusText(status: string) {
  if (status === "blocking") return "阻断"
  if (status === "ok") return "通过"
  if (status === "warning") return "提醒"
  return status || "-"
}

function jsStringLiteral(value: string) {
  return JSON.stringify(value.trim())
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return "-"
  if (value < 1024) return `${value} B`
  const kb = value / 1024
  if (kb < 1024) return `${kb.toFixed(1)} KB`
  const mb = kb / 1024
  if (mb < 1024) return `${mb.toFixed(1)} MB`
  return `${(mb / 1024).toFixed(1)} GB`
}

function downloadJsonFile(filename: string, value: unknown) {
  const blob = new Blob([JSON.stringify(value, null, 2)], {
    type: "application/json;charset=utf-8",
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function printDeliveryReport() {
  window.print()
}

function formatTemplatePreviewConfirm(preview: DigitalStoreTemplatePreview) {
  const productNames = preview.products.slice(0, 4).map((item) => {
    return `${item.action === "update" ? "更新" : "新建"}${item.name}`
  })
  const promotionNames = preview.promotions.slice(0, 4).map((item) => {
    return `${item.action === "update" ? "更新" : "新建"}${item.name}`
  })
  const warnings = preview.warnings.map((item) => item.message)
  return [
    `将应用行业样板：${preview.template.name || preview.template.code}${preview.template.version ? ` v${preview.template.version}` : ""}`,
    `店长资料：${preview.profileAction === "update" ? "更新当前配置" : "创建新配置"}`,
    `产品：新建 ${preview.productCreateTotal}，更新 ${preview.productUpdateTotal}`,
    productNames.length ? `产品明细：${productNames.join("、")}` : "",
    `活动：新建 ${preview.promotionCreateTotal}，更新 ${preview.promotionUpdateTotal}`,
    promotionNames.length ? `活动明细：${promotionNames.join("、")}` : "",
    `行业风险规则：${preview.riskRules.length} 组`,
    `验收场景：${preview.acceptanceItems.length} 项`,
    warnings.length ? `提示：${warnings.join("；")}` : "",
    "",
    "确认应用后会同步相关 FAQ 知识索引。是否继续？",
  ]
    .filter(Boolean)
    .join("\n")
}

export default function DashboardStoreSetupPage() {
  const [profile, setProfile] = useState<DigitalStoreProfile | null>(null)
  const [deliveryReport, setDeliveryReport] = useState<DigitalStoreDeliveryReport | null>(null)
  const [latestDeliveryRecord, setLatestDeliveryRecord] = useState<DigitalStoreDeliveryRecord | null>(null)
  const [setupStatus, setSetupStatus] = useState<DigitalStoreSetupStatus | null>(null)
  const [maintenanceStatus, setMaintenanceStatus] = useState<DigitalStoreMaintenanceStatus | null>(null)
  const [knowledgeAssistant, setKnowledgeAssistant] = useState<DigitalStoreKnowledgeAssistant | null>(null)
  const [templateEffect, setTemplateEffect] = useState<DigitalStoreTemplateEffect | null>(null)
  const [templates, setTemplates] = useState<DigitalStoreTemplate[]>([])
  const [latestWebhookTest, setLatestWebhookTest] = useState<DigitalStoreWebhookTestResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [runningAction, setRunningAction] = useState<string | null>(null)
  const [origin, setOrigin] = useState("")
  const templateImportInputRef = useRef<HTMLInputElement | null>(null)

  const loadState = useCallback(async () => {
    setLoading(true)
    try {
      const [nextProfile, nextSetupStatus, nextTemplates] = await Promise.all([
        fetchDigitalStoreProfile(),
        fetchDigitalStoreSetupStatus(),
        fetchDigitalStoreTemplates(),
      ])
      setProfile(nextProfile)
      setSetupStatus(nextSetupStatus)
      setTemplates(nextTemplates)
      const [nextMaintenanceStatus, nextKnowledgeAssistant, nextTemplateEffect] = await Promise.allSettled([
        fetchDigitalStoreMaintenanceStatus(),
        fetchDigitalStoreKnowledgeAssistant(),
        fetchDigitalStoreTemplateEffect(),
      ])
      setMaintenanceStatus(nextMaintenanceStatus.status === "fulfilled" ? nextMaintenanceStatus.value : null)
      setKnowledgeAssistant(nextKnowledgeAssistant.status === "fulfilled" ? nextKnowledgeAssistant.value : null)
      setTemplateEffect(nextTemplateEffect.status === "fulfilled" ? nextTemplateEffect.value : null)
      const optionalResults = [nextMaintenanceStatus, nextKnowledgeAssistant, nextTemplateEffect]
      optionalResults.forEach((result) => {
        if (result.status === "rejected") {
          console.warn("[store-setup] optional setup panel failed", result.reason)
        }
      })
      if (origin) {
        try {
          const nextReport = await fetchDigitalStoreDeliveryReport(origin)
          setDeliveryReport(nextReport)
          setLatestDeliveryRecord(nextReport.latestRecord || null)
        } catch (error) {
          setDeliveryReport(null)
          setLatestDeliveryRecord(null)
          console.warn("[store-setup] delivery report failed", error)
        }
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载初始化状态失败")
    } finally {
      setLoading(false)
    }
  }, [origin])

  useEffect(() => {
    void loadState()
  }, [loadState])

  useEffect(() => {
    setOrigin(window.location.origin)
  }, [])

  async function runAction(
    key: string,
    action: () => Promise<unknown>,
    successText: string
  ) {
    if (runningAction) return
    setRunningAction(key)
    try {
      const nextSuccessText = await action()
      toast.success(typeof nextSuccessText === "string" ? nextSuccessText : successText)
      await loadState()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "操作失败")
    } finally {
      setRunningAction(null)
    }
  }

  async function handleApplyTemplate(template: DigitalStoreTemplate) {
    if (runningAction) return
    setRunningAction(`template-${template.code}`)
    try {
      const preview = await previewDigitalStoreTemplate(template.code)
      if (!window.confirm(formatTemplatePreviewConfirm(preview))) {
        return
      }
      await applyDigitalStoreTemplate(template.code)
      toast.success(`${template.name}样板已初始化`)
      await loadState()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "应用行业样板失败")
    } finally {
      setRunningAction(null)
    }
  }

  async function handleExportTemplate(template: DigitalStoreTemplate) {
    await runAction(
      `export-template-${template.code}`,
      async () => {
        const result = await exportDigitalStoreTemplate(template.code)
        downloadJsonFile(`digital-store-template-${template.code}.json`, result)
      },
      "已导出行业模板 JSON"
    )
  }

  async function handleCopyTemplateImprovementPack() {
    const content = templateEffect?.improvementMarkdown?.trim()
    if (!content) {
      toast.info("暂无可复制的模板改进包")
      return
    }
    try {
      await navigator.clipboard.writeText(content)
      toast.success("模板改进包已复制")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "复制模板改进包失败")
    }
  }

  async function handleImportTemplateFile(file: File | undefined) {
    if (!file || runningAction) return
    setRunningAction("import-template-json")
    try {
      const text = await file.text()
      const payload = JSON.parse(text) as DigitalStoreTemplateImportPayload
      const preview = await previewImportedDigitalStoreTemplate(payload)
      if (!window.confirm(formatTemplatePreviewConfirm(preview))) {
        return
      }
      await applyImportedDigitalStoreTemplate(payload)
      toast.success(`${preview.template.name || preview.template.code} 模板已导入并应用`)
      await loadState()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "导入行业模板失败")
    } finally {
      setRunningAction(null)
      if (templateImportInputRef.current) {
        templateImportInputRef.current.value = ""
      }
    }
  }

  async function handleEnsureRuntime() {
    await runAction(
      "ensure-runtime",
      async () => {
        await ensureDigitalStoreRuntime()
      },
      "已生成数字店长 Agent 和 Web 聊天渠道"
    )
  }

  async function handleCreateDeliveryRecord() {
    await runAction(
      "delivery-record",
      async () => {
        const record = await createDigitalStoreDeliveryRecord({
          publicBaseUrl: origin,
          acceptanceStatus: deliveryReport?.ready ? "passed" : "pending",
          acceptanceSummary: deliveryReport?.ready
            ? "交付配置完整，自动化或人工验收通过。"
            : "交付配置仍有缺口，等待补齐后复验。",
        })
        setLatestDeliveryRecord(record)
      },
      "已保存交付记录"
    )
  }

  async function handleTestWebhookNotify() {
    if (runningAction) return
    setRunningAction("webhook-test")
    try {
      const result = await testDigitalStoreWebhookNotifyScenarios()
      setLatestWebhookTest(result)
      if (!result.enabled) {
        toast.info(result.message || "外部通知尚未启用")
      } else if (result.sent) {
        toast.success(result.message || `已发送 ${result.sentTotal || 0} 类关键事件测试`)
      } else {
        toast.error(result.message || "关键通知测试未全部发送")
      }
      await loadState()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "关键通知测试失败")
    } finally {
      setRunningAction(null)
    }
  }

  async function handleCleanupDemoData() {
    const firstConfirmed = window.confirm(
      "将清理测试会话、消息、销售线索、跟进、工单、通知、检索反馈和运行日志。产品、活动、知识库、模型、Agent、渠道、客户档案和交付记录会保留。是否继续？"
    )
    if (!firstConfirmed) return
    const secondConfirmed = window.confirm(
      "请再次确认：该操作会删除当前部署里的演示运营流水，适合正式交付前清场。删除后不可在后台恢复。"
    )
    if (!secondConfirmed) return

    await runAction(
      "cleanup-demo-data",
      async () => {
        const result = await cleanupDigitalStoreDemoData()
        const total = Object.values(result.deleted || {}).reduce((sum, count) => sum + count, 0)
        return result.message || `已清理 ${total} 条演示数据`
      },
      "演示数据清理完成"
    )
  }

  const setupSteps = useMemo<SetupStep[]>(() => {
    const profileDone = Boolean(profile?.initialized && profile.brandName && profile.aiManagerName)
    const knowledgeDone = Boolean(setupStatus?.knowledgeFAQId || profile?.knowledgeFAQId)
    const productTotal = Number(setupStatus?.productTotal || 0)
    const promotionTotal = Number(setupStatus?.promotionTotal || 0)
    const productUnsynced = Number(setupStatus?.productKnowledgeUnsyncedTotal || 0)
    const productFailed = Number(setupStatus?.productKnowledgeFailedTotal || 0)
    const promotionUnsynced = Number(setupStatus?.promotionKnowledgeUnsyncedTotal || 0)
    const promotionFailed = Number(setupStatus?.promotionKnowledgeFailedTotal || 0)
    const productKnowledgeReady = productTotal > 0 && productUnsynced === 0 && productFailed === 0
    const promotionKnowledgeReady = promotionTotal > 0 && promotionUnsynced === 0 && promotionFailed === 0
    return [
      {
        key: "profile",
        title: "品牌与店长",
        description: profileDone
          ? `${profile?.brandName || "-"} / ${profile?.aiManagerName || "-"}`
          : "品牌、门店、人设、预约和转人工规则",
        done: profileDone,
        href: "/dashboard/digital-store",
        icon: <BotMessageSquareIcon />,
      },
      {
        key: "products",
        title: "产品库",
        description:
          productTotal > 0
            ? productKnowledgeReady
              ? `${productTotal} 个产品，FAQ 已同步 ${setupStatus?.productKnowledgeSyncedTotal || 0}/${productTotal}`
              : `${productTotal} 个产品，未同步 ${productUnsynced}，索引失败 ${productFailed}`
            : "产品卖点、适用人群、价格区间",
        done: productKnowledgeReady,
        href: "/dashboard/products",
        icon: <PackageIcon />,
      },
      {
        key: "promotions",
        title: "活动库",
        description:
          promotionTotal > 0
            ? promotionKnowledgeReady
              ? `${promotionTotal} 个活动，FAQ 已同步 ${setupStatus?.promotionKnowledgeSyncedTotal || 0}/${promotionTotal}`
              : `${promotionTotal} 个活动，未同步 ${promotionUnsynced}，索引失败 ${promotionFailed}`
            : "优惠权益、到店礼、预约话术",
        done: promotionKnowledgeReady,
        href: "/dashboard/promotions",
        icon: <BadgePercentIcon />,
      },
      {
        key: "llm",
        title: "AI 模型",
        description:
          setupStatus?.llmConfigId && setupStatus?.embeddingConfigId
            ? `聊天：${setupStatus.llmConfigName || `模型 #${setupStatus.llmConfigId}`}；向量：${setupStatus.embeddingConfigName || `模型 #${setupStatus.embeddingConfigId}`}`
            : setupStatus?.llmConfigId
              ? "已启用聊天模型，还需要启用 embedding 模型用于知识检索"
              : setupStatus?.embeddingConfigId
                ? "已启用 embedding 模型，还需要启用聊天模型用于客户回复"
                : "启用聊天模型和 embedding 模型",
        done: Boolean(setupStatus?.llmConfigId && setupStatus?.embeddingConfigId),
        href: "/dashboard/ai-configs",
        icon: <SparklesIcon />,
      },
      {
        key: "agent",
        title: "数字店长 Agent",
        description: setupStatus?.agentId
          ? `${setupStatus.agentName || `Agent #${setupStatus.agentId}`}${setupStatus.workflowPublished ? "，流程已发布" : "，待发布流程"}`
          : "一键生成默认接待 Agent 和流程",
        done: Boolean(setupStatus?.agentId && setupStatus.workflowPublished),
        href: "/dashboard/ai-agents",
        icon: <MessageSquareMoreIcon />,
      },
      {
        key: "human-handoff",
        title: "人工接待",
        description: setupStatus?.humanHandoff?.message || "配置顾问组、排班和可自动分配顾问",
        done: Boolean(setupStatus?.humanHandoff?.ready),
        href: "/dashboard/agents",
        icon: <UserRoundCheckIcon />,
      },
      {
        key: "channel",
        title: "Web 聊天渠道",
        description: setupStatus?.webChannelId
          ? `${setupStatus.webChannelName || "Web 渠道"} / ${setupStatus.webChannelCode || "-"}`
          : "生成客户聊天入口",
        done: Boolean(setupStatus?.webChannelId),
        href: "/dashboard/channels",
        icon: <RouterIcon />,
      },
      {
        key: "knowledge",
        title: "知识同步",
        description: knowledgeDone
          ? `店长配置已同步到 FAQ #${profile?.knowledgeFAQId}`
          : "把店长配置同步为可检索知识",
        done: knowledgeDone,
        href: "/dashboard/knowledge",
        icon: <DatabaseZapIcon />,
      },
    ]
  }, [profile, setupStatus])

  const completedCount = setupSteps.filter((step) => step.done).length
  const complete = completedCount === setupSteps.length
  const webEntry = deliveryReport?.webEntry?.channelCode
    ? deliveryReport.webEntry
    : setupStatus?.webEntry
  const humanHandoff = deliveryReport?.humanHandoff || setupStatus?.humanHandoff
  const webChannelCode = webEntry?.channelCode || setupStatus?.webChannelCode || ""
  const chatUrl = useMemo(() => {
    if (deliveryReport?.chatUrl) return deliveryReport.chatUrl
    if (webEntry?.chatUrl) return webEntry.chatUrl
    if (!origin || !webChannelCode) return ""
    const url = new URL("/support/chat/", origin)
    url.searchParams.set("channelId", webChannelCode)
    return url.toString()
  }, [deliveryReport?.chatUrl, origin, webChannelCode, webEntry?.chatUrl])
  const embedSnippet = useMemo(() => {
    if (deliveryReport?.embedSnippet) return deliveryReport.embedSnippet
    if (webEntry?.embedSnippet) return webEntry.embedSnippet
    if (!origin || !webChannelCode) return ""
    return `<script>
  window.AgentDeskConfig = {
    channelId: ${jsStringLiteral(webChannelCode)},
    baseUrl: ${jsStringLiteral(origin)},
    title: ${jsStringLiteral(webEntry?.title || "AI数字店长")},
    subtitle: ${jsStringLiteral(webEntry?.subtitle || "")},
    themeColor: ${jsStringLiteral(webEntry?.themeColor || "#2563eb")},
    position: ${jsStringLiteral(webEntry?.position || "right")},
    width: ${jsStringLiteral(webEntry?.width || "380px")}
  };
</script>
<script async src="${origin}/sdk/agent-desk-sdk.min.js"></script>`
  }, [deliveryReport?.embedSnippet, origin, webChannelCode, webEntry])
  const latestAcceptanceFailures = (latestDeliveryRecord?.acceptanceResults || []).filter(
    (item) => !item.passed
  )

  async function copyText(text: string, successText: string) {
    if (!text) return
    try {
      await navigator.clipboard.writeText(text)
      toast.success(successText)
    } catch {
      toast.error("复制失败")
    }
  }

  return (
    <DashboardPage>
      <DashboardToolbar
        actions={
          <>
            <Button
              variant="outline"
              onClick={() => void loadState()}
              disabled={loading || Boolean(runningAction)}
            >
              <RefreshCwIcon className={loading ? "animate-spin" : undefined} />
              刷新
            </Button>
            {templates[0] ? (
              <Button
                onClick={() => void handleApplyTemplate(templates[0])}
                disabled={Boolean(runningAction)}
              >
                <SparklesIcon />
                {runningAction === `template-${templates[0].code}` ? "初始化中" : "导入行业样板"}
              </Button>
            ) : null}
            <Button
              variant="outline"
              onClick={() => void handleEnsureRuntime()}
              disabled={Boolean(runningAction) || !profile?.initialized}
            >
              <RouterIcon />
              {runningAction === "ensure-runtime" ? "生成中" : "生成接待运行时"}
            </Button>
          </>
        }
      >
        <div>
          <h1 className="text-lg font-semibold">商家交付初始化</h1>
          <p className="text-sm text-muted-foreground">
            {complete
              ? "当前部署已具备数字店长接待、推荐和留资的基础数据。"
              : "把品牌、产品、活动和知识库准备好后即可进入客户接待。"}
          </p>
        </div>
      </DashboardToolbar>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="space-y-4">
          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm text-muted-foreground">初始化进度</div>
                <div className="mt-1 text-2xl font-semibold">
                  {completedCount}/{setupSteps.length}
                </div>
                {profile?.templateCode ? (
                  <div className="mt-1 text-xs text-muted-foreground">
                    当前样板：{profile.templateCode}
                    {profile.templateVersion ? ` v${profile.templateVersion}` : ""}
                    {profile.templateAppliedAt ? ` / ${profile.templateAppliedAt}` : ""}
                  </div>
                ) : null}
              </div>
              <Badge variant={complete ? "default" : "secondary"}>
                {complete ? "可接待" : "待完善"}
              </Badge>
            </div>
            <div className="mt-4 h-2 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${(completedCount / setupSteps.length) * 100}%` }}
              />
            </div>
          </section>

          <div className="grid gap-3 md:grid-cols-2">
            {setupSteps.map((step) => (
              <Link
                key={step.key}
                href={step.href}
                className="rounded-md border border-border/70 bg-card p-4 transition-colors hover:bg-muted/40"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 gap-3">
                    <div
                      className={cn(
                        "flex size-9 shrink-0 items-center justify-center rounded-md border",
                        step.done
                          ? "border-primary/30 bg-primary/10 text-primary"
                          : "border-border bg-background text-muted-foreground"
                      )}
                    >
                      {step.icon}
                    </div>
                    <div className="min-w-0">
                      <div className="font-medium">{step.title}</div>
                      <div className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                        {step.description}
                      </div>
                    </div>
                  </div>
                  {step.done ? (
                    <BadgeCheckIcon className="size-5 shrink-0 text-primary" />
                  ) : (
                    <ExternalLinkIcon className="size-4 shrink-0 text-muted-foreground" />
                  )}
                </div>
              </Link>
            ))}
          </div>
        </div>

        <aside className="space-y-4">
          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center gap-2 font-medium">
              <BoxesIcon className="size-4" />
              样板导入
            </div>
            <div className="mt-3 space-y-3">
              <input
                ref={templateImportInputRef}
                type="file"
                accept="application/json,.json"
                className="hidden"
                onChange={(event) => void handleImportTemplateFile(event.target.files?.[0])}
              />
              {templates.length ? (
                templates.map((template) => (
                  <div
                    key={template.code}
                    className="rounded-md border border-border/70 bg-background p-3"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-medium">{template.name}</span>
                          <Badge variant="secondary">{template.industry}</Badge>
                          {template.version ? (
                            <Badge variant="outline">v{template.version}</Badge>
                          ) : null}
                        </div>
                        <p className="mt-2 text-xs leading-5 text-muted-foreground">
                          {template.description}
                        </p>
                      </div>
                      <SparklesIcon
                        className={cn(
                          "mt-0.5 size-4 shrink-0 text-muted-foreground",
                          runningAction === `template-${template.code}` && "animate-pulse text-primary"
                        )}
                      />
                    </div>
                    <div className="mt-3 grid grid-cols-2 gap-2">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void handleApplyTemplate(template)}
                        disabled={Boolean(runningAction)}
                      >
                        <SparklesIcon />
                        应用样板
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void handleExportTemplate(template)}
                        disabled={Boolean(runningAction)}
                      >
                        <DownloadIcon />
                        导出 JSON
                      </Button>
                    </div>
                  </div>
                ))
              ) : (
                <div className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
                  暂无可用行业样板
                </div>
              )}
              <Button
                type="button"
                variant="outline"
                className="w-full justify-start"
                onClick={() => templateImportInputRef.current?.click()}
                disabled={Boolean(runningAction)}
              >
                <UploadIcon />
                导入 JSON 模板
              </Button>
              <Button
                variant="outline"
                className="w-full justify-start"
                onClick={() =>
                  void runAction(
                    "sync-profile",
                    syncDigitalStoreKnowledge,
                    "店长配置已同步到知识库"
                  )
                }
                disabled={Boolean(runningAction) || !profile?.initialized}
              >
                <DatabaseZapIcon />
                同步店长知识
              </Button>
            </div>
          </section>

          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="font-medium">下一步</div>
            <div className="mt-3 grid gap-2">
              <Link
                href="/dashboard/ai-configs"
                className={buttonVariants({ variant: "outline" })}
              >
                配置 AI 模型
              </Link>
              <Link
                href="/dashboard/ai-agents"
                className={buttonVariants({ variant: "outline" })}
              >
                检查 Agent
              </Link>
              <Link
                href="/dashboard/channels"
                className={buttonVariants({ variant: "outline" })}
              >
                接入聊天渠道
              </Link>
            </div>
          </section>

          <section className="delivery-report-print rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 font-medium">
                  <DatabaseZapIcon className="size-4" />
                  知识库导入助手
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {knowledgeAssistant?.industry || "当前行业"} 必备 FAQ 检查
                </div>
              </div>
              <Badge variant={knowledgeAssistant?.missingTotal ? "secondary" : "default"}>
                待补 {knowledgeAssistant?.missingTotal ?? 0}
              </Badge>
            </div>
            <div className="mt-3 grid grid-cols-2 gap-2 text-center text-xs">
              <div className="rounded-md bg-muted/50 px-2 py-2">
                <div className="font-medium text-foreground">
                  {knowledgeAssistant?.coveredTotal ?? 0}
                </div>
                <div className="text-muted-foreground">已覆盖</div>
              </div>
              <div className="rounded-md bg-muted/50 px-2 py-2">
                <div className="font-medium text-foreground">
                  {knowledgeAssistant?.missingTotal ?? 0}
                </div>
                <div className="text-muted-foreground">待补充</div>
              </div>
            </div>
            <div className="mt-3 space-y-2">
              {(knowledgeAssistant?.items || [])
                .filter((item) => !item.covered)
                .slice(0, 5)
                .map((item) => (
                  <div
                    key={item.key}
                    className="rounded-md border border-border/70 bg-background px-3 py-2"
                  >
                    <div className="line-clamp-2 text-xs font-medium">{item.question}</div>
                    <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                      {item.reason}
                    </div>
                    <div className="mt-2 flex items-center justify-between gap-2">
                      <div className="line-clamp-1 text-xs text-muted-foreground">
                        {item.keywords.slice(0, 3).join(" / ")}
                      </div>
                      {item.actionHref ? (
                        <Link
                          href={item.actionHref}
                          className={buttonVariants({ variant: "outline", size: "sm" })}
                        >
                          {item.actionLabel || "去补 FAQ"}
                        </Link>
                      ) : null}
                    </div>
                  </div>
                ))}
              {knowledgeAssistant && knowledgeAssistant.missingTotal === 0 ? (
                <div className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
                  当前行业必备 FAQ 已基本覆盖。
                </div>
              ) : null}
            </div>
          </section>

          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 font-medium">
                  <SparklesIcon className="size-4" />
                  模板效果回收
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {templateEffect?.templateCode
                    ? `${templateEffect.templateCode} ${templateEffect.templateVersion || ""}`
                    : "当前店铺未记录模板来源"}
                </div>
              </div>
              <div className="flex flex-wrap items-center justify-end gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void handleCopyTemplateImprovementPack()}
                  disabled={!templateEffect?.improvementMarkdown}
                >
                  <CopyIcon className="mr-2 size-4" />
                  复制改进包
                </Button>
                <Badge
                  variant={
                    (templateEffect?.missingQuestionTotal || 0) +
                      (templateEffect?.negativeFeedbackTotal || 0) >
                    0
                      ? "secondary"
                      : "default"
                  }
                >
                  近 {templateEffect?.days ?? 30} 天
                </Badge>
              </div>
            </div>
            <div className="mt-3 grid grid-cols-3 gap-2 text-center text-xs">
              <div className="rounded-md bg-muted/50 px-2 py-2">
                <div className="font-medium text-foreground">
                  {templateEffect?.retrieveTotal ?? 0}
                </div>
                <div className="text-muted-foreground">检索</div>
              </div>
              <div className="rounded-md bg-muted/50 px-2 py-2">
                <div className="font-medium text-foreground">
                  {templateEffect?.missingQuestionTotal ?? 0}
                </div>
                <div className="text-muted-foreground">缺口</div>
              </div>
              <div className="rounded-md bg-muted/50 px-2 py-2">
                <div className="font-medium text-foreground">
                  {templateEffect?.negativeFeedbackTotal ?? 0}
                </div>
                <div className="text-muted-foreground">负反馈</div>
              </div>
            </div>
            <div className="mt-3 space-y-3">
              <div className="rounded-md border border-border/70 bg-background p-3">
                <div className="text-xs font-medium">高频模板缺口</div>
                <div className="mt-2 space-y-2">
                  {(templateEffect?.missingQuestions || []).slice(0, 3).map((item) => (
                    <div key={`missing-${item.question}`} className="rounded-md border px-3 py-2">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0">
                          <div className="line-clamp-2 text-xs font-medium">{item.question}</div>
                          <div className="mt-1 text-xs text-muted-foreground">
                            {item.answerStatusName || "待处理"} / {item.latestAt || "-"}
                          </div>
                        </div>
                        <Badge variant="outline">{item.count}</Badge>
                      </div>
                      {item.actionHref ? (
                        <Link
                          href={item.actionHref}
                          className={cn(
                            buttonVariants({ variant: "outline", size: "sm" }),
                            "mt-2"
                          )}
                        >
                          {item.actionLabel || "查看日志"}
                        </Link>
                      ) : null}
                    </div>
                  ))}
                  {templateEffect && templateEffect.missingQuestions.length === 0 ? (
                    <div className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
                      暂无无答案、兜底或风控问题。
                    </div>
                  ) : null}
                </div>
              </div>
              <div className="rounded-md border border-border/70 bg-background p-3">
                <div className="text-xs font-medium">高频负反馈</div>
                <div className="mt-2 space-y-2">
                  {(templateEffect?.negativeFeedbacks || []).slice(0, 3).map((item) => (
                    <div key={`feedback-${item.question}`} className="rounded-md border px-3 py-2">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0">
                          <div className="line-clamp-2 text-xs font-medium">{item.question}</div>
                          <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                            {item.feedbackReason ||
                              item.feedbackTypeName ||
                              item.answerStatusName ||
                              "未填写原因"}
                          </div>
                        </div>
                        <Badge variant="outline">{item.count}</Badge>
                      </div>
                      {item.actionHref ? (
                        <Link
                          href={item.actionHref}
                          className={cn(
                            buttonVariants({ variant: "outline", size: "sm" }),
                            "mt-2"
                          )}
                        >
                          {item.actionLabel || "查看反馈"}
                        </Link>
                      ) : null}
                    </div>
                  ))}
                  {templateEffect && templateEffect.negativeFeedbacks.length === 0 ? (
                    <div className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
                      暂无负反馈问题。
                    </div>
                  ) : null}
                </div>
              </div>
              {templateEffect?.suggestions?.length ? (
                <div className="space-y-2">
                  {templateEffect.suggestions.slice(0, 3).map((item) => (
                    <div
                      key={item}
                      className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground"
                    >
                      {item}
                    </div>
                  ))}
                </div>
              ) : null}
            </div>
          </section>

          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 font-medium">
                  <DatabaseZapIcon className="size-4" />
                  运维与升级
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  备份、恢复演练和版本更新检查
                </div>
              </div>
              <Badge variant={maintenanceStatus?.status === "ok" ? "default" : "secondary"}>
                {maintenanceStatus?.status === "ok" ? "正常" : "需关注"}
              </Badge>
            </div>
            <div className="mt-3 space-y-3">
              <div className="rounded-md border border-border/70 bg-background px-3 py-2 text-xs">
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium">最近备份</span>
                  <span className="text-muted-foreground">
                    {maintenanceStatus?.latestBackup?.createdAt ||
                      maintenanceStatus?.latestBackup?.timestamp ||
                      "未发现"}
                  </span>
                </div>
                {maintenanceStatus?.latestBackup ? (
                  <>
                    <div className="mt-1 truncate text-muted-foreground">
                      {maintenanceStatus.latestBackup.path}
                    </div>
                    <div className="mt-2 grid grid-cols-2 gap-2 text-muted-foreground">
                      <div className="rounded-md bg-muted/50 px-2 py-1">
                        MySQL：{maintenanceStatus.latestBackup.hasMysqlDump ? "有" : "无"}
                      </div>
                      <div className="rounded-md bg-muted/50 px-2 py-1">
                        data：{maintenanceStatus.latestBackup.hasDataArchive ? "有" : "无"}
                      </div>
                      <div className="rounded-md bg-muted/50 px-2 py-1">
                        配置：{maintenanceStatus.latestBackup.hasConfigSnapshot ? "有" : "无"}
                      </div>
                      <div className="rounded-md bg-muted/50 px-2 py-1">
                        大小：{formatBytes(maintenanceStatus.latestBackup.sizeBytes)}
                      </div>
                    </div>
                  </>
                ) : null}
              </div>
              {maintenanceStatus?.warnings?.length ? (
                <div className="space-y-2">
                  {maintenanceStatus.warnings.map((item) => (
                    <div
                      key={item.key}
                      className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground"
                    >
                      <div className="font-medium text-foreground">{item.label}</div>
                      <div className="mt-1 leading-5">{item.message}</div>
                    </div>
                  ))}
                </div>
              ) : null}
              <div className="space-y-2">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start"
                  disabled={!maintenanceStatus?.backupCommand}
                  onClick={() =>
                    void copyText(maintenanceStatus?.backupCommand || "", "已复制备份命令")
                  }
                >
                  <CopyIcon />
                  复制备份命令
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start"
                  disabled={!maintenanceStatus?.restoreDryRunCommand}
                  onClick={() =>
                    void copyText(
                      maintenanceStatus?.restoreDryRunCommand || "",
                      "已复制恢复演练命令"
                    )
                  }
                >
                  <CopyIcon />
                  复制恢复演练命令
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start"
                  disabled={!maintenanceStatus?.upgradeRunbook}
                  onClick={() =>
                    void copyText(
                      maintenanceStatus?.upgradeRunbook || "",
                      "已复制升级 Runbook"
                    )
                  }
                >
                  <CopyIcon />
                  复制升级 Runbook
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start"
                  disabled={!maintenanceStatus?.upgradeCommands?.length}
                  onClick={() =>
                    void copyText(
                      maintenanceStatus?.upgradeCommands?.join("\n") || "",
                      "已复制升级检查命令"
                    )
                  }
                >
                  <CopyIcon />
                  复制升级检查命令
                </Button>
              </div>
            </div>
          </section>

          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="font-medium">聊天入口</div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {webChannelCode ? `渠道 ${webChannelCode}` : "生成接待运行时后可获取入口"}
                </div>
              </div>
              {webChannelCode ? <Badge variant="secondary">已生成</Badge> : null}
            </div>
            {webChannelCode ? (
              <div className="mt-3 space-y-3">
                <div className="rounded-md border border-border/70 bg-background p-3">
                  <div className="flex items-center gap-3">
                    <div
                      className="flex size-10 shrink-0 items-center justify-center rounded-full text-white"
                      style={{ backgroundColor: webEntry?.themeColor || "#2563eb" }}
                    >
                      <BotMessageSquareIcon className="size-5" />
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">
                        {webEntry?.title || "AI数字店长"}
                      </div>
                      <div className="mt-0.5 truncate text-xs text-muted-foreground">
                        {webEntry?.subtitle || profile?.brandName || "欢迎咨询门店服务"}
                      </div>
                    </div>
                  </div>
                  <div className="mt-3 grid gap-2 text-xs text-muted-foreground">
                    <div className="flex justify-between gap-3">
                      <span>主题色</span>
                      <span className="flex min-w-0 items-center gap-2 text-right text-foreground">
                        <span
                          className="size-3 rounded-full border border-border"
                          style={{ backgroundColor: webEntry?.themeColor || "#2563eb" }}
                        />
                        {webEntry?.themeColor || "#2563eb"}
                      </span>
                    </div>
                    <div className="flex justify-between gap-3">
                      <span>位置/宽度</span>
                      <span className="text-right text-foreground">
                        {webEntry?.position || "right"} / {webEntry?.width || "380px"}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="rounded-md border bg-muted/40 px-3 py-2 font-mono text-xs break-all">
                  {chatUrl}
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void copyText(chatUrl, "已复制聊天链接")}
                  >
                    <CopyIcon />
                    复制链接
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => window.open(chatUrl, "_blank", "noopener,noreferrer")}
                  >
                    <ExternalLinkIcon />
                    打开
                  </Button>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start"
                  onClick={() => void copyText(embedSnippet, "已复制嵌入代码")}
                >
                  <CopyIcon />
                  复制网站嵌入代码
                </Button>
              </div>
            ) : (
              <Button
                type="button"
                variant="outline"
                className="mt-3 w-full justify-start"
                onClick={() => void handleEnsureRuntime()}
                disabled={Boolean(runningAction) || !profile?.initialized}
              >
                <RouterIcon />
                生成接待运行时
              </Button>
            )}
          </section>

          <section className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 font-medium">
                  <FileTextIcon className="size-4" />
                  交付报告
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {deliveryReport?.ready ? "配置完整，可复制给交付资料" : "用于交付前确认缺口"}
                </div>
              </div>
              <Badge variant={deliveryReport?.ready ? "default" : "secondary"}>
                {deliveryReport?.ready ? "可交付" : "待完善"}
              </Badge>
            </div>
            <div className="mt-3 space-y-3">
              <div className="grid gap-2 text-xs text-muted-foreground">
                <div className="flex justify-between gap-3">
                  <span>生成时间</span>
                  <span className="text-right text-foreground">
                    {deliveryReport?.generatedAt || "-"}
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span>品牌</span>
                  <span className="text-right text-foreground">
                    {deliveryReport?.brandName || profile?.brandName || "-"}
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span>门店</span>
                  <span className="text-right text-foreground">
                    {deliveryReport?.storeName || profile?.storeName || "-"}
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span>后台地址</span>
                  <span className="max-w-48 truncate text-right text-foreground">
                    {deliveryReport?.dashboardUrl || "-"}
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span>聊天入口</span>
                  <span className="max-w-48 truncate text-right text-foreground">
                    {deliveryReport?.chatUrl || chatUrl || "-"}
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span>验收命令</span>
                  <span className="max-w-48 truncate text-right font-mono text-foreground">
                    {deliveryReport?.acceptanceCommand || "-"}
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span>外部通知</span>
                  <span className="text-right text-foreground">
                    {deliveryReport?.notificationStatus?.enabled
                      ? `已启用 / ${deliveryReport.notificationStatus.format || "generic"}`
                      : deliveryReport?.notificationStatus?.status || "未配置"}
                  </span>
                </div>
              </div>
              {deliveryReport?.notificationStatus?.message ? (
                <div className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
                  {deliveryReport.notificationStatus.message}
                </div>
              ) : null}
              {deliveryReport?.missingSteps?.length ? (
                <div className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
                  {deliveryReport.missingSteps.join("、")}
                </div>
              ) : null}
              {deliveryReport?.items?.length ? (
                <div className="space-y-2">
                  <div className="text-xs font-medium text-muted-foreground">配置检查</div>
                  <div className="space-y-2">
                    {deliveryReport.items.map((item) => (
                      <div
                        key={item.label}
                        className="rounded-md border border-border/70 bg-background px-3 py-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="min-w-0">
                            <div className="truncate text-xs font-medium">{item.label}</div>
                            <div className="mt-1 line-clamp-1 text-xs text-muted-foreground">
                              {item.value || "-"}
                            </div>
                          </div>
                          <div className="flex shrink-0 items-center gap-2">
                            <Badge variant={item.status === "完成" ? "default" : "secondary"}>
                              {item.status || "-"}
                            </Badge>
                            {item.status !== "完成" && item.actionHref ? (
                              <Link
                                href={item.actionHref}
                                className={buttonVariants({ variant: "outline", size: "sm" })}
                              >
                                {item.actionLabel || "去处理"}
                              </Link>
                            ) : null}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {humanHandoff ? (
                <div className="rounded-md border border-border/70 bg-background px-3 py-2">
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-2 text-xs font-medium">
                      <UserRoundCheckIcon className="size-3.5 shrink-0" />
                      <span className="truncate">人工接待</span>
                    </div>
                    <Badge variant={humanHandoff.ready ? "default" : "secondary"}>
                      {humanHandoff.ready ? "可接待" : "待配置"}
                    </Badge>
                  </div>
                  <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                    {humanHandoff.message || "-"}
                  </div>
                  <div className="mt-2 grid grid-cols-3 gap-2 text-center text-xs">
                    <div className="rounded-md bg-muted/50 px-2 py-1">
                      <div className="font-medium text-foreground">
                        {humanHandoff.agentTeamIds?.length || 0}
                      </div>
                      <div className="text-muted-foreground">顾问组</div>
                    </div>
                    <div className="rounded-md bg-muted/50 px-2 py-1">
                      <div className="font-medium text-foreground">
                        {humanHandoff.activeTeamIds?.length || 0}
                      </div>
                      <div className="text-muted-foreground">排班中</div>
                    </div>
                    <div className="rounded-md bg-muted/50 px-2 py-1">
                      <div className="font-medium text-foreground">
                        {humanHandoff.candidateProfiles || 0}
                      </div>
                      <div className="text-muted-foreground">可分配</div>
                    </div>
                  </div>
                </div>
              ) : null}
              {deliveryReport?.modelHealthChecks?.length ? (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                    <SparklesIcon className="size-3.5" />
                    模型与检索健康
                  </div>
                  <div className="space-y-2">
                    {deliveryReport.modelHealthChecks.map((item) => (
                      <div
                        key={item.key}
                        className="rounded-md border border-border/70 bg-background px-3 py-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="truncate text-xs font-medium">{item.label}</div>
                          <div className="flex shrink-0 items-center gap-2">
                            <Badge variant={securityCheckBadgeVariant(item.status)}>
                              {securityCheckStatusText(item.status)}
                            </Badge>
                            {item.status !== "ok" && item.actionHref ? (
                              <Link
                                href={item.actionHref}
                                className={buttonVariants({ variant: "outline", size: "sm" })}
                              >
                                {item.actionLabel || "去处理"}
                              </Link>
                            ) : null}
                          </div>
                        </div>
                        <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                          {item.message}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {deliveryReport?.securityChecks?.length ? (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                    <ShieldCheckIcon className="size-3.5" />
                    上线安全自检
                  </div>
                  <div className="space-y-2">
                    {deliveryReport.securityChecks.map((item) => (
                      <div
                        key={item.key}
                        className="rounded-md border border-border/70 bg-background px-3 py-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="truncate text-xs font-medium">{item.label}</div>
                          <div className="flex shrink-0 items-center gap-2">
                            <Badge variant={securityCheckBadgeVariant(item.status)}>
                              {securityCheckStatusText(item.status)}
                            </Badge>
                            {item.status !== "ok" && item.actionHref ? (
                              <Link
                                href={item.actionHref}
                                className={buttonVariants({ variant: "outline", size: "sm" })}
                              >
                                {item.actionLabel || "去处理"}
                              </Link>
                            ) : null}
                          </div>
                        </div>
                        <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                          {item.message}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {deliveryReport?.acceptanceItems?.length ? (
                <div className="space-y-2">
                  <div className="text-xs font-medium text-muted-foreground">上线验收清单</div>
                  <div className="space-y-2">
                    {deliveryReport.acceptanceItems.map((item) => (
                      <div
                        key={item.code}
                        className="rounded-md border border-border/70 bg-background px-3 py-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="truncate text-xs font-medium">
                            {item.code} {item.title}
                          </div>
                          <Badge variant={item.blocking ? "default" : "secondary"}>
                            {item.blocking ? "阻断" : "观察"}
                          </Badge>
                        </div>
                        <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                          {item.customerAsk}
                        </div>
                        <div className="mt-1 hidden text-xs text-muted-foreground delivery-report-print-only">
                          <div>期望：{item.expectation}</div>
                          <div>后台检查：{item.consoleCheck}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {deliveryReport?.embedSnippet ? (
                <div className="hidden space-y-2 delivery-report-print-only">
                  <div className="text-xs font-medium text-muted-foreground">网站嵌入代码</div>
                  <pre className="whitespace-pre-wrap break-words rounded-md border border-border/70 bg-background px-3 py-2 text-xs">
                    {deliveryReport.embedSnippet}
                  </pre>
                </div>
              ) : null}
              <Button
                type="button"
                variant="outline"
                className="delivery-report-no-print w-full justify-start"
                disabled={!deliveryReport?.markdown}
                onClick={() => void copyText(deliveryReport?.markdown || "", "已复制交付报告")}
              >
                <CopyIcon />
                复制交付报告
              </Button>
              <Button
                type="button"
                variant="outline"
                className="delivery-report-no-print w-full justify-start"
                disabled={!deliveryReport?.acceptanceRunbook}
                onClick={() =>
                  void copyText(deliveryReport?.acceptanceRunbook || "", "已复制验收执行清单")
                }
              >
                <CopyIcon />
                复制验收执行清单
              </Button>
              <Button
                type="button"
                variant="outline"
                className="delivery-report-no-print w-full justify-start"
                disabled={!deliveryReport}
                onClick={printDeliveryReport}
              >
                <PrinterIcon />
                打印 / 保存 PDF
              </Button>
              <Button
                type="button"
                variant="outline"
                className="delivery-report-no-print w-full justify-start"
                disabled={Boolean(runningAction)}
                onClick={() => void handleTestWebhookNotify()}
              >
                <MessageSquareMoreIcon />
                {runningAction === "webhook-test" ? "发送中" : "发送关键通知测试"}
              </Button>
              {latestWebhookTest?.scenarios?.length ? (
                <div className="space-y-2 rounded-md border border-border/70 bg-background px-3 py-2">
                  <div className="flex items-center justify-between gap-2 text-xs">
                    <span className="font-medium">最近通知测试</span>
                    <div className="flex shrink-0 items-center gap-2">
                      {latestWebhookTest.sentTotal || latestWebhookTest.failedTotal ? (
                        <span className="text-muted-foreground">
                          成功 {latestWebhookTest.sentTotal || 0} / 失败 {latestWebhookTest.failedTotal || 0}
                        </span>
                      ) : null}
                      <span className="text-muted-foreground">{latestWebhookTest.testedAt || "-"}</span>
                    </div>
                  </div>
                  <div className="space-y-1.5">
                    {latestWebhookTest.scenarios.map((item) => (
                      <div
                        key={item.key}
                        className="rounded-md bg-muted/40 px-2 py-1.5 text-xs"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className="min-w-0 truncate text-muted-foreground">
                            {item.title}
                          </span>
                          <Badge variant={item.sent ? "default" : "secondary"}>
                            {item.sent ? "已发送" : latestWebhookTest.enabled ? "发送失败" : "未发送"}
                          </Badge>
                        </div>
                        {item.message ? (
                          <div className="mt-1 line-clamp-2 text-muted-foreground">
                            {item.message}
                          </div>
                        ) : null}
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              <Button
                type="button"
                variant="outline"
                className="delivery-report-no-print w-full justify-start"
                disabled={Boolean(runningAction) || !deliveryReport}
                onClick={() => void handleCreateDeliveryRecord()}
              >
                <FileTextIcon />
                {runningAction === "delivery-record" ? "保存中" : "保存交付记录"}
              </Button>
              <Button
                type="button"
                variant="outline"
                className="delivery-report-no-print w-full justify-start border-destructive/30 text-destructive hover:bg-destructive/10 hover:text-destructive"
                disabled={Boolean(runningAction)}
                onClick={() => void handleCleanupDemoData()}
              >
                <Trash2Icon />
                {runningAction === "cleanup-demo-data" ? "清理中" : "清理演示数据"}
              </Button>
              <div className="delivery-report-no-print rounded-md border border-dashed px-3 py-2 text-xs leading-5 text-muted-foreground">
                清理测试会话、线索、工单、通知和检索日志；保留产品、活动、知识、模型、Agent、渠道、客户档案和交付记录。
              </div>
              {latestDeliveryRecord ? (
                <div className="rounded-md bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                  <div className="flex items-center justify-between gap-2">
                    <span>最近记录：{latestDeliveryRecord.createdAt || "-"}</span>
                    <Badge
                      variant={
                        latestDeliveryRecord.acceptanceStatus === "failed"
                          ? "destructive"
                          : latestDeliveryRecord.acceptanceStatus === "passed"
                            ? "default"
                            : "secondary"
                      }
                    >
                      {latestDeliveryRecord.acceptanceStatus || "-"}
                    </Badge>
                  </div>
                  <div className="mt-1 line-clamp-2">
                    {latestDeliveryRecord.acceptanceSummary || "暂无验收摘要"} / {latestDeliveryRecord.createUserName || "-"}
                  </div>
                  {latestDeliveryRecord.scenarioTotal > 0 ? (
                    <div className="mt-2 grid grid-cols-3 gap-2 text-center">
                      <div className="rounded-md bg-background px-2 py-1">
                        <div className="font-medium text-foreground">
                          {latestDeliveryRecord.scenarioTotal}
                        </div>
                        <div>场景</div>
                      </div>
                      <div className="rounded-md bg-background px-2 py-1">
                        <div className="font-medium text-foreground">
                          {latestDeliveryRecord.passedTotal}
                        </div>
                        <div>通过</div>
                      </div>
                      <div className="rounded-md bg-background px-2 py-1">
                        <div className="font-medium text-foreground">
                          {latestDeliveryRecord.failedTotal}
                        </div>
                        <div>失败</div>
                      </div>
                    </div>
                  ) : null}
                  {latestDeliveryRecord.acceptanceFinishedAt ? (
                    <div className="mt-2">
                      自动验收结束：{latestDeliveryRecord.acceptanceFinishedAt}
                    </div>
                  ) : null}
                  {latestAcceptanceFailures.length > 0 ? (
                    <div className="mt-2 space-y-2">
                      <div className="font-medium text-foreground">失败场景定位</div>
                      {latestAcceptanceFailures.map((item) => (
                        <div key={item.code} className="rounded-md bg-background px-2 py-2">
                          <div className="flex items-center justify-between gap-2">
                            <div className="min-w-0 truncate font-medium text-foreground">
                              {item.code} {item.title}
                            </div>
                            <Badge variant="destructive">
                              {item.failureType || "failed"}
                            </Badge>
                          </div>
                          <div className="mt-1 line-clamp-2">
                            {item.detail || item.reason || "未提供失败原因"}
                          </div>
                          {item.missingKeywords?.length ? (
                            <div className="mt-1">
                              缺失关键词：{item.missingKeywords.join(" / ")}
                            </div>
                          ) : null}
                          {item.matchedBanned ? (
                            <div className="mt-1">
                              命中禁用词：{item.matchedBanned}
                            </div>
                          ) : null}
                          {item.suggestion ? (
                            <div className="mt-1">
                              建议：{item.suggestion}
                            </div>
                          ) : null}
                          {item.reply ? (
                            <div className="mt-1 line-clamp-2">
                              回复片段：{item.reply}
                            </div>
                          ) : null}
                          {item.conversationUrl ? (
                            <a
                              href={item.conversationUrl}
                              className="mt-2 inline-flex text-xs font-medium text-primary hover:underline"
                            >
                              打开对应会话
                            </a>
                          ) : null}
                        </div>
                      ))}
                    </div>
                  ) : null}
                </div>
              ) : null}
            </div>
          </section>
        </aside>
      </div>
      <style jsx global>{`
        @media print {
          @page {
            margin: 12mm;
          }

          body * {
            visibility: hidden !important;
          }

          .delivery-report-print,
          .delivery-report-print * {
            visibility: visible !important;
          }

          .delivery-report-print {
            position: absolute !important;
            top: 0 !important;
            left: 0 !important;
            width: 100% !important;
            border: 0 !important;
            border-radius: 0 !important;
            background: #fff !important;
            color: #111 !important;
            box-shadow: none !important;
          }

          .delivery-report-no-print {
            display: none !important;
          }

          .delivery-report-print-only {
            display: block !important;
          }

          .delivery-report-print .line-clamp-1,
          .delivery-report-print .line-clamp-2 {
            display: block !important;
            overflow: visible !important;
            -webkit-line-clamp: unset !important;
            -webkit-box-orient: initial !important;
          }

          .delivery-report-print a {
            color: inherit !important;
            text-decoration: none !important;
          }

          .delivery-report-print pre {
            white-space: pre-wrap !important;
          }
        }
      `}</style>
    </DashboardPage>
  )
}
