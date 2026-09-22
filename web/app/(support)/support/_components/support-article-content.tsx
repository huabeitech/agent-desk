"use client"

import { Children, isValidElement, useEffect, useId, useMemo, useRef, useState, type MouseEvent, type ReactNode } from "react"
import { useTheme } from "next-themes"
import rehypeHighlight from "rehype-highlight"
import ReactMarkdown, { defaultUrlTransform, type Components } from "react-markdown"
import remarkBreaks from "remark-breaks"
import remarkGfm from "remark-gfm"
import { toast } from "sonner"

import { useImageLightboxOptional, type ImageLightboxContextValue, type ImageLightboxItem } from "@/components/image-lightbox"
import { SafeRichHTML } from "@/components/safe-rich-html"
import { useI18n } from "@/i18n/provider"
import { articleHeadingId } from "@/lib/support-article"

let mermaidRenderQueue = Promise.resolve()

function enqueueMermaidRender(task: () => Promise<void>) {
  const next = mermaidRenderQueue.then(task, task)
  mermaidRenderQueue = next.catch(() => undefined)
  return next
}

function registerArticleLightboxItem(element: HTMLElement) {
  element.dataset.supportLightboxItem = ""
}

function unregisterArticleLightboxItem(element: HTMLElement) {
  delete element.dataset.supportLightboxItem
}

function resolveArticleLightboxItem(element: HTMLElement): ImageLightboxItem | null {
  const svg = element.querySelector<SVGSVGElement>("svg")
  if (svg) {
    const figure = element.closest<HTMLElement>(".support-mermaid")
    return {
      src: `mermaid:${svg.id}`,
      alt: element.querySelector<HTMLElement>("[role='img']")?.getAttribute("aria-label") ?? "",
      svg: svg.outerHTML,
      backgroundColor: figure ? window.getComputedStyle(figure).backgroundColor : undefined,
    }
  }
  if (element instanceof HTMLImageElement) {
    const src = element.getAttribute("src")
    return src ? { src, alt: element.alt } : null
  }
  return null
}

function openArticleLightbox(lightbox: ImageLightboxContextValue, target: HTMLElement) {
  const article = target.closest(".typeset-support-docs")
  if (!article) return
  const elements = Array.from(article.querySelectorAll<HTMLElement>("[data-support-lightbox-item]"))
  const entries = elements.flatMap((element) => {
    const item = resolveArticleLightboxItem(element)
    return item ? [{ element, item }] : []
  })
  const index = entries.findIndex((entry) => entry.element === target)
  if (index < 0) return
  const items = entries.map((entry) => entry.item)
  if (typeof lightbox.openGallery === "function") {
    lightbox.openGallery(items, index)
  } else {
    const item = items[index]
    lightbox.open(item.src, item.alt)
  }
}

type MarkdownNode = {
  children?: MarkdownNode[]
  properties?: Record<string, unknown>
  tagName?: string
  type: string
  value?: string
}

function nodeText(node: MarkdownNode): string {
  if (node.type === "text") return node.value ?? ""
  return node.children?.map(nodeText).join("") ?? ""
}

function rehypeArticleHeadingIds() {
  return (tree: MarkdownNode) => {
    let headingIndex = 0
    const visit = (node: MarkdownNode) => {
      if (node.type === "element" && (node.tagName === "h2" || node.tagName === "h3")) {
        node.properties = {
          ...node.properties,
          className: "scroll-mt-20",
          id: articleHeadingId(nodeText(node), headingIndex),
        }
        headingIndex += 1
      }
      node.children?.forEach(visit)
    }
    visit(tree)
  }
}

function supportArticleUrlTransform(url: string, key: string) {
  const transformed = defaultUrlTransform(url)
  if (!transformed || (key !== "href" && key !== "src")) return transformed
  if (transformed.startsWith("/")) return transformed
  try {
    const parsed = new URL(transformed, "https://support.local")
    return parsed.protocol === "http:" || parsed.protocol === "https:" ? transformed : ""
  } catch {
    return ""
  }
}

function reactNodeText(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") return String(node)
  if (Array.isArray(node)) return node.map(reactNodeText).join("")
  if (isValidElement<{ children?: ReactNode }>(node)) return reactNodeText(node.props.children)
  return ""
}

function mermaidDefinition(children: ReactNode) {
  const code = Children.toArray(children).find(
    (child) => isValidElement<{ className?: string }>(child) && child.props.className?.split(/\s+/).includes("language-mermaid"),
  )
  if (!isValidElement<{ children?: ReactNode }>(code)) return null
  return reactNodeText(code.props.children).replace(/\n$/, "")
}

