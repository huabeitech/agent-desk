"use client"

import { useCallback, useEffect, useState, useSyncExternalStore } from "react"
import { createPortal } from "react-dom"

import { cn } from "@/lib/utils"

import { htmlToMarkdown, markdownToHtml } from "./convert"
import { HtmlEditor } from "./html-editor"
import { MarkdownEditor } from "./markdown-editor"
import {
  CONTENT_MODE_OPTIONS,
  type ContentMode,
  type ContentValue,
  type UploadImageHandler,
} from "./types"
import { useI18n } from "@/i18n/provider"

type ContentEditorProps = {
  value: ContentValue
  onChange: (next: ContentValue) => void
  placeholder?: string
  disabled?: boolean
  onUploadImage?: UploadImageHandler
  height?: number | string
  allowedModes?: ReadonlyArray<ContentMode>
}

function normalizeHeight(height?: number | string) {
  if (typeof height === "number") {
    return `${height}px`
  }
  if (typeof height === "string" && height.trim()) {
    return height
  }
  return "400px"
}

function getModeLabel(mode: ContentMode) {
  return mode === "markdown" ? "Markdown" : "HTML"
}

function convertContent(mode: ContentMode, raw: string) {
  if (mode === "markdown") {
    return markdownToHtml(raw)
  }
  return htmlToMarkdown(raw)
}

function subscribeMounted() {
  return () => undefined
}

export function ContentEditor({
  value,
  onChange,
  placeholder,
  disabled = false,
  onUploadImage,
  height,
  allowedModes = CONTENT_MODE_OPTIONS,
}: ContentEditorProps) {
  const t = useI18n()
  const editorHeight = normalizeHeight(height)
  const [fullscreen, setFullscreen] = useState(false)
  const mounted = useSyncExternalStore(subscribeMounted, () => true, () => false)
  const normalizedAllowedModes = allowedModes.length > 0 ? allowedModes : CONTENT_MODE_OPTIONS
  const activeMode = normalizedAllowedModes.includes(value.mode)
    ? value.mode
    : normalizedAllowedModes[0]

  useEffect(() => {
    if (!fullscreen) {
      return
    }

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = "hidden"

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setFullscreen(false)
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener("keydown", handleKeyDown)
    }
  }, [fullscreen])

  const handleModeChange = useCallback(
    (nextMode: ContentMode) => {
      if (
        disabled ||
        normalizedAllowedModes.length <= 1 ||
        nextMode === activeMode ||
        !normalizedAllowedModes.includes(nextMode)
      ) {
        return
      }
      const currentText = value.raw.trim()
      if (!currentText) {
        onChange({ mode: nextMode, raw: "" })
        return
      }

      const confirmed = window.confirm(
        t("editor.modeSwitchConfirm", { mode: getModeLabel(nextMode) })
      )
      if (!confirmed) {
        return
      }

      onChange({
        mode: nextMode,
        raw: convertContent(activeMode, value.raw),
      })
    },
    [activeMode, disabled, normalizedAllowedModes, onChange, t, value.raw]
  )

  useEffect(() => {
    if (value.mode !== activeMode) {
      onChange({ mode: activeMode, raw: value.raw })
    }
  }, [activeMode, onChange, value.mode, value.raw])

  const content = (
    <div
      className={cn(
        "w-full",
        fullscreen && "fixed inset-0 z-[10000] overflow-hidden bg-background p-4"
      )}
    >
      {activeMode === "markdown" ? (
        <MarkdownEditor
          value={value.raw}
          onChange={(nextRaw) => onChange({ mode: "markdown", raw: nextRaw })}
          mode={activeMode}
          allowedModes={normalizedAllowedModes}
          onModeChange={handleModeChange}
          fullscreen={fullscreen}
          onToggleFullscreen={() => setFullscreen((current) => !current)}
          placeholder={placeholder}
          disabled={disabled}
          onUploadImage={onUploadImage}
          height={fullscreen ? "calc(100vh - 2rem)" : editorHeight}
        />
      ) : (
        <HtmlEditor
          value={value.raw}
          onChange={(nextRaw) => onChange({ mode: "html", raw: nextRaw })}
          mode={activeMode}
          allowedModes={normalizedAllowedModes}
          onModeChange={handleModeChange}
          fullscreen={fullscreen}
          onToggleFullscreen={() => setFullscreen((current) => !current)}
          placeholder={placeholder}
          disabled={disabled}
          onUploadImage={onUploadImage}
          height={fullscreen ? "calc(100vh - 2rem)" : editorHeight}
        />
      )}
    </div>
  )

  if (fullscreen && mounted) {
    return createPortal(content, document.body)
  }

  return content
}

export type { ContentMode, ContentValue, UploadImageHandler, UploadImageResult } from "./types"
