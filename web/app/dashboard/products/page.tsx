"use client"

import { useRef, useState } from "react"
import {
  BanIcon,
  CheckCircle2Icon,
  DownloadIcon,
  FileUpIcon,
  RefreshCwIcon,
} from "lucide-react"
import { toast } from "sonner"

import {
  createDashboardStatusColumn,
  createDashboardStatusToggleAction,
  DashboardCrudPage,
  type DashboardCrudColumn,
} from "@/components/dashboard/crud"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  createProduct,
  deleteProduct,
  fetchProduct,
  fetchProducts,
  importProducts,
  reindexProduct,
  updateProduct,
  updateProductStatus,
  type AdminProduct,
  type ProductImportResult,
  type SaveProductPayload,
} from "@/lib/api/product"
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

function priceText(item: AdminProduct) {
  if (item.priceMin > 0 && item.priceMax > 0) {
    return `¥${item.priceMin.toLocaleString()} - ¥${item.priceMax.toLocaleString()}`
  }
  if (item.priceMin > 0) {
    return `¥${item.priceMin.toLocaleString()}起`
  }
  if (item.priceMax > 0) {
    return `¥${item.priceMax.toLocaleString()}左右`
  }
  return "-"
}

function statusLabel(status: Status) {
  if (status === Status.Disabled) return "禁用"
  if (status === Status.Deleted) return "已删除"
  return "启用"
}

function payloadFromValues(values: Record<string, string | number | boolean | string[] | number[]>) {
  return {
    name: String(values.name ?? ""),
    category: String(values.category ?? ""),
    priceMin: Number(values.priceMin ?? 0),
    priceMax: Number(values.priceMax ?? 0),
    sellingPoints: String(values.sellingPoints ?? ""),
    suitablePeople: String(values.suitablePeople ?? ""),
    unsuitablePeople: String(values.unsuitablePeople ?? ""),
    scenarios: String(values.scenarios ?? ""),
    specs: String(values.specs ?? ""),
    industryAttributes: String(values.industryAttributes ?? ""),
    imageUrl: String(values.imageUrl ?? ""),
    priority: Number(values.priority ?? 0),
    knowledgeBaseId: Number(values.knowledgeBaseId ?? 0),
    status: Number(values.status ?? Status.Ok),
    remark: String(values.remark ?? ""),
  }
}

