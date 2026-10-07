"use client"

import { useEffect, useRef, useState } from "react"
import Image from "next/image"
import { UploadIcon, XIcon } from "lucide-react"
import { toast } from "sonner"

import { uploadAsset } from "@/lib/api/admin"
import { useI18n } from "@/i18n/provider"
import { cn } from "@/lib/utils"

export type ImageInputProps = {
  value?: string
  onChange?: (value: string) => void
  disabled?: boolean
  accept?: string
  maxSize?: number
  prefix?: string
  previewValue?: string
  placeholder?: string
  className?: string
  upload?: (file: File, prefix?: string) => Promise<{ assetId?: string; url: string }>
}

export function ImageInput({
  value,
  onChange,
  disabled,
  accept = "image/*",
  maxSize = 5 * 1024 * 1024,
  prefix,
  previewValue,
  placeholder,
  className,
  upload = uploadAsset,
}: ImageInputProps) {
  const t = useI18n()
  const [uploading, setUploading] = useState(false)
  const [uploadedPreview, setUploadedPreview] = useState("")
  const fileInputRef = useRef<HTMLInputElement>(null)
  const resolvedPlaceholder = placeholder ?? t("upload.imagePlaceholder")

  useEffect(() => {
    setUploadedPreview("")
  }, [previewValue])

  function handleClick() {
    if (disabled || uploading) {
      return
    }
    fileInputRef.current?.click()
  }

  function handleClear(event: React.MouseEvent) {
    event.stopPropagation()
    setUploadedPreview("")
    onChange?.("")
  }

  async function handleFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }

    if (!file.type.startsWith("image/")) {
      toast.error(t("upload.chooseImage"))
      return
    }

    if (file.size > maxSize) {
      const maxSizeMB = (maxSize / 1024 / 1024).toFixed(0)
      toast.error(t("upload.imageTooLarge", { maxSize: maxSizeMB }))
      return
    }

    setUploading(true)
    try {
      const result = await upload(file, prefix)
      setUploadedPreview(result.url)
      onChange?.(result.assetId || result.url)
      toast.success(t("upload.imageUploaded"))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("upload.imageUploadFailed"))
    } finally {
      setUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
    }
  }

  const isDisabled = disabled || uploading

  return (
    <div className="relative w-fit">
      <input
        ref={fileInputRef}
        type="file"
        accept={accept}
        className="hidden"
        onChange={handleFileChange}
        disabled={isDisabled}
      />
      <div
        onClick={handleClick}
        className={cn(
          "group relative flex size-24 cursor-pointer items-center justify-center overflow-hidden rounded-lg border-2 border-dashed border-input bg-muted transition-colors",
          "hover:border-primary hover:bg-muted/50",
          "focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none",
          isDisabled && "cursor-not-allowed opacity-50",
          className
        )}
        tabIndex={isDisabled ? -1 : 0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault()
            handleClick()
          }
        }}
        role="button"
        aria-label={value ? t("upload.replaceImage") : resolvedPlaceholder}
      >
        {value ? (
          <>
            <Image
              src={uploadedPreview || previewValue || resolveImagePreviewURL(value)}
              alt={t("upload.uploadedImage")}
              fill
              sizes="96px"
              unoptimized
              className="object-cover"
            />
            <div className="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
              <span className="text-xs text-white">{t("upload.replaceImage")}</span>
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center gap-1 text-muted-foreground">
            <UploadIcon className="size-4" />
            <span className="text-xs">{uploading ? t("upload.uploading") : resolvedPlaceholder}</span>
          </div>
        )}
        {uploading && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/50">
            <div className="size-6 animate-spin rounded-full border-2 border-white border-t-transparent" />
          </div>
        )}
      </div>
      {value && !isDisabled && (
        <button
          type="button"
          onClick={handleClear}
          className="absolute -right-1.5 -top-1.5 flex size-5 items-center justify-center rounded-full bg-red-600 text-white shadow-sm transition-colors hover:bg-red-700 focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none"
          aria-label={t("upload.deleteImage")}
        >
          <XIcon className="size-3" />
        </button>
      )}
    </div>
  )
}

function resolveImagePreviewURL(value: string) {
  if (/^https?:\/\//i.test(value) || value.startsWith("/")) {
    return value
  }
  return ""
}
