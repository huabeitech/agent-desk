"use client"

import { type ReactNode } from "react"
import Link from "next/link"
import { EyeIcon, MessageCircle, ThumbsUpIcon } from "lucide-react"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { useI18n } from "@/i18n/provider"
import { postHref, type PostListItem } from "@/lib/api/support-community"
import { cn, formatDateTime } from "@/lib/utils"

export function PostCard({ item }: { item: PostListItem }) {
  const t = useI18n()
  const authorName = item.user.displayName || t("supportPublic.common.user")
  const avatarText = authorName.trim().slice(0, 1).toUpperCase()

  return (
    <li className="px-4 py-3">
      <div className="flex min-w-0 items-center gap-2">
        <Avatar className="size-6 shrink-0">
          <AvatarImage src={item.user.avatar} alt={authorName} />
          <AvatarFallback className="bg-muted text-[11px] font-medium text-muted-foreground">{avatarText}</AvatarFallback>
        </Avatar>
        <div className="flex min-w-0 items-center gap-1 text-xs md:text-sm">
          <span className="max-w-32 truncate text-muted-foreground">{authorName}</span>
          <span className="text-muted-foreground">·</span>
          <span className="truncate text-muted-foreground">{formatDateTime(item.createdAt)}</span>
        </div>
      </div>

      <div className="mt-2 space-y-2">
        <h2 className="text-[15px] leading-6 font-semibold break-all text-foreground sm:text-base">
          <Link
            href={postHref(item.id)}
            className="transition-colors hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {item.acceptedCommentId > 0 ? (
              <span className="relative -top-px mr-1 inline-flex h-5 shrink-0 items-center align-middle whitespace-nowrap rounded-md bg-primary/10 px-2 text-[11px] leading-none font-medium text-primary ring-1 ring-primary/20">
                {t("supportPublic.comment.accepted")}
              </span>
            ) : null}
            {item.title}
          </Link>
        </h2>
        {item.summary ? (
          <Link
            href={postHref(item.id)}
            className="block overflow-hidden text-[15px] leading-6 break-all text-muted-foreground transition-colors hover:text-foreground/80 sm:text-sm sm:leading-normal"
            style={{ display: "-webkit-box", WebkitBoxOrient: "vertical", WebkitLineClamp: 3 }}
          >
            {item.summary}
          </Link>
        ) : null}
      </div>

      <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
        <span className="inline-flex max-w-full items-center rounded-full bg-accent px-2.5 py-1 text-xs text-muted-foreground">
          <span className="truncate">{item.categoryName || t("supportPublic.common.uncategorized")}</span>
        </span>
        <div className="ml-auto flex items-center gap-4 text-xs text-muted-foreground">
          <PostMetric icon={<MessageCircle className="size-4" />} value={item.commentCount} label={t("supportPublic.posts.comments")} />
          <PostMetric icon={<ThumbsUpIcon className="size-4" />} value={item.reactionCount} label={t("supportPublic.posts.likes")} />
          <PostMetric className="hidden sm:inline-flex" icon={<EyeIcon className="size-4" />} value={item.viewCount} label={t("supportPublic.posts.views")} />
        </div>
      </div>
    </li>
  )
}

export function PostMetric({ icon, value, label, className }: { icon: ReactNode; value: number; label: string; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex min-h-8 items-center gap-1.5 transition-colors",
        className
      )}
      title={label}
      aria-label={`${label}: ${value}`}
    >
      {icon}
      {value}
    </span>
  )
}

export function PostStatusPill({ status }: { status: string }) {
  const t = useI18n()
  if (status === "resolved") return <span className="inline-flex h-5 items-center rounded bg-emerald-50 px-1.5 text-[11px] font-medium text-emerald-700">{t("supportPublic.status.resolved")}</span>
  if (status === "closed") return <span className="inline-flex h-5 items-center rounded bg-muted px-1.5 text-[11px] font-medium text-muted-foreground">{t("supportPublic.status.closed")}</span>
  return <span className="inline-flex h-5 items-center rounded bg-amber-50 px-1.5 text-[11px] font-medium text-amber-700">{t("supportPublic.status.normal")}</span>
}

export function PostListLoading() {
  return (
    <div className="divide-y divide-border" aria-hidden="true">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="px-4 py-3">
          <div className="flex items-center gap-2">
            <div className="size-6 animate-pulse rounded-full bg-muted" />
            <div className="h-4 w-28 animate-pulse rounded bg-muted" />
          </div>
          <div className="mt-3 h-5 w-4/5 animate-pulse rounded bg-muted" />
          <div className="mt-2 h-4 w-2/3 animate-pulse rounded bg-muted" />
          <div className="mt-3 h-6 w-20 animate-pulse rounded-full bg-muted" />
        </div>
      ))}
    </div>
  )
}
