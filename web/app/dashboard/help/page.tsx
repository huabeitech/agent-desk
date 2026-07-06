"use client"

import Link from "next/link"
import {
  AlertTriangleIcon,
  ArrowRightIcon,
  BadgeCheckIcon,
  BookOpenCheckIcon,
  BotMessageSquareIcon,
  ClipboardListIcon,
  FileTextIcon,
  MessageSquareWarningIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  SparklesIcon,
  UsersIcon,
} from "lucide-react"

import { DashboardPage, DashboardToolbar } from "@/components/dashboard-page"
import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const dailyCards = [
  {
    title: "老板每天看什么",
    icon: <BadgeCheckIcon className="size-4" />,
    href: "/dashboard",
    action: "打开经营总览",
    items: [
      "看今日咨询、留资、高意向、预约、成交和转人工数量。",
      "先处理逾期跟进、今日预约、售后/投诉和 AI 负反馈。",
      "复制或发送经营日报给门店群，确认当天负责人。",
    ],
  },
  {
    title: "顾问每天怎么跟",
    icon: <ClipboardListIcon className="size-4" />,
    href: "/dashboard/sales-leads",
    action: "打开销售线索",
    items: [
      "优先查看今日任务、逾期、高意向、预约和售后风险视图。",
      "根据自动标签、最近客户消息和跟进建议决定先联系谁。",
      "每次联系后记录跟进内容、下一步动作和下次跟进时间。",
    ],
  },
  {
    title: "运营每天怎么修知识",
    icon: <MessageSquareWarningIcon className="size-4" />,
    href: "/dashboard/knowledge",
    action: "打开知识库",
    items: [
      "从首页负反馈明细进入检索日志，筛选仅负反馈、无答案和兜底回复。",
      "把重复问题生成 FAQ 草稿，人工确认答案后启用并重建索引。",
      "补充产品、活动、售后、价格边界和禁用承诺的标准口径。",
    ],
  },
] as const

const weeklyCards = [
  {
    title: "周复盘",
    description: "每周看一次趋势，判断数字店长是否真的带来线索和预约。",
    checks: ["看经营趋势近 7 天", "检查热门产品和高频问题", "确认负反馈是否已沉淀 FAQ"],
  },
  {
    title: "话术优化",
    description: "用入口或话术版本对比，保留能带来高意向和预约的版本。",
    checks: ["对比 sourceChannel", "看高意向率和预约率", "停用无效或投诉高的话术"],
  },
  {
    title: "交付巡检",
    description: "每周确认模型、知识索引、通知和备份状态，减少上线后隐患。",
    checks: ["检查模型与检索健康", "发送关键通知测试", "确认最近备份和恢复 dry-run 命令"],
  },
] as const

const riskCards = [
  {
    title: "客户要求最低价、现货、保证效果",
    action: "不要承诺，按门店确认和官方政策回答，必要时转人工。",
  },
  {
    title: "客户表达投诉、退款、退货、差评",
    action: "优先安抚，确认订单和联系方式，生成售后工单并同步负责人。",
  },
  {
    title: "AI 回答被点踩或引用错误",
    action: "从检索日志生成 FAQ 草稿，人工确认后启用，避免错误继续发生。",
  },
  {
    title: "没有顾问在线但客户要人工",
    action: "保持 AI 接待，沉淀未分配待跟进线索，下一班次优先处理。",
  },
] as const

const launchChecks = [
  "交付报告没有阻断项，模型、Embedding、向量库、产品和活动索引都通过。",
  "客户聊天入口、网站嵌入代码、品牌标题和主题色与交付报告一致。",
  "外部通知已完成高意向、预约、转人工、未分配、售后风险测试。",
  "至少跑完一轮行业验收矩阵，并保存交付记录和 PDF 报告。",
]

