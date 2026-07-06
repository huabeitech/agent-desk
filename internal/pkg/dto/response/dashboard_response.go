package response

type DashboardOverviewResponse struct {
	Range             string                        `json:"range"`
	GeneratedAt       string                        `json:"generatedAt"`
	Summary           DashboardSummaryResponse      `json:"summary"`
	ConversationStats DashboardSectionStatsResponse `json:"conversationStats"`
	AgentStats        DashboardAgentStatsResponse   `json:"agentStats"`
	AIStats           DashboardAIStatsResponse      `json:"aiStats"`
	DigitalStoreStats DashboardDigitalStoreResponse `json:"digitalStoreStats"`
	Alerts            []DashboardAlertResponse      `json:"alerts"`
	QuickLinks        []DashboardQuickLinkResponse  `json:"quickLinks"`
}

type DashboardSummaryResponse struct {
	TodayNewConversations        int64   `json:"todayNewConversations"`
	ProcessingConversations      int64   `json:"processingConversations"`
	PendingDispatchConversations int64   `json:"pendingDispatchConversations"`
	OnlineAgents                 int64   `json:"onlineAgents"`
	AIServiceRate                float64 `json:"aiServiceRate"`
}

type DashboardSectionStatsResponse struct {
	StatusDistribution []DashboardStatusDistributionItem `json:"statusDistribution"`
	Trend              []DashboardTrendItem              `json:"trend"`
}

type DashboardStatusDistributionItem struct {
	Status int    `json:"status"`
	Label  string `json:"label"`
	Count  int64  `json:"count"`
}

type DashboardTrendItem struct {
	Date        string `json:"date"`
	NewCount    int64  `json:"newCount"`
	ClosedCount int64  `json:"closedCount"`
}

type DashboardAgentStatsResponse struct {
	OnlineAgents  int64                       `json:"onlineAgents"`
	BusyAgents    int64                       `json:"busyAgents"`
	OfflineAgents int64                       `json:"offlineAgents"`
	TeamLoads     []DashboardTeamLoadResponse `json:"teamLoads"`
}

type DashboardTeamLoadResponse struct {
	TeamID                  int64   `json:"teamId"`
	TeamName                string  `json:"teamName"`
	TotalAgents             int64   `json:"totalAgents"`
	OnlineAgents            int64   `json:"onlineAgents"`
	BusyAgents              int64   `json:"busyAgents"`
	OfflineAgents           int64   `json:"offlineAgents"`
	WaitingConversations    int64   `json:"waitingConversations"`
	ProcessingConversations int64   `json:"processingConversations"`
	MaxConcurrentCapacity   int64   `json:"maxConcurrentCapacity"`
	LoadRate                float64 `json:"loadRate"`
	HasScheduleNow          bool    `json:"hasScheduleNow"`
}

type DashboardAIStatsResponse struct {
	EnabledAIAgents                 int64   `json:"enabledAiAgents"`
	EnabledChannels                 int64   `json:"enabledChannels"`
	TodayKnowledgeRetrieves         int64   `json:"todayKnowledgeRetrieves"`
	TodayKnowledgeRetrieveFailCount int64   `json:"todayKnowledgeRetrieveFailCount"`
	TodayKnowledgeRetrieveFailRate  float64 `json:"todayKnowledgeRetrieveFailRate"`
	TodaySkillRunFailCount          int64   `json:"todaySkillRunFailCount"`
	TodayAIHandoffCount             int64   `json:"todayAiHandoffCount"`
}

type DashboardDigitalStoreResponse struct {
	TodayConsultations    int64                      `json:"todayConsultations"`
	TodayLeads            int64                      `json:"todayLeads"`
	LeadConversionRate    float64                    `json:"leadConversionRate"`
	TodayHighIntentLeads  int64                      `json:"todayHighIntentLeads"`
	TodayAppointmentLeads int64                      `json:"todayAppointmentLeads"`
	TodayConvertedLeads   int64                      `json:"todayConvertedLeads"`
	PendingFollowUpLeads  int64                      `json:"pendingFollowUpLeads"`
	ActiveProducts        int64                      `json:"activeProducts"`
	ActivePromotions      int64                      `json:"activePromotions"`
	TodayHandoffs         int64                      `json:"todayHandoffs"`
	TopLeadProducts       []DashboardTopItemResponse `json:"topLeadProducts"`
	Summary               string                     `json:"summary"`
}

