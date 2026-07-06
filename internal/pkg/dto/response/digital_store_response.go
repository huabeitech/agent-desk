package response

type DigitalStoreProfileResponse struct {
	BrandName            string `json:"brandName"`
	Industry             string `json:"industry"`
	StoreName            string `json:"storeName"`
	StoreAddress         string `json:"storeAddress"`
	BusinessHours        string `json:"businessHours"`
	ContactPhone         string `json:"contactPhone"`
	ServiceWeChat        string `json:"serviceWechat"`
	EnterpriseWebhookURL string `json:"enterpriseWebhookUrl"`
	AIManagerName        string `json:"aiManagerName"`
	AIPersona            string `json:"aiPersona"`
	ReplyStyle           string `json:"replyStyle"`
	ForbiddenClaims      string `json:"forbiddenClaims"`
	HandoffPolicy        string `json:"handoffPolicy"`
	AppointmentPolicy    string `json:"appointmentPolicy"`
	KnowledgeBaseID      int64  `json:"knowledgeBaseId"`
	KnowledgeFAQID       int64  `json:"knowledgeFAQId"`
	TemplateCode         string `json:"templateCode"`
	TemplateVersion      string `json:"templateVersion"`
	TemplateAppliedAt    string `json:"templateAppliedAt,omitempty"`
	Initialized          bool   `json:"initialized"`
	UpdatedAt            string `json:"updatedAt,omitempty"`
}

type DigitalStoreSetupStatusResponse struct {
	ProfileInitialized              bool                              `json:"profileInitialized"`
	KnowledgeBaseID                 int64                             `json:"knowledgeBaseId"`
	KnowledgeFAQID                  int64                             `json:"knowledgeFAQId"`
	ProductTotal                    int64                             `json:"productTotal"`
	PromotionTotal                  int64                             `json:"promotionTotal"`
	ProductKnowledgeSyncedTotal     int64                             `json:"productKnowledgeSyncedTotal"`
	ProductKnowledgeUnsyncedTotal   int64                             `json:"productKnowledgeUnsyncedTotal"`
	ProductKnowledgeFailedTotal     int64                             `json:"productKnowledgeFailedTotal"`
	PromotionKnowledgeSyncedTotal   int64                             `json:"promotionKnowledgeSyncedTotal"`
	PromotionKnowledgeUnsyncedTotal int64                             `json:"promotionKnowledgeUnsyncedTotal"`
	PromotionKnowledgeFailedTotal   int64                             `json:"promotionKnowledgeFailedTotal"`
	LLMConfigID                     int64                             `json:"llmConfigId"`
	LLMConfigName                   string                            `json:"llmConfigName"`
	EmbeddingConfigID               int64                             `json:"embeddingConfigId"`
	EmbeddingConfigName             string                            `json:"embeddingConfigName"`
	AgentID                         int64                             `json:"agentId"`
	AgentName                       string                            `json:"agentName"`
	WorkflowPublished               bool                              `json:"workflowPublished"`
	WebChannelID                    int64                             `json:"webChannelId"`
	WebChannelCode                  string                            `json:"webChannelCode"`
	WebChannelName                  string                            `json:"webChannelName"`
	WebEntry                        DigitalStoreWebEntryResponse      `json:"webEntry"`
	HumanHandoff                    DigitalStoreHumanHandoffResponse  `json:"humanHandoff"`
	ModelHealthChecks               []DigitalStoreHealthCheckResponse `json:"modelHealthChecks"`
	Ready                           bool                              `json:"ready"`
	MissingSteps                    []string                          `json:"missingSteps"`
}

type DigitalStoreHumanHandoffResponse struct {
	Ready              bool    `json:"ready"`
	AgentTeamIDs       []int64 `json:"agentTeamIds"`
	ActiveTeamIDs      []int64 `json:"activeTeamIds"`
	AgentProfileTotal  int64   `json:"agentProfileTotal"`
	AutoAssignProfiles int64   `json:"autoAssignProfiles"`
	EligibleProfiles   int     `json:"eligibleProfiles"`
	CandidateProfiles  int     `json:"candidateProfiles"`
	Message            string  `json:"message"`
}

