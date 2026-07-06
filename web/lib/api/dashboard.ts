import { request } from "@/lib/api/client"

export type DashboardRange = "today" | "7d" | "30d"

export type DashboardStatusDistributionItem = {
  status: number
  label: string
  count: number
}

export type DashboardTrendItem = {
  date: string
  newCount: number
  closedCount: number
}

export type DashboardTeamLoad = {
  teamId: number
  teamName: string
  totalAgents: number
  onlineAgents: number
  busyAgents: number
  offlineAgents: number
  waitingConversations: number
  processingConversations: number
  maxConcurrentCapacity: number
  loadRate: number
  hasScheduleNow: boolean
}

export type DashboardAlert = {
  id: string
  level: "info" | "warning" | "error"
  title: string
  description: string
  count: number
  link: string
}

export type DashboardQuickLink = {
  title: string
  description: string
  link: string
}

export type DashboardTopItem = {
  name: string
  count: number
}

export type DashboardOverview = {
  range: DashboardRange
  generatedAt: string
  summary: {
    todayNewConversations: number
    processingConversations: number
    pendingDispatchConversations: number
    onlineAgents: number
    aiServiceRate: number
  }
  conversationStats: {
    statusDistribution: DashboardStatusDistributionItem[]
    trend: DashboardTrendItem[]
  }
  agentStats: {
    onlineAgents: number
    busyAgents: number
    offlineAgents: number
    teamLoads: DashboardTeamLoad[]
  }
  aiStats: {
    enabledAiAgents: number
    enabledChannels: number
    todayKnowledgeRetrieves: number
    todayKnowledgeRetrieveFailCount: number
    todayKnowledgeRetrieveFailRate: number
    todaySkillRunFailCount: number
    todayAiHandoffCount: number
  }
  digitalStoreStats: {
    todayConsultations: number
    todayLeads: number
    leadConversionRate: number
    todayHighIntentLeads: number
    todayAppointmentLeads: number
    todayConvertedLeads: number
    pendingFollowUpLeads: number
    activeProducts: number
    activePromotions: number
    todayHandoffs: number
    topLeadProducts: DashboardTopItem[]
    summary: string
  }
  alerts: DashboardAlert[]
  quickLinks: DashboardQuickLink[]
}

export type DashboardReportLead = {
  id: number
  customerName: string
  phone: string
  wechat: string
  city: string
  interestedProducts: string
  demandSummary: string
  buyingStage: string
  appointmentAt?: string
  appointmentTimeText: string
  appointmentStore: string
  appointmentPeople: number
  status: string
  ownerUserId: number
  ownerUserName?: string
  nextFollowUpAt?: string
  followUpState?: string
  createdAt: string
}

export type DashboardReportTicket = {
  id: number
  ticketNo: string
  title: string
  description: string
  status: string
  currentAssigneeId: number
  currentAssigneeName?: string
  conversationId: number
  customerId: number
  latestProgress?: string
  latestProgressAt?: string
  handledAt?: string
  createdAt: string
  updatedAt: string
}

export type DashboardAIFeedback = {
  id: number
  retrieveLogId: number
  knowledgeBaseId: number
  feedbackType: number
  feedbackTypeName: string
  feedbackReason: string
  question: string
  answerStatus: number
  answerStatusName: string
  modelName: string
  createdAt: string
}

export type DashboardFAQDraft = {
  id: number
  knowledgeBaseId: number
  question: string
  answer: string
  remark: string
  createdAt: string
  updatedAt: string
}

export type DashboardAIQualityTodo = {
  key: string
  title: string
  description: string
  count: number
  level: "info" | "warning" | "error" | string
  actionHref?: string
  actionLabel?: string
}

export type DashboardAIRiskAnswer = {
  id: number
  knowledgeBaseId: number
  question: string
  answerStatus: number
  answerStatusName: string
  hitCount: number
  topScore: string
  modelName: string
  createdAt: string
  actionHref: string
}

