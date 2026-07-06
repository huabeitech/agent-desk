import { request, requestBlob } from "@/lib/api/client"
import type { AdminCustomer } from "@/lib/api/customer"
import type { PageResult } from "@/lib/api/admin"

export type SalesLeadIntent = "unknown" | "low" | "medium" | "high"
export type SalesLeadStage =
  | "unknown"
  | "consulting"
  | "comparing"
  | "appointment"
  | "ready_to_buy"
  | "after_sales"
export type SalesLeadStatus =
  | "new"
  | "following"
  | "visited"
  | "converted"
  | "invalid"
  | "closed"

export type AdminSalesLead = {
  id: number
  customerId: number
  conversationId: number
  customerName: string
  phone: string
  wechat: string
  city: string
  addressHint: string
  budgetMin: number
  budgetMax: number
  interestedProducts: string
  demandSummary: string
  intentLevel: SalesLeadIntent
  buyingStage: SalesLeadStage
  appointmentAt?: string
  appointmentTimeText: string
  appointmentStore: string
  appointmentPeople: number
  appointmentRemark: string
  sourceChannel: string
  ownerUserId: number
  ownerUserName?: string
  status: SalesLeadStatus
  nextFollowUpAt?: string
  lastMessageId: number
  lastMessageSummary: string
  lastCustomerMessage: string
  mergeKey: string
  mergeReason: string
  mergedAt?: string
  remark: string
  autoTags: string[]
  autoTagDetails: SalesLeadAutoTag[]
  createdAt: string
  updatedAt: string
  customer?: AdminCustomer
}

export type SalesLeadAutoTag = {
  label: string
  level: string
  reason: string
  actionLabel: string
  actionUrl?: string
}

export type LeadFollowUp = {
  id: number
  leadId: number
  operatorId: number
  operatorName: string
  content: string
  nextAction: string
  nextFollowUpAt?: string
  createdAt?: string
}

export type SalesLeadFollowUpAdvice = {
  customerSummary: string
  nextAction: string
  script: string
  copyText: string
  riskHints: string[]
}

export type SalesLeadDetail = {
  lead: AdminSalesLead
  followUps?: LeadFollowUp[]
  followUpAdvice?: SalesLeadFollowUpAdvice
}

export type SalesLeadFollowUpReminderLead = {
  id: number
  customerName: string
  phone: string
  wechat: string
  intentLevel: SalesLeadIntent
  status: SalesLeadStatus
  ownerUserId: number
  ownerUserName?: string
  nextFollowUpAt?: string
  followUpState: "overdue" | "today" | "scheduled" | "none" | string
  demandSummary: string
  actionUrl: string
}

export type SalesLeadFollowUpReminderSummary = {
  generatedAt: string
  overdueCount: number
  todayCount: number
  dueCount: number
  unassignedDueCount: number
  missingScheduleCount: number
  previewLeads: SalesLeadFollowUpReminderLead[]
  message: string
  notificationSent: boolean
}

export type SalesLeadAppointmentItem = {
  id: number
  customerName: string
  phone: string
  wechat: string
  intentLevel: SalesLeadIntent
  status: SalesLeadStatus
  ownerUserId: number
  ownerUserName?: string
  appointmentAt?: string
  appointmentTimeText: string
  appointmentStore: string
  appointmentPeople: number
  demandSummary: string
  appointmentState: "overdue" | "today" | "upcoming" | "unscheduled" | string
  actionUrl: string
}

export type SalesLeadAppointmentSummary = {
  generatedAt: string
  days: number
  overdueCount: number
  todayCount: number
  upcomingCount: number
  unscheduledCount: number
  unassignedCount: number
  previewAppointments: SalesLeadAppointmentItem[]
  message: string
  notificationSent: boolean
}

export type ClaimUnassignedSalesLeadsRequest = {
  keyword?: string
  status?: string
  intent?: string
  taskView?: string
  followUpStatus?: string
  appointmentStatus?: string
  limit?: number
}

export type ClaimUnassignedSalesLeadsResult = {
  claimedCount: number
  leadIds: number[]
  message: string
}

export type SalesLeadCRMSyncResult = {
  leadId: number
  generatedAt: string
  webhookEnabled: boolean
  sent: boolean
  title: string
  message: string
  webhookEventType: string
}

