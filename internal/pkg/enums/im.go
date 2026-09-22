package enums

type IMConversationStatus int

const (
	IMConversationStatusAIServing IMConversationStatus = 1
	IMConversationStatusPending   IMConversationStatus = 2
	IMConversationStatusActive    IMConversationStatus = 3
	IMConversationStatusClosed    IMConversationStatus = 4
)

var imConversationStatusLabelMap = map[IMConversationStatus]string{
	IMConversationStatusAIServing: "AI接待中",
	IMConversationStatusPending:   "待接入",
	IMConversationStatusActive:    "处理中",
	IMConversationStatusClosed:    "已关闭",
}

var IMConversationStatusValues = []IMConversationStatus{
	IMConversationStatusAIServing,
	IMConversationStatusPending,
	IMConversationStatusActive,
	IMConversationStatusClosed,
}

func GetIMConversationStatusLabel(status IMConversationStatus) string {
	return imConversationStatusLabelMap[status]
}

type IMConversationServiceMode int

const (
	IMConversationServiceModeAIOnly    IMConversationServiceMode = 1
	IMConversationServiceModeHumanOnly IMConversationServiceMode = 2
	IMConversationServiceModeAIFirst   IMConversationServiceMode = 3
)

var IMConversationServiceModeValues = []IMConversationServiceMode{
	IMConversationServiceModeAIOnly,
	IMConversationServiceModeHumanOnly,
	IMConversationServiceModeAIFirst,
}

var imConversationServiceModeLabelMap = map[IMConversationServiceMode]string{
	IMConversationServiceModeAIOnly:    "仅AI",
	IMConversationServiceModeHumanOnly: "仅人工",
	IMConversationServiceModeAIFirst:   "AI优先",
}

func GetIMConversationServiceModeLabel(mode IMConversationServiceMode) string {
	return imConversationServiceModeLabelMap[mode]
}

type IMSenderType string

const (
	IMSenderTypeAgent    IMSenderType = "agent"    // 客服
	IMSenderTypeCustomer IMSenderType = "customer" // 客户
	IMSenderTypeAI       IMSenderType = "ai"       // AI
	IMSenderTypeSystem   IMSenderType = "system"   // 系统
	IMSenderTypeNote     IMSenderType = "note"     // 内部便签
)

var imSenderTypeLabelMap = map[IMSenderType]string{
	IMSenderTypeAgent:    "客服",
	IMSenderTypeCustomer: "客户",
	IMSenderTypeAI:       "AI",
	IMSenderTypeSystem:   "系统",
	IMSenderTypeNote:     "便签",
}

func GetIMSenderTypeLabel(senderType IMSenderType) string {
	return imSenderTypeLabelMap[senderType]
}

type IMParticipantType string

const (
	IMParticipantTypeCustomer IMParticipantType = "customer"
	IMParticipantTypeAgent    IMParticipantType = "agent"
	IMParticipantTypeAI       IMParticipantType = "ai"
	IMParticipantTypeSystem   IMParticipantType = "system"
)

var imParticipantTypeLabelMap = map[IMParticipantType]string{
	IMParticipantTypeCustomer: "客户",
	IMParticipantTypeAgent:    "客服",
	IMParticipantTypeAI:       "AI",
	IMParticipantTypeSystem:   "系统",
}

func GetIMParticipantTypeLabel(participantType IMParticipantType) string {
	return imParticipantTypeLabelMap[participantType]
}

type IMAssignmentType string

const (
	IMAssignmentTypeAssign   IMAssignmentType = "assign"
	IMAssignmentTypeTransfer IMAssignmentType = "transfer"
)

var imAssignmentTypeLabelMap = map[IMAssignmentType]string{
	IMAssignmentTypeAssign:   "分配",
	IMAssignmentTypeTransfer: "转接",
}

func GetIMAssignmentTypeLabel(assignType IMAssignmentType) string {
	return imAssignmentTypeLabelMap[assignType]
}

type IMAssignmentStatus int

const (
	IMAssignmentStatusActive   IMAssignmentStatus = 0
	IMAssignmentStatusInactive IMAssignmentStatus = 1
)

var imAssignmentStatusLabelMap = map[IMAssignmentStatus]string{
	IMAssignmentStatusActive:   "进行中",
	IMAssignmentStatusInactive: "已结束",
}

func GetIMAssignmentStatusLabel(status IMAssignmentStatus) string {
	return imAssignmentStatusLabelMap[status]
}

type IMMessageType string

const (
	IMMessageTypeText       IMMessageType = "text"
	IMMessageTypeImage      IMMessageType = "image"
	IMMessageTypeAttachment IMMessageType = "attachment"
	IMMessageTypeHTML       IMMessageType = "html"
	IMMessageTypeNote       IMMessageType = "note"
)

var imMessageTypeLabelMap = map[IMMessageType]string{
	IMMessageTypeText:       "文本",
	IMMessageTypeImage:      "图片",
	IMMessageTypeAttachment: "附件",
	IMMessageTypeHTML:       "富文本",
	IMMessageTypeNote:       "便签",
}

func GetIMMessageTypeLabel(messageType IMMessageType) string {
	return imMessageTypeLabelMap[messageType]
}

type IMMessageStatus int

