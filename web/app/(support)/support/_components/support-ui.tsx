import { CheckCircle2Icon, CircleHelpIcon, SearchIcon } from "lucide-react"
import { type ReactNode } from "react"

import { Input } from "@/components/ui/input"
import { useI18n } from "@/i18n/provider"
import { cn } from "@/lib/utils"

export function SupportSearchInput({
  value,
  onChange,
  placeholder,
  compact = false,
  hero = false,
}: {
  value: string
  onChange: (value: string) => void
  placeholder: string
  compact?: boolean
  hero?: boolean
}) {
  return (
    <div className="relative flex-1">
      <SearchIcon className={cn("pointer-events-none absolute top-1/2 -translate-y-1/2 text-muted-foreground", hero ? "left-4 size-5" : "left-3 size-4")} />
      <Input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={cn(
          "bg-card",
          hero && "h-13 rounded-md border-white bg-white pl-12 pr-4 text-base shadow-[0_12px_30px_rgba(36,117,252,.12)] focus-visible:ring-primary/25 dark:border-border dark:bg-card",
          compact && "h-9 pl-9",
          !hero && !compact && "h-11 pl-9"
        )}
        placeholder={placeholder}
      />
    </div>
  )
}

export function SupportFormField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="grid gap-2">
      <span className="text-sm font-medium">{label}</span>
      {children}
    </label>
  )
}

export function PostStatusBadge({ status }: { status: string }) {
  const t = useI18n()
  if (status === "resolved") {
    return (
      <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] leading-none font-medium text-emerald-700 ring-1 ring-emerald-200">
        <CheckCircle2Icon className="mr-1 size-3" />
        {t("supportPublic.status.resolved")}
      </span>
    )
  }
  if (status === "closed") {
    return (
      <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[11px] leading-none font-medium text-muted-foreground ring-1 ring-border">
        {t("supportPublic.status.closed")}
      </span>
    )
  }
  return (
    <span className="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-[11px] leading-none font-medium text-amber-700 ring-1 ring-amber-200">
      <CircleHelpIcon className="mr-1 size-3" />
      {t("supportPublic.status.normal")}
    </span>
  )
}

export function SupportInfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border bg-card p-4 shadow-sm">
      <div className="text-2xl font-semibold">{value}</div>
      <div className="mt-1 text-sm text-muted-foreground">{label}</div>
    </div>
  )
}
