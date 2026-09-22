"use client"

import { GlobeIcon } from "lucide-react"
import { useAppLocale, useI18n } from "@/i18n/provider"
import { type AppLocale } from "@/i18n/config"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const languageOptions: Array<{
  value: AppLocale
  labelKey: string
  flag: string
}> = [
  { value: "en-US", labelKey: "language.enUS", flag: "🇺🇸" },
  { value: "vi-VN", labelKey: "language.viVN", flag: "🇻🇳" },
  { value: "zh-CN", labelKey: "language.zhCN", flag: "🇨🇳" },
]

export function LanguageToggle({
  variant = "outline",
  size = "sm",
}: {
  variant?: "outline" | "ghost"
  size?: "sm" | "icon" | "icon-sm"
}) {
  const t = useI18n()
  const { locale, setLocale } = useAppLocale()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant={variant} size={size} />}
        aria-label={t("language.selectLanguage")}
        title={t("language.selectLanguage")}
      >
        <GlobeIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48 min-w-48">
        <DropdownMenuRadioGroup
          value={locale}
          onValueChange={(value) => setLocale(value as AppLocale)}
        >
          {languageOptions.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              <span className="mr-1 text-sm">{option.flag}</span>
              {t(option.labelKey)}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
