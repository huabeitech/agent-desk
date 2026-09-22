"use client"

import {
  ChevronRightIcon,
  ChevronsUpDownIcon,
  Loader2Icon,
  TagIcon,
} from "lucide-react"
import { useMemo, useState, type ComponentProps } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useI18n } from "@/i18n/provider"
import type { TagTree } from "@/lib/api/admin"
import {
  buildTagPathMap,
  flattenTagTree,
  flattenVisibleTagTree,
  type FlatTagNode,
} from "@/lib/tag-tree"
import { cn } from "@/lib/utils"

type CommonTagSelectorProps = {
  tags: TagTree[]
  placeholder: string
  searchPlaceholder?: string
  emptyText?: string
  loadingText?: string
  disabled?: boolean
  loading?: boolean
  excludeIds?: number[]
  align?: "start" | "center" | "end"
  className?: string
  triggerClassName?: string
  triggerVariant?: ComponentProps<typeof Button>["variant"]
  triggerSize?: ComponentProps<typeof Button>["size"]
  contentClassName?: string
  showSelectedBadges?: boolean
  selectedCountText?: (count: number) => string
  triggerText?: string
  pendingTagId?: number | null
}

type MultipleTagSelectorProps = CommonTagSelectorProps & {
  mode: "multiple"
  value?: number[]
  onChange: (value: number[]) => void
}

type SingleTagSelectorProps = CommonTagSelectorProps & {
  mode: "single"
  value?: number | null
  onChange: (value: number) => void
  rootOption?: {
    value: number
    label: string
  }
}

export type TagSelectorProps = MultipleTagSelectorProps | SingleTagSelectorProps

function isSelected(props: TagSelectorProps, tagId: number) {
  if (props.mode === "single") {
    return props.value === tagId
  }
  return new Set(props.value ?? []).has(tagId)
}

function getSelectedTags(flatTags: FlatTagNode[], value?: number[]) {
  const selectedIds = new Set(value ?? [])
  return flatTags.filter((tag) => selectedIds.has(tag.id))
}

