"use client"

import type { ReactNode } from "react"

import { Bubble, BubbleContent } from "@/components/ui/bubble"
import { cn } from "@/lib/utils"

export type ConversationMessageVariant =
  | "customer"
  | "agent"
  | "ai"
  | "system"
  | "recalled"
  | "note"

type ConversationMessageBubbleProps = {
  variant: ConversationMessageVariant
  children: ReactNode
  className?: string
}

function getBubbleVariant(variant: ConversationMessageVariant) {
  switch (variant) {
    case "customer":
      return "secondary" as const
    case "system":
      return "muted" as const
    case "ai":
      return "tinted" as const
    case "agent":
      return "default" as const
    case "note":
      return "tinted" as const
    case "recalled":
      return "outline" as const
    default:
      return "secondary" as const
  }
}

function getContentClassName(variant: ConversationMessageVariant) {
  switch (variant) {
    case "agent":
      return "bg-emerald-600 text-white"
    case "note":
      return "border border-amber-500/30 bg-amber-500/10 text-amber-950 dark:text-amber-200"
    case "system":
      return "border-dashed text-muted-foreground"
    case "recalled":
      return "border-dashed bg-muted/40 text-muted-foreground"
    default:
      return undefined
  }
}

export function ConversationMessageBubble({
  variant,
  children,
  className,
}: ConversationMessageBubbleProps) {
  return (
    <Bubble variant={getBubbleVariant(variant)}>
      <BubbleContent className={cn("rounded-2xl px-4 py-3 leading-6", getContentClassName(variant), className)}>
        {children}
      </BubbleContent>
    </Bubble>
  )
}