export default function DashboardHelpPage() {
  return (
    <DashboardPage>
      <DashboardToolbar
        actions={
          <div className="flex flex-wrap gap-2">
            <Link
              href="/dashboard/store-setup"
              className={buttonVariants({ variant: "outline" })}
            >
              <SparklesIcon />
              交付初始化
            </Link>
            <Link href="/dashboard" className={buttonVariants({ variant: "default" })}>
              <BotMessageSquareIcon />
              今日经营
            </Link>
          </div>
        }
      >
        <div>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <BookOpenCheckIcon className="size-4" />
            AI 数字店长运营手册
          </div>
          <h1 className="mt-1 text-2xl font-semibold tracking-normal">上线后每天怎么用</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            这页给老板、运营和门店顾问做日常 SOP。目标是让商家不用问开发，也能完成看报表、跟线索、修知识、做复盘和处理风险。
          </p>
        </div>
      </DashboardToolbar>

      <section className="grid gap-3 lg:grid-cols-3">
        {dailyCards.map((card) => (
          <div key={card.title} className="rounded-md border border-border/70 bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-2 font-medium">
                {card.icon}
                {card.title}
              </div>
              <Link href={card.href} className={buttonVariants({ variant: "ghost", size: "sm" })}>
                {card.action}
                <ArrowRightIcon />
              </Link>
            </div>
            <div className="mt-3 space-y-2">
              {card.items.map((item) => (
                <div key={item} className="flex gap-2 text-sm leading-6 text-muted-foreground">
                  <span className="mt-2 size-1.5 shrink-0 rounded-full bg-primary" />
                  <span>{item}</span>
                </div>
              ))}
            </div>
          </div>
        ))}
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
        <div className="rounded-md border border-border/70 bg-card p-4">
          <div className="flex items-center gap-2 font-medium">
            <RefreshCwIcon className="size-4" />
            每周运营节奏
          </div>
          <div className="mt-3 grid gap-3 md:grid-cols-3">
            {weeklyCards.map((card) => (
              <div key={card.title} className="rounded-md border border-border/70 bg-background p-3">
                <div className="text-sm font-medium">{card.title}</div>
                <p className="mt-2 text-sm leading-6 text-muted-foreground">{card.description}</p>
                <div className="mt-3 space-y-1.5">
                  {card.checks.map((item) => (
                    <div key={item} className="flex items-center gap-2 text-xs text-muted-foreground">
                      <BadgeCheckIcon className="size-3.5 text-primary" />
                      {item}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-md border border-border/70 bg-card p-4">
          <div className="flex items-center gap-2 font-medium">
            <ShieldCheckIcon className="size-4" />
            上线前必须确认
          </div>
          <div className="mt-3 space-y-2">
            {launchChecks.map((item, index) => (
              <div key={item} className="flex gap-3 rounded-md bg-muted/40 px-3 py-2 text-sm">
                <Badge variant="secondary" className="h-6 min-w-6 justify-center">
                  {index + 1}
                </Badge>
                <span className="leading-6">{item}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="rounded-md border border-border/70 bg-card p-4">
        <div className="flex items-center gap-2 font-medium">
          <AlertTriangleIcon className="size-4" />
          风险处理口径
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-2">
          {riskCards.map((card) => (
            <div key={card.title} className="rounded-md border border-border/70 bg-background p-3">
              <div className="text-sm font-medium">{card.title}</div>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{card.action}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-md border border-border/70 bg-card p-4">
        <div className="flex items-center gap-2 font-medium">
          <UsersIcon className="size-4" />
          角色分工
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-3">
          <div className="rounded-md bg-muted/40 p-3 text-sm leading-6">
            <div className="font-medium">老板</div>
            <p className="mt-1 text-muted-foreground">每天看经营日报和趋势复盘，决定重点产品、活动和顾问跟进优先级。</p>
          </div>
          <div className="rounded-md bg-muted/40 p-3 text-sm leading-6">
            <div className="font-medium">运营</div>
            <p className="mt-1 text-muted-foreground">维护产品、活动、FAQ、风险口径和模板，处理 AI 负反馈并发布知识。</p>
          </div>
          <div className="rounded-md bg-muted/40 p-3 text-sm leading-6">
            <div className="font-medium">顾问</div>
            <p className="mt-1 text-muted-foreground">处理会话和销售线索，记录跟进结果，补齐联系方式、预算、预约和成交状态。</p>
          </div>
        </div>
      </section>

      <section className="rounded-md border border-border/70 bg-card p-4">
        <div className="flex items-center gap-2 font-medium">
          <FileTextIcon className="size-4" />
          常用入口
        </div>
        <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {[
            ["经营总览", "/dashboard"],
            ["销售线索", "/dashboard/sales-leads"],
            ["会话工作台", "/dashboard/conversations"],
            ["知识库", "/dashboard/knowledge"],
            ["产品库", "/dashboard/products"],
            ["活动库", "/dashboard/promotions"],
            ["交付初始化", "/dashboard/store-setup"],
            ["店长配置", "/dashboard/digital-store"],
          ].map(([label, href]) => (
            <Link
              key={href}
              href={href}
              className={cn(buttonVariants({ variant: "outline" }), "justify-between")}
            >
              {label}
              <ArrowRightIcon />
            </Link>
          ))}
        </div>
      </section>
    </DashboardPage>
  )
}
