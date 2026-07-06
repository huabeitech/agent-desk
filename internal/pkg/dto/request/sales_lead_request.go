package request

type SalesLeadListRequest struct {
	Page              int    `json:"page"`
	Limit             int    `json:"limit"`
	Keyword           string `json:"keyword"`
	Status            string `json:"status"`
	Intent            string `json:"intent"`
	TaskView          string `json:"taskView"`
	FollowUpStatus    string `json:"followUpStatus"`
	AppointmentStatus string `json:"appointmentStatus"`
	OwnerUserID       *int64 `json:"ownerUserId"`
}

func (r SalesLeadListRequest) GetPage() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

func (r SalesLeadListRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 20
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

func (r SalesLeadListRequest) Offset() int {
	return (r.GetPage() - 1) * r.GetLimit()
}

type UpdateSalesLeadRequest struct {
	ID                  int64  `json:"id"`
	CustomerName        string `json:"customerName"`
	Phone               string `json:"phone"`
	WeChat              string `json:"wechat"`
	City                string `json:"city"`
	AddressHint         string `json:"addressHint"`
	BudgetMin           int64  `json:"budgetMin"`
	BudgetMax           int64  `json:"budgetMax"`
	InterestedProducts  string `json:"interestedProducts"`
	DemandSummary       string `json:"demandSummary"`
	IntentLevel         string `json:"intentLevel"`
	BuyingStage         string `json:"buyingStage"`
	AppointmentAt       string `json:"appointmentAt"`
	AppointmentTimeText string `json:"appointmentTimeText"`
	AppointmentStore    string `json:"appointmentStore"`
	AppointmentPeople   int    `json:"appointmentPeople"`
	AppointmentRemark   string `json:"appointmentRemark"`
	OwnerUserID         int64  `json:"ownerUserId"`
	Status              string `json:"status"`
	Remark              string `json:"remark"`
}

type AssignSalesLeadRequest struct {
	ID          int64 `json:"id"`
	OwnerUserID int64 `json:"ownerUserId"`
}

type UpdateSalesLeadStatusRequest struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Remark string `json:"remark"`
}

type SyncSalesLeadToCRMRequest struct {
	ID     int64  `json:"id"`
	Remark string `json:"remark"`
}

type ClaimUnassignedSalesLeadsRequest struct {
	Keyword           string `json:"keyword"`
	Status            string `json:"status"`
	Intent            string `json:"intent"`
	TaskView          string `json:"taskView"`
	FollowUpStatus    string `json:"followUpStatus"`
	AppointmentStatus string `json:"appointmentStatus"`
	Limit             int    `json:"limit"`
}

func (r ClaimUnassignedSalesLeadsRequest) ToListRequest() SalesLeadListRequest {
	unassigned := int64(-1)
	return SalesLeadListRequest{
		Page:              1,
		Limit:             r.GetLimit(),
		Keyword:           r.Keyword,
		Status:            r.Status,
		Intent:            r.Intent,
		TaskView:          r.TaskView,
		FollowUpStatus:    r.FollowUpStatus,
		AppointmentStatus: r.AppointmentStatus,
		OwnerUserID:       &unassigned,
	}
}

func (r ClaimUnassignedSalesLeadsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 20
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

type CreateLeadFollowUpRequest struct {
	LeadID         int64  `json:"leadId"`
	Content        string `json:"content"`
	NextAction     string `json:"nextAction"`
	NextFollowUpAt string `json:"nextFollowUpAt"`
}

type SalesLeadFollowUpReminderRequest struct {
	OwnerUserID *int64 `json:"ownerUserId"`
	Limit       int    `json:"limit"`
}

func (r SalesLeadFollowUpReminderRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	if r.Limit > 50 {
		return 50
	}
	return r.Limit
}

type SalesLeadAppointmentSummaryRequest struct {
	OwnerUserID *int64 `json:"ownerUserId"`
	Days        int    `json:"days"`
	Limit       int    `json:"limit"`
}

func (r SalesLeadAppointmentSummaryRequest) GetDays() int {
	if r.Days <= 0 {
		return 7
	}
	if r.Days > 30 {
		return 30
	}
	return r.Days
}

func (r SalesLeadAppointmentSummaryRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 8
	}
	if r.Limit > 50 {
		return 50
	}
	return r.Limit
}
