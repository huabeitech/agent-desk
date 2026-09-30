"use client"

import { createElement } from "react"

import {
  Building2Icon,
  Gamepad2Icon,
  MessagesSquareIcon,
  MessageSquareMoreIcon,
  SendIcon,
  type LucideIcon,
} from "lucide-react"

type Translate = (key: string, values?: Record<string, string | number>) => string

// CHANNEL_TYPE_LABEL_KEYS 为渠道类型后端标识到 i18n 键的映射。
const CHANNEL_TYPE_LABEL_KEYS: Record<string, string> = {
  web: "channel.typeWeb",
  wechat_mp: "channel.typeWechatMp",
  wxwork_kf: "channel.typeWxworkKf",
  telegram: "channel.typeTelegram",
  zalo_oa: "channel.typeZaloOa",
  discord: "channel.typeDiscord",
}

// getChannelTypeLabel 返回渠道类型的本地化显示名；未知类型回退为原始标识。
export function getChannelTypeLabel(channelType: string, t: Translate): string {
  const key = CHANNEL_TYPE_LABEL_KEYS[channelType]
  return key ? t(key) : channelType
}

// CHANNEL_TYPE_ICONS 为渠道类型到 Lucide 图标的映射。
const CHANNEL_TYPE_ICONS: Record<string, LucideIcon> = {
  web: Building2Icon,
  wechat_mp: MessagesSquareIcon,
  wxwork_kf: MessageSquareMoreIcon,
  telegram: SendIcon,
  zalo_oa: SendIcon,
  discord: Gamepad2Icon,
}

export function ChannelTypeIcon({
  channelType,
  className,
}: {
  channelType: string
  className?: string
}) {
  const Icon = CHANNEL_TYPE_ICONS[channelType] ?? Building2Icon
  return createElement(Icon, { className, "aria-hidden": true })
}