export function TagSelector(props: TagSelectorProps) {
  const t = useI18n()
  const [open, setOpen] = useState(false)
  const {
    tags,
    placeholder,
    searchPlaceholder = t("common.searchKeyword"),
    emptyText = t("common.emptyOptions"),
    loadingText = t("common.loading"),
    disabled = false,
    loading = false,
    excludeIds,
    align = "start",
    className,
    triggerClassName,
    triggerVariant = "outline",
    triggerSize,
    contentClassName,
    showSelectedBadges = props.mode === "multiple",
    selectedCountText,
    triggerText,
    pendingTagId = null,
  } = props
  const [query, setQuery] = useState("")
  const [collapsedIds, setCollapsedIds] = useState<Set<number>>(new Set())

  const flatTags = useMemo(
    () => flattenTagTree(tags, { excludeIds }),
    [excludeIds, tags]
  )
  const visibleFlatTags = useMemo(
    () => flattenVisibleTagTree(tags, { excludeIds, collapsedIds: [...collapsedIds] }),
    [collapsedIds, excludeIds, tags]
  )

  const selectedTags = useMemo(
    () => (props.mode === "multiple" ? getSelectedTags(flatTags, props.value) : []),
    [flatTags, props]
  )

  const rootTag = useMemo<FlatTagNode | null>(() => {
    if (props.mode !== "single" || !props.rootOption) {
      return null
    }
    return {
      id: props.rootOption.value,
      parentId: 0,
      name: props.rootOption.label,
      remark: "",
      sortNo: 0,
      status: 0,
      createdAt: "",
      updatedAt: "",
      children: [],
      depth: 0,
      path: props.rootOption.label,
      searchableText: `${props.rootOption.label} ${props.rootOption.value}`,
    }
  }, [props])

  const allSelectableTags = useMemo(
    () => (props.mode === "single" && rootTag ? [rootTag, ...flatTags] : flatTags),
    [flatTags, props.mode, rootTag]
  )
  const normalizedQuery = query.trim().toLowerCase()
  const visibleSelectableTags = useMemo(() => {
    const visibleTags = props.mode === "single" && rootTag
      ? [rootTag, ...visibleFlatTags]
      : visibleFlatTags

    if (!normalizedQuery) {
      return visibleTags
    }

    return allSelectableTags.filter((tag) =>
      tag.searchableText.toLowerCase().includes(normalizedQuery)
    )
  }, [allSelectableTags, normalizedQuery, props.mode, rootTag, visibleFlatTags])
  const singleSelected = props.mode === "single"
    ? allSelectableTags.find((tag) => tag.id === props.value)
    : null
  const triggerLabel =
    triggerText ??
    (props.mode === "multiple"
      ? selectedTags.length > 0
        ? selectedCountText?.(selectedTags.length) ?? `${placeholder} (${selectedTags.length})`
        : placeholder
      : singleSelected?.path ?? placeholder)

  function handleSelect(tagId: number) {
    if (props.mode === "single") {
      props.onChange(tagId)
      setOpen(false)
      return
    }

    const selectedIds = new Set(props.value ?? [])
    if (selectedIds.has(tagId)) {
      props.onChange((props.value ?? []).filter((item) => item !== tagId))
      return
    }
    props.onChange([...(props.value ?? []), tagId])
  }

  function toggleCollapsed(tagId: number) {
    setCollapsedIds((current) => {
      const next = new Set(current)
      if (next.has(tagId)) {
        next.delete(tagId)
      } else {
        next.add(tagId)
      }
      return next
    })
  }

  return (
    <div className={className}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              type="button"
              variant={triggerVariant}
              size={triggerSize}
              role="combobox"
              aria-expanded={open}
              disabled={disabled}
              className={cn("w-full justify-between font-normal", triggerClassName)}
            />
          }
        >
          <span className="flex min-w-0 items-center gap-2">
            <TagIcon className="size-4 shrink-0 text-muted-foreground" />
            <span className="truncate">{triggerLabel}</span>
          </span>
          <ChevronsUpDownIcon className="ml-2 size-4 shrink-0 opacity-50" />
        </PopoverTrigger>
        <PopoverContent
          align={align}
          className={cn("w-(--radix-popover-trigger-width) min-w-72 p-0", contentClassName)}
        >
          <Command shouldFilter={false}>
            <CommandInput
              value={query}
              onValueChange={setQuery}
              placeholder={searchPlaceholder}
            />
            <CommandList>
              {loading ? <CommandEmpty>{loadingText}</CommandEmpty> : null}
              {!loading && visibleSelectableTags.length === 0 ? (
                <CommandEmpty>{emptyText}</CommandEmpty>
              ) : null}
              {!loading ? (
                <CommandGroup>
                  {visibleSelectableTags.map((tag) => {
                    const checked = isSelected(props, tag.id)
                    const pending = pendingTagId === tag.id
                    const hasChildren = tag.children.length > 0
                    const collapsed = collapsedIds.has(tag.id)

                    return (
                      <CommandItem
                        key={tag.id}
                        value={tag.searchableText}
                        disabled={disabled || pendingTagId !== null}
                        onSelect={() => handleSelect(tag.id)}
                        className={cn(
                          props.mode === "single" &&
                            checked &&
                            "bg-muted text-foreground"
                        )}
                      >
                        <div
                          className="flex min-w-0 flex-1 items-center gap-1.5"
                          style={{ paddingLeft: `${tag.depth * 14}px` }}
                          title={tag.path}
                        >
                          {hasChildren && !normalizedQuery ? (
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon-sm"
                              className="size-5 shrink-0"
                              aria-label={collapsed ? t("common.expand") : t("common.collapse")}
                              onMouseDown={(event) => event.preventDefault()}
                              onClick={(event) => {
                                event.preventDefault()
                                event.stopPropagation()
                                toggleCollapsed(tag.id)
                              }}
                            >
                              <ChevronRightIcon
                                className={cn(
                                  "size-3.5 transition-transform",
                                  !collapsed && "rotate-90"
                                )}
                              />
                            </Button>
                          ) : (
                            <span className="size-5 shrink-0" />
                          )}
                          {pending ? (
                            <Loader2Icon className="size-4 shrink-0 animate-spin" />
                          ) : props.mode === "multiple" ? (
                            <Checkbox
                              checked={checked}
                              tabIndex={-1}
                              aria-hidden="true"
                              className="pointer-events-none"
                            />
                          ) : (
                            <span className="size-4 shrink-0" />
                          )}
                          <span className="min-w-0 flex-1 truncate">{tag.name}</span>
                        </div>
                      </CommandItem>
                    )
                  })}
                </CommandGroup>
              ) : null}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>

      {showSelectedBadges && props.mode === "multiple" && selectedTags.length > 0 ? (
        <TagBadges ids={props.value ?? []} tags={tags} className="mt-2" />
      ) : null}
    </div>
  )
}

type TagBadgesProps = {
  ids?: number[]
  tags: TagTree[]
  fallbackTags?: Array<{ id: number; name: string }>
  className?: string
}

export function TagBadges({
  ids,
  tags,
  fallbackTags = [],
  className,
}: TagBadgesProps) {
  const tagPathMap = buildTagPathMap(tags)
  const fallbackMap = new Map(fallbackTags.map((tag) => [tag.id, tag.name]))
  const safeIds = ids ?? []

  if (safeIds.length === 0) {
    return null
  }

  return (
    <div className={cn("flex flex-wrap items-center gap-1.5", className)}>
      {safeIds.map((id) => (
        <Badge
          key={id}
          variant="outline"
          className="max-w-full px-2 text-[12px] font-normal"
        >
          <span className="break-all">{tagPathMap.get(id) ?? fallbackMap.get(id) ?? `#${id}`}</span>
        </Badge>
      ))}
    </div>
  )
}