const (
	IMMessageStatusSending   IMMessageStatus = 1
	IMMessageStatusSent      IMMessageStatus = 2
	IMMessageStatusDelivered IMMessageStatus = 3
	IMMessageStatusRead      IMMessageStatus = 4
	IMMessageStatusFailed    IMMessageStatus = 5
	IMMessageStatusRecalled  IMMessageStatus = 6
)

var imMessageStatusLabelMap = map[IMMessageStatus]string{
	IMMessageStatusSending:   "发送中",
	IMMessageStatusSent:      "已发送",
	IMMessageStatusDelivered: "已送达",
	IMMessageStatusRead:      "已读",
	IMMessageStatusFailed:    "发送失败",
	IMMessageStatusRecalled:  "已撤回",
}

func GetIMMessageStatusLabel(status IMMessageStatus) string {
	return imMessageStatusLabelMap[status]
}

type IMEventType string

const (
	IMEventTypeCreate        IMEventType = "create"
	IMEventTypeAssign        IMEventType = "assign"
	IMEventTypeTransfer      IMEventType = "transfer"
	IMEventTypeClose         IMEventType = "close"
	IMEventTypeMessageSend   IMEventType = "message_send"
	IMEventTypeMessageRecall IMEventType = "message_recall"
)

var imEventTypeLabelMap = map[IMEventType]string{
	IMEventTypeCreate:        "创建会话",
	IMEventTypeAssign:        "分配会话",
	IMEventTypeTransfer:      "转接会话",
	IMEventTypeClose:         "关闭会话",
	IMEventTypeMessageSend:   "发送消息",
	IMEventTypeMessageRecall: "撤回消息",
}

func GetIMEventTypeLabel(eventType IMEventType) string {
	return imEventTypeLabelMap[eventType]
}

type AIAgentHandoffMode int

const (
	AIAgentHandoffModeWaitPool        AIAgentHandoffMode = 1
	AIAgentHandoffModeDefaultTeamPool AIAgentHandoffMode = 2
	AIAgentHandoffModeAIHoldAndNotify AIAgentHandoffMode = 3
)

var AIAgentHandoffModeValues = []AIAgentHandoffMode{
	AIAgentHandoffModeWaitPool,
	AIAgentHandoffModeDefaultTeamPool,
	AIAgentHandoffModeAIHoldAndNotify,
}

var aiAgentHandoffModeLabelMap = map[AIAgentHandoffMode]string{
	AIAgentHandoffModeWaitPool:        "进入待接入池",
	AIAgentHandoffModeDefaultTeamPool: "进入默认客服组待接入池",
	AIAgentHandoffModeAIHoldAndNotify: "AI继续接待并提醒人工",
}

func GetAIAgentHandoffModeLabel(mode AIAgentHandoffMode) string {
	return aiAgentHandoffModeLabelMap[mode]
}

type AIAgentFallbackMode int

const (
	AIAgentFallbackModeNoAnswer     AIAgentFallbackMode = 1
	AIAgentFallbackModeSuggestRetry AIAgentFallbackMode = 2
	AIAgentFallbackModeHandoff      AIAgentFallbackMode = 3
)

var AIAgentFallbackModeValues = []AIAgentFallbackMode{
	AIAgentFallbackModeNoAnswer,
	AIAgentFallbackModeSuggestRetry,
	AIAgentFallbackModeHandoff,
}

var aiAgentFallbackModeLabelMap = map[AIAgentFallbackMode]string{
	AIAgentFallbackModeNoAnswer:     "直接说明知识不足",
	AIAgentFallbackModeSuggestRetry: "引导用户补充信息",
	AIAgentFallbackModeHandoff:      "转人工客服",
}

func GetAIAgentFallbackModeLabel(mode AIAgentFallbackMode) string {
	return aiAgentFallbackModeLabelMap[mode]
}

const (
	IMRealtimeEventConnected               = "connected"
	IMRealtimeEventPong                    = "pong"
	IMRealtimeEventSubscribed              = "subscribed"
	IMRealtimeEventUnsubscribed            = "unsubscribed"
	IMRealtimeEventResyncRequired          = "resyncRequired"
	IMRealtimeEventMessageCreated          = "message.created"
	IMRealtimeEventMessageRecalled         = "message.recalled"
	IMRealtimeEventConversationCreated     = "conversation.created"
	IMRealtimeEventConversationUpdated     = "conversation.updated"
	IMRealtimeEventConversationAssigned    = "conversation.assigned"
	IMRealtimeEventConversationTransferred = "conversation.transferred"
	IMRealtimeEventConversationClosed      = "conversation.closed"
	IMRealtimeEventConversationRead        = "conversation.read"
	IMRealtimeEventNotificationCreated     = "notification.created"
	IMRealtimeEventCustomerSessionRefresh  = "customer_session.refresh"
)

const (
	IMRealtimeClientTypePing        = "ping"
	IMRealtimeClientTypeSubscribe   = "subscribe"
	IMRealtimeClientTypeUnsubscribe = "unsubscribe"
	IMRealtimeClientTypeAck         = "ack"
)

const (
	IMRealtimeResyncReasonInvalidPayload         = "invalid_payload"
	IMRealtimeResyncReasonUnsupportedMessageType = "unsupported_message_type"
	IMRealtimeResyncReasonManual                 = "manual"
)
