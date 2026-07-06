package response

import "agent-desk/internal/pkg/enums"

type ProductResponse struct {
	ID                 int64        `json:"id"`
	Name               string       `json:"name"`
	Category           string       `json:"category"`
	PriceMin           int64        `json:"priceMin"`
	PriceMax           int64        `json:"priceMax"`
	SellingPoints      string       `json:"sellingPoints"`
	SuitablePeople     string       `json:"suitablePeople"`
	UnsuitablePeople   string       `json:"unsuitablePeople"`
	Scenarios          string       `json:"scenarios"`
	Specs              string       `json:"specs"`
	IndustryAttributes string       `json:"industryAttributes"`
	ImageURL           string       `json:"imageUrl"`
	Priority           int          `json:"priority"`
	KnowledgeBaseID    int64        `json:"knowledgeBaseId"`
	KnowledgeFAQID     int64        `json:"knowledgeFAQId"`
	Status             enums.Status `json:"status"`
	Remark             string       `json:"remark"`
	CreatedAt          string       `json:"createdAt,omitempty"`
	UpdatedAt          string       `json:"updatedAt,omitempty"`
}

type ProductImportResultResponse struct {
	Total   int                        `json:"total"`
	Created int                        `json:"created"`
	Updated int                        `json:"updated"`
	Skipped int                        `json:"skipped"`
	Failed  int                        `json:"failed"`
	Errors  []ProductImportRowResponse `json:"errors"`
}

type ProductImportRowResponse struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}