type DigitalStoreWebEntryResponse struct {
	ChannelID    int64  `json:"channelId"`
	ChannelCode  string `json:"channelCode"`
	ChannelName  string `json:"channelName"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	ThemeColor   string `json:"themeColor"`
	Position     string `json:"position"`
	Width        string `json:"width"`
	ChatURL      string `json:"chatUrl"`
	EmbedSnippet string `json:"embedSnippet"`
}

type DigitalStoreDeliveryReportItem struct {
	Label       string `json:"label"`
	Status      string `json:"status"`
	Value       string `json:"value"`
	ActionHref  string `json:"actionHref,omitempty"`
	ActionLabel string `json:"actionLabel,omitempty"`
}

type DigitalStoreAcceptanceItem struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	CustomerAsk  string `json:"customerAsk"`
	Expectation  string `json:"expectation"`
	ConsoleCheck string `json:"consoleCheck"`
	Blocking     bool   `json:"blocking"`
}

type DigitalStoreNotificationStatusResponse struct {
	Enabled              bool   `json:"enabled"`
	Configured           bool   `json:"configured"`
	Format               string `json:"format"`
	HasSecret            bool   `json:"hasSecret"`
	ProfileWebhookURLSet bool   `json:"profileWebhookUrlSet"`
	Status               string `json:"status"`
	Message              string `json:"message"`
}

type DigitalStoreSecurityCheckResponse struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	ActionHref  string `json:"actionHref,omitempty"`
	ActionLabel string `json:"actionLabel,omitempty"`
}

type DigitalStoreHealthCheckResponse struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	ActionHref  string `json:"actionHref,omitempty"`
	ActionLabel string `json:"actionLabel,omitempty"`
}

type DigitalStoreWebhookTestResponse struct {
	DigitalStoreNotificationStatusResponse
	Sent        bool                                      `json:"sent"`
	TestedAt    string                                    `json:"testedAt"`
	SentTotal   int                                       `json:"sentTotal"`
	FailedTotal int                                       `json:"failedTotal"`
	Scenarios   []DigitalStoreWebhookTestScenarioResponse `json:"scenarios,omitempty"`
}

type DigitalStoreWebhookTestScenarioResponse struct {
	Key       string `json:"key"`
	EventType string `json:"eventType"`
	Title     string `json:"title"`
	Sent      bool   `json:"sent"`
	Message   string `json:"message"`
}

type DigitalStoreDemoDataCleanupResponse struct {
	CleanedAt string           `json:"cleanedAt"`
	Message   string           `json:"message"`
	Deleted   map[string]int64 `json:"deleted"`
}

type DigitalStoreMaintenanceStatusResponse struct {
	CheckedAt            string                                   `json:"checkedAt"`
	Status               string                                   `json:"status"`
	BackupRoot           string                                   `json:"backupRoot"`
	BackupCommand        string                                   `json:"backupCommand"`
	RestoreDryRunCommand string                                   `json:"restoreDryRunCommand"`
	UpgradeCommands      []string                                 `json:"upgradeCommands"`
	UpgradeRunbook       string                                   `json:"upgradeRunbook"`
	LatestBackup         *DigitalStoreBackupSnapshotResponse      `json:"latestBackup,omitempty"`
	Warnings             []DigitalStoreMaintenanceWarningResponse `json:"warnings"`
}

type DigitalStoreBackupSnapshotResponse struct {
	Path                   string `json:"path"`
	Timestamp              string `json:"timestamp"`
	CreatedAt              string `json:"createdAt"`
	ProjectDir             string `json:"projectDir"`
	ComposeFile            string `json:"composeFile"`
	HasManifest            bool   `json:"hasManifest"`
	HasMySQLDump           bool   `json:"hasMysqlDump"`
	HasDataArchive         bool   `json:"hasDataArchive"`
	HasDockerConfigArchive bool   `json:"hasDockerConfigArchive"`
	HasConfigSnapshot      bool   `json:"hasConfigSnapshot"`
	SizeBytes              int64  `json:"sizeBytes"`
}

type DigitalStoreMaintenanceWarningResponse struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Message string `json:"message"`
}

type DigitalStoreDeliveryReportResponse struct {
	GeneratedAt        string                                 `json:"generatedAt"`
	BrandName          string                                 `json:"brandName"`
	StoreName          string                                 `json:"storeName"`
	Ready              bool                                   `json:"ready"`
	DashboardURL       string                                 `json:"dashboardUrl"`
	ChatURL            string                                 `json:"chatUrl"`
	EmbedSnippet       string                                 `json:"embedSnippet"`
	WebEntry           DigitalStoreWebEntryResponse           `json:"webEntry"`
	HumanHandoff       DigitalStoreHumanHandoffResponse       `json:"humanHandoff"`
	AcceptanceCommand  string                                 `json:"acceptanceCommand"`
	AcceptanceItems    []DigitalStoreAcceptanceItem           `json:"acceptanceItems"`
	NotificationStatus DigitalStoreNotificationStatusResponse `json:"notificationStatus"`
	SecurityChecks     []DigitalStoreSecurityCheckResponse    `json:"securityChecks"`
	ModelHealthChecks  []DigitalStoreHealthCheckResponse      `json:"modelHealthChecks"`
	Items              []DigitalStoreDeliveryReportItem       `json:"items"`
	MissingSteps       []string                               `json:"missingSteps"`
	LatestRecord       *DigitalStoreDeliveryRecordResponse    `json:"latestRecord,omitempty"`
	Markdown           string                                 `json:"markdown"`
	AcceptanceRunbook  string                                 `json:"acceptanceRunbook"`
}

type DigitalStoreDeliveryRecordResponse struct {
	ID                   int64                                          `json:"id"`
	BrandName            string                                         `json:"brandName"`
	StoreName            string                                         `json:"storeName"`
	Ready                bool                                           `json:"ready"`
	AcceptanceStatus     string                                         `json:"acceptanceStatus"`
	AcceptanceSummary    string                                         `json:"acceptanceSummary"`
	AcceptanceCommand    string                                         `json:"acceptanceCommand"`
	ScenarioTotal        int                                            `json:"scenarioTotal"`
	PassedTotal          int                                            `json:"passedTotal"`
	FailedTotal          int                                            `json:"failedTotal"`
	AcceptanceStartedAt  string                                         `json:"acceptanceStartedAt,omitempty"`
	AcceptanceFinishedAt string                                         `json:"acceptanceFinishedAt,omitempty"`
	DashboardURL         string                                         `json:"dashboardUrl"`
	ChatURL              string                                         `json:"chatUrl"`
	WebChannelCode       string                                         `json:"webChannelCode"`
	CreatedAt            string                                         `json:"createdAt"`
	CreateUserName       string                                         `json:"createUserName"`
	AcceptanceResults    []DigitalStoreAcceptanceScenarioResultResponse `json:"acceptanceResults,omitempty"`
}

type DigitalStoreAcceptanceScenarioResultResponse struct {
	Code             string   `json:"code"`
	Title            string   `json:"title"`
	Passed           bool     `json:"passed"`
	Reason           string   `json:"reason"`
	FailureType      string   `json:"failureType"`
	Detail           string   `json:"detail"`
	Suggestion       string   `json:"suggestion"`
	ConversationID   int64    `json:"conversationId"`
	ConversationURL  string   `json:"conversationUrl"`
	Reply            string   `json:"reply"`
	ExpectedKeywords []string `json:"expectedKeywords"`
	MatchedKeywords  []string `json:"matchedKeywords"`
	MissingKeywords  []string `json:"missingKeywords"`
	BannedKeywords   []string `json:"bannedKeywords"`
	MatchedBanned    string   `json:"matchedBanned"`
}

type DigitalStoreTemplateResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Industry    string `json:"industry"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type DigitalStoreTemplateExportResponse struct {
	SchemaVersion   string                                  `json:"schemaVersion"`
	ExportedAt      string                                  `json:"exportedAt"`
	Template        DigitalStoreTemplateResponse            `json:"template"`
	Profile         DigitalStoreProfileResponse             `json:"profile"`
	Products        []DigitalStoreTemplateProductResponse   `json:"products"`
	Promotions      []DigitalStoreTemplatePromotionResponse `json:"promotions"`
	RiskRules       []DigitalStoreIndustryRiskRuleResponse  `json:"riskRules"`
	AcceptanceItems []DigitalStoreAcceptanceItem            `json:"acceptanceItems"`
}

type DigitalStoreTemplatePreviewResponse struct {
	Template             DigitalStoreTemplateResponse           `json:"template"`
	Profile              DigitalStoreProfileResponse            `json:"profile"`
	ProfileAction        string                                 `json:"profileAction"`
	ProductCreateTotal   int                                    `json:"productCreateTotal"`
	ProductUpdateTotal   int                                    `json:"productUpdateTotal"`
	PromotionCreateTotal int                                    `json:"promotionCreateTotal"`
	PromotionUpdateTotal int                                    `json:"promotionUpdateTotal"`
	Products             []DigitalStoreTemplatePreviewItem      `json:"products"`
	Promotions           []DigitalStoreTemplatePreviewItem      `json:"promotions"`
	RiskRules            []DigitalStoreIndustryRiskRuleResponse `json:"riskRules"`
	AcceptanceItems      []DigitalStoreAcceptanceItem           `json:"acceptanceItems"`
	Warnings             []DigitalStoreTemplatePreviewWarning   `json:"warnings"`
}

type DigitalStoreIndustryRiskRuleResponse struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	ForbiddenClaims []string `json:"forbiddenClaims"`
	HandoffTriggers []string `json:"handoffTriggers"`
}

