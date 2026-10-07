"use client"

import { useSyncExternalStore } from "react"
import { LaptopIcon, MoonIcon, SunIcon } from "lucide-react"
import { useTheme } from "@/components/theme-provider"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useI18n } from "@/i18n/provider"
import { cn } from "@/lib/utils"

type ThemeMode = "light" | "dark" | "system"
type ThemeToggleButtonVariant = "outline" | "ghost"
type ThemeToggleButtonSize = "sm" | "icon-sm"

const themeOptions: Array<{
  value: ThemeMode
  labelKey: string
  icon: typeof SunIcon
}> = [
  { value: "system", labelKey: "theme.system", icon: LaptopIcon },
  { value: "dark", labelKey: "theme.dark", icon: MoonIcon },
  { value: "light", labelKey: "theme.light", icon: SunIcon },
]

export function ThemeToggle({
  variant = "outline",
  size = "sm",
  className,
}: {
  variant?: ThemeToggleButtonVariant
  size?: ThemeToggleButtonSize
  className?: string
}) {
  const t = useI18n()
  const { theme, setTheme } = useTheme()
  const mounted = useSyncExternalStore(
    () => () => {},
    () => true,
    () => false
  )

  const activeTheme = mounted
    ? ((theme as ThemeMode | undefined) ?? "system")
    : "system"
  const ActiveIcon =
    themeOptions.find((option) => option.value === activeTheme)?.icon ??
    LaptopIcon

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        render={
          <Button
            variant={variant}
            size={size}
            className={cn(
              "rounded-md text-muted-foreground hover:text-foreground",
              className
            )}
          />
        }
        aria-label={t("theme.toggle")}
      >
        <ActiveIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40 min-w-40">
        <DropdownMenuRadioGroup
          value={activeTheme}
          onValueChange={(value) => setTheme(value as ThemeMode)}
        >
          {themeOptions.map((option) => {
            const Icon = option.icon
            return (
              <DropdownMenuRadioItem
                key={option.value}
                value={option.value}
                className="h-8 px-2"
              >
                <Icon />
                {t(option.labelKey)}
              </DropdownMenuRadioItem>
            )
          })}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
