"use client"

import { Fragment, useCallback, useEffect, useMemo, useState, type ReactNode } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { ArrowRightIcon, ChevronDownIcon, ChevronRightIcon, FileTextIcon, FolderIcon, FolderOpenIcon, ThumbsDownIcon, ThumbsUpIcon } from "lucide-react"
import { toast } from "sonner"

import { useImageLightboxOptional } from "@/components/image-lightbox"
import { Button } from "@/components/ui/button"
import { Breadcrumb, BreadcrumbItem, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb"
import { SupportArticleContent } from "@/app/(support)/support/_components/support-article-content"
import { PublicArticleToc } from "@/app/(support)/support/_components/support-article-toc"
import { SupportHeader } from "@/app/(support)/support/_components/support-header"
import { SupportDocLink, type DocPageNavigationHandler, useSupportDocRoute } from "@/app/(support)/support/_components/support-help-navigation"
import { SupportSearchInput } from "@/app/(support)/support/_components/support-ui"
import { EmptyState } from "@/components/empty-state"
import { useSupportDocReader as useSupportDocReaderData } from "@/app/(support)/support/_components/use-support-help-reader"
import { useI18n } from "@/i18n/provider"
import { fetchDocPages, submitDocPageFeedback, type DocPage } from "@/lib/api/support"
import { articleHeadingId } from "@/lib/support-article"
import { cn, formatDateTime } from "@/lib/utils"

export function SupportDocsList() {
  return <SupportDocsReader />
}

export function DocPageDetail() {
  return <SupportDocsReader />
}

function SupportDocsReader() {
  const t = useI18n()
  const pathname = usePathname()
  const [query, setQuery] = useState("")
  const [searchResults, setSearchResults] = useState<DocPage[]>([])
  const [searchLoading, setSearchLoading] = useState(false)
  const { activePath, navigate, replace } = useSupportDocRoute(pathname)
  const { page, pages, expanded, setExpanded, navigationLoading, pageLoading, failed } = useSupportDocReaderData(activePath, replace)
  const navigateToDocPage = useCallback<DocPageNavigationHandler>((event, targetPage) => {
    navigate(event, targetPage)
  }, [navigate])

  useEffect(() => {
    if (!page) return
    document.title = `${page.title} · ${t("supportPublic.help.title")}`
  }, [page, t])

  useEffect(() => {
    const keyword = query.trim()
    if (!keyword) return
    const timer = window.setTimeout(() => {
      void fetchDocPages({ keyword, limit: 50 })
        .then((result) => setSearchResults(result.results))
        .catch(() => setSearchResults([]))
        .finally(() => setSearchLoading(false))
    }, 250)
    return () => window.clearTimeout(timer)
  }, [query])

  const visiblePages = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return pages
    const matched = new Set<number>()
    pages.forEach((item) => {
      if (`${item.title} ${item.summary} ${item.slug} ${(item.tags || []).join(" ")}`.toLowerCase().includes(keyword)) {
        matched.add(item.id)
        let parentId = item.parentId
        while (parentId) {
          matched.add(parentId)
          parentId = pages.find((candidate) => candidate.id === parentId)?.parentId ?? 0
        }
      }
    })
    return pages.filter((item) => matched.has(item.id))
  }, [pages, query])

  return (
    <SupportDocsFrame
      navigation={<DocNavigation pages={visiblePages} rootPages={visiblePages.filter((item) => !item.parentId)} searchResults={searchResults.map((item) => pages.find((candidate) => candidate.id === item.id) || item).filter((item) => Boolean(item.docPath))} title={query} expanded={expanded} selectedPageId={page?.id ?? 0} loading={navigationLoading || searchLoading} failed={failed} onTitleChange={(value) => { setQuery(value); setSearchResults([]); setSearchLoading(Boolean(value.trim())) }} onExpandedChange={setExpanded} onNavigate={navigateToDocPage} />}
      toc={<PublicArticleToc articleId="doc-page-detail-preview" content={page?.content ?? ""} contentType={page?.contentType} />}
    >
      <div aria-busy={pageLoading} className={cn(page && pageLoading && "opacity-60 transition-opacity")}>
        {page ? <DocArticle page={page} pages={pages} previewId="doc-page-detail-preview" onNavigate={navigateToDocPage} /> : <div className="grid min-h-[60svh] place-items-center"><EmptyState text={pageLoading ? t("supportPublic.loading.page") : failed ? t("supportPublic.empty.pageNotFound") : t("supportPublic.empty.noPages")} /></div>}
      </div>
    </SupportDocsFrame>
  )
}

