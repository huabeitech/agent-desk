import { CheckCircle2Icon, SearchIcon } from "lucide-react"
import { type ReactNode } from "react"

import { Badge } from "@/components/ui/badge"
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
  if (status === "resolved") return <Badge className="bg-emerald-600 text-white"><CheckCircle2Icon /> {t("supportPublic.status.resolved")}</Badge>
  if (status === "closed") return <Badge variant="outline">{t("supportPublic.status.closed")}</Badge>
  return <Badge variant="secondary">{t("supportPublic.status.normal")}</Badge>
}

export function SupportInfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border bg-card p-4 shadow-sm">
      <div className="text-2xl font-semibold">{value}</div>
      <div className="mt-1 text-sm text-muted-foreground">{label}</div>
    </div>
  )
}