export default function DashboardProductsPage() {
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const [importing, setImporting] = useState(false)
  const [lastImportResult, setLastImportResult] = useState<ProductImportResult | null>(null)

  function handleDownloadTemplate() {
    const rows = [
      [
        "产品名称",
        "品类",
        "最低价",
        "最高价",
        "核心卖点",
        "适合人群",
        "不适合人群",
        "使用场景",
        "规格参数",
        "行业属性",
        "图片链接",
        "推荐优先级",
        "知识库ID",
        "状态",
        "备注",
      ],
      [
        "慕斯脊护支撑款",
        "床垫",
        "12000",
        "18000",
        "分区承托、偏硬支撑、贴合腰背",
        "老人、腰背压力明显、喜欢支撑感的人群",
        "喜欢特别软睡感的人群",
        "老人房、腰背支撑、试躺预约",
        "1.5m/1.8m，可到店确认规格",
        "睡感：偏硬；支撑：分区承托；到店体验：建议试躺确认",
        "",
        "90",
        "",
        "启用",
        "样例行，可删除后导入",
      ],
    ]
    downloadCSVFile("product-import-template.csv", rows)
  }

  function handleDownloadImportErrors() {
    const errors = lastImportResult?.errors ?? []
    if (errors.length === 0) return
    downloadCSVFile("product-import-errors.csv", [
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
      const result = await importProducts(file)
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
      toast.error(error instanceof Error ? error.message : "导入产品失败")
    } finally {
      setImporting(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
    }
  }

  const columns: DashboardCrudColumn<AdminProduct>[] = [
    {
      key: "name",
      label: "产品",
      render: (item) => (
        <div className="min-w-44">
          <div className="font-medium">{item.name}</div>
          <div className="text-xs text-muted-foreground">
            FAQ #{item.knowledgeFAQId || "-"}
          </div>
        </div>
      ),
    },
    {
      key: "category",
      label: "品类",
      className: "w-28",
      render: (item) => <Badge variant="outline">{item.category || "未分类"}</Badge>,
    },
    {
      key: "price",
      label: "价格",
      className: "w-40",
      render: (item) => <span className="text-muted-foreground">{priceText(item)}</span>,
    },
    {
      key: "sellingPoints",
      label: "卖点",
      render: (item) => (
        <div className="line-clamp-2 max-w-[320px] text-muted-foreground">
          {item.sellingPoints || "-"}
        </div>
      ),
    },
    {
      key: "suitablePeople",
      label: "适合人群",
      render: (item) => (
        <div className="line-clamp-2 max-w-[300px] text-muted-foreground">
          {item.suitablePeople || "-"}
        </div>
      ),
    },
    {
      key: "industryAttributes",
      label: "行业属性",
      render: (item) => (
        <div className="line-clamp-2 max-w-[260px] text-muted-foreground">
          {item.industryAttributes || "-"}
        </div>
      ),
    },
    {
      key: "priority",
      label: "优先级",
      className: "w-20",
      render: (item) => item.priority,
    },
    createDashboardStatusColumn<AdminProduct, Status>({
      label: "状态",
      className: "w-24",
      getStatus: (item) => item.status as Status,
      getLabel: statusLabel,
      getBadgeVariant: (status) => (status === Status.Ok ? "default" : "secondary"),
    }),
  ]

  return (
    <DashboardCrudPage<AdminProduct, SaveProductPayload>
      filters={[
        {
          name: "keyword",
          label: "关键词",
          placeholder: "产品、卖点、人群、场景",
          defaultValue: "",
          trim: true,
          className: "w-full sm:w-72",
        },
        {
          name: "category",
          label: "品类",
          placeholder: "床垫、床架、电动床",
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
      fetchList={fetchProducts}
      getItemId={(item) => item.id}
      createItem={createProduct}
      updateItem={(item, payload) => updateProduct({ id: item.id, ...payload })}
      deleteItem={(item) => deleteProduct(item.id)}
      canDelete={(item) => item.status !== Status.Deleted}
      form={{
        fetchDetail: fetchProduct,
        fields: [
          {
            name: "name",
            label: "产品名称",
            placeholder: "例如：慕斯脊护支撑款",
            required: true,
            requiredMessage: "产品名称不能为空",
            trim: true,
          },
          {
            name: "category",
            label: "品类",
            placeholder: "床垫",
            trim: true,
          },
          {
            name: "priceMin",
            label: "最低价",
            type: "number",
            valueType: "number",
            min: 0,
            defaultValue: 0,
          },
          {
            name: "priceMax",
            label: "最高价",
            type: "number",
            valueType: "number",
            min: 0,
            defaultValue: 0,
          },
          {
            name: "sellingPoints",
            label: "核心卖点",
            type: "textarea",
            rows: 3,
            trim: true,
          },
          {
            name: "suitablePeople",
            label: "适合人群",
            type: "textarea",
            rows: 3,
            trim: true,
          },
          {
            name: "unsuitablePeople",
            label: "不适合人群",
            type: "textarea",
            rows: 3,
            trim: true,
          },
          {
            name: "scenarios",
            label: "使用场景",
            type: "textarea",
            rows: 3,
            trim: true,
          },
          {
            name: "specs",
            label: "规格参数",
            type: "textarea",
            rows: 3,
            trim: true,
          },
          {
            name: "industryAttributes",
            label: "行业属性",
            type: "textarea",
            rows: 3,
            trim: true,
            placeholder: "例如：课时/班型、诊疗项目、装修面积、车型配置、睡感/尺寸等",
          },
          {
            name: "priority",
            label: "推荐优先级",
            type: "number",
            valueType: "number",
            defaultValue: 0,
          },
          {
            name: "knowledgeBaseId",
            label: "FAQ知识库ID",
            type: "number",
            valueType: "number",
            min: 0,
            defaultValue: 0,
            description: "留空或填 0 时自动使用第一个 FAQ 知识库",
          },
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
          {
            name: "imageUrl",
            label: "图片链接",
            trim: true,
          },
          {
            name: "remark",
            label: "备注",
            type: "textarea",
            rows: 3,
            trim: true,
          },
        ],
        transformSubmitValues: payloadFromValues,
        labels: {
          createTitle: "新建产品",
          editTitle: "编辑产品",
          create: "创建",
          save: "保存",
          saving: "保存中",
          cancel: "取消",
          loadingDetail: "加载产品详情",
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
            await reindexProduct(item.id)
            toast.success(`已重建 ${item.name} 的推荐索引`)
            await reload()
          },
        },
        createDashboardStatusToggleAction<AdminProduct, Status>({
          disabled: (item) => item.status === Status.Deleted,
          icon: (item) => (item.status === Status.Ok ? <BanIcon /> : <CheckCircle2Icon />),
          label: (item) => (item.status === Status.Ok ? "禁用" : "启用"),
          getNextStatus: (item) => (item.status === Status.Ok ? Status.Disabled : Status.Ok),
          updateStatus: (item, nextStatus) => updateProductStatus(item.id, nextStatus),
          successMessage: (item, nextStatus) =>
            `${item.name} 已${nextStatus === Status.Ok ? "启用" : "禁用"}`,
          errorMessage: "更新产品状态失败",
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
        create: "新建产品",
        query: "查询",
        loading: "加载产品中",
        empty: "暂无产品",
        actions: "操作",
        edit: "编辑",
        delete: "删除",
        processing: "处理中",
        moreActions: (item) => `${item.name} 更多操作`,
        loadFailed: "加载产品失败",
        saveFailed: "保存产品失败",
        deleteFailed: "删除产品失败",
        created: (payload) => `已创建 ${payload.name}`,
        updated: (item) => `已更新 ${item.name}`,
        deleted: (item) => `已删除 ${item.name}`,
      }}
    />
  )
}