export type DashboardPendingQuestionGroup = {
  question: string
  count: number
  noAnswerCount: number
  fallbackCount: number
  blockedCount: number
  negativeFeedbackCount: number
  latestRetrieveLogId: number
  knowledgeBaseId: number
  latestAt: string
  actionHref: string
  actionLabel: string
}

export type DashboardAIQualityReport = {
  range: DashboardRange
  generatedAt: string
  startDate: string
  endDate: string
  retrieveTotal: number
  retrieveHitTotal: number
  retrieveHitRate: number
  noAnswerCount: number
  fallbackCount: number
  blockedCount: number
  riskAnswerCount: number
  negativeFeedbackCount: number
  feedbackCount: number
  negativeFeedbackRate: number
  pendingFaqDraftCount: number
  todoTotal: number
  todos: DashboardAIQualityTodo[]
  topQuestions: DashboardTopItem[]
  unansweredQuestions: DashboardTopItem[]
  topNegativeReasons: DashboardTopItem[]
  pendingQuestionGroups: DashboardPendingQuestionGroup[]
  recentNegativeFeedbacks: DashboardAIFeedback[]
  pendingFaqDrafts: DashboardFAQDraft[]
  recentRiskAnswerSamples: DashboardAIRiskAnswer[]
  knowledgeSuggestions: string[]
}

export type DashboardSalesFunnelStep = {
  key: string
  label: string
  count: number
  rate: number
  dropOffCount: number
  dropOffRate: number
  actionHref?: string
}

export type DashboardAdvisorEfficiency = {
  ownerUserId: number
  ownerUserName: string
  assignedLeadCount: number
  followUpCount: number
  overdueFollowUpCount: number
  todayFollowUpCount: number
  convertedLeadCount: number
  invalidLeadCount: number
  conversionRate: number
  invalidRate: number
  averageFirstFollowUpMinutes: number
  invalidReasons: DashboardTopItem[]
}

export type DashboardSalesFunnelReport = {
  range: DashboardRange
  generatedAt: string
  startDate: string
  endDate: string
  conversationTotal: number
  leadTotal: number
  leadConversionRate: number
  closedConversionRate: number
  appointmentTotal: number
  visitedTotal: number
  convertedTotal: number
  invalidTotal: number
  unassignedTotal: number
  overdueFollowUpTotal: number
  invalidReasons: DashboardTopItem[]
  steps: DashboardSalesFunnelStep[]
  advisorStats: DashboardAdvisorEfficiency[]
  suggestions: string[]
}

export type DashboardBusinessTrendItem = {
  date: string
  conversationCount: number
  leadCount: number
  highIntentCount: number
  appointmentCount: number
  visitedCount: number
  convertedCount: number
  handoffCount: number
  negativeFeedbackCount: number
}

export type DashboardBusinessTrendReport = {
  range: DashboardRange
  generatedAt: string
  startDate: string
  endDate: string
  conversationTotal: number
  leadTotal: number
  leadConversionRate: number
  highIntentTotal: number
  appointmentTotal: number
  visitedTotal: number
  convertedTotal: number
  handoffTotal: number
  negativeFeedbackTotal: number
  pendingFaqDraftCount: number
  series: DashboardBusinessTrendItem[]
  topProducts: DashboardTopItem[]
  topChannels: DashboardTopItem[]
  topQuestions: DashboardTopItem[]
  topUnansweredQuestions: DashboardTopItem[]
  topNegativeReasons: DashboardTopItem[]
  advisorStats: DashboardAdvisorEfficiency[]
  suggestions: string[]
  reportMarkdown: string
}

