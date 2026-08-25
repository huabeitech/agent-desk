import { type ReactNode } from "react"

import { SupportAIChatWidget } from "@/app/(support)/support/_components/support-ai-chat-widget"
import { SupportHeader, type SupportHeaderSection } from "@/app/(support)/support/_components/support-header"
import { cn } from "@/lib/utils"

export function SupportPageShell({ children, section = "home" }: { children: ReactNode; section?: SupportHeaderSection }) {
  return (
    // <div className="min-h-svh overflow-x-clip bg-[#f7f9fc] text-foreground dark:bg-background">
    <div className="min-h-svh overflow-x-clip bg-[#F4F6FF] text-foreground dark:bg-background">
      <SupportHeader section={section} />
      {children}
      <SupportAIChatWidget />
    </div>
  )
}

export function SupportPageContent({ children, className, width = "standard" }: { children?: ReactNode; className?: string; width?: "standard" | "docs" }) {
  return <div className={cn("mx-auto px-5 sm:px-6 md:px-8 lg:px-10", width === "docs" ? "max-w-(--support-docs-max-width)" : "max-w-(--support-shell-max-width)", className)}>{children}</div>
}
