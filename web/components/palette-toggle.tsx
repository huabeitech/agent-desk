"use client"

import { useEffect, useState } from "react"
import { CircleIcon, DropletsIcon, PaletteIcon } from "lucide-react"

import { useI18n } from "@/i18n/provider"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type PaletteMode = "plain" | "blue" | "green" | "gray"

const PALETTE_STORAGE_KEY = "dashboard_palette"
const DEFAULT_PALETTE: PaletteMode = "plain"

const paletteOptions: Array<{
  value: PaletteMode
  labelKey: string
  swatch: string
}> = [
  {
    value: "plain",
    labelKey: "palette.plain",
    swatch: "bg-[#2475fc]",
  },
  {
    value: "green",
    labelKey: "palette.green",
    swatch: "bg-teal-700",
  },
  {
    value: "gray",
    labelKey: "palette.gray",
    swatch: "bg-slate-500",
  },
  {
    value: "blue",
    labelKey: "palette.blue",
    swatch: "bg-blue-600",
  },
]

function readPalette(): PaletteMode {
  if (typeof window === "undefined") {
    return DEFAULT_PALETTE
  }

  const stored = window.localStorage.getItem(PALETTE_STORAGE_KEY)
  return stored === "plain" ||
    stored === "blue" ||
    stored === "green" ||
    stored === "gray"
    ? stored
    : DEFAULT_PALETTE
}

function applyPalette(value: PaletteMode) {
  document.documentElement.dataset.palette = value
  window.localStorage.setItem(PALETTE_STORAGE_KEY, value)
}

export function PaletteToggle() {
  const t = useI18n()
  const [palette, setPalette] = useState<PaletteMode>(DEFAULT_PALETTE)

  useEffect(() => {
    const paletteTask = window.setTimeout(() => {
      const storedPalette = readPalette()
      setPalette(storedPalette)
      applyPalette(storedPalette)
    }, 0)
    return () => window.clearTimeout(paletteTask)
  }, [])

  function handleChange(value: string) {
    const nextPalette: PaletteMode =
      value === "blue" || value === "green" || value === "gray"
        ? value
        : "plain"
    setPalette(nextPalette)
    applyPalette(nextPalette)
  }

  const ActiveIcon =
    palette === "plain"
      ? CircleIcon
      : palette === "green"
        ? DropletsIcon
        : PaletteIcon

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="outline" size="sm" />}
        aria-label={t("palette.toggle")}
      >
        <ActiveIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52 min-w-52">
        <DropdownMenuRadioGroup value={palette} onValueChange={handleChange}>
          {paletteOptions.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              <span className={`size-2.5 rounded-full ${option.swatch}`} />
              <span className="flex-1">{t(option.labelKey)}</span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
