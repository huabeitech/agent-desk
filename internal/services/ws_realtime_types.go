package services

import (
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/openidentity"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	realtimeWriteWait      = 10 * time.Second
	realtimePongWait       = 60 * time.Second
	realtimePingPeriod     = 30 * time.Second
	realtimeMaxMessageSize = 8 << 10
	realtimeSendBufferSize = 64
)

const (
	realtimeRoleUser         = "user"
	realtimeRoleAdmin        = "admin"
	realtimeRoleNotification = "notification"
)

const (
	realtimeTopicUserPrefix         = "user:"
	realtimeTopicGuestPrefix        = "guest:"
	realtimeTopicAdminPrefix        = "admin:"
	realtimeTopicConversationPrefix = "conversation:"
	realtimeTopicNotificationPrefix = "notification:"
	realtimeTopicAdminAll           = "admin:all"
)

type RealtimeEvent struct {
	EventID string               `json:"eventId"`
	Type    string               `json:"type"`
	Topic   string               `json:"topic,omitempty"`
	Data    RealtimeEventPayload `json:"data,omitempty"`
	At      string               `json:"at"`
}

type RealtimeDomainEvent interface {
	EventType() string
	EventPayload() RealtimeEventPayload
}

type RealtimeEventPayload interface {
	realtimeEventPayload()
}

type RealtimeConnectedPayload struct {
	ConnID       string   `json:"connId,omitempty"`
	UserID       int64    `json:"userId,omitempty"`
	GuestID      string   `json:"guestId,omitempty"`
	Role         string   `json:"role,omitempty"`
	TerminalType string   `json:"terminalType,omitempty"`
	Topics       []string `json:"topics,omitempty"`
}

func (RealtimeConnectedPayload) realtimeEventPayload() {}

type RealtimeConnectedEvent struct {
	Payload RealtimeConnectedPayload
}

func (e RealtimeConnectedEvent) EventType() string {
	return enums.IMRealtimeEventConnected
}

