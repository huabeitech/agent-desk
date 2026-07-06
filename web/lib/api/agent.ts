import { request } from "@/lib/api/client"

export type Paging = {
  page: number
  limit: number
  total: number
}

export type PageResult<T> = {
  results: T[]
  page: Paging
}

export type CursorResult<T> = {
  results: T[]
  cursor: string
  hasMore: boolean
}

export type AgentConversationTag = {
  id: number
  name: string
}

export type AgentConversationParticipant = {
  id: number
  participantType: string
  participantId: number
  externalParticipantId?: string
  joinedAt?: string
  leftAt?: string
  status: number
}

export type AgentConversation = {
  id: number
  aiAgentId?: number
  channelId?: number
  customerId?: number
  customerName: string
  status: number
  serviceMode: number
  priority: number
  currentAssigneeId: number
  currentAssigneeName?: string
  currentTeamId: number
  currentTeamName?: string
  lastMessageId: number
  lastMessageAt?: string
  lastActiveAt?: string
  lastMessageSummary?: string
  customerUnreadCount: number
  agentUnreadCount: number
  customerLastReadMessageId: number
  customerLastReadAt?: string
  agentLastReadMessageId: number
  agentLastReadAt?: string
  customerOnline: boolean
  closedAt?: string
  tags?: AgentConversationTag[]
  participants?: AgentConversationParticipant[]
}

export type AgentConversationDetail = AgentConversation & {
  participants?: AgentConversationParticipant[]
}

export type AgentMessage = {
  id: number
  conversationId: number
  workflowRunId?: number
  clientMsgId?: string
  senderType: string
  senderId: number
  senderName?: string
  senderAvatar?: string
  messageType: string
  content: string
  payload?: string
  sendStatus: number
  sentAt?: string
  deliveredAt?: string
  readAt?: string
  customerRead: boolean
  customerReadAt?: string
  agentRead: boolean
  agentReadAt?: string
  recalledAt?: string
  quotedMessageId?: number
}

export type AgentAsset = {
  id: number
  assetId: string
  provider: string
  storageKey: string
  filename: string
  fileSize: number
  mimeType: string
  status: number
  url: string
  createdAt: string
  updatedAt: string
  createUserId: number
  createUserName: string
  updateUserId: number
  updateUserName: string
}

function toQueryString(query?: Record<string, string | number | undefined>) {
  if (!query) {
    return ""
  }

  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === "") {
      return
    }
    params.set(key, String(value))
  })
  const output = params.toString()
  return output ? `?${output}` : ""
}

export function fetchAgentConversations(
  query?: Record<string, string | number | undefined>
) {
  return request<PageResult<AgentConversation>>(
    `/api/dashboard/conversation/conversations${toQueryString(query)}`
  )
}

export function fetchAgentConversationDetail(id: number) {
  return request<AgentConversationDetail>(`/api/dashboard/conversation/${id}`)
}

export function fetchAgentMessages(
  query?: Record<string, string | number | undefined>
) {
  return request<CursorResult<AgentMessage>>(
    `/api/dashboard/conversation/message_list${toQueryString(query)}`
  )
}

export function sendAgentMessage(payload: {
  conversationId: number
  messageType: string
  content: string
  payload?: string
  clientMsgId?: string
}) {
  return request<AgentMessage>("/api/dashboard/conversation/send_message", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function recallAgentMessage(messageId: number) {
  return request<AgentMessage>("/api/dashboard/conversation/recall_message", {
    method: "POST",
    body: JSON.stringify({ messageId }),
  })
}

export function markAgentMessageRead(conversationId: number, messageId = 0) {
  return request<void>("/api/dashboard/conversation/read", {
    method: "POST",
    body: JSON.stringify({ conversationId, messageId }),
  })
}

export function uploadAgentConversationImage(conversationId: number, file: File) {
  const formData = new FormData()
  formData.set("conversationId", String(conversationId))
  formData.set("file", file)
  return request<AgentAsset>("/api/dashboard/conversation/upload_image", {
    method: "POST",
    body: formData,
  })
}

export function uploadAgentConversationAttachment(conversationId: number, file: File) {
  const formData = new FormData()
  formData.set("conversationId", String(conversationId))
  formData.set("file", file)
  return request<AgentAsset>("/api/dashboard/conversation/upload_attachment", {
    method: "POST",
    body: formData,
  })
}

export function closeAgentConversation(
  conversationId: number,
  closeReason: string
) {
  return request<void>("/api/dashboard/conversation/close", {
    method: "POST",
    body: JSON.stringify({ conversationId, closeReason }),
  })
}

export function assignAgentConversation(
  conversationId: number,
  assigneeId: number,
  reason: string
) {
  return request<void>("/api/dashboard/conversation/assign", {
    method: "POST",
    body: JSON.stringify({ conversationId, assigneeId, reason }),
  })
}

export function transferAgentConversation(
  conversationId: number,
  toUserId: number,
  reason: string
) {
  return request<void>("/api/dashboard/conversation/transfer", {
    method: "POST",
    body: JSON.stringify({ conversationId, toUserId, reason }),
  })
}

export function resumeAIConversation(conversationId: number, reason: string) {
  return request<void>("/api/dashboard/conversation/resume_ai", {
    method: "POST",
    body: JSON.stringify({ conversationId, reason }),
  })
}

export function linkConversationToCustomer(payload: {
  conversationId: number
  customerId: number
}) {
  return request<void>("/api/dashboard/conversation/link_customer", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function addConversationTag(payload: {
  conversationId: number
  tagId: number
}) {
  return request<void>("/api/dashboard/conversation/add_tag", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function removeConversationTag(payload: {
  conversationId: number
  tagId: number
}) {
  return request<void>("/api/dashboard/conversation/remove_tag", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}
