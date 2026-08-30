import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

export function EmptyState({
  text,
  className,
  compact = false,
}: {
  text: ReactNode
  className?: string
  compact?: boolean
}) {
  return (
    <div className={cn("rounded-md border border-dashed bg-card p-8 text-center text-sm text-muted-foreground", compact && "border-0 p-5", className)}>
      {text}
    </div>
  )
}
