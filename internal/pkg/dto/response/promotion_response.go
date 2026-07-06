package response

import "agent-desk/internal/pkg/enums"

type PromotionResponse struct {
	ID                 int64        `json:"id"`
	Name               string       `json:"name"`
	PromotionType      string       `json:"promotionType"`
	Description        string       `json:"description"`
	ApplicableProducts string       `json:"applicableProducts"`
	StartAt            string       `json:"startAt,omitempty"`
	EndAt              string       `json:"endAt,omitempty"`
	DiscountRule       string       `json:"discountRule"`
	StoreBenefit       string       `json:"storeBenefit"`
	AppointmentBenefit string       `json:"appointmentBenefit"`
	ScriptSuggestion   string       `json:"scriptSuggestion"`
	Priority           int          `json:"priority"`
	KnowledgeBaseID    int64        `json:"knowledgeBaseId"`
	KnowledgeFAQID     int64        `json:"knowledgeFAQId"`
	Status             enums.Status `json:"status"`
	Remark             string       `json:"remark"`
	CreatedAt          string       `json:"createdAt,omitempty"`
	UpdatedAt          string       `json:"updatedAt,omitempty"`
}

type PromotionImportResultResponse struct {
	Total   int                          `json:"total"`
	Created int                          `json:"created"`
	Updated int                          `json:"updated"`
	Skipped int                          `json:"skipped"`
	Failed  int                          `json:"failed"`
	Errors  []PromotionImportRowResponse `json:"errors"`
}

type PromotionImportRowResponse struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}
