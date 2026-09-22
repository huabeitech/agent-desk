"use client"

import Image from "next/image"
import Link from "next/link"
import { useEffect, useState } from "react"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useAppLocale, useI18n } from "@/i18n/provider"
import { fetchPublicConfig, type PublicConfig } from "@/lib/api/config"
import enUSMessages from "@/messages/en-US.json"
import viVNMessages from "@/messages/vi-VN.json"
import zhCNMessages from "@/messages/zh-CN.json"

type LegalPageType = "terms" | "privacy"

type LegalSection = {
  title: string
  body: string
}

type LegalDocument = {
  title: string
  description: string
  updatedAt: string
  relatedLabel: string
  relatedLink: string
  sections: LegalSection[]
}

const messages: Record<string, { legal: Record<LegalPageType, LegalDocument> }> = {
  "zh-CN": zhCNMessages,
  "en-US": enUSMessages,
  "vi-VN": viVNMessages as unknown as { legal: Record<LegalPageType, LegalDocument> },
}

export function LegalDocumentPage({ type }: { type: LegalPageType }) {
  const t = useI18n()
  const { locale } = useAppLocale()
  const [config, setConfig] = useState<PublicConfig | null>(null)
  const document = messages[locale].legal[type] as LegalDocument
  const relatedHref = type === "terms" ? "/legal/privacy" : "/legal/terms"

  useEffect(() => {
    fetchPublicConfig()
      .then(setConfig)
      .catch(() => setConfig(null))
  }, [])

  const brandName = config?.companyName || t("app.brand")
  const brandLogo = config?.companyLogoUrl || "/images/logo.svg"

  return (
    <main className="min-h-svh bg-muted px-6 py-8 md:px-10">
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
        <header className="flex items-center gap-4">
          <div className="flex items-center gap-2 font-medium">
            <Image
              src={brandLogo}
              alt={brandName}
              width={32}
              height={32}
              className="size-8 object-contain"
              priority
              unoptimized={brandLogo.startsWith("http")}
            />
            <span>{brandName}</span>
          </div>
        </header>

        <Card className="bg-card/95">
          <CardHeader className="gap-3 border-b px-6 py-6 md:px-8">
            <div className="flex flex-col gap-2">
              <CardTitle className="text-3xl font-semibold tracking-tight">
                {document.title}
              </CardTitle>
              <p className="text-sm leading-6 text-muted-foreground">
                {document.description}
              </p>
            </div>
            <p className="text-xs text-muted-foreground">{document.updatedAt}</p>
          </CardHeader>
          <CardContent className="space-y-8 px-6 py-7 md:px-8">
            {document.sections.map((section) => (
              <section key={section.title} className="space-y-2">
                <h2 className="text-lg font-semibold tracking-tight">
                  {section.title}
                </h2>
                <p className="text-sm leading-7 text-muted-foreground">
                  {section.body}
                </p>
              </section>
            ))}
          </CardContent>
        </Card>

        <footer className="flex justify-end text-sm text-muted-foreground">
          <p>
            {document.relatedLabel}{" "}
            <Link
              href={relatedHref}
              className="font-medium text-foreground underline-offset-4 hover:underline"
            >
              {document.relatedLink}
            </Link>
          </p>
        </footer>
      </div>
    </main>
  )
}