function SupportDocsFrame({
  children,
  navigation,
  toc,
}: {
  children: ReactNode
  navigation: ReactNode
  toc: ReactNode
}) {
  const t = useI18n()
  return (
    <main className="min-h-svh bg-background text-foreground">
      <SupportHeader
        section="help"
        mobileNavigation={navigation ? {
          title: t("supportPublic.help.navigation"),
          content: navigation,
        } : undefined}
      />

      <div className="support-docs-grid mx-auto max-w-[var(--support-docs-max-width)]">
        {navigation ? <aside className="hidden border-r xl:sticky xl:top-14 xl:block xl:h-[calc(100svh-3.5rem)] xl:overflow-y-auto">{navigation}</aside> : null}
        <div className="min-w-0 px-5 py-9 sm:px-6 sm:py-12 md:px-8 lg:px-10 2xl:px-12">{children}</div>
        {toc ? <div className="hidden 2xl:block">{toc}</div> : null}
      </div>
    </main>
  )
}

function DocNavigation({
  pages,
  rootPages,
  searchResults = [],
  title,
  expanded,
  selectedPageId,
  loading,
  failed,
  onTitleChange,
  onExpandedChange,
  onNavigate,
}: {
  pages: DocPage[]
  rootPages: DocPage[]
  searchResults?: DocPage[]
  title: string
  expanded: Set<number>
  selectedPageId: number
  loading: boolean
  failed: boolean
  onTitleChange: (value: string) => void
  onExpandedChange: (value: Set<number>) => void
  onNavigate: DocPageNavigationHandler
}) {
  const t = useI18n()
  return (
    <div className="p-4">
      <SupportSearchInput value={title} onChange={onTitleChange} placeholder={t("supportPublic.help.searchPlaceholder")} compact />
      <div className="mt-4 grid gap-0.5">
        {title.trim() ? searchResults.map((page) => (
          <SupportDocLink key={page.id} page={page} onNavigate={onNavigate} className={cn("rounded-md px-2.5 py-2 text-sm transition-colors hover:bg-muted", selectedPageId === page.id && "bg-primary/10 text-primary")}>
            <span className="block line-clamp-2 font-medium leading-5">{page.title}</span>
            {page.summary ? <span className="mt-1 block line-clamp-2 text-xs leading-5 text-muted-foreground">{page.summary}</span> : null}
          </SupportDocLink>
        )) : rootPages.map((page) => (
          <PublicDocPageNode key={page.id} page={page} depth={0} pages={pages} expanded={expanded} selectedPageId={selectedPageId} onToggle={(id) => {
            const next = new Set(expanded)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            onExpandedChange(next)
          }} onNavigate={onNavigate} />
        ))}
        {loading ? <div className="px-2 py-8 text-center text-sm text-muted-foreground">{t("supportPublic.loading.navigation")}</div> : null}
        {!loading && (title.trim() ? !searchResults.length : !pages.length) ? <EmptyState text={failed ? t("supportPublic.empty.pagesFailed") : t("supportPublic.empty.noPagesMatched")} compact /> : null}
      </div>
    </div>
  )
}

