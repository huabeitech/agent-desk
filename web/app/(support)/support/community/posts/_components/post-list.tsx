"use client"

import { useState } from "react"
import { useSearchParams } from "next/navigation"

import { CommunityFrame } from "@/app/(support)/support/_components/community-frame"
import { CommunityPostList } from "@/app/(support)/support/_components/community-post-list"
import { SupportEmptyState as EmptyState } from "@/app/(support)/support/_components/support-ui"
import { useCommunityCategoryRoute } from "@/app/(support)/support/_components/support-community-route"
import { PostListLoading } from "@/app/(support)/support/community/posts/_components/post-ui"
import { useI18n } from "@/i18n/provider"
import { cn } from "@/lib/utils"

export function PostList() {
  const t = useI18n()
  const searchParams = useSearchParams()
  const categoryRoute = useCommunityCategoryRoute()
  const title = searchParams.get("title") || ""
  const [status, setStatus] = useState(searchParams.get("status") || "all")

  const resetKey = [
    categoryRoute.activeCategoryId,
    categoryRoute.categorySlug,
    status,
    title,
  ].join(":")
  const query = {
    categoryId: categoryRoute.activeCategoryId === "all" ? undefined : categoryRoute.activeCategoryId,
    status: status === "all" ? undefined : status,
    title,
  }
  const statusOptions = [
    { value: "all", label: t("supportPublic.status.all") },
    { value: "normal", label: t("supportPublic.status.normal") },
    { value: "resolved", label: t("supportPublic.status.resolved") },
  ]
  const categoryUnavailable = Boolean(categoryRoute.categorySlug && !categoryRoute.categoriesLoading && !categoryRoute.activeCategory)

  return (
    <CommunityFrame categoryRoute={categoryRoute}>
      <div className="flex justify-between border-b border-border px-4 py-3">
        <div className="text-base font-bold">{t("supportPublic.posts.title")}</div>
        <div className="inline-flex flex-wrap items-center gap-1 rounded-lg bg-muted p-1">
            {statusOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                className={cn(
                  "inline-flex h-5 items-center rounded-md px-3 text-sm font-medium whitespace-nowrap transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  status === option.value
                    ? "bg-background text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
                aria-pressed={status === option.value}
                onClick={() => setStatus(option.value)}
              >
                {option.label}
              </button>
            ))}
        </div>
      </div>
      <div>
        {categoryRoute.categoriesLoading && categoryRoute.categorySlug ? <PostListLoading /> : null}
        {categoryUnavailable ? <EmptyState text={t("supportPublic.empty.noPostsMatched")} /> : null}
        {!categoryRoute.categoriesLoading && !categoryUnavailable ? (
          <CommunityPostList
            resetKey={resetKey}
            emptyText={t("supportPublic.empty.noPostsMatched")}
            query={query}
          />
        ) : null}
      </div>
    </CommunityFrame>
  )
}
