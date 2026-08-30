"use client"

import { useEffect, useMemo, useRef, useState, type MouseEvent as ReactMouseEvent } from "react"

import { useI18n } from "@/i18n/provider"
import { articleHeadingId, markdownHeadingText } from "@/lib/support-article"
import { cn } from "@/lib/utils"

export function PublicArticleToc({
  articleId,
  content,
  contentType = "markdown",
  stickyOffset = "header",
}: {
  articleId: string
  content: string
  contentType?: string
  stickyOffset?: "header" | "content"
}) {
  const t = useI18n()
  const tocRef = useRef<HTMLElement>(null)
  const headings = useMemo(() => getArticleTocHeadings(content, contentType), [content, contentType])
  const [activeId, setActiveId] = useState("")

  useEffect(() => {
    if (!headings.length || !articleId) return
    let frame = 0
    const findHeadingElement = (headingId: string) => document.getElementById(articleId)?.querySelector<HTMLElement>(`#${CSS.escape(headingId)}`) ?? null
    const syncActiveHeading = () => {
      frame = 0
      const anchorOffset = 88
      let nextId = headings[0].id
      for (const heading of headings) {
        const element = findHeadingElement(heading.id)
        if (!element || element.getBoundingClientRect().top > anchorOffset) break
        nextId = heading.id
      }
      setActiveId((current) => current === nextId ? current : nextId)
    }
    const scheduleSync = () => {
      if (frame) return
      frame = window.requestAnimationFrame(syncActiveHeading)
    }
    const syncHashTarget = () => {
      const hashId = decodeURIComponent(window.location.hash.slice(1))
      if (headings.some((heading) => heading.id === hashId)) {
        findHeadingElement(hashId)?.scrollIntoView()
      }
    }
    const initialFrame = window.requestAnimationFrame(() => {
      syncHashTarget()
      syncActiveHeading()
    })
    const delayedSync = window.setTimeout(() => {
      syncHashTarget()
      syncActiveHeading()
    }, 50)
    window.addEventListener("scroll", scheduleSync, { passive: true })
    window.addEventListener("resize", scheduleSync)
    return () => {
      window.cancelAnimationFrame(initialFrame)
      window.clearTimeout(delayedSync)
      if (frame) window.cancelAnimationFrame(frame)
      window.removeEventListener("scroll", scheduleSync)
      window.removeEventListener("resize", scheduleSync)
    }
  }, [articleId, headings])

  const scrollToHeading = (event: ReactMouseEvent<HTMLAnchorElement>, headingId: string) => {
    const element = document.getElementById(articleId)?.querySelector<HTMLElement>(`#${CSS.escape(headingId)}`)
    if (!element) return
    event.preventDefault()
    element.scrollIntoView()
    window.history.replaceState(null, "", `#${encodeURIComponent(headingId)}`)
    setActiveId(headingId)
  }

  useEffect(() => {
    const container = tocRef.current
    if (!container || !activeId) return
    const activeLink = Array.from(container.querySelectorAll<HTMLAnchorElement>("[data-toc-id]"))
      .find((link) => link.dataset.tocId === activeId)
    if (!activeLink) return
    const containerRect = container.getBoundingClientRect()
    const linkRect = activeLink.getBoundingClientRect()
    if (linkRect.top >= containerRect.top && linkRect.bottom <= containerRect.bottom) return
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches
    container.scrollTo({
      top: activeLink.offsetTop - container.clientHeight / 2 + activeLink.clientHeight / 2,
      behavior: reducedMotion ? "auto" : "smooth",
    })
  }, [activeId])

  return (
    <aside
      ref={tocRef}
      className={cn(
        "sticky overflow-y-auto px-5 py-12",
        stickyOffset === "content" ? "top-[5.5rem] max-h-[calc(100svh-5.625rem)]" : "top-14 max-h-[calc(100svh-3.5rem)]"
      )}
    >
      <div>
        <div className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{t("supportPublic.help.toc")}</div>
        {headings.length ? headings.map((item, index) => (
          <a
            key={`${item.title}-${index}`}
            href={`#${item.id}`}
            data-toc-id={item.id}
            aria-current={activeId === item.id ? "location" : undefined}
            onClick={(event) => scrollToHeading(event, item.id)}
            className={cn(
              "block border-l py-1.5 pl-3 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-foreground",
              item.level === 3 && "pl-6",
              activeId === item.id && "border-primary bg-muted/50 font-medium text-foreground"
            )}
          >
            <span className="line-clamp-3">{item.title}</span>
          </a>
        )) : <div className="text-sm text-muted-foreground">{t("supportPublic.help.noToc")}</div>}
      </div>
    </aside>
  )
}

export function hasArticleTocHeadings(content: string, contentType?: string) {
  return getArticleTocHeadings(content, contentType).length > 0
}

function getArticleTocHeadings(content: string, contentType = "markdown") {
  return contentType === "html"
    ? Array.from(content.matchAll(/<h([23])[^>]*>([\s\S]*?)<\/h\1>/gi)).map((match, index) => {
        const title = match[2].replace(/<[^>]+>/g, "").trim()
        return { level: Number(match[1]), title, id: articleHeadingId(title, index) }
      })
    : Array.from(content.matchAll(/^(#{2,3})\s+(.+)$/gm)).map((match, index) => {
        const title = markdownHeadingText(match[2])
        return { level: match[1].length, title, id: articleHeadingId(title, index) }
      })
}
