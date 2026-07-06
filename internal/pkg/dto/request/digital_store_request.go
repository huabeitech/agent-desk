package request

type DigitalStoreProfileRequest struct {
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
	Initialized          bool   `json:"initialized"`
}

type DigitalStoreDeliveryRecordCreateRequest struct {
	PublicBaseURL     string `json:"publicBaseUrl"`
	AcceptanceStatus  string `json:"acceptanceStatus"`
	AcceptanceSummary string `json:"acceptanceSummary"`
}

type DigitalStoreAcceptanceScenarioResultRequest struct {
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

type DigitalStoreAcceptanceResultCreateRequest struct {
	PublicBaseURL string                                        `json:"publicBaseUrl"`
	Command       string                                        `json:"command"`
	ScenarioTotal int                                           `json:"scenarioTotal"`
	PassedTotal   int                                           `json:"passedTotal"`
	FailedTotal   int                                           `json:"failedTotal"`
	StartedAt     string                                        `json:"startedAt"`
	FinishedAt    string                                        `json:"finishedAt"`
	Results       []DigitalStoreAcceptanceScenarioResultRequest `json:"results"`
}

type DigitalStoreApplyTemplateRequest struct {
	TemplateCode string `json:"templateCode"`
}

type DigitalStoreTemplateImportRequest struct {
	SchemaVersion string                                `json:"schemaVersion"`
	ExportedAt    string                                `json:"exportedAt"`
	Template      DigitalStoreTemplateImportMetaRequest `json:"template"`
	Profile       DigitalStoreProfileRequest            `json:"profile"`
	Products      []SaveProductRequest                  `json:"products"`
	Promotions    []SavePromotionRequest                `json:"promotions"`
}

type DigitalStoreTemplateImportMetaRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Industry    string `json:"industry"`
	Version     string `json:"version"`
	Description string `json:"description"`
}
