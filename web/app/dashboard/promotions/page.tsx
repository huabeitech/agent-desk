"use client"

import { useRef, useState } from "react"
import {
  BanIcon,
  CheckCircle2Icon,
  DownloadIcon,
  FileUpIcon,
  RefreshCwIcon,
  SparklesIcon,
} from "lucide-react"
import { toast } from "sonner"

import {
  createDashboardStatusColumn,
  createDashboardStatusToggleAction,
  DashboardCrudPage,
  type DashboardCrudActionState,
  type DashboardCrudColumn,
} from "@/components/dashboard/crud"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  createPromotion,
  deletePromotion,
  fetchPromotion,
  fetchPromotions,
  importPromotions,
  reindexPromotion,
  seedMusePromotions,
  updatePromotion,
  updatePromotionStatus,
  type AdminPromotion,
  type PromotionImportResult,
  type SavePromotionPayload,
} from "@/lib/api/promotion"
import { Status, StatusLabels } from "@/lib/generated/enums"

function escapeCSVCell(value: string | number) {
  return `"${String(value).replaceAll('"', '""')}"`
}

function downloadCSVFile(filename: string, rows: Array<Array<string | number>>) {
  const csv = rows.map((row) => row.map(escapeCSVCell).join(",")).join("\n")
  const blob = new Blob([`\ufeff${csv}`], { type: "text/csv;charset=utf-8" })
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function statusLabel(status: Status) {
  if (status === Status.Disabled) return "禁用"
  if (status === Status.Deleted) return "已删除"
  return "启用"
}

function dateRangeText(item: AdminPromotion) {
  if (item.startAt && item.endAt) return `${item.startAt.slice(0, 10)} - ${item.endAt.slice(0, 10)}`
  if (item.startAt) return `${item.startAt.slice(0, 10)} 起`
  if (item.endAt) return `${item.endAt.slice(0, 10)} 前`
  return "长期有效"
}

function payloadFromValues(values: Record<string, string | number | boolean | string[] | number[]>) {
  return {
    name: String(values.name ?? ""),
    promotionType: String(values.promotionType ?? ""),
    description: String(values.description ?? ""),
    applicableProducts: String(values.applicableProducts ?? ""),
    startAt: String(values.startAt ?? ""),
    endAt: String(values.endAt ?? ""),
    discountRule: String(values.discountRule ?? ""),
    storeBenefit: String(values.storeBenefit ?? ""),
    appointmentBenefit: String(values.appointmentBenefit ?? ""),
    scriptSuggestion: String(values.scriptSuggestion ?? ""),
    priority: Number(values.priority ?? 0),
    knowledgeBaseId: Number(values.knowledgeBaseId ?? 0),
    status: Number(values.status ?? Status.Ok),
    remark: String(values.remark ?? ""),
  }
}

export default function DashboardPromotionsPage() {
  const actionStateRef = useRef<DashboardCrudActionState | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const [importing, setImporting] = useState(false)
  const [lastImportResult, setLastImportResult] = useState<PromotionImportResult | null>(null)

  function handleDownloadTemplate() {
    const rows = [
      [
        "活动名称",
        "活动类型",
        "活动描述",
        "适用产品",
        "开始时间",
        "结束时间",
        "优惠规则",
        "到店权益",
        "预约权益",
        "话术建议",
        "推荐优先级",
        "知识库ID",
        "状态",
        "备注",
      ],
      [
        "周末预约试躺礼",
        "预约权益",
        "提前预约周末试躺，可安排睡眠顾问预留体验时段",
        "慕斯脊护支撑款、慕斯云感舒睡款",
        "2026-07-01",
        "2026-07-31",
        "具体成交价和叠加优惠以门店顾问确认为准",
        "到店可享免费睡眠咨询和软硬度试躺对比",
        "提前预约并留下手机号，可领取护睡礼包",
        "客户提到周末、到店、试躺时，优先邀请预约并留资",
        "90",
        "",
        "启用",
        "样例行，可删除后导入",
      ],
    ]
    downloadCSVFile("promotion-import-template.csv", rows)
  }

  function handleDownloadImportErrors() {
    const errors = lastImportResult?.errors ?? []
    if (errors.length === 0) return
    downloadCSVFile("promotion-import-errors.csv", [
      ["行号", "错误原因"],
      ...errors.map((item) => [item.row, item.message]),
    ])
  }

  async function handleImportFile(file: File, reload: () => Promise<void>) {
    if (!file || importing) return
    if (!file.name.toLowerCase().endsWith(".csv")) {
      toast.error("只支持 CSV 文件")
      return
    }
    setImporting(true)
    try {
      const result = await importPromotions(file)
      setLastImportResult(result)
      toast.success(
        `导入完成：新增 ${result.created}，更新 ${result.updated}，失败 ${result.failed}`
      )
      if (result.errors.length > 0) {
        toast.warning(
          result.errors
            .slice(0, 3)
            .map((item) => `第 ${item.row} 行：${item.message}`)
            .join("；")
        )
      }
      await reload()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "导入活动失败")
    } finally {
      setImporting(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
    }
  }

  const columns: DashboardCrudColumn<AdminPromotion>[] = [
    {
      key: "name",
      label: "活动",
      render: (item) => (
        <div className="min-w-48">
          <div className="font-medium">{item.name}</div>
          <div className="text-xs text-muted-foreground">FAQ #{item.knowledgeFAQId || "-"}</div>
        </div>
      ),
    },
    {
      key: "promotionType",
      label: "类型",
      className: "w-28",
      render: (item) => <Badge variant="outline">{item.promotionType || "促销活动"}</Badge>,
    },
    {
      key: "dateRange",
      label: "有效期",
      className: "w-44",
      render: (item) => <span className="text-muted-foreground">{dateRangeText(item)}</span>,
    },
    {
      key: "benefit",
      label: "权益",
      render: (item) => (
        <div className="line-clamp-2 max-w-[360px] text-muted-foreground">
          {item.appointmentBenefit || item.storeBenefit || item.discountRule || "-"}
        </div>
      ),
    },
    {
      key: "products",
      label: "适用产品",
      render: (item) => (
        <div className="line-clamp-2 max-w-[260px] text-muted-foreground">
          {item.applicableProducts || "全部产品"}
        </div>
      ),
    },
    {
      key: "priority",
      label: "优先级",
      className: "w-20",
      render: (item) => item.priority,
    },
    createDashboardStatusColumn<AdminPromotion, Status>({
      label: "状态",
      className: "w-24",
      getStatus: (item) => item.status as Status,
      getLabel: statusLabel,
      getBadgeVariant: (status) => (status === Status.Ok ? "default" : "secondary"),
    }),
  ]

  async function handleSeedMuse() {
    try {
      await seedMusePromotions()
      toast.success("已导入慕斯活动样板")
      await actionStateRef.current?.onRefresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "导入样板失败")
    }
  }

  return (
    <div className="flex flex-1 flex-col">
      <div className="flex justify-end border-b border-border/70 px-4 py-3 lg:px-5">
        <Button variant="outline" onClick={handleSeedMuse}>
          <SparklesIcon />
          导入慕斯样板
        </Button>
      </div>
      <DashboardCrudPage<AdminPromotion, SavePromotionPayload>
        filters={[
          {
            name: "keyword",
            label: "关键词",
            placeholder: "活动、权益、产品、话术",
            defaultValue: "",
            trim: true,
            className: "w-full sm:w-72",
          },
          {
            name: "promotionType",
            label: "类型",
            placeholder: "预约权益、组合权益",
            defaultValue: "",
            trim: true,
            className: "w-full sm:w-44",
          },
          {
            name: "status",
            label: "状态",
            type: "select",
            defaultValue: "all",
            allValue: "all",
            valueType: "number",
            options: [
              { value: "all", label: "全部状态" },
              { value: String(Status.Ok), label: "启用" },
              { value: String(Status.Disabled), label: "禁用" },
            ],
            className: "w-full sm:w-36",
          },
        ]}
        columns={columns}
        fetchList={fetchPromotions}
        getItemId={(item) => item.id}
        createItem={createPromotion}
        updateItem={(item, payload) => updatePromotion({ id: item.id, ...payload })}
        deleteItem={(item) => deletePromotion(item.id)}
        canDelete={(item) => item.status !== Status.Deleted}
        onActionStateChange={(state) => {
          actionStateRef.current = state
        }}
        form={{
          fetchDetail: fetchPromotion,
          fields: [
            { name: "name", label: "活动名称", placeholder: "周末预约试躺礼", required: true, requiredMessage: "活动名称不能为空", trim: true },
            { name: "promotionType", label: "活动类型", placeholder: "预约权益", trim: true },
            { name: "startAt", label: "开始日期", placeholder: "2026-07-01", trim: true },
            { name: "endAt", label: "结束日期", placeholder: "2026-07-31", trim: true },
            { name: "applicableProducts", label: "适用产品", type: "textarea", rows: 3, trim: true },
            { name: "description", label: "活动说明", type: "textarea", rows: 3, trim: true },
            { name: "discountRule", label: "优惠规则", type: "textarea", rows: 3, trim: true },
            { name: "storeBenefit", label: "到店权益", type: "textarea", rows: 3, trim: true },
            { name: "appointmentBenefit", label: "预约权益", type: "textarea", rows: 3, trim: true },
            { name: "scriptSuggestion", label: "话术建议", type: "textarea", rows: 3, trim: true },
            { name: "priority", label: "推荐优先级", type: "number", valueType: "number", defaultValue: 0 },
            { name: "knowledgeBaseId", label: "FAQ知识库ID", type: "number", valueType: "number", min: 0, defaultValue: 0, description: "留空或填 0 时自动使用第一个 FAQ 知识库" },
            {
              name: "status",
              label: "状态",
              type: "select",
              valueType: "number",
              defaultValue: Status.Ok,
              options: [
                { value: String(Status.Ok), label: StatusLabels[Status.Ok] ?? "启用" },
                { value: String(Status.Disabled), label: StatusLabels[Status.Disabled] ?? "禁用" },
              ],
            },
            { name: "remark", label: "备注", type: "textarea", rows: 3, trim: true },
          ],
          transformSubmitValues: payloadFromValues,
          labels: {
            createTitle: "新建活动",
            editTitle: "编辑活动",
            create: "创建",
            save: "保存",
            saving: "保存中",
            cancel: "取消",
            loadingDetail: "加载活动详情",
            required: "必填项不能为空",
            invalidNumber: "请输入有效数字",
            minValue: (min) => `不能小于 ${min}`,
            maxValue: (max) => `不能大于 ${max}`,
          },
        }}
        rowActions={[
          {
            key: "reindex",
            icon: <RefreshCwIcon />,
            label: "重建索引",
            run: async ({ item, reload }) => {
              await reindexPromotion(item.id)
              toast.success(`已重建 ${item.name} 的活动索引`)
              await reload()
            },
          },
          createDashboardStatusToggleAction<AdminPromotion, Status>({
            disabled: (item) => item.status === Status.Deleted,
            icon: (item) => (item.status === Status.Ok ? <BanIcon /> : <CheckCircle2Icon />),
            label: (item) => (item.status === Status.Ok ? "禁用" : "启用"),
            getNextStatus: (item) => (item.status === Status.Ok ? Status.Disabled : Status.Ok),
            updateStatus: (item, nextStatus) => updatePromotionStatus(item.id, nextStatus),
            successMessage: (item, nextStatus) => `${item.name} 已${nextStatus === Status.Ok ? "启用" : "禁用"}`,
            errorMessage: "更新活动状态失败",
          }),
        ]}
        renderToolbarActions={({ reload, loading }) => (
          <>
            <Button variant="outline" onClick={handleDownloadTemplate} disabled={loading || importing}>
              <DownloadIcon />
              模板
            </Button>
            {lastImportResult?.errors?.length ? (
              <Button
                variant="outline"
                onClick={handleDownloadImportErrors}
                disabled={loading || importing}
              >
                <DownloadIcon />
                错误明细
              </Button>
            ) : null}
            <Button
              variant="outline"
              onClick={() => fileInputRef.current?.click()}
              disabled={loading || importing}
            >
              <FileUpIcon className={importing ? "animate-pulse" : undefined} />
              {importing ? "导入中" : "导入CSV"}
            </Button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,text/csv"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0]
                if (file) {
                  void handleImportFile(file, reload)
                }
              }}
            />
          </>
        )}
        labels={{
          refresh: "刷新",
          create: "新建活动",
          query: "查询",
          loading: "加载活动中",
          empty: "暂无活动",
          actions: "操作",
          edit: "编辑",
          delete: "删除",
          processing: "处理中",
          moreActions: (item) => `${item.name} 更多操作`,
          loadFailed: "加载活动失败",
          saveFailed: "保存活动失败",
          deleteFailed: "删除活动失败",
          created: (payload) => `已创建 ${payload.name}`,
          updated: (item) => `已更新 ${item.name}`,
          deleted: (item) => `已删除 ${item.name}`,
        }}
      />
    </div>
  )
}