type DashboardTopItemResponse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DashboardDailyBusinessReportResponse struct {
	ReportDate                        string                          `json:"reportDate"`
	ConversationCount                 int64                           `json:"conversationCount"`
	AIReplyCount                      int64                           `json:"aiReplyCount"`
	HandoffCount                      int64                           `json:"handoffCount"`
	LeadCount                         int64                           `json:"leadCount"`
	LeadConversionRate                float64                         `json:"leadConversionRate"`
	HighIntentCount                   int64                           `json:"highIntentCount"`
	AppointmentCount                  int64                           `json:"appointmentCount"`
	ConvertedCount                    int64                           `json:"convertedCount"`
	UnresolvedCount                   int64                           `json:"unresolvedCount"`
	UnassignedPriorityLeadCount       int64                           `json:"unassignedPriorityLeadCount"`
	OverdueFollowUpCount              int64                           `json:"overdueFollowUpCount"`
	TodayFollowUpCount                int64                           `json:"todayFollowUpCount"`
	UnscheduledHotLeads               int64                           `json:"unscheduledHotLeads"`
	OverdueAppointmentCount           int64                           `json:"overdueAppointmentCount"`
	TodayAppointmentCount             int64                           `json:"todayAppointmentCount"`
	UnscheduledAppointmentCount       int64                           `json:"unscheduledAppointmentCount"`
	PendingAfterSalesTicketCount      int64                           `json:"pendingAfterSalesTicketCount"`
	TodayAfterSalesTicketCount        int64                           `json:"todayAfterSalesTicketCount"`
	TodayHandledAfterSalesTicketCount int64                           `json:"todayHandledAfterSalesTicketCount"`
	AIFeedbackCount                   int64                           `json:"aiFeedbackCount"`
	AIFeedbackLikeCount               int64                           `json:"aiFeedbackLikeCount"`
	AIFeedbackNegativeCount           int64                           `json:"aiFeedbackNegativeCount"`
	AIFeedbackNegativeRate            float64                         `json:"aiFeedbackNegativeRate"`
	ActiveProductCount                int64                           `json:"activeProductCount"`
	ActivePromotionCount              int64                           `json:"activePromotionCount"`
	TopLeadProducts                   []DashboardTopItemResponse      `json:"topLeadProducts"`
	TopQuestions                      []DashboardTopItemResponse      `json:"topQuestions"`
	UnansweredQuestions               []DashboardTopItemResponse      `json:"unansweredQuestions"`
	TopAIFeedbackReasons              []DashboardTopItemResponse      `json:"topAiFeedbackReasons"`
	RecentNegativeAIFeedbacks         []DashboardAIFeedbackResponse   `json:"recentNegativeAiFeedbacks"`
	PendingFAQDraftCount              int64                           `json:"pendingFaqDraftCount"`
	PendingFAQDrafts                  []DashboardFAQDraftResponse     `json:"pendingFaqDrafts"`
	HighIntentLeads                   []DashboardReportLeadResponse   `json:"highIntentLeads"`
	PriorityFollowUps                 []DashboardReportLeadResponse   `json:"priorityFollowUps"`
	AfterSalesTickets                 []DashboardReportTicketResponse `json:"afterSalesTickets"`
	Summary                           string                          `json:"summary"`
	Highlights                        []string                        `json:"highlights"`
	FollowUpSuggestions               []string                        `json:"followUpSuggestions"`
	KnowledgeSuggestions              []string                        `json:"knowledgeSuggestions"`
}

type DashboardDailyBusinessReportPushResponse struct {
	ReportDate       string `json:"reportDate"`
	GeneratedAt      string `json:"generatedAt"`
	WebhookEnabled   bool   `json:"webhookEnabled"`
	DailyEnabled     bool   `json:"dailyEnabled"`
	Sent             bool   `json:"sent"`
	Title            string `json:"title"`
	Message          string `json:"message"`
	WebhookEventType string `json:"webhookEventType"`
}

