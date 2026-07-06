package request

type AgentConversationFilter string

const (
	AgentConversationFilterAIServing AgentConversationFilter = "ai_serving"
	AgentConversationFilterMine      AgentConversationFilter = "mine"
	AgentConversationFilterActive    AgentConversationFilter = "active"
	AgentConversationFilterPending   AgentConversationFilter = "pending"
	AgentConversationFilterClosed    AgentConversationFilter = "closed"
)

type ConversationListRequest struct {
	Status            int    `json:"status"`
	ServiceMode       int    `json:"serviceMode"`
	CurrentAssigneeID int64  `json:"currentAssigneeId"`
	Keyword           string `json:"keyword"`
	TagID             int64  `json:"tagId"`
}

type AssignConversationRequest struct {
	ConversationID int64  `json:"conversationId"`
	AssigneeID     int64  `json:"assigneeId"`
	Reason         string `json:"reason"`
}

type DispatchConversationRequest struct {
	ConversationID int64 `json:"conversationId"`
}

type TransferConversationRequest struct {
	ConversationID int64  `json:"conversationId"`
	ToUserID       int64  `json:"toUserId"`
	Reason         string `json:"reason"`
}

type CloseConversationRequest struct {
	ConversationID int64  `json:"conversationId"`
	CloseReason    string `json:"closeReason"`
}

type ResumeAIConversationRequest struct {
	ConversationID int64  `json:"conversationId"`
	Reason         string `json:"reason"`
}

type ConversationFollowUpAdviceRequest struct {
	ConversationID int64 `json:"conversationId"`
}

type ReadConversationRequest struct {
	ConversationID int64 `json:"conversationId"`
	MessageID      int64 `json:"messageId"`
}

type AddConversationTagRequest struct {
	ConversationID int64 `json:"conversationId"`
	TagID          int64 `json:"tagId"`
}

type RemoveConversationTagRequest struct {
	ConversationID int64 `json:"conversationId"`
	TagID          int64 `json:"tagId"`
}

// LinkConversationCustomerRequest 将客服会话关联到 CRM 客户（并同步访客身份映射）。
type LinkConversationCustomerRequest struct {
	ConversationID int64 `json:"conversationId"`
	CustomerID     int64 `json:"customerId"`
}
