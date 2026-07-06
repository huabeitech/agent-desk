"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import {
  CalendarClockIcon,
  ClipboardIcon,
  ExternalLinkIcon,
  GitMergeIcon,
  MessageSquareTextIcon,
  RefreshCcwIcon,
  SaveIcon,
  SendIcon,
  UserRoundIcon,
} from "lucide-react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { OptionCombobox } from "@/components/option-combobox"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { fetchUsersAll, type AdminUser } from "@/lib/api/admin"
import {
  createLeadFollowUp,
  fetchSalesLeadDetail,
  updateSalesLead,
  type AdminSalesLead,
  type LeadFollowUp,
  type SalesLeadDetail,
  type SalesLeadIntent,
  type SalesLeadStage,
  type SalesLeadStatus,
  type UpdateSalesLeadPayload,
} from "@/lib/api/sales-lead"
import { cn } from "@/lib/utils"

type SalesLeadDetailDialogProps = {
  leadId: number | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onChanged: () => void
}

type LeadForm = UpdateSalesLeadPayload

const statusOptions: Array<{ value: SalesLeadStatus; label: string }> = [
  { value: "new", label: "新线索" },
  { value: "following", label: "跟进中" },
  { value: "visited", label: "已到店" },
  { value: "converted", label: "已转化" },
  { value: "invalid", label: "无效" },
  { value: "closed", label: "已关闭" },
]

const intentOptions: Array<{ value: SalesLeadIntent; label: string }> = [
  { value: "high", label: "高意向" },
  { value: "medium", label: "中意向" },
  { value: "low", label: "低意向" },
  { value: "unknown", label: "未知" },
]

const stageOptions: Array<{ value: SalesLeadStage; label: string }> = [
  { value: "consulting", label: "咨询了解" },
  { value: "comparing", label: "对比决策" },
  { value: "appointment", label: "预约到店" },
  { value: "ready_to_buy", label: "准备购买" },
  { value: "after_sales", label: "售后问题" },
  { value: "unknown", label: "未知" },
]

function buildForm(lead: AdminSalesLead): LeadForm {
  return {
    id: lead.id,
    customerName: lead.customerName || "",
    phone: lead.phone || "",
    wechat: lead.wechat || "",
    city: lead.city || "",
    addressHint: lead.addressHint || "",
    budgetMin: lead.budgetMin || 0,
    budgetMax: lead.budgetMax || 0,
    interestedProducts: lead.interestedProducts || "",
    demandSummary: lead.demandSummary || "",
    intentLevel: lead.intentLevel || "unknown",
    buyingStage: lead.buyingStage || "unknown",
    appointmentAt: toDateTimeLocalValue(lead.appointmentAt),
    appointmentTimeText: lead.appointmentTimeText || "",
    appointmentStore: lead.appointmentStore || "",
    appointmentPeople: lead.appointmentPeople || 0,
    appointmentRemark: lead.appointmentRemark || "",
    ownerUserId: lead.ownerUserId || 0,
    status: lead.status || "new",
    remark: lead.remark || "",
  }
}

function toDateTimeLocalValue(value?: string) {
  if (!value) return ""
  return value.replace(" ", "T").slice(0, 16)
}

function fromDateTimeLocalValue(value: string) {
  if (!value) return ""
  const normalized = value.replace("T", " ")
  return normalized.length === 16 ? `${normalized}:00` : normalized
}

function formatMoney(value: number) {
  return value > 0 ? `¥${value.toLocaleString()}` : "-"
}

function statusLabel(value: SalesLeadStatus) {
  return statusOptions.find((option) => option.value === value)?.label ?? value
}

function intentLabel(value: SalesLeadIntent) {
  return intentOptions.find((option) => option.value === value)?.label ?? value
}

function stageLabel(value: SalesLeadStage) {
  return stageOptions.find((option) => option.value === value)?.label ?? value
}

function mergeKeyLabel(value?: string) {
  switch (value) {
    case "new":
      return "新建"
    case "conversation":
      return "同会话"
    case "phone":
      return "同手机号"
    case "wechat":
      return "同微信"
    case "customer":
      return "同客户"
    default:
      return value || "系统记录"
  }
}

