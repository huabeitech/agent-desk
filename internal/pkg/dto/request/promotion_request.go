package request

type PromotionListRequest struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Keyword       string `json:"keyword"`
	PromotionType string `json:"promotionType"`
	ActiveOnly    bool   `json:"activeOnly"`
	Status        *int   `json:"status"`
}

func (r PromotionListRequest) GetPage() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

func (r PromotionListRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 20
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

func (r PromotionListRequest) Offset() int {
	return (r.GetPage() - 1) * r.GetLimit()
}

type SavePromotionRequest struct {
	ID                 int64  `json:"id"`
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
	KnowledgeBaseID    int64  `json:"knowledgeBaseId"`
	Status             int    `json:"status"`
	Remark             string `json:"remark"`
}

type DeletePromotionRequest struct {
	ID int64 `json:"id"`
}

type UpdatePromotionStatusRequest struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}

type ReindexPromotionRequest struct {
	ID int64 `json:"id"`
}
