"use client"

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"

import {
  DEFAULT_LOCALE,
  type AppLocale,
  configureLocale,
} from "@/i18n/config"
import { translateMessage } from "@/i18n/messages"
import { fetchPublicConfig, type PublicConfig } from "@/lib/api/config"

type LocaleContextValue = {
  locale: AppLocale
  setLocale: (locale: AppLocale) => void
  t: (key: string, values?: Record<string, string | number>) => string
}

const LocaleContext = createContext<LocaleContextValue>({
  locale: DEFAULT_LOCALE,
  setLocale: () => {},
  t: (key) => key,
})

function applyBranding(locale: AppLocale, config?: PublicConfig | null) {
  if (typeof document === "undefined") return

  const rawTitle = translateMessage(locale, "app.metadataTitle")
  const brandName = config?.companyName?.trim()
  if (brandName) {
    if (brandName.toLowerCase().includes("desk") || brandName.toLowerCase().includes("crove")) {
      document.title = `${brandName} - ${rawTitle}`
    } else {
      document.title = `${brandName} Desk - ${rawTitle}`
    }
  } else {
    document.title = rawTitle
  }

  const faviconUrl = config?.companyFaviconUrl || config?.companyLogoUrl || "/favicon.svg"
  let link = document.querySelector<HTMLLinkElement>("link[rel~='icon']")
  if (!link) {
    link = document.createElement("link")
    link.rel = "icon"
    document.head.appendChild(link)
  }
  link.href = faviconUrl
}

export function AppI18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<AppLocale>(DEFAULT_LOCALE)
  const [publicConfig, setPublicConfig] = useState<PublicConfig | null>(null)
  const [isLocaleReady, setIsLocaleReady] = useState(false)

  const handleSetLocale = (newLocale: AppLocale) => {
    const next = configureLocale(newLocale)
    setLocaleState(next)
    try {
      window.localStorage.setItem("app_locale", next)
    } catch (_) {}
    document.documentElement.lang = next
    applyBranding(next, publicConfig)
  }

  useEffect(() => {
    let cancelled = false
    let savedLocale: string | null = null
    try {
      savedLocale = window.localStorage.getItem("app_locale")
    } catch (_) {}

    fetchPublicConfig()
      .then((cfg) => {
        if (cancelled) return
        setPublicConfig(cfg)
        const targetLocale = savedLocale ? configureLocale(savedLocale) : configureLocale(cfg.language)
        setLocaleState(targetLocale)
        document.documentElement.lang = targetLocale
        applyBranding(targetLocale, cfg)
        setIsLocaleReady(true)
      })
      .catch(() => {
        if (cancelled) return
        const targetLocale = savedLocale ? configureLocale(savedLocale) : configureLocale(DEFAULT_LOCALE)
        setLocaleState(targetLocale)
        document.documentElement.lang = targetLocale
        applyBranding(targetLocale, null)
        setIsLocaleReady(true)
      })

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    applyBranding(locale, publicConfig)
  }, [locale, publicConfig])

  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      t: (key, values) => translateMessage(locale, key, values),
      setLocale: handleSetLocale,
    }),
    [locale]
  )

  if (!isLocaleReady) {
    return null
  }

  return (
    <LocaleContext.Provider value={value}>
      {children}
    </LocaleContext.Provider>
  )
}

export function useAppLocale() {
  return useContext(LocaleContext)
}

export function useI18n() {
  return useContext(LocaleContext).t
}
