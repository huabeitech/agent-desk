"use client"

import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
} from "react"

import { THEME_STORAGE_KEY } from "@/lib/theme"

export type ThemeMode = "light" | "dark" | "system"
type ResolvedTheme = Exclude<ThemeMode, "system">

type ThemeContextValue = {
  theme: ThemeMode
  resolvedTheme: ResolvedTheme
  setTheme: (theme: ThemeMode) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)
const subscribers = new Set<() => void>()

function readTheme(): ThemeMode {
  const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY)
  return storedTheme === "light" || storedTheme === "dark" || storedTheme === "system"
    ? storedTheme
    : "system"
}

function subscribe(listener: () => void) {
  subscribers.add(listener)
  return () => subscribers.delete(listener)
}

function notifyThemeChange() {
  subscribers.forEach((listener) => listener())
}

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}

function resolveTheme(theme: ThemeMode): ResolvedTheme {
  return theme === "system" ? getSystemTheme() : theme
}

function readResolvedTheme(): ResolvedTheme {
  return resolveTheme(readTheme())
}

function applyTheme(theme: ThemeMode) {
  const resolvedTheme = resolveTheme(theme)
  const root = document.documentElement
  root.classList.remove("light", "dark")
  root.classList.add(resolvedTheme)
  root.style.colorScheme = resolvedTheme
  return resolvedTheme
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useSyncExternalStore(subscribe, readTheme, (): ThemeMode => "system")
  const resolvedTheme = useSyncExternalStore(
    subscribe,
    readResolvedTheme,
    (): ResolvedTheme => "light"
  )

  const setTheme = useCallback((nextTheme: ThemeMode) => {
    window.localStorage.setItem(THEME_STORAGE_KEY, nextTheme)
    applyTheme(nextTheme)
    notifyThemeChange()
  }, [])

  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)")
    const handleSystemThemeChange = () => {
      if (readTheme() === "system") {
        applyTheme("system")
        notifyThemeChange()
      }
    }
    const handleStorage = (event: StorageEvent) => {
      if (event.key === THEME_STORAGE_KEY) {
        applyTheme(readTheme())
        notifyThemeChange()
      }
    }

    mediaQuery.addEventListener("change", handleSystemThemeChange)
    window.addEventListener("storage", handleStorage)
    return () => {
      mediaQuery.removeEventListener("change", handleSystemThemeChange)
      window.removeEventListener("storage", handleStorage)
    }
  }, [])

  const value = useMemo(
    () => ({ theme, resolvedTheme, setTheme }),
    [resolvedTheme, setTheme, theme]
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  const theme = useContext(ThemeContext)
  if (!theme) {
    throw new Error("useTheme must be used within ThemeProvider")
  }
  return theme
}