type DigitalStoreTemplatePreviewItem struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	ExistingID int64  `json:"existingId,omitempty"`
	Reason     string `json:"reason"`
}

type DigitalStoreTemplatePreviewWarning struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

type DigitalStoreKnowledgeAssistantResponse struct {
	GeneratedAt     string                               `json:"generatedAt"`
	Industry        string                               `json:"industry"`
	KnowledgeBaseID int64                                `json:"knowledgeBaseId"`
	CoveredTotal    int                                  `json:"coveredTotal"`
	MissingTotal    int                                  `json:"missingTotal"`
	Items           []DigitalStoreKnowledgeAssistantItem `json:"items"`
}

type DigitalStoreKnowledgeAssistantItem struct {
	Key          string   `json:"key"`
	Question     string   `json:"question"`
	Reason       string   `json:"reason"`
	Required     bool     `json:"required"`
	Covered      bool     `json:"covered"`
	MatchedFAQID int64    `json:"matchedFaqId,omitempty"`
	Keywords     []string `json:"keywords"`
	ActionHref   string   `json:"actionHref,omitempty"`
	ActionLabel  string   `json:"actionLabel,omitempty"`
}

type DigitalStoreTemplateEffectResponse struct {
	GeneratedAt           string                           `json:"generatedAt"`
	TemplateCode          string                           `json:"templateCode"`
	TemplateVersion       string                           `json:"templateVersion"`
	TemplateAppliedAt     string                           `json:"templateAppliedAt,omitempty"`
	Industry              string                           `json:"industry"`
	KnowledgeBaseID       int64                            `json:"knowledgeBaseId"`
	Days                  int                              `json:"days"`
	RetrieveTotal         int64                            `json:"retrieveTotal"`
	MissingQuestionTotal  int64                            `json:"missingQuestionTotal"`
	NegativeFeedbackTotal int64                            `json:"negativeFeedbackTotal"`
	MissingQuestions      []DigitalStoreTemplateEffectItem `json:"missingQuestions"`
	NegativeFeedbacks     []DigitalStoreTemplateEffectItem `json:"negativeFeedbacks"`
	Suggestions           []string                         `json:"suggestions"`
	ImprovementMarkdown   string                           `json:"improvementMarkdown"`
}

