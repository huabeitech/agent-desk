import { request } from "@/lib/api/client"

export type DigitalStoreProfile = {
  brandName: string
  industry: string
  storeName: string
  storeAddress: string
  businessHours: string
  contactPhone: string
  serviceWechat: string
  enterpriseWebhookUrl: string
  aiManagerName: string
  aiPersona: string
  replyStyle: string
  forbiddenClaims: string
  handoffPolicy: string
  appointmentPolicy: string
  knowledgeBaseId: number
  knowledgeFAQId: number
  templateCode: string
  templateVersion: string
  templateAppliedAt?: string
  initialized: boolean
  updatedAt?: string
}

export type SaveDigitalStoreProfilePayload = Omit<
  DigitalStoreProfile,
  "knowledgeFAQId" | "templateCode" | "templateVersion" | "templateAppliedAt" | "updatedAt"
>

export type DigitalStoreSetupStatus = {
  profileInitialized: boolean
  knowledgeBaseId: number
  knowledgeFAQId: number
  productTotal: number
  promotionTotal: number
  productKnowledgeSyncedTotal: number
  productKnowledgeUnsyncedTotal: number
  productKnowledgeFailedTotal: number
  promotionKnowledgeSyncedTotal: number
  promotionKnowledgeUnsyncedTotal: number
  promotionKnowledgeFailedTotal: number
  llmConfigId: number
  llmConfigName: string
  embeddingConfigId: number
  embeddingConfigName: string
  agentId: number
  agentName: string
  workflowPublished: boolean
  webChannelId: number
  webChannelCode: string
  webChannelName: string
  webEntry: DigitalStoreWebEntry
  humanHandoff: DigitalStoreHumanHandoff
  modelHealthChecks: DigitalStoreHealthCheck[]
  ready: boolean
  missingSteps: string[]
}

export type DigitalStoreHumanHandoff = {
  ready: boolean
  agentTeamIds: number[]
  activeTeamIds: number[]
  agentProfileTotal: number
  autoAssignProfiles: number
  eligibleProfiles: number
  candidateProfiles: number
  message: string
}

export type DigitalStoreWebEntry = {
  channelId: number
  channelCode: string
  channelName: string
  title: string
  subtitle: string
  themeColor: string
  position: string
  width: string
  chatUrl: string
  embedSnippet: string
}

export type DigitalStoreDeliveryReportItem = {
  label: string
  status: string
  value: string
  actionHref?: string
  actionLabel?: string
}

export type DigitalStoreAcceptanceItem = {
  code: string
  title: string
  customerAsk: string
  expectation: string
  consoleCheck: string
  blocking: boolean
}

export type DigitalStoreNotificationStatus = {
  enabled: boolean
  configured: boolean
  format: string
  hasSecret: boolean
  profileWebhookUrlSet: boolean
  status: string
  message: string
}

export type DigitalStoreSecurityCheck = {
  key: string
  label: string
  status: "ok" | "warning" | "blocking" | string
  message: string
  actionHref?: string
  actionLabel?: string
}

export type DigitalStoreHealthCheck = {
  key: string
  label: string
  status: "ok" | "warning" | "blocking" | string
  message: string
  actionHref?: string
  actionLabel?: string
}

export type DigitalStoreWebhookTestResult = DigitalStoreNotificationStatus & {
  sent: boolean
  testedAt: string
  sentTotal: number
  failedTotal: number
  scenarios?: DigitalStoreWebhookTestScenario[]
}

export type DigitalStoreWebhookTestScenario = {
  key: string
  eventType: string
  title: string
  sent: boolean
  message: string
}

export type DigitalStoreDemoDataCleanupResult = {
  cleanedAt: string
  message: string
  deleted: Record<string, number>
}

export type DigitalStoreMaintenanceStatus = {
  checkedAt: string
  status: "ok" | "warning" | string
  backupRoot: string
  backupCommand: string
  restoreDryRunCommand: string
  upgradeCommands: string[]
  upgradeRunbook: string
  latestBackup?: DigitalStoreBackupSnapshot | null
  warnings: DigitalStoreMaintenanceWarning[]
}

export type DigitalStoreBackupSnapshot = {
  path: string
  timestamp: string
  createdAt: string
  projectDir: string
  composeFile: string
  hasManifest: boolean
  hasMysqlDump: boolean
  hasDataArchive: boolean
  hasDockerConfigArchive: boolean
  hasConfigSnapshot: boolean
  sizeBytes: number
}

export type DigitalStoreMaintenanceWarning = {
  key: string
  label: string
  message: string
}

export type DigitalStoreKnowledgeAssistant = {
  generatedAt: string
  industry: string
  knowledgeBaseId: number
  coveredTotal: number
  missingTotal: number
  items: DigitalStoreKnowledgeAssistantItem[]
}