function DocArticle({ page, pages, previewId, onNavigate }: { page: DocPage; pages: DocPage[]; previewId: string; onNavigate: DocPageNavigationHandler }) {
  const t = useI18n()
  const lightbox = useImageLightboxOptional()
  const [feedbackPending, setFeedbackPending] = useState(false)
  const breadcrumbs = docPageBreadcrumbs(pages, page)
  const currentIndex = pages.findIndex((item) => item.id === page.id)
  const previousPage = currentIndex > 0 ? pages[currentIndex - 1] : null
  const nextPage = currentIndex >= 0 ? pages[currentIndex + 1] : null
  const submitFeedback = async (helpful: boolean) => {
    if (feedbackPending) return
    setFeedbackPending(true)
    try {
      await submitDocPageFeedback(page.id, helpful)
      toast.success(t("supportPublic.toast.feedbackSaved"))
    } finally {
      setFeedbackPending(false)
    }
  }
  useEffect(() => {
    if (page.contentType !== "html") return
    const container = document.getElementById(previewId)
    if (!container) return
    const cleanup: Array<() => void> = []
    container.querySelectorAll<HTMLElement>("h2, h3").forEach((heading, index) => {
      heading.id = articleHeadingId(heading.textContent || "", index)
      heading.classList.add("scroll-mt-20")
    })
    container.querySelectorAll<HTMLPreElement>("pre").forEach((block) => {
      block.classList.add("group", "relative")
      const button = document.createElement("button")
      button.type = "button"
      button.className = "not-typeset absolute right-2 top-2 rounded-md border border-border bg-background/90 px-2 py-1 text-xs text-muted-foreground opacity-0 shadow-sm transition-opacity group-hover:opacity-100 focus:opacity-100"
      button.dataset.notTypeset = "true"
      button.textContent = t("supportPublic.help.copyCode")
      button.setAttribute("aria-label", t("supportPublic.help.copyCode"))
      const copy = () => void navigator.clipboard.writeText(block.querySelector("code")?.textContent || block.textContent || "").then(() => toast.success(t("supportPublic.toast.codeCopied")))
      button.addEventListener("click", copy)
      block.appendChild(button)
      cleanup.push(() => { button.removeEventListener("click", copy); button.remove() })
    })
    const articleImages = Array.from(container.querySelectorAll<HTMLImageElement>("img"))
    articleImages.forEach((image, imageIndex) => {
      if (!lightbox) return
      image.classList.add("cursor-zoom-in")
      const open = () => lightbox.openGallery(articleImages.map((item) => ({
        src: item.currentSrc || item.src,
        alt: item.alt,
      })), imageIndex)
      image.addEventListener("click", open)
      cleanup.push(() => image.removeEventListener("click", open))
    })
    container.querySelectorAll<HTMLTableElement>("table").forEach((table) => {
      if (table.parentElement?.classList.contains("typeset-scroll")) return
      const wrapper = document.createElement("div")
      wrapper.className = "typeset-scroll"
      table.before(wrapper)
      wrapper.appendChild(table)
      cleanup.push(() => {
        wrapper.before(table)
        wrapper.remove()
      })
    })
    return () => cleanup.forEach((dispose) => dispose())
  }, [lightbox, page.content, page.contentType, previewId, t])
  return (
    <article className="mx-auto max-w-[var(--support-article-width)]">
      <Breadcrumb>
        <BreadcrumbList className="gap-y-1">
          <BreadcrumbItem>
            <Link href="/support/docs" className="transition-colors hover:text-foreground">{t("supportPublic.help.title")}</Link>
          </BreadcrumbItem>
          {breadcrumbs.map((item) => (
            <Fragment key={item.id}>
              <BreadcrumbSeparator />
              <BreadcrumbItem className="min-w-0">
                {item.id === page.id ? (
                  <BreadcrumbPage className="truncate">{item.title}</BreadcrumbPage>
                ) : (
                  <SupportDocLink page={item} onNavigate={onNavigate} className="truncate transition-colors hover:text-foreground">
                    {item.title}
                  </SupportDocLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          ))}
        </BreadcrumbList>
      </Breadcrumb>
      <h1 className="mt-6 text-balance text-3xl font-bold tracking-tight sm:text-4xl">{page.title}</h1>
      <div className="my-3 text-xs text-muted-foreground">{t("supportPublic.help.updatedAt", { date: formatDateTime(page.publishedAt || page.updatedAt) })}</div>
      <SupportArticleContent id={previewId} content={page.content} contentType={page.contentType} />
      <ChildPageLinks pages={pages.filter((item) => item.parentId === page.id)} onNavigate={onNavigate} />
      <div className="mt-12 flex flex-col gap-4 border-t pt-6 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="text-sm font-medium">{t("supportPublic.help.feedbackTitle")}</div>
          <div className="mt-1 text-sm text-muted-foreground">{t("supportPublic.help.feedbackDescription")}</div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" disabled={feedbackPending} onClick={() => void submitFeedback(true)}><ThumbsUpIcon />{t("supportPublic.actions.helpful")}</Button>
          <Button variant="outline" disabled={feedbackPending} onClick={() => void submitFeedback(false)}><ThumbsDownIcon />{t("supportPublic.actions.notHelpful")}</Button>
        </div>
      </div>
      {(previousPage || nextPage) ? <nav className="mt-8 grid gap-3 sm:grid-cols-2">
        {previousPage ? <ArticlePager page={previousPage} direction="previous" onNavigate={onNavigate} /> : <span />}
        {nextPage ? <ArticlePager page={nextPage} direction="next" onNavigate={onNavigate} /> : null}
      </nav> : null}
    </article>
  )
}

function docPageBreadcrumbs(pages: DocPage[], page: DocPage) {
  const pagesById = new Map(pages.map((item) => [item.id, item]))
  const ancestors: DocPage[] = []
  const visited = new Set<number>([page.id])
  let parentId = page.parentId
  while (parentId && !visited.has(parentId)) {
    const parent = pagesById.get(parentId)
    if (!parent) break
    ancestors.unshift(parent)
    visited.add(parent.id)
    parentId = parent.parentId
  }
  return [...ancestors, page]
}

function ArticlePager({ page, direction, onNavigate }: { page: DocPage; direction: "previous" | "next"; onNavigate: DocPageNavigationHandler }) {
  const t = useI18n()
  return <SupportDocLink page={page} onNavigate={onNavigate} className={cn("group rounded-md border px-4 py-3 transition-colors hover:border-primary/40 hover:bg-muted/50", direction === "next" && "text-right")}>
    <span className="text-xs text-muted-foreground">{t(`supportPublic.help.${direction}`)}</span>
    <span className="mt-1 flex items-center justify-between gap-3 text-sm font-medium text-primary">{direction === "previous" ? <ChevronRightIcon className="size-4 rotate-180" /> : null}<span className={cn("truncate", direction === "next" && "ml-auto")}>{page.title}</span>{direction === "next" ? <ChevronRightIcon className="size-4" /> : null}</span>
  </SupportDocLink>
}

function PublicDocPageNode({
  page,
  depth,
  pages,
  expanded,
  selectedPageId,
  onToggle,
  onNavigate,
}: {
  page: DocPage
  depth: number
  pages: DocPage[]
  expanded: Set<number>
  selectedPageId: number
  onToggle: (id: number) => void
  onNavigate: DocPageNavigationHandler
}) {
  const t = useI18n()
  const open = expanded.has(page.id)
  const children = pages.filter((item) => item.parentId === page.id)
  const hasChildren = children.length > 0
  const selected = selectedPageId === page.id
  return (
    <div className={cn(depth === 0 && "mt-1 first:mt-0")}>
      <div
        className={cn(
          "group relative flex min-h-9 w-full items-center rounded-md pr-2 text-sm text-muted-foreground transition-colors hover:bg-muted/70 hover:text-foreground",
          depth === 0 && hasChildren && "font-semibold text-foreground",
          selected && "bg-primary/10 font-medium text-primary before:absolute before:inset-y-1.5 before:left-0 before:w-0.5 before:rounded-full before:bg-primary"
        )}
        style={{ paddingLeft: `${depth * 16 + 4}px` }}
      >
        {hasChildren ? (
          <button
            type="button"
            className="flex size-7 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-background/80 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            onClick={() => onToggle(page.id)}
            aria-expanded={open}
            aria-label={open ? t("supportPublic.a11y.collapse") : t("supportPublic.a11y.expand")}
          >
            {open ? <ChevronDownIcon className="size-4" /> : <ChevronRightIcon className="size-4" />}
          </button>
        ) : <span className="size-7 shrink-0" />}
        <SupportDocLink page={page} onNavigate={onNavigate} className="flex min-w-0 flex-1 items-center gap-2 py-1.5 text-left leading-5" aria-current={selected ? "page" : undefined}>
          {hasChildren ? (open ? <FolderOpenIcon className="size-4 shrink-0" /> : <FolderIcon className="size-4 shrink-0" />) : <FileTextIcon className="size-3.5 shrink-0 opacity-70" />}
          <span className="line-clamp-2">{page.title}</span>
        </SupportDocLink>
      </div>
      {open && hasChildren ? (
        <div className="relative before:absolute before:inset-y-1 before:w-px before:bg-border/80" style={{ marginLeft: `${depth * 16 + 17}px` }}>
          <div style={{ marginLeft: `${-(depth * 16 + 17)}px` }}>
            {children.map((child) => (
              <PublicDocPageNode key={child.id} page={child} depth={depth + 1} pages={pages} expanded={expanded} selectedPageId={selectedPageId} onToggle={onToggle} onNavigate={onNavigate} />
            ))}
          </div>
        </div>
      ) : null}
    </div>
  )
}

function ChildPageLinks({ pages, onNavigate }: { pages: DocPage[]; onNavigate: DocPageNavigationHandler }) {
  const t = useI18n()
  if (!pages.length) return null
  return (
    <section className="mt-10 border-t pt-6" aria-labelledby="support-child-pages-title">
      <h2 id="support-child-pages-title" className="mb-4 text-sm font-semibold tracking-tight text-foreground">
        {t("supportPublic.help.childPages")}
      </h2>
      <div className="grid gap-3 sm:grid-cols-2">
        {pages.map((page) => (
          <SupportDocLink
            key={page.id}
            page={page}
            onNavigate={onNavigate}
            className="group flex min-w-0 items-start gap-3 rounded-md border border-border/70 bg-card p-4 shadow-xs transition-[border-color,background-color,box-shadow,transform] hover:-translate-y-0.5 hover:border-primary/30 hover:bg-primary/[0.025] hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          >
            <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border/60 bg-muted/60 text-muted-foreground transition-colors group-hover:border-primary/20 group-hover:bg-primary/10 group-hover:text-primary">
              <FileTextIcon className="size-4" aria-hidden="true" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block text-sm font-semibold leading-6 text-foreground transition-colors group-hover:text-primary">
                {page.title}
              </span>
              {page.summary ? <span className="mt-0.5 line-clamp-3 text-sm leading-5 text-muted-foreground">{page.summary}</span> : null}
            </span>
            <ArrowRightIcon className="mt-2 size-4 shrink-0 text-muted-foreground/60 transition-[color,transform] group-hover:translate-x-0.5 group-hover:text-primary" aria-hidden="true" />
          </SupportDocLink>
        ))}
      </div>
    </section>
  )
}
