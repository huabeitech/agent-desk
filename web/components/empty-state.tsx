import type { ReactNode } from "react"
import { InboxIcon } from "lucide-react"

import { cn } from "@/lib/utils"

export function EmptyState({
  text,
  description,
  icon,
  action,
  className,
  compact = false,
}: {
  text: ReactNode
  description?: ReactNode
  icon?: ReactNode
  action?: ReactNode
  className?: string
  compact?: boolean
}) {
  return (
    <div
      role="status"
      className={cn(
        "flex flex-col items-center justify-center rounded-md bg-muted/20 px-6 py-10 text-center",
        compact && "border-0 bg-transparent px-4 py-6",
        className
      )}
    >
      <div className={cn("mb-4 flex size-11 items-center justify-center rounded-md border bg-background text-muted-foreground", compact && "mb-3 size-9")}>
        {icon ?? <InboxIcon aria-hidden="true" className={cn("size-5", compact && "size-4")} />}
      </div>
      <p className="text-sm font-medium text-foreground">{text}</p>
      {description ? <p className="mt-1 max-w-md text-sm leading-6 text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  )
}