export type DigitalStoreKnowledgeAssistantItem = {
  key: string
  question: string
  reason: string
  required: boolean
  covered: boolean
  matchedFaqId?: number
  keywords: string[]
  actionHref?: string
  actionLabel?: string
}

export type DigitalStoreTemplateEffect = {
  generatedAt: string
  templateCode: string
  templateVersion: string
  templateAppliedAt?: string
  industry: string
  knowledgeBaseId: number
  days: number
  retrieveTotal: number
  missingQuestionTotal: number
  negativeFeedbackTotal: number
  missingQuestions: DigitalStoreTemplateEffectItem[]
  negativeFeedbacks: DigitalStoreTemplateEffectItem[]
  suggestions: string[]
  improvementMarkdown: string
}

export type DigitalStoreTemplateEffectItem = {
  question: string
  count: number
  latestAt?: string
  feedbackReason?: string
  feedbackTypeName?: string
  answerStatusName?: string
  actionHref?: string
  actionLabel?: string
  createFaqActionHref?: string
}

export type DigitalStoreDeliveryReport = {
  generatedAt: string
  brandName: string
  storeName: string
  ready: boolean
  dashboardUrl: string
  chatUrl: string
  embedSnippet: string
  webEntry: DigitalStoreWebEntry
  humanHandoff: DigitalStoreHumanHandoff
  acceptanceCommand: string
  acceptanceItems: DigitalStoreAcceptanceItem[]
  notificationStatus: DigitalStoreNotificationStatus
  securityChecks: DigitalStoreSecurityCheck[]
  modelHealthChecks: DigitalStoreHealthCheck[]
  items: DigitalStoreDeliveryReportItem[]
  missingSteps: string[]
  latestRecord?: DigitalStoreDeliveryRecord | null
  markdown: string
  acceptanceRunbook: string
}

export type DigitalStoreDeliveryRecord = {
  id: number
  brandName: string
  storeName: string
  ready: boolean
  acceptanceStatus: string
  acceptanceSummary: string
  acceptanceCommand: string
  scenarioTotal: number
  passedTotal: number
  failedTotal: number
  acceptanceStartedAt?: string
  acceptanceFinishedAt?: string
  dashboardUrl: string
  chatUrl: string
  webChannelCode: string
  createdAt: string
  createUserName: string
  acceptanceResults?: DigitalStoreAcceptanceScenarioResult[]
}

export type DigitalStoreAcceptanceScenarioResult = {
  code: string
  title: string
  passed: boolean
  reason: string
  failureType: string
  detail: string
  suggestion: string
  conversationId: number
  conversationUrl: string
  reply: string
  expectedKeywords: string[]
  matchedKeywords: string[]
  missingKeywords: string[]
  bannedKeywords: string[]
  matchedBanned: string
}

export type DigitalStoreTemplate = {
  code: string
  name: string
  industry: string
  version: string
  description: string
}

export type DigitalStoreTemplateExport = {
  schemaVersion: string
  exportedAt: string
  template: DigitalStoreTemplate
  profile: DigitalStoreProfile
  products: DigitalStoreTemplateProduct[]
  promotions: DigitalStoreTemplatePromotion[]
  riskRules: DigitalStoreIndustryRiskRule[]
  acceptanceItems: DigitalStoreAcceptanceItem[]
}

export type DigitalStoreTemplateImportPayload = DigitalStoreTemplateExport

export type DigitalStoreTemplatePreview = {
  template: DigitalStoreTemplate
  profile: DigitalStoreProfile
  profileAction: "create" | "update" | string
  productCreateTotal: number
  productUpdateTotal: number
  promotionCreateTotal: number
  promotionUpdateTotal: number
  products: DigitalStoreTemplatePreviewItem[]
  promotions: DigitalStoreTemplatePreviewItem[]
  riskRules: DigitalStoreIndustryRiskRule[]
  acceptanceItems: DigitalStoreAcceptanceItem[]
  warnings: DigitalStoreTemplatePreviewWarning[]
}

export type DigitalStoreIndustryRiskRule = {
  key: string
  label: string
  forbiddenClaims: string[]
  handoffTriggers: string[]
}

export type DigitalStoreTemplatePreviewItem = {
  name: string
  action: "create" | "update" | string
  existingId?: number
  reason: string
}

export type DigitalStoreTemplatePreviewWarning = {
  key: string
  message: string
}

export type DigitalStoreTemplateProduct = {
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
  status: number
  remark: string
}

export type DigitalStoreTemplatePromotion = {
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
  status: number
  remark: string
}

export function fetchDigitalStoreProfile() {
  return request<DigitalStoreProfile>("/api/dashboard/digital-store/profile")
}