type DashboardAIQualityReportResponse struct {
	Range                   string                          `json:"range"`
	GeneratedAt             string                          `json:"generatedAt"`
	StartDate               string                          `json:"startDate"`
	EndDate                 string                          `json:"endDate"`
	RetrieveTotal           int64                           `json:"retrieveTotal"`
	RetrieveHitTotal        int64                           `json:"retrieveHitTotal"`
	RetrieveHitRate         float64                         `json:"retrieveHitRate"`
	NoAnswerCount           int64                           `json:"noAnswerCount"`
	FallbackCount           int64                           `json:"fallbackCount"`
	BlockedCount            int64                           `json:"blockedCount"`
	RiskAnswerCount         int64                           `json:"riskAnswerCount"`
	NegativeFeedbackCount   int64                           `json:"negativeFeedbackCount"`
	FeedbackCount           int64                           `json:"feedbackCount"`
	NegativeFeedbackRate    float64                         `json:"negativeFeedbackRate"`
	PendingFAQDraftCount    int64                           `json:"pendingFaqDraftCount"`
	TodoTotal               int64                           `json:"todoTotal"`
	Todos                   []DashboardAIQualityTodoItem    `json:"todos"`
	TopQuestions            []DashboardTopItemResponse      `json:"topQuestions"`
	UnansweredQuestions     []DashboardTopItemResponse      `json:"unansweredQuestions"`
	TopNegativeReasons      []DashboardTopItemResponse      `json:"topNegativeReasons"`
	PendingQuestionGroups   []DashboardPendingQuestionGroup `json:"pendingQuestionGroups"`
	RecentNegativeFeedbacks []DashboardAIFeedbackResponse   `json:"recentNegativeFeedbacks"`
	PendingFAQDrafts        []DashboardFAQDraftResponse     `json:"pendingFaqDrafts"`
	RecentRiskAnswerSamples []DashboardAIRiskAnswerItem     `json:"recentRiskAnswerSamples"`
	KnowledgeSuggestions    []string                        `json:"knowledgeSuggestions"`
}

type DashboardSalesFunnelReportResponse struct {
	Range                string                       `json:"range"`
	GeneratedAt          string                       `json:"generatedAt"`
	StartDate            string                       `json:"startDate"`
	EndDate              string                       `json:"endDate"`
	ConversationTotal    int64                        `json:"conversationTotal"`
	LeadTotal            int64                        `json:"leadTotal"`
	LeadConversionRate   float64                      `json:"leadConversionRate"`
	ClosedConversionRate float64                      `json:"closedConversionRate"`
	AppointmentTotal     int64                        `json:"appointmentTotal"`
	VisitedTotal         int64                        `json:"visitedTotal"`
	ConvertedTotal       int64                        `json:"convertedTotal"`
	InvalidTotal         int64                        `json:"invalidTotal"`
	UnassignedTotal      int64                        `json:"unassignedTotal"`
	OverdueFollowUpTotal int64                        `json:"overdueFollowUpTotal"`
	InvalidReasons       []DashboardTopItemResponse   `json:"invalidReasons"`
	Steps                []DashboardSalesFunnelStep   `json:"steps"`
	AdvisorStats         []DashboardAdvisorEfficiency `json:"advisorStats"`
	Suggestions          []string                     `json:"suggestions"`
}

type DashboardSalesFunnelStep struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	Count        int64   `json:"count"`
	Rate         float64 `json:"rate"`
	DropOffCount int64   `json:"dropOffCount"`
	DropOffRate  float64 `json:"dropOffRate"`
	ActionHref   string  `json:"actionHref,omitempty"`
}

