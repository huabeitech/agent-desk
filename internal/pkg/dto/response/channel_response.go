package response

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/utils"
)

type ChannelResponse struct {
	ID                            int64        `json:"id"`
	ChannelType                   string       `json:"channelType"`
	ChannelID                     string       `json:"channelId"`
	AIAgentID                     int64        `json:"aiAgentId"`
	AIAgentRolloutPercent         int          `json:"aiAgentRolloutPercent"`
	PreviousAIAgentRolloutPercent int          `json:"previousAiAgentRolloutPercent"`
	AIReplyPlaceholder            string       `json:"aiReplyPlaceholder"`
	AIReplyTimeoutSeconds         int          `json:"aiReplyTimeoutSeconds"`
	AIReplyTimeoutNotice          string       `json:"aiReplyTimeoutNotice"`
	AIAgentName                   string       `json:"aiAgentName,omitempty"`
	Name                          string       `json:"name"`
	ConfigJSON                    string       `json:"configJson"`
	Status                        enums.Status `json:"status"`
	Remark                        string       `json:"remark"`
}

type WxWorkKFAccountResponse struct {
	OpenKfID        string `json:"openKfId"`
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	ManagePrivilege bool   `json:"managePrivilege"`
}

type ChannelMessageOutboxResponse struct {
	ID             int64  `json:"id"`
	ChannelType    string `json:"channelType"`
	ConversationID int64  `json:"conversationId"`
	MessageID      int64  `json:"messageId"`
	Payload        string `json:"payload"`
	SendStatus     string `json:"sendStatus"`
	RetryCount     int    `json:"retryCount"`
	NextRetryAt    string `json:"nextRetryAt"`
	LastError      string `json:"lastError"`
	SentAt         string `json:"sentAt"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	CreateUserName string `json:"createUserName"`
	UpdateUserName string `json:"updateUserName"`
}

func BuildChannelResponse(item *models.Channel) ChannelResponse {
	if item == nil {
		return ChannelResponse{}
	}
	return ChannelResponse{
		ID:                            item.ID,
		ChannelType:                   item.ChannelType,
		ChannelID:                     item.ChannelID,
		AIAgentID:                     item.AIAgentID,
		AIAgentRolloutPercent:         item.AIAgentRolloutPercent,
		PreviousAIAgentRolloutPercent: item.PreviousAIAgentRolloutPercent,
		AIReplyPlaceholder:            item.AIReplyPlaceholder,
		AIReplyTimeoutSeconds:         item.AIReplyTimeoutSeconds,
		AIReplyTimeoutNotice:          item.AIReplyTimeoutNotice,
		Name:                          item.Name,
		ConfigJSON:                    item.ConfigJSON,
		Status:                        item.Status,
		Remark:                        item.Remark,
	}
}

func BuildChannelMessageOutboxResponse(item *models.ChannelMessageOutbox) ChannelMessageOutboxResponse {
	if item == nil {
		return ChannelMessageOutboxResponse{}
	}
	return ChannelMessageOutboxResponse{
		ID:             item.ID,
		ChannelType:    item.ChannelType,
		ConversationID: item.ConversationID,
		MessageID:      item.MessageID,
		Payload:        item.Payload,
		SendStatus:     item.SendStatus,
		RetryCount:     item.RetryCount,
		NextRetryAt:    utils.FormatTimePtr(item.NextRetryAt),
		LastError:      item.LastError,
		SentAt:         utils.FormatTimePtr(item.SentAt),
		CreatedAt:      utils.FormatTime(item.CreatedAt),
		UpdatedAt:      utils.FormatTime(item.UpdatedAt),
		CreateUserName: item.CreateUserName,
		UpdateUserName: item.UpdateUserName,
	}
}