function MermaidDiagram({ definition }: { definition: string }) {
  const t = useI18n()
  const { resolvedTheme } = useTheme()
  const lightbox = useImageLightboxOptional()
  const previewRef = useRef<HTMLButtonElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const reactId = useId()
  const diagramId = useMemo(() => `support-mermaid-${reactId.replace(/[^a-zA-Z0-9_-]/g, "")}`, [reactId])
  const renderKey = `${definition}\n${resolvedTheme}`
  const [readyRenderKey, setReadyRenderKey] = useState<string | null>(null)
  const isReady = readyRenderKey === renderKey

  useEffect(() => {
    const container = containerRef.current
    const preview = previewRef.current
    if (!container) return
    let cancelled = false
    const status = document.createElement("div")
    status.className = "support-mermaid-status"
    status.setAttribute("role", "status")
    status.textContent = t("supportPublic.help.mermaidLoading")
    container.setAttribute("aria-busy", "true")
    container.replaceChildren(status)

    void enqueueMermaidRender(async () => {
      try {
        const { default: mermaid } = await import("mermaid")
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "strict",
          theme: resolvedTheme === "dark" ? "dark" : "default",
          themeVariables: resolvedTheme === "dark" ? {
            actorBkg: "#27272a",
            actorBorder: "#8b5cf6",
            actorLineColor: "#a1a1aa",
            actorTextColor: "#f4f4f5",
            edgeLabelBackground: "#18181b",
            labelBoxBkgColor: "#27272a",
            labelBoxBorderColor: "#8b5cf6",
            labelTextColor: "#f4f4f5",
            lineColor: "#a1a1aa",
            loopTextColor: "#f4f4f5",
            noteBkgColor: "#292524",
            noteTextColor: "#f4f4f5",
            primaryTextColor: "#f4f4f5",
            signalColor: "#d4d4d8",
            signalTextColor: "#f4f4f5",
            tertiaryTextColor: "#f4f4f5",
          } : undefined,
          fontFamily: "var(--font-inter), Arial, sans-serif",
          flowchart: { useMaxWidth: true },
        })
        const valid = await mermaid.parse(definition, { suppressErrors: true })
        if (!valid) throw new Error("Invalid Mermaid definition")
        const { svg } = await mermaid.render(diagramId, definition)
        if (cancelled) return
        container.innerHTML = svg
        container.setAttribute("aria-busy", "false")
        if (lightbox && preview) {
          registerArticleLightboxItem(preview)
          setReadyRenderKey(renderKey)
        }
      } catch {
        if (cancelled) return
        const error = document.createElement("div")
        error.className = "support-mermaid-status support-mermaid-error"
        error.setAttribute("role", "alert")
        error.textContent = t("supportPublic.help.mermaidError")
        container.setAttribute("aria-busy", "false")
        container.replaceChildren(error)
      }
    })

    return () => {
      cancelled = true
      if (preview) unregisterArticleLightboxItem(preview)
    }
  }, [definition, diagramId, lightbox, renderKey, resolvedTheme, t])

  return (
    <figure className="support-mermaid not-typeset" data-not-typeset>
      <button
        ref={previewRef}
        type="button"
        disabled={!isReady}
        className="support-mermaid-preview"
        aria-label={t("supportPublic.help.mermaidPreview")}
        onClick={(event) => {
          if (lightbox) openArticleLightbox(lightbox, event.currentTarget)
        }}
      >
        <div
          ref={containerRef}
          className="support-mermaid-canvas"
          role="img"
          aria-label={t("supportPublic.help.mermaidDiagram")}
        />
      </button>
    </figure>
  )
}

type SupportArticleContentProps = {
  content: string
  contentType?: string
  id: string
  articleHeadingIds?: boolean
}

export function SupportArticleContent({ content, contentType = "markdown", id, articleHeadingIds = true }: SupportArticleContentProps) {
  const t = useI18n()
  const lightbox = useImageLightboxOptional()
  const components = useMemo<Components>(() => ({
    a: ({ children, href, ...props }) => (
      <a {...props} href={href} target="_blank" rel="noreferrer noopener">
        {children}
      </a>
    ),
    img: ({ alt, src, ...props }) => (
      // Markdown images can use arbitrary remote hosts, so Next Image cannot optimize them safely.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        {...props}
        alt={alt ?? ""}
        src={src}
        ref={(image) => {
          if (image && lightbox && typeof src === "string") {
            registerArticleLightboxItem(image)
            return () => unregisterArticleLightboxItem(image)
          }
        }}
        className={lightbox && typeof src === "string" ? "cursor-zoom-in" : undefined}
        onClick={lightbox && typeof src === "string" ? (event) => openArticleLightbox(lightbox, event.currentTarget) : undefined}
      />
    ),
    pre: ({ children, ...props }) => {
      const definition = mermaidDefinition(children)
      if (definition !== null) return <MermaidDiagram definition={definition} />
      const copy = (event: MouseEvent<HTMLButtonElement>) => {
        const code = event.currentTarget.parentElement?.querySelector("code")?.textContent ?? ""
        void navigator.clipboard.writeText(code).then(() => toast.success(t("supportPublic.toast.codeCopied")))
      }
      return (
        <pre {...props} className="group relative">
          {children}
          <button
            type="button"
            className="not-typeset absolute right-2 top-2 rounded-md border border-border bg-background/90 px-2 py-1 text-xs text-muted-foreground opacity-0 shadow-sm transition-opacity group-hover:opacity-100 focus:opacity-100"
            data-not-typeset
            aria-label={t("supportPublic.help.copyCode")}
            onClick={copy}
          >
            {t("supportPublic.help.copyCode")}
          </button>
        </pre>
      )
    },
    table: ({ children, ...props }) => (
      <div className="typeset-scroll">
        <table {...props}>{children}</table>
      </div>
    ),
  }), [lightbox, t])

  if (contentType === "html") {
    return <SafeRichHTML id={id} html={content} articleHeadingIds={articleHeadingIds} unstyled className="typeset typeset-support-docs" />
  }

  return (
    <div id={id} className="typeset typeset-support-docs">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        rehypePlugins={[
          ...(articleHeadingIds ? [rehypeArticleHeadingIds] : []),
          [rehypeHighlight, { detect: false, plainText: ["text", "txt", "plaintext"] }],
        ]}
        skipHtml
        urlTransform={supportArticleUrlTransform}
        components={components}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
