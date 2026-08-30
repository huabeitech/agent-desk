import { type CSSProperties, type ReactNode } from "react"

import { SupportPageContent, SupportPageShell, type SupportPageMobileNavigation } from "@/app/(support)/support/_components/support-page-shell"
import { type SupportHeaderSection } from "@/app/(support)/support/_components/support-header"
import { cn } from "@/lib/utils"

type SupportPageLayoutProps = {
  children: ReactNode
  section?: SupportHeaderSection
  startAside?: ReactNode
  endAside?: ReactNode
  startAsideWidth?: string
  endAsideWidth?: string
  startAsideBreakpoint?: "lg" | "xl"
  endAsideBreakpoint?: "xl" | "2xl"
  startAsideClassName?: string
  endAsideClassName?: string
  mobileNavigation?: SupportPageMobileNavigation
  contentClassName?: string
  fullBleed?: boolean
}

const startAsideColumnClasses = {
  lg: "lg:grid-cols-[var(--support-layout-start-aside-width)_minmax(0,1fr)]",
  xl: "xl:grid-cols-[var(--support-layout-start-aside-width)_minmax(0,1fr)]",
}

const endAsideColumnClasses = {
  xl: "xl:grid-cols-[minmax(0,1fr)_var(--support-layout-end-aside-width)]",
  "2xl": "2xl:grid-cols-[minmax(0,1fr)_var(--support-layout-end-aside-width)]",
}

const threeColumnClasses = {
  xl: "xl:grid-cols-[var(--support-layout-start-aside-width)_minmax(0,1fr)_var(--support-layout-end-aside-width)]",
  "2xl": "2xl:grid-cols-[var(--support-layout-start-aside-width)_minmax(0,1fr)_var(--support-layout-end-aside-width)]",
}

const endAsideVisibilityClasses = {
  xl: "hidden xl:block",
  "2xl": "hidden 2xl:block",
}

export function SupportPageLayout({
  children,
  section = "home",
  startAside,
  endAside,
  startAsideWidth = "20rem",
  endAsideWidth = "var(--support-doc-toc-width)",
  startAsideBreakpoint = "lg",
  endAsideBreakpoint = "2xl",
  startAsideClassName,
  endAsideClassName,
  mobileNavigation,
  contentClassName,
  fullBleed = false,
}: SupportPageLayoutProps) {
  const hasColumns = Boolean(startAside || endAside)
  const endAsideStartsGridAtSameBreakpoint = startAside && endAside && startAsideBreakpoint === endAsideBreakpoint
  const layoutStyle = {
    "--support-layout-start-aside-width": startAsideWidth,
    "--support-layout-end-aside-width": endAsideWidth,
  } as CSSProperties

  return (
    <SupportPageShell section={section} mobileNavigation={mobileNavigation}>
      {fullBleed ? children : (
        <SupportPageContent
          width="docs"
          style={layoutStyle}
          className={cn(
            hasColumns && "grid items-start gap-6",
            startAside && !endAsideStartsGridAtSameBreakpoint && startAsideColumnClasses[startAsideBreakpoint],
            endAside && !startAside && endAsideColumnClasses[endAsideBreakpoint],
            startAside && endAside && threeColumnClasses[endAsideBreakpoint],
            contentClassName
          )}
        >
          {startAside ? <div className={cn("min-w-0 self-stretch", startAsideClassName)}>{startAside}</div> : null}
          {children}
          {endAside ? <div className={cn("min-w-0 self-stretch", endAsideVisibilityClasses[endAsideBreakpoint], endAsideClassName)}>{endAside}</div> : null}
        </SupportPageContent>
      )}
    </SupportPageShell>
  )
}