function autoTagVariant(level?: string) {
  if (level === "danger") return "destructive" as const
  if (level === "warning" || level === "hot") return "secondary" as const
  return "outline" as const
}

function userLabel(user: AdminUser) {
  return user.nickname || user.username || `#${user.id}`
}

function customerKeyword(lead: AdminSalesLead) {
  return lead.customer?.primaryMobile ||
    lead.phone ||
    lead.customer?.name ||
    lead.customerName ||
    ""
}

function Field({
  label,
  children,
  className,
}: {
  label: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <div className={cn("space-y-1.5", className)}>
      <Label>{label}</Label>
      {children}
    </div>
  )
}

export function SalesLeadDetailDialog({
  leadId,
  open,
  onOpenChange,
  onChanged,
}: SalesLeadDetailDialogProps) {
  const [detail, setDetail] = useState<SalesLeadDetail | null>(null)
  const [form, setForm] = useState<LeadForm | null>(null)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [followUpContent, setFollowUpContent] = useState("")
  const [nextAction, setNextAction] = useState("")
  const [nextFollowUpAt, setNextFollowUpAt] = useState("")
  const [followUpSaving, setFollowUpSaving] = useState(false)
  const seqRef = useRef(0)

  const loadDetail = useCallback(async (targetLeadId = leadId) => {
    if (!open || !targetLeadId) {
      return
    }
    const seq = seqRef.current + 1
    seqRef.current = seq
    setLoading(true)
    try {
      const data = await fetchSalesLeadDetail(targetLeadId)
      if (seqRef.current !== seq) return
      setDetail(data)
      setForm(buildForm(data.lead))
    } catch (error) {
      if (seqRef.current !== seq) return
      toast.error(error instanceof Error ? error.message : "加载线索详情失败")
    } finally {
      if (seqRef.current === seq) {
        setLoading(false)
      }
    }
  }, [leadId, open])

  useEffect(() => {
    if (!open) {
      setDetail(null)
      setForm(null)
      setFollowUpContent("")
      setNextAction("")
      setNextFollowUpAt("")
      setLoading(false)
      setSaving(false)
      setFollowUpSaving(false)
      return
    }
    void loadDetail(leadId)
  }, [leadId, loadDetail, open])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    fetchUsersAll()
      .then((data) => {
        if (!cancelled) {
          setUsers(data.filter((user) => user.status === 0))
        }
      })
      .catch((error) => {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : "加载负责人失败")
        }
      })
    return () => {
      cancelled = true
    }
  }, [open])

  function patchForm(patch: Partial<LeadForm>) {
    setForm((current) => (current ? { ...current, ...patch } : current))
  }

  async function handleSave() {
    if (!form || saving) return
    setSaving(true)
    try {
      await updateSalesLead({
        ...form,
        customerName: form.customerName.trim(),
        phone: form.phone.trim(),
        wechat: form.wechat.trim(),
        city: form.city.trim(),
        addressHint: form.addressHint.trim(),
        interestedProducts: form.interestedProducts.trim(),
        demandSummary: form.demandSummary.trim(),
        appointmentAt: fromDateTimeLocalValue(form.appointmentAt),
        appointmentTimeText: form.appointmentTimeText.trim(),
        appointmentStore: form.appointmentStore.trim(),
        appointmentRemark: form.appointmentRemark.trim(),
        remark: form.remark.trim(),
      })
      toast.success("线索已保存")
      await loadDetail(form.id)
      onChanged()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "保存线索失败")
    } finally {
      setSaving(false)
    }
  }

  async function handleCreateFollowUp() {
    if (!detail || followUpSaving) return
    const content = followUpContent.trim()
    if (!content) {
      toast.error("请输入跟进内容")
      return
    }
    setFollowUpSaving(true)
    try {
      await createLeadFollowUp({
        leadId: detail.lead.id,
        content,
        nextAction: nextAction.trim(),
        nextFollowUpAt: fromDateTimeLocalValue(nextFollowUpAt),
      })
      toast.success("跟进已记录")
      setFollowUpContent("")
      setNextAction("")
      setNextFollowUpAt("")
      await loadDetail(detail.lead.id)
      onChanged()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "记录跟进失败")
    } finally {
      setFollowUpSaving(false)
    }
  }

  const lead = detail?.lead
  const followUps: LeadFollowUp[] = detail?.followUps ?? []
  const followUpAdvice = detail?.followUpAdvice

  async function handleCopyFollowUpAdvice() {
    if (!followUpAdvice?.copyText) return
    try {
      await navigator.clipboard.writeText(followUpAdvice.copyText)
      toast.success("跟进摘要已复制")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "复制跟进摘要失败")
    }
  }

  function handleUseAdviceAsFollowUp() {
    if (!followUpAdvice) return
    setFollowUpContent(followUpAdvice.script || followUpAdvice.customerSummary || "")
    setNextAction(followUpAdvice.nextAction || "")
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-5xl">
        <DialogHeader>
          <div className="flex flex-wrap items-center justify-between gap-3 pr-8">
            <div className="min-w-0">
              <DialogTitle>{lead?.customerName || "销售线索"}</DialogTitle>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                {lead ? (
                  <>
                    <Badge>{intentLabel(lead.intentLevel)}</Badge>
                    <Badge variant="outline">{stageLabel(lead.buyingStage)}</Badge>
                    <Badge variant="secondary">{statusLabel(lead.status)}</Badge>
                    {(lead.autoTags ?? []).slice(0, 6).map((tag) => (
                      <Badge key={tag} variant="outline">
                        {tag}
                      </Badge>
                    ))}
                    <span>#{lead.id}</span>
                  </>
                ) : null}
              </div>
            </div>
            <div className="flex items-center gap-2">
              {lead?.conversationId ? (
                <Link
                  className={buttonVariants({ variant: "outline" })}
                  href={`/dashboard/conversations?conversationId=${lead.conversationId}`}
                >
                    <ExternalLinkIcon />
                    会话
                </Link>
              ) : null}
              {lead?.customerId ? (
                <Link
                  className={buttonVariants({ variant: "outline" })}
                  href={`/dashboard/customers?keyword=${encodeURIComponent(customerKeyword(lead))}`}
                >
                  <UserRoundIcon />
                  客户
                </Link>
              ) : null}
              <Button
                variant="outline"
                onClick={() => void loadDetail(leadId)}
                disabled={loading}
              >
                <RefreshCcwIcon className={loading ? "animate-spin" : undefined} />
                刷新
              </Button>
            </div>
          </div>
        </DialogHeader>

        {loading && !form ? (
          <div className="py-12 text-center text-muted-foreground">加载中</div>
        ) : form && lead ? (
          <div className="space-y-5">
            <div className="grid gap-3 rounded-lg border p-3 sm:grid-cols-4">
              <div>
                <div className="text-xs text-muted-foreground">预算</div>
                <div className="mt-1 font-medium">
                  {formatMoney(lead.budgetMin)} - {formatMoney(lead.budgetMax)}
                </div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">意向产品</div>
                <div className="mt-1 truncate font-medium">
                  {lead.interestedProducts || "-"}
                </div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">负责人</div>
                <div className="mt-1 font-medium">{lead.ownerUserName || "-"}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">创建时间</div>
                <div className="mt-1 font-medium">{lead.createdAt || "-"}</div>
              </div>
            </div>

            {(lead.autoTagDetails ?? []).length > 0 ? (
              <div className="rounded-lg border bg-muted/30 p-3">
                <div className="text-sm font-medium">自动标签说明</div>
                <div className="mt-3 grid gap-2 md:grid-cols-2">
                  {(lead.autoTagDetails ?? []).slice(0, 8).map((tag) => (
                    <div key={tag.label} className="rounded-md border bg-background px-3 py-2">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <Badge variant={autoTagVariant(tag.level)}>{tag.label}</Badge>
                        {tag.actionUrl ? (
                          <Link className="text-xs text-primary hover:underline" href={tag.actionUrl}>
                            {tag.actionLabel || "去处理"}
                          </Link>
                        ) : (
                          <span className="text-xs text-muted-foreground">{tag.actionLabel || "-"}</span>
                        )}
                      </div>
                      <div className="mt-2 text-sm text-muted-foreground">
                        {tag.reason || "系统根据线索状态自动生成。"}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            {lead.mergeReason ? (
              <div className="rounded-lg border bg-muted/30 p-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 font-medium">
                      <GitMergeIcon className="size-4" />
                      归并依据
                    </div>
                    <div className="mt-2 text-sm leading-6 text-muted-foreground">
                      {lead.mergeReason}
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-wrap items-center gap-2">
                    <Badge variant="secondary">{mergeKeyLabel(lead.mergeKey)}</Badge>
                    {lead.mergedAt ? (
                      <span className="text-xs text-muted-foreground">{lead.mergedAt}</span>
                    ) : null}
                  </div>
                </div>
              </div>
            ) : null}

            {lead?.lastCustomerMessage || lead?.lastMessageSummary ? (
              <div className="rounded-lg border p-3">
                <div className="flex items-center gap-2 font-medium">
                  <MessageSquareTextIcon className="size-4" />
                  最近会话
                </div>
                <div className="mt-3 grid gap-3 md:grid-cols-2">
                  <div className="rounded-md bg-muted/40 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">最近客户消息</div>
                    <div className="mt-1 leading-6">
                      {lead.lastCustomerMessage || "-"}
                    </div>
                  </div>
                  <div className="rounded-md bg-muted/40 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">会话摘要</div>
                    <div className="mt-1 leading-6">
                      {lead.lastMessageSummary || "-"}
                    </div>
                  </div>
                </div>
              </div>
            ) : null}

            {followUpAdvice ? (
              <div className="rounded-lg border p-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 font-medium">
                      <MessageSquareTextIcon className="size-4" />
                      顾问跟进建议
                    </div>
                    <div className="mt-2 text-sm text-muted-foreground">
                      {followUpAdvice.customerSummary || "-"}
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-wrap gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => void handleCopyFollowUpAdvice()}
                      disabled={!followUpAdvice.copyText}
                    >
                      <ClipboardIcon className="size-4" />
                      复制摘要
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={handleUseAdviceAsFollowUp}
                    >
                      <SendIcon className="size-4" />
                      填入跟进
                    </Button>
                  </div>
                </div>
                <div className="mt-3 grid gap-3 lg:grid-cols-[1fr_1.4fr]">
                  <div className="rounded-md bg-muted/40 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">建议下一步</div>
                    <div className="mt-1 leading-6">{followUpAdvice.nextAction || "-"}</div>
                  </div>
                  <div className="rounded-md bg-muted/40 p-3 text-sm">
                    <div className="text-xs text-muted-foreground">建议话术</div>
                    <div className="mt-1 leading-6">{followUpAdvice.script || "-"}</div>
                  </div>
                </div>
                {followUpAdvice.riskHints.length > 0 ? (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {followUpAdvice.riskHints.map((hint) => (
                      <Badge key={hint} variant="secondary">{hint}</Badge>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}

            <div className="grid gap-4 lg:grid-cols-[1.35fr_1fr]">
              <div className="space-y-4">
                <div className="grid gap-3 sm:grid-cols-3">
                  <Field label="客户姓名">
                    <Input
                      value={form.customerName}
                      onChange={(event) => patchForm({ customerName: event.target.value })}
                    />
                  </Field>
                  <Field label="手机号">
                    <Input
                      value={form.phone}
                      onChange={(event) => patchForm({ phone: event.target.value })}
                    />
                  </Field>
                  <Field label="微信">
                    <Input
                      value={form.wechat}
                      onChange={(event) => patchForm({ wechat: event.target.value })}
                    />
                  </Field>
                  <Field label="城市">
                    <Input
                      value={form.city}
                      onChange={(event) => patchForm({ city: event.target.value })}
                    />
                  </Field>
                  <Field label="地址/小区" className="sm:col-span-2">
                    <Input
                      value={form.addressHint}
                      onChange={(event) => patchForm({ addressHint: event.target.value })}
                    />
                  </Field>
                  <Field label="预算下限">
                    <Input
                      type="number"
                      min={0}
                      value={form.budgetMin}
                      onChange={(event) =>
                        patchForm({ budgetMin: Number(event.target.value) || 0 })
                      }
                    />
                  </Field>
                  <Field label="预算上限">
                    <Input
                      type="number"
                      min={0}
                      value={form.budgetMax}
                      onChange={(event) =>
                        patchForm({ budgetMax: Number(event.target.value) || 0 })
                      }
                    />
                  </Field>
                  <Field label="意向产品">
                    <Input
                      value={form.interestedProducts}
                      onChange={(event) =>
                        patchForm({ interestedProducts: event.target.value })
                      }
                    />
                  </Field>
                  <Field label="意向等级">
                    <OptionCombobox
                      value={form.intentLevel}
                      onChange={(value) =>
                        patchForm({ intentLevel: value as SalesLeadIntent })
                      }
                      placeholder="选择意向"
                      options={intentOptions}
                    />
                  </Field>
                  <Field label="购买阶段">
                    <OptionCombobox
                      value={form.buyingStage}
                      onChange={(value) =>
                        patchForm({ buyingStage: value as SalesLeadStage })
                      }
                      placeholder="选择阶段"
                      options={stageOptions}
                    />
                  </Field>
                  <Field label="状态">
                    <OptionCombobox
                      value={form.status}
                      onChange={(value) =>
                        patchForm({ status: value as SalesLeadStatus })
                      }
                      placeholder="选择状态"
                      options={statusOptions}
                    />
                  </Field>
                  <Field label="负责人">
                    <OptionCombobox
                      value={String(form.ownerUserId || 0)}
                      onChange={(value) =>
                        patchForm({ ownerUserId: Number(value) || 0 })
                      }
                      placeholder="选择负责人"
                      options={[
                        { value: "0", label: "未分配" },
                        ...users.map((user) => ({
                          value: String(user.id),
                          label: userLabel(user),
                        })),
                      ]}
                    />
                  </Field>
                </div>

                <div className="grid gap-3 sm:grid-cols-2">
                  <Field label="需求摘要" className="sm:col-span-2">
                    <Textarea
                      value={form.demandSummary}
                      onChange={(event) =>
                        patchForm({ demandSummary: event.target.value })
                      }
                      rows={4}
                    />
                  </Field>
                  <Field label="内部备注" className="sm:col-span-2">
                    <Textarea
                      value={form.remark}
                      onChange={(event) => patchForm({ remark: event.target.value })}
                      rows={3}
                    />
                  </Field>
                </div>
              </div>

              <div className="space-y-4">
                <div className="rounded-lg border p-3">
                  <div className="mb-3 flex items-center gap-2 font-medium">
                    <UserRoundIcon className="size-4" />
                    客户档案
                  </div>
                  {lead.customerId ? (
                    <div className="space-y-2 text-sm">
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-muted-foreground">客户</span>
                        <span className="min-w-0 truncate font-medium">
                          {lead.customer?.name || lead.customerName || `#${lead.customerId}`}
                        </span>
                      </div>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-muted-foreground">客户ID</span>
                        <span className="font-medium">#{lead.customerId}</span>
                      </div>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-muted-foreground">主手机号</span>
                        <span className="min-w-0 truncate font-medium">
                          {lead.customer?.primaryMobile || lead.phone || "-"}
                        </span>
                      </div>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-muted-foreground">主邮箱</span>
                        <span className="min-w-0 truncate font-medium">
                          {lead.customer?.primaryEmail || "-"}
                        </span>
                      </div>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-muted-foreground">最近活跃</span>
                        <span className="min-w-0 truncate font-medium">
                          {lead.customer?.lastActiveAt || "-"}
                        </span>
                      </div>
                      {lead.customer?.remark ? (
                        <div className="rounded-md bg-muted/50 p-2 text-muted-foreground">
                          {lead.customer.remark}
                        </div>
                      ) : null}
                      <Link
                        className={cn(buttonVariants({ variant: "outline", size: "sm" }), "w-full")}
                        href={`/dashboard/customers?keyword=${encodeURIComponent(customerKeyword(lead))}`}
                      >
                        <ExternalLinkIcon />
                        打开客户管理
                      </Link>
                    </div>
                  ) : (
                    <div className="rounded-md border border-dashed p-3 text-sm text-muted-foreground">
                      当前线索尚未绑定客户档案，保存手机号、微信或姓名后系统会尝试自动创建或复用客户。
                    </div>
                  )}
                </div>

                <div className="rounded-lg border p-3">
                  <div className="mb-3 flex items-center gap-2 font-medium">
                    <CalendarClockIcon className="size-4" />
                    预约信息
                  </div>
                  <div className="grid gap-3">
                    <Field label="预约时间">
                      <Input
                        type="datetime-local"
                        value={form.appointmentAt}
                        onChange={(event) =>
                          patchForm({ appointmentAt: event.target.value })
                        }
                      />
                    </Field>
                    <Field label="时间描述">
                      <Input
                        value={form.appointmentTimeText}
                        onChange={(event) =>
                          patchForm({ appointmentTimeText: event.target.value })
                        }
                      />
                    </Field>
                    <Field label="预约门店">
                      <Input
                        value={form.appointmentStore}
                        onChange={(event) =>
                          patchForm({ appointmentStore: event.target.value })
                        }
                      />
                    </Field>
                    <Field label="到店人数">
                      <Input
                        type="number"
                        min={0}
                        value={form.appointmentPeople}
                        onChange={(event) =>
                          patchForm({ appointmentPeople: Number(event.target.value) || 0 })
                        }
                      />
                    </Field>
                    <Field label="预约备注">
                      <Textarea
                        value={form.appointmentRemark}
                        onChange={(event) =>
                          patchForm({ appointmentRemark: event.target.value })
                        }
                        rows={3}
                      />
                    </Field>
                  </div>
                </div>
              </div>
            </div>

            <Separator />

            <div className="grid gap-4 lg:grid-cols-[1fr_1.1fr]">
              <div className="space-y-3">
                <div className="font-medium">新增跟进</div>
                <Field label="跟进内容">
                  <Textarea
                    value={followUpContent}
                    onChange={(event) => setFollowUpContent(event.target.value)}
                    rows={4}
                  />
                </Field>
                <div className="grid gap-3 sm:grid-cols-2">
                  <Field label="下一步动作">
                    <Input
                      value={nextAction}
                      onChange={(event) => setNextAction(event.target.value)}
                    />
                  </Field>
                  <Field label="下次跟进">
                    <Input
                      type="datetime-local"
                      value={nextFollowUpAt}
                      onChange={(event) => setNextFollowUpAt(event.target.value)}
                    />
                  </Field>
                </div>
                <Button
                  onClick={() => void handleCreateFollowUp()}
                  disabled={followUpSaving}
                >
                  <SendIcon />
                  {followUpSaving ? "记录中" : "记录跟进"}
                </Button>
              </div>

              <div className="space-y-3">
                <div className="font-medium">跟进记录</div>
                <div className="max-h-72 space-y-3 overflow-y-auto pr-1">
                  {followUps.length > 0 ? (
                    followUps.map((item) => (
                      <div key={item.id} className="rounded-lg border p-3">
                        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
                          <span>{item.operatorName || "-"}</span>
                          <span>{item.createdAt || "-"}</span>
                        </div>
                        <div className="mt-2 whitespace-pre-wrap">{item.content}</div>
                        {item.nextAction || item.nextFollowUpAt ? (
                          <div className="mt-2 text-xs text-muted-foreground">
                            {item.nextAction || "-"}
                            {item.nextFollowUpAt ? ` / ${item.nextFollowUpAt}` : ""}
                          </div>
                        ) : null}
                      </div>
                    ))
                  ) : (
                    <div className="rounded-lg border border-dashed py-10 text-center text-muted-foreground">
                      暂无跟进记录
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="py-12 text-center text-muted-foreground">线索不存在</div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
          <Button onClick={() => void handleSave()} disabled={!form || saving}>
            <SaveIcon />
            {saving ? "保存中" : "保存线索"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
