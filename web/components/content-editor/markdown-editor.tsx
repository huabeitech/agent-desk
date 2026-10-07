"use client"

import {
  type ComponentProps,
  forwardRef,
  useId,
  useImperativeHandle,
  useMemo,
  useRef,
} from "react"
import { Maximize2Icon, Minimize2Icon } from "lucide-react"
import { config, MdEditor, NormalToolbar, type ExposeParam } from "md-editor-rt"
import { useTheme } from "@/components/theme-provider"

import "./markdown-editor.css"

import { EditorModeSwitch } from "./editor-mode-switch"
import type { ContentMode, UploadImageHandler } from "./types"
import { useI18n } from "@/i18n/provider"
import { cn } from "@/lib/utils"

config({
  codeMirrorExtensions(extensions) {
    return extensions.filter((extension) => extension.type !== "linkShortener")
  },
})

export type MarkdownEditorRef = {
  focus: () => void
}

type MarkdownEditorProps = {
  value: string
  onChange: (nextValue: string) => void
  mode: ContentMode
  allowedModes: ReadonlyArray<ContentMode>
  onModeChange: (nextMode: ContentMode) => void
  fullscreen: boolean
  onToggleFullscreen: () => void
  placeholder?: string
  disabled?: boolean
  onUploadImage?: UploadImageHandler
  height: string
  scrollMode: "editor" | "document"
}

type MdEditorToolbars = NonNullable<ComponentProps<typeof MdEditor>["toolbars"]>

export const MarkdownEditor = forwardRef<MarkdownEditorRef, MarkdownEditorProps>(
  function MarkdownEditor(
    {
      value,
      onChange,
      mode,
      allowedModes,
      onModeChange,
      fullscreen,
      onToggleFullscreen,
      placeholder = "",
      disabled = false,
      onUploadImage,
      height,
      scrollMode,
    },
    ref
  ) {
    const t = useI18n()
    const editorId = useId()
    const editorRef = useRef<ExposeParam>(null)
    const { resolvedTheme } = useTheme()
    const showModeSwitch = allowedModes.length > 1
    const defToolbars = useMemo(
      () => {
        const fullscreenToolbar = (
          <NormalToolbar
            key="toggle-fullscreen"
            title={fullscreen ? t("editor.exitFullscreen") : t("editor.fullscreen")}
            disabled={disabled}
            onClick={onToggleFullscreen}
          >
            {fullscreen ? (
              <Minimize2Icon className="h-[16px] w-[16px]" />
            ) : (
              <Maximize2Icon className="h-[16px] w-[16px]" />
            )}
          </NormalToolbar>
        )

        if (!showModeSwitch) {
          return [fullscreenToolbar]
        }

        return [
          <EditorModeSwitch
            key="mode-switch"
            value={mode}
            allowedModes={allowedModes}
            disabled={disabled}
            onChange={onModeChange}
          />,
          fullscreenToolbar,
        ]
      },
      [allowedModes, disabled, fullscreen, mode, onModeChange, onToggleFullscreen, showModeSwitch, t]
    )
    const toolbars = useMemo(
      () => [
        ...(showModeSwitch ? [0, "-"] : []),
        "bold",
        "underline",
        "italic",
        "strikeThrough",
        "-",
        "title",
        "quote",
        "unorderedList",
        "orderedList",
        "-",
        "codeRow",
        "code",
        "link",
        "image",
        "-",
        "revoke",
        "next",
        showModeSwitch ? 1 : 0,
        "preview",
        "previewOnly",
      ] as MdEditorToolbars,
      [showModeSwitch]
    )

    useImperativeHandle(ref, () => ({
      focus() {
        editorRef.current?.focus()
      },
    }))
    return (
      <div
        className={cn("w-full rounded-lg border bg-background", scrollMode === "document" && "flex min-h-full flex-col")}
        style={scrollMode === "document" ? undefined : { height }}
      >
        <div className={cn("content-editor-markdown", scrollMode === "editor" && "h-full", scrollMode === "document" && "content-editor-markdown-document flex min-h-full flex-1 flex-col")}>
          <MdEditor
            ref={editorRef}
            id={editorId}
            value={value}
            onChange={onChange}
            theme={resolvedTheme === "dark" ? "dark" : "light"}
            preview={false}
            toolbars={toolbars}
            defToolbars={defToolbars}
            footers={[]}
            noMermaid
            noKatex
            noHighlight
            placeholder={placeholder}
            disabled={disabled}
            style={scrollMode === "document" ? undefined : { height: "100%" }}
            onUploadImg={
              onUploadImage
                ? async (files, callback) => {
                    const uploadedUrls: string[] = []
                    for (const file of files) {
                      const uploaded = await onUploadImage(file)
                      if (uploaded?.url) {
                        uploadedUrls.push(uploaded.url)
                      }
                    }
                    callback(uploadedUrls)
                  }
                : undefined
            }
          />
        </div>
      </div>
    )
  }
)