export function fetchDigitalStoreTemplates() {
  return request<DigitalStoreTemplate[]>("/api/dashboard/digital-store/templates")
}

export function exportDigitalStoreTemplate(templateCode: string) {
  return request<DigitalStoreTemplateExport>(
    `/api/dashboard/digital-store/templates/export?templateCode=${encodeURIComponent(templateCode)}`
  )
}

export function previewDigitalStoreTemplate(templateCode: string) {
  return request<DigitalStoreTemplatePreview>(
    `/api/dashboard/digital-store/templates/preview?templateCode=${encodeURIComponent(templateCode)}`
  )
}

export function previewImportedDigitalStoreTemplate(payload: DigitalStoreTemplateImportPayload) {
  return request<DigitalStoreTemplatePreview>(
    "/api/dashboard/digital-store/templates/import_preview",
    {
      method: "POST",
      body: JSON.stringify(payload),
    }
  )
}

export function fetchDigitalStoreDeliveryReport(publicBaseUrl?: string) {
  const query = publicBaseUrl
    ? `?publicBaseUrl=${encodeURIComponent(publicBaseUrl)}`
    : ""
  return request<DigitalStoreDeliveryReport>(
    `/api/dashboard/digital-store/delivery_report${query}`
  )
}

export function fetchLatestDigitalStoreDeliveryRecord() {
  return request<DigitalStoreDeliveryRecord | null>(
    "/api/dashboard/digital-store/delivery_records/latest"
  )
}

export function createDigitalStoreDeliveryRecord(payload: {
  publicBaseUrl?: string
  acceptanceStatus?: string
  acceptanceSummary?: string
}) {
  return request<DigitalStoreDeliveryRecord>(
    "/api/dashboard/digital-store/delivery_records/create",
    {
      method: "POST",
      body: JSON.stringify(payload),
    }
  )
}

export function fetchDigitalStoreSetupStatus() {
  return request<DigitalStoreSetupStatus>("/api/dashboard/digital-store/setup_status")
}

export function fetchDigitalStoreKnowledgeAssistant() {
  return request<DigitalStoreKnowledgeAssistant>(
    "/api/dashboard/digital-store/knowledge_assistant"
  )
}

export function fetchDigitalStoreTemplateEffect() {
  return request<DigitalStoreTemplateEffect>(
    "/api/dashboard/digital-store/template_effect"
  )
}

export function fetchDigitalStoreMaintenanceStatus() {
  return request<DigitalStoreMaintenanceStatus>(
    "/api/dashboard/digital-store/maintenance_status"
  )
}

export function saveDigitalStoreProfile(payload: SaveDigitalStoreProfilePayload) {
  return request<DigitalStoreProfile>("/api/dashboard/digital-store/profile", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function seedMuseDigitalStoreProfile() {
  return request<DigitalStoreProfile>("/api/dashboard/digital-store/seed_muse", {
    method: "POST",
    body: JSON.stringify({}),
  })
}

export function applyDigitalStoreTemplate(templateCode: string) {
  return request<DigitalStoreProfile>("/api/dashboard/digital-store/apply_template", {
    method: "POST",
    body: JSON.stringify({ templateCode }),
  })
}

export function applyImportedDigitalStoreTemplate(payload: DigitalStoreTemplateImportPayload) {
  return request<DigitalStoreProfile>(
    "/api/dashboard/digital-store/apply_imported_template",
    {
      method: "POST",
      body: JSON.stringify(payload),
    }
  )
}

export function syncDigitalStoreKnowledge() {
  return request<DigitalStoreProfile>("/api/dashboard/digital-store/sync_knowledge", {
    method: "POST",
    body: JSON.stringify({}),
  })
}

export function ensureDigitalStoreRuntime() {
  return request<DigitalStoreSetupStatus>("/api/dashboard/digital-store/ensure_runtime", {
    method: "POST",
    body: JSON.stringify({}),
  })
}

export function testDigitalStoreWebhookNotify() {
  return request<DigitalStoreWebhookTestResult>(
    "/api/dashboard/digital-store/test_webhook_notify",
    {
      method: "POST",
      body: JSON.stringify({}),
    }
  )
}

export function testDigitalStoreWebhookNotifyScenarios() {
  return request<DigitalStoreWebhookTestResult>(
    "/api/dashboard/digital-store/test_webhook_notify_scenarios",
    {
      method: "POST",
      body: JSON.stringify({}),
    }
  )
}

export function cleanupDigitalStoreDemoData() {
  return request<DigitalStoreDemoDataCleanupResult>(
    "/api/dashboard/digital-store/cleanup_demo_data",
    {
      method: "POST",
      body: JSON.stringify({}),
    }
  )
}
