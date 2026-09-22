import { DEFAULT_LOCALE, type AppLocale, readStoredLocale } from "@/i18n/config"
import enUSMessages from "@/messages/en-US.json"
import viVNMessages from "@/messages/vi-VN.json"
import zhCNMessages from "@/messages/zh-CN.json"

const messages = {
  "zh-CN": zhCNMessages,
  "en-US": enUSMessages,
  "vi-VN": viVNMessages,
} satisfies Record<AppLocale, typeof zhCNMessages>

export function translateMessage(
  locale: AppLocale,
  key: string,
  values?: Record<string, string | number>
): string {
  const value = getMessageValue(messages[locale], key)
  if (typeof value === "string") {
    return formatMessage(value, values)
  }
  const fallback = getMessageValue(messages[DEFAULT_LOCALE], key)
  return typeof fallback === "string" ? formatMessage(fallback, values) : key
}

export function translateCurrentMessage(
  key: string,
  values?: Record<string, string | number>
): string {
  return translateMessage(readStoredLocale(), key, values)
}

function getMessageValue(source: unknown, key: string): unknown {
  let current = source
  for (const part of key.split(".")) {
    if (!current || typeof current !== "object" || !(part in current)) {
      return undefined
    }
    current = (current as Record<string, unknown>)[part]
  }
  return current
}

function formatMessage(
  message: string,
  values?: Record<string, string | number>
): string {
  if (!values) {
    return message
  }
  return message.replace(/\{(\w+)\}/g, (match, key) =>
    Object.prototype.hasOwnProperty.call(values, key) ? String(values[key]) : match
  )
}
