package response

import "agent-desk/internal/pkg/enums"

type ConversationTagResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ConversationParticipantResponse struct {
	ID                    int64        `json:"id"`
	ParticipantType       string       `json:"participantType"`
	ParticipantID         int64        `json:"participantId"`
	ExternalParticipantID string       `json:"externalParticipantId,omitempty"`
	JoinedAt              string       `json:"joinedAt,omitempty"`
	LeftAt                string       `json:"leftAt,omitempty"`
	Status                enums.Status `json:"status"`
}

type ConversationResponse struct {
	ID                        int64                           `json:"id"`
	AIAgentID                 int64                           `json:"aiAgentId"`
	ChannelID                 int64                           `json:"channelId"`
	CustomerID                int64                           `json:"customerId"`
	CustomerName              string                          `json:"customerName"`
	Status                    enums.IMConversationStatus      `json:"status"`
	ServiceMode               enums.IMConversationServiceMode `json:"serviceMode"`
	Priority                  int                             `json:"priority"`
	CurrentAssigneeID         int64                           `json:"currentAssigneeId"`
	CurrentAssigneeName       string                          `json:"currentAssigneeName,omitempty"`
	CurrentTeamID             int64                           `json:"currentTeamId"`
	CurrentTeamName           string                          `json:"currentTeamName,omitempty"`
	LastMessageID             int64                           `json:"lastMessageId"`
	LastMessageAt             string                          `json:"lastMessageAt,omitempty"`
	LastActiveAt              string                          `json:"lastActiveAt,omitempty"`
	LastMessageSummary        string                          `json:"lastMessageSummary,omitempty"`
	CustomerUnreadCount       int                             `json:"customerUnreadCount"`
	AgentUnreadCount          int                             `json:"agentUnreadCount"`
	CustomerLastReadMessageID int64                           `json:"customerLastReadMessageId"`
	CustomerLastReadAt        string                          `json:"customerLastReadAt,omitempty"`
	AgentLastReadMessageID    int64                           `json:"agentLastReadMessageId"`
	AgentLastReadAt           string                          `json:"agentLastReadAt,omitempty"`
	CustomerOnline            bool                            `json:"customerOnline"`
	ClosedAt                  string                          `json:"closedAt,omitempty"`
	ClosedBy                  int64                           `json:"closedBy"`
	ClosedByName              string                          `json:"closedByName,omitempty"`
	CloseReason               string                          `json:"closeReason,omitempty"`
}

type ConversationDetailResponse struct {
	ConversationResponse
	Participants []ConversationParticipantResponse `json:"participants,omitempty"`
}

type ConversationFollowUpAdviceResponse struct {
	ConversationID int64  `json:"conversationId"`
	LeadID         int64  `json:"leadId,omitempty"`
	Source         string `json:"source"`
	SalesLeadFollowUpAdviceResult
}