type DashboardBusinessTrendReportResponse struct {
	Range                  string                       `json:"range"`
	GeneratedAt            string                       `json:"generatedAt"`
	StartDate              string                       `json:"startDate"`
	EndDate                string                       `json:"endDate"`
	ConversationTotal      int64                        `json:"conversationTotal"`
	LeadTotal              int64                        `json:"leadTotal"`
	LeadConversionRate     float64                      `json:"leadConversionRate"`
	HighIntentTotal        int64                        `json:"highIntentTotal"`
	AppointmentTotal       int64                        `json:"appointmentTotal"`
	VisitedTotal           int64                        `json:"visitedTotal"`
	ConvertedTotal         int64                        `json:"convertedTotal"`
	HandoffTotal           int64                        `json:"handoffTotal"`
	NegativeFeedbackTotal  int64                        `json:"negativeFeedbackTotal"`
	PendingFAQDraftCount   int64                        `json:"pendingFaqDraftCount"`
	Series                 []DashboardBusinessTrendItem `json:"series"`
	TopProducts            []DashboardTopItemResponse   `json:"topProducts"`
	TopChannels            []DashboardTopItemResponse   `json:"topChannels"`
	TopQuestions           []DashboardTopItemResponse   `json:"topQuestions"`
	TopUnansweredQuestions []DashboardTopItemResponse   `json:"topUnansweredQuestions"`
	TopNegativeReasons     []DashboardTopItemResponse   `json:"topNegativeReasons"`
	AdvisorStats           []DashboardAdvisorEfficiency `json:"advisorStats"`
	Suggestions            []string                     `json:"suggestions"`
	ReportMarkdown         string                       `json:"reportMarkdown"`
}

type DashboardBusinessTrendItem struct {
	Date                  string `json:"date"`
	ConversationCount     int64  `json:"conversationCount"`
	LeadCount             int64  `json:"leadCount"`
	HighIntentCount       int64  `json:"highIntentCount"`
	AppointmentCount      int64  `json:"appointmentCount"`
	VisitedCount          int64  `json:"visitedCount"`
	ConvertedCount        int64  `json:"convertedCount"`
	HandoffCount          int64  `json:"handoffCount"`
	NegativeFeedbackCount int64  `json:"negativeFeedbackCount"`
}

type DashboardABTestReportResponse struct {
	Range                 string                         `json:"range"`
	GeneratedAt           string                         `json:"generatedAt"`
	StartDate             string                         `json:"startDate"`
	EndDate               string                         `json:"endDate"`
	VariantTotal          int64                          `json:"variantTotal"`
	LeadTotal             int64                          `json:"leadTotal"`
	FeedbackTotal         int64                          `json:"feedbackTotal"`
	NegativeFeedbackTotal int64                          `json:"negativeFeedbackTotal"`
	NegativeFeedbackRate  float64                        `json:"negativeFeedbackRate"`
	Variants              []DashboardABTestVariantResult `json:"variants"`
	Suggestions           []string                       `json:"suggestions"`
}

type DashboardABTestVariantResult struct {
	VariantCode       string  `json:"variantCode"`
	VariantName       string  `json:"variantName"`
	LeadCount         int64   `json:"leadCount"`
	HighIntentCount   int64   `json:"highIntentCount"`
	HighIntentRate    float64 `json:"highIntentRate"`
	AppointmentCount  int64   `json:"appointmentCount"`
	AppointmentRate   float64 `json:"appointmentRate"`
	VisitedCount      int64   `json:"visitedCount"`
	VisitRate         float64 `json:"visitRate"`
	ConvertedCount    int64   `json:"convertedCount"`
	ConversionRate    float64 `json:"conversionRate"`
	InvalidCount      int64   `json:"invalidCount"`
	InvalidRate       float64 `json:"invalidRate"`
	QualityRiskLevel  string  `json:"qualityRiskLevel"`
	QualityRiskReason string  `json:"qualityRiskReason"`
	TopProduct        string  `json:"topProduct"`
	RecommendedAction string  `json:"recommendedAction"`
}

type DashboardAdvisorEfficiency struct {
	OwnerUserID                 int64                      `json:"ownerUserId"`
	OwnerUserName               string                     `json:"ownerUserName"`
	AssignedLeadCount           int64                      `json:"assignedLeadCount"`
	FollowUpCount               int64                      `json:"followUpCount"`
	OverdueFollowUpCount        int64                      `json:"overdueFollowUpCount"`
	TodayFollowUpCount          int64                      `json:"todayFollowUpCount"`
	ConvertedLeadCount          int64                      `json:"convertedLeadCount"`
	InvalidLeadCount            int64                      `json:"invalidLeadCount"`
	ConversionRate              float64                    `json:"conversionRate"`
	InvalidRate                 float64                    `json:"invalidRate"`
	AverageFirstFollowUpMinutes int64                      `json:"averageFirstFollowUpMinutes"`
	InvalidReasons              []DashboardTopItemResponse `json:"invalidReasons"`
}

