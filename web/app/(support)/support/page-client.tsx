"use client"

import { useState } from "react"
import Link from "next/link"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { SupportPageShell } from "@/app/(support)/support/_components/support-page-shell"
import { SupportSearchInput } from "@/app/(support)/support/_components/support-ui"
import { useI18n } from "@/i18n/provider"
import { postsHref } from "@/lib/api/support-community"
import { cn } from "@/lib/utils"

export function SupportCenterHome() {
  const t = useI18n()
  const [query, setQuery] = useState("")

  return (
    <SupportPageShell>
      <section className="relative bg-[radial-gradient(circle_at_50%_-30%,#ddecff,transparent_55%)] px-5 py-12 sm:px-8 sm:py-18 dark:border-border dark:bg-[radial-gradient(circle_at_50%_-30%,rgba(36,117,252,.26),transparent_55%)]">
        <div className="relative mx-auto max-w-3xl text-center">
          <Badge variant="secondary" className="mb-5 bg-white/70 px-3 py-1 text-primary dark:bg-card/80">
            {t("supportPublic.home.badge")}
          </Badge>
          <h1 className="text-balance text-3xl font-semibold tracking-tight sm:text-5xl">
            {t("supportPublic.home.title")}
          </h1>
          <p className="mx-auto mt-4 max-w-xl text-pretty text-sm leading-6 text-muted-foreground sm:text-base">
            {t("supportPublic.home.description")}
          </p>
          <div className="relative mx-auto mt-8 flex max-w-2xl flex-col gap-2 sm:flex-row">
            <SupportSearchInput
              value={query}
              onChange={setQuery}
              placeholder={t("supportPublic.home.searchPlaceholder")}
              hero
            />
            <Link
              className={cn(buttonVariants({ size: "lg" }), "h-13 rounded-md px-6")}
              href={`${postsHref()}${query ? `?title=${encodeURIComponent(query)}` : ""}`}
            >
              {t("supportPublic.actions.search")}
            </Link>
          </div>
        </div>
      </section>

    </SupportPageShell>
  )
}
