"use client"

import { FileTextIcon, LayoutGridIcon } from "lucide-react"

import { useI18n } from "@/i18n/provider"
import type { Category } from "@/lib/api/support-community"
import { cn } from "@/lib/utils"

type CommunityCategoryNavProps = {
  categories: Category[]
  active: number | "all"
  loading?: boolean
  failed?: boolean
  onChange: (value: number | "all") => void
  onRetry?: () => void
  onSelected?: () => void
}

export function CommunityCategoryNav(props: CommunityCategoryNavProps) {
  const t = useI18n()
  return (
    <aside className="hidden min-w-0 self-start overflow-hidden rounded-md bg-card xl:sticky xl:top-[5.5rem] xl:block xl:max-h-[calc(100svh-5.625rem)]" aria-label={t("supportPublic.posts.categoryNavigation")}>
      <nav className="max-h-[calc(100svh-5.625rem)] overflow-y-auto">
        <div className="grid gap-0.5 px-3 py-4">
          <CommunityCategoryMenuContent {...props} />
        </div>
      </nav>
    </aside>
  )
}

export function CommunityCategoryMenuContent({ categories, active, loading = false, failed = false, onChange, onRetry, onSelected }: CommunityCategoryNavProps) {
  const t = useI18n()
  const choose = (value: number | "all") => {
    onChange(value)
    onSelected?.()
  }

  return (
    <>
      <button type="button" data-support-mobile-menu-close="true" data-category-id="all" onClick={() => choose("all")} className={categoryItemClassName(active === "all")} aria-pressed={active === "all"}>
        <LayoutGridIcon aria-hidden="true" className="size-4 shrink-0" />
        <span className="truncate">{t("supportPublic.common.allCategories")}</span>
      </button>
      {categories.map((category) => (
        <button key={category.id} type="button" data-support-mobile-menu-close="true" data-category-id={category.id} onClick={() => choose(category.id)} className={categoryItemClassName(active === category.id)} aria-pressed={active === category.id}>
          <FileTextIcon aria-hidden="true" className="size-4 shrink-0" />
          <span className="truncate">{category.name}</span>
        </button>
      ))}
      {loading ? <div className="px-2.5 py-2 text-xs text-muted-foreground">{t("supportPublic.loading.categories")}</div> : null}
      {failed ? <button type="button" className="mt-2 px-2.5 text-left text-sm text-destructive underline-offset-4 hover:underline" onClick={onRetry}>{t("supportPublic.posts.categoriesFailed")}</button> : null}
    </>
  )
}

function categoryItemClassName(active: boolean) {
  return cn(
    "relative flex min-w-0 items-center gap-2 rounded-md text-left text-sm transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
    active ? "bg-primary/10 font-medium text-primary" : "text-muted-foreground",
    "min-h-9 px-2.5 py-1.5"
  )
}