type DashboardAIQualityTodoItem struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Count       int64  `json:"count"`
	Level       string `json:"level"`
	ActionHref  string `json:"actionHref,omitempty"`
	ActionLabel string `json:"actionLabel,omitempty"`
}

type DashboardPendingQuestionGroup struct {
	Question              string `json:"question"`
	Count                 int64  `json:"count"`
	NoAnswerCount         int64  `json:"noAnswerCount"`
	FallbackCount         int64  `json:"fallbackCount"`
	BlockedCount          int64  `json:"blockedCount"`
	NegativeFeedbackCount int64  `json:"negativeFeedbackCount"`
	LatestRetrieveLogID   int64  `json:"latestRetrieveLogId"`
	KnowledgeBaseID       int64  `json:"knowledgeBaseId"`
	LatestAt              string `json:"latestAt"`
	ActionHref            string `json:"actionHref"`
	ActionLabel           string `json:"actionLabel"`
}

type DashboardAIRiskAnswerItem struct {
	ID               int64  `json:"id"`
	KnowledgeBaseID  int64  `json:"knowledgeBaseId"`
	Question         string `json:"question"`
	AnswerStatus     int    `json:"answerStatus"`
	AnswerStatusName string `json:"answerStatusName"`
	HitCount         int    `json:"hitCount"`
	TopScore         string `json:"topScore"`
	ModelName        string `json:"modelName"`
	CreatedAt        string `json:"createdAt"`
	ActionHref       string `json:"actionHref"`
}

type DashboardReportLeadResponse struct {
	ID                  int64  `json:"id"`
	CustomerName        string `json:"customerName"`
	Phone               string `json:"phone"`
	WeChat              string `json:"wechat"`
	City                string `json:"city"`
	InterestedProducts  string `json:"interestedProducts"`
	DemandSummary       string `json:"demandSummary"`
	BuyingStage         string `json:"buyingStage"`
	AppointmentAt       string `json:"appointmentAt,omitempty"`
	AppointmentTimeText string `json:"appointmentTimeText"`
	AppointmentStore    string `json:"appointmentStore"`
	AppointmentPeople   int    `json:"appointmentPeople"`
	Status              string `json:"status"`
	OwnerUserID         int64  `json:"ownerUserId"`
	OwnerUserName       string `json:"ownerUserName,omitempty"`
	NextFollowUpAt      string `json:"nextFollowUpAt,omitempty"`
	FollowUpState       string `json:"followUpState,omitempty"`
	CreatedAt           string `json:"createdAt"`
}

type DashboardReportTicketResponse struct {
	ID                  int64  `json:"id"`
	TicketNo            string `json:"ticketNo"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	Status              string `json:"status"`
	CurrentAssigneeID   int64  `json:"currentAssigneeId"`
	CurrentAssigneeName string `json:"currentAssigneeName,omitempty"`
	ConversationID      int64  `json:"conversationId"`
	CustomerID          int64  `json:"customerId"`
	LatestProgress      string `json:"latestProgress,omitempty"`
	LatestProgressAt    string `json:"latestProgressAt,omitempty"`
	HandledAt           string `json:"handledAt,omitempty"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

type DashboardAIFeedbackResponse struct {
	ID               int64  `json:"id"`
	RetrieveLogID    int64  `json:"retrieveLogId"`
	KnowledgeBaseID  int64  `json:"knowledgeBaseId"`
	FeedbackType     int    `json:"feedbackType"`
	FeedbackTypeName string `json:"feedbackTypeName"`
	FeedbackReason   string `json:"feedbackReason"`
	Question         string `json:"question"`
	AnswerStatus     int    `json:"answerStatus"`
	AnswerStatusName string `json:"answerStatusName"`
	ModelName        string `json:"modelName"`
	CreatedAt        string `json:"createdAt"`
}

type DashboardFAQDraftResponse struct {
	ID              int64  `json:"id"`
	KnowledgeBaseID int64  `json:"knowledgeBaseId"`
	Question        string `json:"question"`
	Answer          string `json:"answer"`
	Remark          string `json:"remark"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type DashboardAlertResponse struct {
	ID          string `json:"id"`
	Level       string `json:"level"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Count       int64  `json:"count"`
	Link        string `json:"link"`
}

type DashboardQuickLinkResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
}