func (e RealtimeConnectedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeTopicsPayload struct {
	Topics []string `json:"topics,omitempty"`
}

func (RealtimeTopicsPayload) realtimeEventPayload() {}

type RealtimeSubscribedEvent struct {
	Payload RealtimeTopicsPayload
}

func (e RealtimeSubscribedEvent) EventType() string {
	return enums.IMRealtimeEventSubscribed
}

func (e RealtimeSubscribedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeUnsubscribedEvent struct {
	Payload RealtimeTopicsPayload
}

func (e RealtimeUnsubscribedEvent) EventType() string {
	return enums.IMRealtimeEventUnsubscribed
}

func (e RealtimeUnsubscribedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimePongEvent struct{}

func (RealtimePongEvent) EventType() string {
	return enums.IMRealtimeEventPong
}

func (RealtimePongEvent) EventPayload() RealtimeEventPayload {
	return nil
}

type RealtimeResyncRequiredPayload struct {
	Reason string `json:"reason,omitempty"`
}

func (RealtimeResyncRequiredPayload) realtimeEventPayload() {}

type RealtimeResyncRequiredEvent struct {
	Payload RealtimeResyncRequiredPayload
}

func (e RealtimeResyncRequiredEvent) EventType() string {
	return enums.IMRealtimeEventResyncRequired
}

func (e RealtimeResyncRequiredEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeMessageCreatedPayload struct {
	ConversationID    int64                      `json:"conversationId,omitempty"`
	MessageID         int64                      `json:"messageId,omitempty"`
	RequestID         string                     `json:"requestId,omitempty"`
	Message           response.MessageResponse   `json:"message,omitempty"`
	Status            enums.IMConversationStatus `json:"status,omitempty"`
	CurrentAssigneeID int64                      `json:"currentAssigneeId,omitempty"`
	SenderType        enums.IMSenderType         `json:"senderType,omitempty"`
	SenderID          int64                      `json:"senderId,omitempty"`
	MessageType       enums.IMMessageType        `json:"messageType,omitempty"`
	Content           string                     `json:"content,omitempty"`
	Payload           string                     `json:"payload,omitempty"`
	SendStatus        enums.IMMessageStatus      `json:"sendStatus,omitempty"`
	SentAt            string                     `json:"sentAt,omitempty"`
}

func (RealtimeMessageCreatedPayload) realtimeEventPayload() {}

type RealtimeMessageCreatedEvent struct {
	Payload RealtimeMessageCreatedPayload
}

func (e RealtimeMessageCreatedEvent) EventType() string {
	return enums.IMRealtimeEventMessageCreated
}

func (e RealtimeMessageCreatedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

// RealtimeMessageUpdatedEvent 通知客户端某条消息内容已被更新（如 AI 占位回复被正式回复替换）。
type RealtimeMessageUpdatedEvent struct {
	Payload RealtimeMessageCreatedPayload
}

func (e RealtimeMessageUpdatedEvent) EventType() string {
	return enums.IMRealtimeEventMessageUpdated
}

func (e RealtimeMessageUpdatedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeMessageRecalledPayload struct {
	ConversationID int64                 `json:"conversationId,omitempty"`
	MessageID      int64                 `json:"messageId,omitempty"`
	SenderType     enums.IMSenderType    `json:"senderType,omitempty"`
	SenderID       int64                 `json:"senderId,omitempty"`
	SendStatus     enums.IMMessageStatus `json:"sendStatus,omitempty"`
	RecalledAt     string                `json:"recalledAt,omitempty"`
}

func (RealtimeMessageRecalledPayload) realtimeEventPayload() {}

type RealtimeMessageRecalledEvent struct {
	Payload RealtimeMessageRecalledPayload
}

func (e RealtimeMessageRecalledEvent) EventType() string {
	return enums.IMRealtimeEventMessageRecalled
}

func (e RealtimeMessageRecalledEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeConversationChangedPayload struct {
	ConversationID            int64                           `json:"conversationId,omitempty"`
	Status                    enums.IMConversationStatus      `json:"status,omitempty"`
	ServiceMode               enums.IMConversationServiceMode `json:"serviceMode,omitempty"`
	CurrentAssigneeID         int64                           `json:"currentAssigneeId,omitempty"`
	CurrentTeamID             int64                           `json:"currentTeamId,omitempty"`
	LastMessageID             int64                           `json:"lastMessageId,omitempty"`
	LastMessageAt             string                          `json:"lastMessageAt,omitempty"`
	LastActiveAt              string                          `json:"lastActiveAt,omitempty"`
	LastMessageSummary        string                          `json:"lastMessageSummary,omitempty"`
	CustomerUnreadCount       int                             `json:"customerUnreadCount,omitempty"`
	AgentUnreadCount          int                             `json:"agentUnreadCount,omitempty"`
	CustomerLastReadMessageID int64                           `json:"customerLastReadMessageId,omitempty"`
	CustomerLastReadAt        string                          `json:"customerLastReadAt,omitempty"`
	AgentLastReadMessageID    int64                           `json:"agentLastReadMessageId,omitempty"`
	AgentLastReadAt           string                          `json:"agentLastReadAt,omitempty"`
}

func (RealtimeConversationChangedPayload) realtimeEventPayload() {}

type RealtimeConversationChangedEvent struct {
	Type    string
	Payload RealtimeConversationChangedPayload
}

func (e RealtimeConversationChangedEvent) EventType() string {
	return e.Type
}

func (e RealtimeConversationChangedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeNotificationCreatedPayload struct {
	Notification response.NotificationResponse `json:"notification"`
}

func (RealtimeNotificationCreatedPayload) realtimeEventPayload() {}

type RealtimeNotificationCreatedEvent struct {
	Payload RealtimeNotificationCreatedPayload
}

func (e RealtimeNotificationCreatedEvent) EventType() string {
	return enums.IMRealtimeEventNotificationCreated
}

func (e RealtimeNotificationCreatedEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type RealtimeCustomerSessionRefreshPayload struct {
	CustomerSessionToken string `json:"customerSessionToken"`
	ExpiresAt            string `json:"expiresAt"`
}

func (RealtimeCustomerSessionRefreshPayload) realtimeEventPayload() {}

type RealtimeCustomerSessionRefreshEvent struct {
	Payload RealtimeCustomerSessionRefreshPayload
}

func (e RealtimeCustomerSessionRefreshEvent) EventType() string {
	return enums.IMRealtimeEventCustomerSessionRefresh
}

func (e RealtimeCustomerSessionRefreshEvent) EventPayload() RealtimeEventPayload {
	return e.Payload
}

type realtimeClientMessage struct {
	Type    string   `json:"type"`
	Topics  []string `json:"topics,omitempty"`
	EventID string   `json:"eventId,omitempty"`
}

type ClientSession struct {
	ID           string
	Conn         *websocket.Conn
	Principal    *dto.AuthPrincipal
	External     *openidentity.ExternalUser
	Role         string
	TerminalType string
	Topics       map[string]struct{}
	Send         chan []byte
	Closed       atomic.Bool
	LastActiveAt atomic.Int64
	closeOnce    sync.Once
}

func (s *ClientSession) enqueue(payload []byte) bool {
	if s == nil || s.Closed.Load() {
		return false
	}
	select {
	case s.Send <- payload:
		return true
	default:
		return false
	}
}

func (s *ClientSession) enqueueEvent(event RealtimeEvent) bool {
	payload, err := json.Marshal(event)
	if err != nil {
		return false
	}
	return s.enqueue(payload)
}

func (s *ClientSession) touch() {
	if s == nil {
		return
	}
	s.LastActiveAt.Store(time.Now().Unix())
}

func (s *ClientSession) topicList() []string {
	if s == nil {
		return nil
	}
	ret := make([]string, 0, len(s.Topics))
	for topic := range s.Topics {
		ret = append(ret, topic)
	}
	return ret
}
