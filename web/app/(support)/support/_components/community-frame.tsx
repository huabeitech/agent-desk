"use client"

import { type ReactNode } from "react"

import { CommunityCategoryMenuContent, CommunityCategoryNav } from "@/app/(support)/support/_components/community-category-nav"
import { SupportPageLayout } from "@/app/(support)/support/_components/support-page-layout"
import { useCommunityCategoryRoute } from "@/app/(support)/support/_components/support-community-route"
import { useI18n } from "@/i18n/provider"

export function CommunityFrame({
  active,
  categoryRoute,
  children,
  toc,
}: {
  active?: number | "all"
  categoryRoute: ReturnType<typeof useCommunityCategoryRoute>
  children: ReactNode
  toc?: ReactNode
}) {
  const t = useI18n()
  const categoryNavigation = (
    <CommunityCategoryMenuContent
      categories={categoryRoute.categories}
      active={active ?? categoryRoute.activeCategoryId}
      loading={categoryRoute.categoriesLoading}
      failed={categoryRoute.categoriesFailed}
      onChange={categoryRoute.changeCategory}
      onRetry={categoryRoute.loadCategories}
    />
  )

  return (
    <SupportPageLayout
      section="community"
      startAsideBreakpoint="xl"
      startAsideWidth="16rem"
      contentClassName="py-6 sm:py-8"
      startAside={(
        <CommunityCategoryNav
          categories={categoryRoute.categories}
          active={active ?? categoryRoute.activeCategoryId}
          loading={categoryRoute.categoriesLoading}
          failed={categoryRoute.categoriesFailed}
          onChange={categoryRoute.changeCategory}
          onRetry={categoryRoute.loadCategories}
        />
      )}
      endAside={toc}
      endAsideClassName="rounded-md bg-card"
      mobileNavigation={{
        title: t("supportPublic.posts.categoryNavigation"),
        content: <div className="grid gap-0.5">{categoryNavigation}</div>,
      }}
    >
      <section className="min-w-0 rounded-md bg-card">
        {children}
      </section>
    </SupportPageLayout>
  )
}
