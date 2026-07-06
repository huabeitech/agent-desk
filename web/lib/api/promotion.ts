import { request } from "@/lib/api/client"
import type { PageResult } from "@/lib/api/admin"

export type AdminPromotion = {
  id: number
  name: string
  promotionType: string
  description: string
  applicableProducts: string
  startAt?: string
  endAt?: string
  discountRule: string
  storeBenefit: string
  appointmentBenefit: string
  scriptSuggestion: string
  priority: number
  knowledgeBaseId: number
  knowledgeFAQId: number
  status: number
  remark: string
  createdAt: string
  updatedAt: string
}

export type SavePromotionPayload = {
  name: string
  promotionType: string
  description: string
  applicableProducts: string
  startAt: string
  endAt: string
  discountRule: string
  storeBenefit: string
  appointmentBenefit: string
  scriptSuggestion: string
  priority: number
  knowledgeBaseId: number
  status: number
  remark: string
}

export type UpdatePromotionPayload = SavePromotionPayload & {
  id: number
}

export type PromotionImportResult = {
  total: number
  created: number
  updated: number
  skipped: number
  failed: number
  errors: Array<{
    row: number
    message: string
  }>
}

export function fetchPromotions(query?: Record<string, string | number | undefined>) {
  return request<PageResult<AdminPromotion>>("/api/dashboard/promotion/list", {
    method: "POST",
    body: JSON.stringify(query ?? {}),
  })
}

export function fetchPromotion(id: number) {
  return request<AdminPromotion>(`/api/dashboard/promotion/${id}`)
}

export function createPromotion(payload: SavePromotionPayload) {
  return request<AdminPromotion>("/api/dashboard/promotion/create", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function updatePromotion(payload: UpdatePromotionPayload) {
  return request<void>("/api/dashboard/promotion/update", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function updatePromotionStatus(id: number, status: number) {
  return request<void>("/api/dashboard/promotion/update_status", {
    method: "POST",
    body: JSON.stringify({ id, status }),
  })
}

export function deletePromotion(id: number) {
  return request<void>("/api/dashboard/promotion/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  })
}

export function reindexPromotion(id: number) {
  return request<void>("/api/dashboard/promotion/reindex", {
    method: "POST",
    body: JSON.stringify({ id }),
  })
}

export function importPromotions(file: File) {
  const form = new FormData()
  form.append("file", file)
  return request<PromotionImportResult>("/api/dashboard/promotion/import", {
    method: "POST",
    body: form,
  })
}

export function seedMusePromotions() {
  return request<void>("/api/dashboard/promotion/seed_muse", {
    method: "POST",
    body: JSON.stringify({}),
  })
}