export type SalesLeadListRequest = {
  page: number
  limit: number
  keyword?: string
  status?: string
  intent?: string
  taskView?: string
  followUpStatus?: string
  appointmentStatus?: string
  ownerUserId?: number
}

export type UpdateSalesLeadPayload = {
  id: number
  customerName: string
  phone: string
  wechat: string
  city: string
  addressHint: string
  budgetMin: number
  budgetMax: number
  interestedProducts: string
  demandSummary: string
  intentLevel: SalesLeadIntent
  buyingStage: SalesLeadStage
  appointmentAt: string
  appointmentTimeText: string
  appointmentStore: string
  appointmentPeople: number
  appointmentRemark: string
  ownerUserId: number
  status: SalesLeadStatus
  remark: string
}

export type UpdateSalesLeadStatusPayload = {
  id: number
  status: SalesLeadStatus
  remark?: string
}

export type CreateLeadFollowUpPayload = {
  leadId: number
  content: string
  nextAction: string
  nextFollowUpAt: string
}

export type SalesLeadFollowUpReminderRequest = {
  ownerUserId?: number
  limit?: number
}

export type SalesLeadAppointmentSummaryRequest = {
  ownerUserId?: number
  days?: number
  limit?: number
}

export function fetchSalesLeads(body: SalesLeadListRequest) {
  return request<PageResult<AdminSalesLead>>("/api/dashboard/sales-lead/list", {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export function fetchSalesLeadDetail(id: number) {
  return request<SalesLeadDetail>(`/api/dashboard/sales-lead/${id}`)
}

export function updateSalesLead(body: UpdateSalesLeadPayload) {
  return request<void>("/api/dashboard/sales-lead/update", {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export function updateSalesLeadStatus(body: UpdateSalesLeadStatusPayload) {
  return request<void>("/api/dashboard/sales-lead/update-status", {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export function syncSalesLeadToCRM(id: number, remark?: string) {
  return request<SalesLeadCRMSyncResult>("/api/dashboard/sales-lead/crm/sync", {
    method: "POST",
    body: JSON.stringify({ id, remark }),
  })
}

export function claimUnassignedSalesLeads(
  body: ClaimUnassignedSalesLeadsRequest = {}
) {
  return request<ClaimUnassignedSalesLeadsResult>(
    "/api/dashboard/sales-lead/claim-unassigned",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  )
}

export function createLeadFollowUp(body: CreateLeadFollowUpPayload) {
  return request<LeadFollowUp>("/api/dashboard/sales-lead/follow-up/create", {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export function fetchSalesLeadFollowUpReminderSummary(
  body: SalesLeadFollowUpReminderRequest = {}
) {
  return request<SalesLeadFollowUpReminderSummary>(
    "/api/dashboard/sales-lead/follow-up/reminder/summary",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  )
}

export function fetchSalesLeadAppointmentSummary(
  body: SalesLeadAppointmentSummaryRequest = {}
) {
  return request<SalesLeadAppointmentSummary>(
    "/api/dashboard/sales-lead/appointment/summary",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  )
}

export function sendSalesLeadAppointmentReminder(
  body: SalesLeadAppointmentSummaryRequest = {}
) {
  return request<SalesLeadAppointmentSummary>(
    "/api/dashboard/sales-lead/appointment/reminder/send",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  )
}

export function sendSalesLeadFollowUpReminder(
  body: SalesLeadFollowUpReminderRequest = {}
) {
  return request<SalesLeadFollowUpReminderSummary>(
    "/api/dashboard/sales-lead/follow-up/reminder/send",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  )
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

function toQueryString(values: Partial<SalesLeadListRequest>) {
  const params = new URLSearchParams()
  Object.entries(values).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "" || value === "all") {
      return
    }
    params.set(key, String(value))
  })
  const query = params.toString()
  return query ? `?${query}` : ""
}

export async function exportSalesLeads(body: Partial<SalesLeadListRequest>) {
  const result = await requestBlob(
    `/api/dashboard/sales-lead/export${toQueryString(body)}`
  )
  downloadBlob(result.blob, result.filename || "sales-leads.csv")
}
