import { request } from "@/lib/api/client"
import type { PageResult } from "@/lib/api/admin"

export type AdminProduct = {
  id: number
  name: string
  category: string
  priceMin: number
  priceMax: number
  sellingPoints: string
  suitablePeople: string
  unsuitablePeople: string
  scenarios: string
  specs: string
  industryAttributes: string
  imageUrl: string
  priority: number
  knowledgeBaseId: number
  knowledgeFAQId: number
  status: number
  remark: string
  createdAt: string
  updatedAt: string
}

export type SaveProductPayload = {
  name: string
  category: string
  priceMin: number
  priceMax: number
  sellingPoints: string
  suitablePeople: string
  unsuitablePeople: string
  scenarios: string
  specs: string
  industryAttributes: string
  imageUrl: string
  priority: number
  knowledgeBaseId: number
  status: number
  remark: string
}

export type UpdateProductPayload = SaveProductPayload & {
  id: number
}

export type ProductImportResult = {
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

export function fetchProducts(query?: Record<string, string | number | undefined>) {
  return request<PageResult<AdminProduct>>("/api/dashboard/product/list", {
    method: "POST",
    body: JSON.stringify(query ?? {}),
  })
}

export function fetchProduct(id: number) {
  return request<AdminProduct>(`/api/dashboard/product/${id}`)
}

export function createProduct(payload: SaveProductPayload) {
  return request<AdminProduct>("/api/dashboard/product/create", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function updateProduct(payload: UpdateProductPayload) {
  return request<void>("/api/dashboard/product/update", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function updateProductStatus(id: number, status: number) {
  return request<void>("/api/dashboard/product/update_status", {
    method: "POST",
    body: JSON.stringify({ id, status }),
  })
}

export function deleteProduct(id: number) {
  return request<void>("/api/dashboard/product/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  })
}

export function reindexProduct(id: number) {
  return request<void>("/api/dashboard/product/reindex", {
    method: "POST",
    body: JSON.stringify({ id }),
  })
}

export function importProducts(file: File) {
  const form = new FormData()
  form.append("file", file)
  return request<ProductImportResult>("/api/dashboard/product/import", {
    method: "POST",
    body: form,
  })
}

export function seedMuseProducts() {
  return request<void>("/api/dashboard/product/seed_muse", {
    method: "POST",
    body: JSON.stringify({}),
  })
}
