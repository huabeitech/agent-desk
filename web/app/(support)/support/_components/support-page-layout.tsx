import { type ReactNode } from "react"

import { SupportPageContent, SupportPageShell, type SupportPageMobileNavigation } from "@/app/(support)/support/_components/support-page-shell"
import { type SupportHeaderSection } from "@/app/(support)/support/_components/support-header"
import { cn } from "@/lib/utils"

type SupportPageLayoutProps = {
  children: ReactNode
  section?: SupportHeaderSection
  sidebar?: ReactNode
  toc?: ReactNode
  mobileNavigation?: SupportPageMobileNavigation
  sidebarBreakpoint?: "lg" | "xl"
  contentClassName?: string
  fullBleed?: boolean
}

const sidebarColumnClasses = {
  lg: "lg:grid-cols-[minmax(0,320px)_minmax(0,1fr)]",
  xl: "xl:grid-cols-[minmax(0,320px)_minmax(0,1fr)]",
}

export function SupportPageLayout({
  children,
  section = "home",
  sidebar,
  toc,
  mobileNavigation,
  sidebarBreakpoint = "lg",
  contentClassName,
  fullBleed = false,
}: SupportPageLayoutProps) {
  const hasColumns = Boolean(sidebar || toc)

  return (
    <SupportPageShell section={section} mobileNavigation={mobileNavigation}>
      {fullBleed ? children : (
        <SupportPageContent
          width="docs"
          className={cn(
            hasColumns && "grid items-start gap-6",
            sidebar && sidebarColumnClasses[sidebarBreakpoint],
            toc && (sidebar ? "2xl:grid-cols-[minmax(0,320px)_minmax(0,1fr)_var(--support-doc-toc-width)]" : "2xl:grid-cols-[minmax(0,1fr)_var(--support-doc-toc-width)]"),
            contentClassName
          )}
        >
          {sidebar}
          {children}
          {toc ? <div className="hidden min-w-0 self-stretch rounded-md bg-card 2xl:block">{toc}</div> : null}
        </SupportPageContent>
      )}
    </SupportPageShell>
  )
}
