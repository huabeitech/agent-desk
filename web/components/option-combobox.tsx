"use client"

import { useState, type ReactNode } from "react"
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react"

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
import { cn } from "@/lib/utils"
import { useI18n } from "@/i18n/provider"

export type ComboboxOption = {
  value: string
  label: string
  disabled?: boolean
  group?: string
  subtitle?: string
  description?: string
}

type CommonOptionComboboxProps = {
  options: ComboboxOption[]
  placeholder: string
  searchPlaceholder?: string
  emptyText?: string
  disabled?: boolean
  triggerClassName?: string
  preserveExternalSelection?: boolean
  renderOptionAction?: (option: ComboboxOption) => ReactNode
}

type OptionComboboxProps = CommonOptionComboboxProps &
  (
    | {
        multiple?: false
        value: string
        onChange: (value: string) => void
        values?: never
        onValuesChange?: never
      }
    | {
        multiple: true
        values: string[]
        onValuesChange: (values: string[]) => void
        value?: never
        onChange?: never
      }
  )

export function OptionCombobox(props: OptionComboboxProps) {
  const {
    options,
    placeholder,
    searchPlaceholder,
    emptyText,
    disabled = false,
    triggerClassName,
    preserveExternalSelection = false,
    renderOptionAction,
  } = props
  const t = useI18n()
  const [open, setOpen] = useState(false)
  const selectedValues = props.multiple ? props.values : [props.value]
  const selectedOptions = options.filter((option) =>
    selectedValues.includes(option.value)
  )
  const selectedLabel =
    selectedOptions.length === 0
      ? placeholder
      : selectedOptions.length === 1
        ? selectedOptions[0].label
        : t("common.selectedCount", { count: selectedOptions.length })
  const optionGroups = Array.from(
    options.reduce((groups, option) => {
      const group = option.group ?? ""
      groups.set(group, [...(groups.get(group) ?? []), option])
      return groups
    }, new Map<string, ComboboxOption[]>()),
  )

  function selectOption(optionValue: string) {
    const option = options.find((item) => item.value === optionValue)
    if (option?.disabled) {
      return
    }
    if (props.multiple) {
      props.onValuesChange(
        props.values.includes(optionValue)
          ? props.values.filter((value) => value !== optionValue)
          : [...props.values, optionValue]
      )
      return
    }
    props.onChange(optionValue)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            role="combobox"
            className={cn("m-0 w-full justify-between font-normal", triggerClassName)}
            disabled={disabled}
          />
        }
      >
        <span className="truncate">{selectedLabel}</span>
        <ChevronsUpDownIcon className="ml-2 size-4 shrink-0 opacity-50" />
      </PopoverTrigger>
      <PopoverContent
        className="w-(--radix-popover-trigger-width) p-0"
        align="start"
        data-workflow-preserve-selection={preserveExternalSelection ? true : undefined}
      >
        <Command>
          <CommandInput placeholder={searchPlaceholder ?? t("common.searchKeyword")} />
          <CommandList>
            <CommandEmpty>{emptyText ?? t("common.emptyOptions")}</CommandEmpty>
            {optionGroups.map(([group, groupOptions]) => (
              <CommandGroup key={group || "__default"} heading={group || undefined}>
                {groupOptions.map((option) => (
                <CommandItem
                  key={option.value}
                  value={`${option.group ?? ""} ${option.label} ${option.value} ${option.subtitle ?? ""} ${option.description ?? ""}`}
                  disabled={option.disabled}
                  onSelect={() => selectOption(option.value)}
                >
                  <div className="flex min-w-0 flex-1 items-center justify-between gap-2">
                    <div className="flex min-w-0 items-start">
                      {props.multiple ? (
                        <Checkbox
                          checked={selectedValues.includes(option.value)}
                          className="pointer-events-none mr-2 mt-0.5"
                          tabIndex={-1}
                        />
                      ) : (
                        <CheckIcon
                          className={cn(
                            "mr-2 mt-0.5 size-4 shrink-0",
                            option.value === props.value ? "opacity-100" : "opacity-0"
                          )}
                        />
                      )}
                      <span className="min-w-0">
                        <span className="block truncate">{option.label}</span>
                        {option.subtitle ? (
                          <span className="mt-0.5 block truncate font-mono text-[11px] leading-4 text-slate-500">
                            {option.subtitle}
                          </span>
                        ) : null}
                        {option.description ? (
                          <span className="mt-0.5 line-clamp-2 text-xs leading-4 text-muted-foreground">
                            {option.description}
                          </span>
                        ) : null}
                      </span>
                    </div>
                    {renderOptionAction ? (
                      <div
                        className="shrink-0"
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={(event) => event.stopPropagation()}
                      >
                        {renderOptionAction(option)}
                      </div>
                    ) : null}
                  </div>
                </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