export type DashboardABTestVariant = {
  variantCode: string
  variantName: string
  leadCount: number
  highIntentCount: number
  highIntentRate: number
  appointmentCount: number
  appointmentRate: number
  visitedCount: number
  visitRate: number
  convertedCount: number
  conversionRate: number
  invalidCount: number
  invalidRate: number
  qualityRiskLevel: string
  qualityRiskReason: string
  topProduct: string
  recommendedAction: string
}

export type DashboardABTestReport = {
  range: DashboardRange
  generatedAt: string
  startDate: string
  endDate: string
  variantTotal: number
  leadTotal: number
  feedbackTotal: number
  negativeFeedbackTotal: number
  negativeFeedbackRate: number
  variants: DashboardABTestVariant[]
  suggestions: string[]
}

export type DashboardDailyBusinessReport = {
  reportDate: string
  conversationCount: number
  aiReplyCount: number
  handoffCount: number
  leadCount: number
  leadConversionRate: number
  highIntentCount: number
  appointmentCount: number
  convertedCount: number
  unresolvedCount: number
  unassignedPriorityLeadCount: number
  overdueFollowUpCount: number
  todayFollowUpCount: number
  unscheduledHotLeads: number
  overdueAppointmentCount: number
  todayAppointmentCount: number
  unscheduledAppointmentCount: number
  pendingAfterSalesTicketCount: number
  todayAfterSalesTicketCount: number
  todayHandledAfterSalesTicketCount: number
  aiFeedbackCount: number
  aiFeedbackLikeCount: number
  aiFeedbackNegativeCount: number
  aiFeedbackNegativeRate: number
  activeProductCount: number
  activePromotionCount: number
  topLeadProducts: DashboardTopItem[]
  topQuestions: DashboardTopItem[]
  unansweredQuestions: DashboardTopItem[]
  topAiFeedbackReasons: DashboardTopItem[]
  recentNegativeAiFeedbacks: DashboardAIFeedback[]
  pendingFaqDraftCount: number
  pendingFaqDrafts: DashboardFAQDraft[]
  highIntentLeads: DashboardReportLead[]
  priorityFollowUps: DashboardReportLead[]
  afterSalesTickets: DashboardReportTicket[]
  summary: string
  highlights: string[]
  followUpSuggestions: string[]
  knowledgeSuggestions: string[]
}

export type DashboardDailyBusinessReportPushResult = {
  reportDate: string
  generatedAt: string
  webhookEnabled: boolean
  dailyEnabled: boolean
  sent: boolean
  title: string
  message: string
  webhookEventType: string
}

export function fetchDashboardOverview(range: DashboardRange) {
  return request<DashboardOverview>(`/api/dashboard/dashboard/overview?range=${range}`)
}

export function fetchDailyBusinessReport(date?: string) {
  const query = date ? `?date=${encodeURIComponent(date)}` : ""
  return request<DashboardDailyBusinessReport>(`/api/dashboard/business-report/daily${query}`)
}

export function sendDailyBusinessReport(date?: string) {
  const query = date ? `?date=${encodeURIComponent(date)}` : ""
  return request<DashboardDailyBusinessReportPushResult>(
    `/api/dashboard/business-report/daily/send${query}`,
    {
      method: "POST",
      body: JSON.stringify({}),
    }
  )
}

export function fetchAIQualityReport(range: DashboardRange = "7d") {
  return request<DashboardAIQualityReport>(
    `/api/dashboard/business-report/ai-quality?range=${range}`
  )
}

export function fetchSalesFunnelReport(range: DashboardRange = "7d") {
  return request<DashboardSalesFunnelReport>(
    `/api/dashboard/business-report/sales-funnel?range=${range}`
  )
}

export function fetchBusinessTrendReport(range: DashboardRange = "7d") {
  return request<DashboardBusinessTrendReport>(
    `/api/dashboard/business-report/trends?range=${range}`
  )
}

export function fetchABTestReport(range: DashboardRange = "7d") {
  return request<DashboardABTestReport>(
    `/api/dashboard/business-report/ab-tests?range=${range}`
  )
}
