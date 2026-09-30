package request

type CreateChannelRequest struct {
	ChannelType           string `json:"channelType"`
	AIAgentID             int64  `json:"aiAgentId"`
	AIAgentRolloutPercent int    `json:"aiAgentRolloutPercent"`
	AIReplyPlaceholder    string `json:"aiReplyPlaceholder"`
	AIReplyTimeoutSeconds int    `json:"aiReplyTimeoutSeconds"`
	AIReplyTimeoutNotice  string `json:"aiReplyTimeoutNotice"`
	Name                  string `json:"name"`
	ConfigJSON            string `json:"configJson"`
	Status                int    `json:"status"`
	Remark                string `json:"remark"`
}

type UpdateChannelRequest struct {
	ID int64 `json:"id"`
	CreateChannelRequest
}

type UpdateChannelStatusRequest struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}

type RollbackChannelAIAgentRolloutRequest struct {
	ID int64 `json:"id"`
}

type DeleteChannelRequest struct {
	ID int64 `json:"id"`
}

type ResetChannelUserTokenSecretRequest struct {
	ID int64 `json:"id"`
}

type TestWxWorkKFReadMessagesRequest struct {
	ID int64 `json:"id"`
}

type ChannelMessageOutboxActionRequest struct {
	ID int64 `json:"id"`
}
