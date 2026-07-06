package response

import "agent-desk/internal/pkg/enums"

type SalesLeadResponse struct {
	ID                  int64                 `json:"id"`
	CustomerID          int64                 `json:"customerId"`
	ConversationID      int64                 `json:"conversationId"`
	CustomerName        string                `json:"customerName"`
	Phone               string                `json:"phone"`
	WeChat              string                `json:"wechat"`
	City                string                `json:"city"`
	AddressHint         string                `json:"addressHint"`
	BudgetMin           int64                 `json:"budgetMin"`
	BudgetMax           int64                 `json:"budgetMax"`
	InterestedProducts  string                `json:"interestedProducts"`
	DemandSummary       string                `json:"demandSummary"`
	IntentLevel         enums.SalesLeadIntent `json:"intentLevel"`
	BuyingStage         enums.SalesLeadStage  `json:"buyingStage"`
	AppointmentAt       string                `json:"appointmentAt,omitempty"`
	AppointmentTimeText string                `json:"appointmentTimeText"`
	AppointmentStore    string                `json:"appointmentStore"`
	AppointmentPeople   int                   `json:"appointmentPeople"`
	AppointmentRemark   string                `json:"appointmentRemark"`
	SourceChannel       string                `json:"sourceChannel"`
	OwnerUserID         int64                 `json:"ownerUserId"`
	OwnerUserName       string                `json:"ownerUserName,omitempty"`
	Status              enums.SalesLeadStatus `json:"status"`
	NextFollowUpAt      string                `json:"nextFollowUpAt,omitempty"`
	LastMessageID       int64                 `json:"lastMessageId"`
	LastMessageSummary  string                `json:"lastMessageSummary"`
	LastCustomerMessage string                `json:"lastCustomerMessage"`
	MergeKey            string                `json:"mergeKey"`
	MergeReason         string                `json:"mergeReason"`
	MergedAt            string                `json:"mergedAt,omitempty"`
	Remark              string                `json:"remark"`
	AutoTags            []string              `json:"autoTags"`
	AutoTagDetails      []SalesLeadAutoTag    `json:"autoTagDetails"`
	CreatedAt           string                `json:"createdAt,omitempty"`
	UpdatedAt           string                `json:"updatedAt,omitempty"`
	Customer            *CustomerResponse     `json:"customer,omitempty"`
}

type SalesLeadAutoTag struct {
	Label       string `json:"label"`
	Level       string `json:"level"`
	Reason      string `json:"reason"`
	ActionLabel string `json:"actionLabel"`
	ActionURL   string `json:"actionUrl,omitempty"`
}

type LeadFollowUpResponse struct {
	ID             int64  `json:"id"`
	LeadID         int64  `json:"leadId"`
	OperatorID     int64  `json:"operatorId"`
	OperatorName   string `json:"operatorName"`
	Content        string `json:"content"`
	NextAction     string `json:"nextAction"`
	NextFollowUpAt string `json:"nextFollowUpAt,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
}

type SalesLeadDetailResponse struct {
	Lead           SalesLeadResponse             `json:"lead"`
	FollowUps      []LeadFollowUpResponse        `json:"followUps,omitempty"`
	FollowUpAdvice SalesLeadFollowUpAdviceResult `json:"followUpAdvice"`
}

type SalesLeadFollowUpAdviceResult struct {
	CustomerSummary string   `json:"customerSummary"`
	NextAction      string   `json:"nextAction"`
	Script          string   `json:"script"`
	CopyText        string   `json:"copyText"`
	RiskHints       []string `json:"riskHints"`
}

type ClaimUnassignedSalesLeadsResponse struct {
	ClaimedCount int64   `json:"claimedCount"`
	LeadIDs      []int64 `json:"leadIds"`
	Message      string  `json:"message"`
}

type SalesLeadCRMSyncResponse struct {
	LeadID           int64  `json:"leadId"`
	GeneratedAt      string `json:"generatedAt"`
	WebhookEnabled   bool   `json:"webhookEnabled"`
	Sent             bool   `json:"sent"`
	Title            string `json:"title"`
	Message          string `json:"message"`
	WebhookEventType string `json:"webhookEventType"`
}

type SalesLeadFollowUpReminderLeadResponse struct {
	ID             int64                 `json:"id"`
	CustomerName   string                `json:"customerName"`
	Phone          string                `json:"phone"`
	WeChat         string                `json:"wechat"`
	IntentLevel    enums.SalesLeadIntent `json:"intentLevel"`
	Status         enums.SalesLeadStatus `json:"status"`
	OwnerUserID    int64                 `json:"ownerUserId"`
	OwnerUserName  string                `json:"ownerUserName,omitempty"`
	NextFollowUpAt string                `json:"nextFollowUpAt,omitempty"`
	FollowUpState  string                `json:"followUpState"`
	DemandSummary  string                `json:"demandSummary"`
	ActionURL      string                `json:"actionUrl"`
}

type SalesLeadFollowUpReminderSummaryResponse struct {
	GeneratedAt          string                                  `json:"generatedAt"`
	OverdueCount         int64                                   `json:"overdueCount"`
	TodayCount           int64                                   `json:"todayCount"`
	DueCount             int64                                   `json:"dueCount"`
	UnassignedDueCount   int64                                   `json:"unassignedDueCount"`
	MissingScheduleCount int64                                   `json:"missingScheduleCount"`
	PreviewLeads         []SalesLeadFollowUpReminderLeadResponse `json:"previewLeads"`
	Message              string                                  `json:"message"`
	NotificationSent     bool                                    `json:"notificationSent"`
}

type SalesLeadAppointmentItemResponse struct {
	ID                  int64                 `json:"id"`
	CustomerName        string                `json:"customerName"`
	Phone               string                `json:"phone"`
	WeChat              string                `json:"wechat"`
	IntentLevel         enums.SalesLeadIntent `json:"intentLevel"`
	Status              enums.SalesLeadStatus `json:"status"`
	OwnerUserID         int64                 `json:"ownerUserId"`
	OwnerUserName       string                `json:"ownerUserName,omitempty"`
	AppointmentAt       string                `json:"appointmentAt,omitempty"`
	AppointmentTimeText string                `json:"appointmentTimeText"`
	AppointmentStore    string                `json:"appointmentStore"`
	AppointmentPeople   int                   `json:"appointmentPeople"`
	DemandSummary       string                `json:"demandSummary"`
	AppointmentState    string                `json:"appointmentState"`
	ActionURL           string                `json:"actionUrl"`
}

type SalesLeadAppointmentSummaryResponse struct {
	GeneratedAt         string                             `json:"generatedAt"`
	Days                int                                `json:"days"`
	OverdueCount        int64                              `json:"overdueCount"`
	TodayCount          int64                              `json:"todayCount"`
	UpcomingCount       int64                              `json:"upcomingCount"`
	UnscheduledCount    int64                              `json:"unscheduledCount"`
	UnassignedCount     int64                              `json:"unassignedCount"`
	PreviewAppointments []SalesLeadAppointmentItemResponse `json:"previewAppointments"`
	Message             string                             `json:"message"`
	NotificationSent    bool                               `json:"notificationSent"`
}