type DigitalStoreTemplateEffectItem struct {
	Question            string `json:"question"`
	Count               int64  `json:"count"`
	LatestAt            string `json:"latestAt,omitempty"`
	FeedbackReason      string `json:"feedbackReason,omitempty"`
	FeedbackTypeName    string `json:"feedbackTypeName,omitempty"`
	AnswerStatusName    string `json:"answerStatusName,omitempty"`
	ActionHref          string `json:"actionHref,omitempty"`
	ActionLabel         string `json:"actionLabel,omitempty"`
	CreateFAQActionHref string `json:"createFaqActionHref,omitempty"`
}

type DigitalStoreTemplateProductResponse struct {
	Name               string `json:"name"`
	Category           string `json:"category"`
	PriceMin           int64  `json:"priceMin"`
	PriceMax           int64  `json:"priceMax"`
	SellingPoints      string `json:"sellingPoints"`
	SuitablePeople     string `json:"suitablePeople"`
	UnsuitablePeople   string `json:"unsuitablePeople"`
	Scenarios          string `json:"scenarios"`
	Specs              string `json:"specs"`
	IndustryAttributes string `json:"industryAttributes"`
	ImageURL           string `json:"imageUrl"`
	Priority           int    `json:"priority"`
	Status             int    `json:"status"`
	Remark             string `json:"remark"`
}

type DigitalStoreTemplatePromotionResponse struct {
	Name               string `json:"name"`
	PromotionType      string `json:"promotionType"`
	Description        string `json:"description"`
	ApplicableProducts string `json:"applicableProducts"`
	StartAt            string `json:"startAt"`
	EndAt              string `json:"endAt"`
	DiscountRule       string `json:"discountRule"`
	StoreBenefit       string `json:"storeBenefit"`
	AppointmentBenefit string `json:"appointmentBenefit"`
	ScriptSuggestion   string `json:"scriptSuggestion"`
	Priority           int    `json:"priority"`
	Status             int    `json:"status"`
	Remark             string `json:"remark"`
}
