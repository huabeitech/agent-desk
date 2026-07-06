package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/pkg/tracex"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"log/slog"
	"slices"
	"strings"
	"time"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
)

var MessageService = newMessageService()

func newMessageService() *messageService {
	return &messageService{}
}

type messageService struct {
}

func (s *messageService) Get(id int64) *models.Message {
	return repositories.MessageRepository.Get(sqls.DB(), id)
}

func (s *messageService) Take(where ...interface{}) *models.Message {
	return repositories.MessageRepository.Take(sqls.DB(), where...)
}

func (s *messageService) Find(cnd *sqls.Cnd) []models.Message {
	return repositories.MessageRepository.Find(sqls.DB(), cnd)
}

// FindByConversationIDCursor 按 id 游标分页：cursor=0 取最新 limit 条；cursor>0 取 id<cursor 的更旧消息。
// 返回的 list 已按 id 升序（时间正序）。nextCursor 为下一页请求传入的游标（本批最小 id）；hasMore 表示可能还有更旧消息。
func (s *messageService) FindByConversationIDCursor(conversationID int64, cursor int64, limit int, senderType, messageType string) (list []models.Message, nextCursor int64, hasMore bool) {
	if limit > 100 {
		limit = 100
	} else if limit <= 0 {
		limit = 20
	}
	cnd := sqls.NewCnd().Eq("conversation_id", conversationID).Limit(limit).Desc("id")
	if cursor > 0 {
		cnd.Lt("id", cursor)
	}
	if strs.IsNotBlank(senderType) {
		cnd.Eq("sender_type", senderType)
	}
	if strs.IsNotBlank(messageType) {
		cnd.Eq("message_type", messageType)
	}
	list = s.Find(cnd)
	nextCursor = cursor
	hasMore = false
	if len(list) > 0 {
		nextCursor = list[len(list)-1].ID
		hasMore = len(list) == limit
	}
	slices.Reverse(list)
	return list, nextCursor, hasMore
}

func (s *messageService) FindOne(cnd *sqls.Cnd) *models.Message {
	return repositories.MessageRepository.FindOne(sqls.DB(), cnd)
}

func (s *messageService) FindPageByParams(params *params.QueryParams) (list []models.Message, paging *sqls.Paging) {
	return repositories.MessageRepository.FindPageByParams(sqls.DB(), params)
}

func (s *messageService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Message, paging *sqls.Paging) {
	return repositories.MessageRepository.FindPageByCnd(sqls.DB(), cnd)
}

// FindPageByCndForImListAscending 与 FindPageByCnd 相同分页条件，将结果按 seq 升序排列（开放 IM 时间正序展示）。
func (s *messageService) FindPageByCndForImListAscending(cnd *sqls.Cnd) (list []models.Message, paging *sqls.Paging) {
	list, paging = s.FindPageByCnd(cnd)
	if len(list) <= 1 {
		return list, paging
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, paging
}

func (s *messageService) Count(cnd *sqls.Cnd) int64 {
	return repositories.MessageRepository.Count(sqls.DB(), cnd)
}

func (s *messageService) Create(t *models.Message) error {
	return repositories.MessageRepository.Create(sqls.DB(), t)
}

func (s *messageService) Update(t *models.Message) error {
	return repositories.MessageRepository.Update(sqls.DB(), t)
}

func (s *messageService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.MessageRepository.Updates(sqls.DB(), id, columns)
}

func (s *messageService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.MessageRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *messageService) Delete(id int64) {
	repositories.MessageRepository.Delete(sqls.DB(), id)
}

func (s *messageService) GetConversationReadTarget(conversationID, messageID int64) (*models.Message, error) {
	if messageID > 0 {
		message := s.Get(messageID)
		if message == nil || message.ConversationID != conversationID {
			return nil, errorsx.InvalidParamI18n("error.e0244")
		}
		return message, nil
	}
	return s.FindOne(sqls.NewCnd().Eq("conversation_id", conversationID).Desc("id")), nil
}

func (s *messageService) SendMessage(conversationID int64, senderType enums.IMSenderType, reqSenderID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal, external *openidentity.ExternalUser) (*models.Message, error) {
	switch senderType {
	case enums.IMSenderTypeAgent:
		return s.sendMessage(conversationID, enums.IMSenderTypeAgent, reqSenderID, clientMsgID, messageType, content, payload, operator, nil, "", 0)
	case enums.IMSenderTypeAI:
		return s.sendMessage(conversationID, enums.IMSenderTypeAI, reqSenderID, clientMsgID, messageType, content, payload, operator, nil, "", 0)
	case enums.IMSenderTypeCustomer:
		return s.sendMessage(conversationID, enums.IMSenderTypeCustomer, 0, clientMsgID, messageType, content, payload, nil, external, "", 0)
	default:
		return nil, errorsx.InvalidParamI18n("error.e0080")
	}
}

func (s *messageService) SendAgentMessage(conversationID int64, reqSenderID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal) (*models.Message, error) {
	return s.SendAgentMessageWithRequestID(conversationID, reqSenderID, clientMsgID, messageType, content, payload, operator, "")
}

func (s *messageService) SendAgentMessageWithRequestID(conversationID int64, reqSenderID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal, requestID string) (*models.Message, error) {
	return s.sendMessage(conversationID, enums.IMSenderTypeAgent, reqSenderID, clientMsgID, messageType, content, payload, operator, nil, requestID, 0)
}

func (s *messageService) RecallAgentMessage(messageID int64, operator *dto.AuthPrincipal) (*models.Message, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if messageID <= 0 {
		return nil, errorsx.InvalidParamI18n("error.e0244")
	}

	message := s.Get(messageID)
	if message == nil {
		return nil, errorsx.InvalidParamI18n("error.e0244")
	}
	if message.SenderType != enums.IMSenderTypeAgent {
		return nil, errorsx.InvalidParamI18n("error.e0091")
	}
	if message.SenderID != operator.UserID {
		return nil, errorsx.ForbiddenI18n("error.e0087")
	}
	if message.RecalledAt != nil || message.SendStatus == enums.IMMessageStatusRecalled {
		return nil, errorsx.InvalidParamI18n("error.e0246")
	}

	conversation, err := s.ValidateConversationSender(message.ConversationID, enums.IMSenderTypeAgent, operator, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	err = sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		updates := map[string]any{
			"send_status":      int(enums.IMMessageStatusRecalled),
			"recalled_at":      now,
			"updated_at":       now,
			"update_user_id":   operator.UserID,
			"update_user_name": operator.Username,
		}
		if err := repositories.MessageRepository.Updates(ctx.Tx, message.ID, updates); err != nil {
			return err
		}

		message.SendStatus = enums.IMMessageStatusRecalled
		message.RecalledAt = &now
		message.UpdatedAt = now
		message.UpdateUserID = operator.UserID
		message.UpdateUserName = operator.Username

		agentReadState, customerReadState := ConversationReadStateService.getConversationReadStates(ctx.Tx, conversation.ID)
		agentUnreadCount, err := ConversationReadStateService.CountUnreadMessages(ctx, conversation.ID, s.readMessageID(agentReadState), enums.IMSenderTypeCustomer)
		if err != nil {
			return err
		}
		customerUnreadCount, err := ConversationReadStateService.CountUnreadMessages(ctx, conversation.ID, s.readMessageID(customerReadState), enums.IMSenderTypeAgent, enums.IMSenderTypeAI)
		if err != nil {
			return err
		}

		conversationUpdates := map[string]any{
			"agent_unread_count":    agentUnreadCount,
			"customer_unread_count": customerUnreadCount,
			"updated_at":            now,
			"update_user_id":        operator.UserID,
			"update_user_name":      operator.Username,
		}
		if conversation.LastMessageID == message.ID {
			lastMessage := repositories.MessageRepository.FindLastUnrecalledByConversationID(ctx.Tx, conversation.ID)
			if lastMessage != nil {
				conversationUpdates["last_message_id"] = lastMessage.ID
				conversationUpdates["last_message_at"] = lastMessage.SentAt
				conversationUpdates["last_message_summary"] = limitText(buildMessageSummary(lastMessage.MessageType, lastMessage.Content), 255)
			} else {
				conversationUpdates["last_message_id"] = 0
				conversationUpdates["last_message_at"] = nil
				conversationUpdates["last_message_summary"] = ""
			}
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversation.ID, conversationUpdates); err != nil {
			return err
		}

		if err := ConversationEventLogService.CreateEvent(ctx, conversation.ID, enums.IMEventTypeMessageRecall, enums.IMSenderTypeAgent, operator.UserID, "客服撤回消息", ""); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if updatedConversation := ConversationService.Get(conversation.ID); updatedConversation != nil {
		WsService.PublishMessageRecalled(updatedConversation, message)
		WsService.PublishConversationChanged(updatedConversation, enums.IMRealtimeEventConversationUpdated)
	}
	return message, nil
}

func (s *messageService) SendAIMessage(conversationID int64, aiAgentID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal) (*models.Message, error) {
	return s.SendAIMessageWithRequestID(conversationID, aiAgentID, clientMsgID, messageType, content, payload, operator, "")
}

func (s *messageService) SendAIMessageWithRequestID(conversationID int64, aiAgentID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal, requestID string) (*models.Message, error) {
	return s.SendAIMessageWithRequestIDAndWorkflowRunID(conversationID, aiAgentID, clientMsgID, messageType, content, payload, operator, requestID, 0)
}

func (s *messageService) SendAIMessageWithRequestIDAndWorkflowRunID(conversationID int64, aiAgentID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal, requestID string, workflowRunID int64) (*models.Message, error) {
	return s.sendMessage(conversationID, enums.IMSenderTypeAI, aiAgentID, clientMsgID, messageType, content, payload, operator, nil, requestID, workflowRunID)
}

func (s *messageService) SendAIServiceNotice(conversationID int64, aiAgentID int64, content string) (*models.Message, error) {
	return s.SendAIServiceNoticeWithRequestID(conversationID, aiAgentID, content, "")
}

func (s *messageService) SendAIServiceNoticeWithRequestID(conversationID int64, aiAgentID int64, content string, requestID string) (*models.Message, error) {
	conversation := ConversationService.Get(conversationID)
	if conversation == nil {
		return nil, errorsx.InvalidParamI18n("error.e0116")
	}
	if conversation.Status == enums.IMConversationStatusClosed {
		return nil, errorsx.InvalidParamI18n("error.e0119")
	}
	return s.sendValidatedMessage(conversation, enums.IMSenderTypeAI, aiAgentID, strs.UUID(), enums.IMMessageTypeText, content, "", &dto.AuthPrincipal{
		UserID:   0,
		Username: "system",
		Nickname: "system",
	}, nil, requestID, 0)
}

func (s *messageService) createAIWelcomeMessage(ctx *sqls.TxContext, conversation *models.Conversation, aiAgent *models.AIAgent, now time.Time) (*models.Message, error) {
	if ctx == nil || conversation == nil || aiAgent == nil || strings.TrimSpace(aiAgent.WelcomeMessage) == "" {
		return nil, nil
	}

	content, payload, summary, err := s.normalizeMessageContent(conversation.ID, enums.IMMessageTypeText, aiAgent.WelcomeMessage, "")
	if err != nil {
		return nil, err
	}
	if strs.IsBlank(content) && strs.IsBlank(payload) {
		return nil, nil
	}

	operator := &dto.AuthPrincipal{
		UserID:   0,
		Username: "system",
		Nickname: "system",
	}
	message := &models.Message{
		ConversationID: conversation.ID,
		ClientMsgID:    strs.UUID(),
		SenderType:     enums.IMSenderTypeAI,
		SenderID:       aiAgent.ID,
		MessageType:    enums.IMMessageTypeText,
		Content:        content,
		Payload:        payload,
		SendStatus:     enums.IMMessageStatusSent,
		SentAt:         &now,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   operator.UserID,
			CreateUserName: operator.Username,
			UpdatedAt:      now,
			UpdateUserID:   operator.UserID,
			UpdateUserName: operator.Username,
		},
	}
	if err := repositories.MessageRepository.Create(ctx.Tx, message); err != nil {
		return nil, err
	}

	if _, err := ConversationReadStateService.MarkAgentRead(ctx, conversation, operator, message); err != nil {
		return nil, err
	}
	agentReadState, customerReadState := ConversationReadStateService.getConversationReadStates(ctx.Tx, conversation.ID)
	agentUnreadCount, err := ConversationReadStateService.CountUnreadMessages(ctx, conversation.ID, s.readMessageID(agentReadState), enums.IMSenderTypeCustomer)
	if err != nil {
		return nil, err
	}
	customerUnreadCount, err := ConversationReadStateService.CountUnreadMessages(ctx, conversation.ID, s.readMessageID(customerReadState), enums.IMSenderTypeAgent, enums.IMSenderTypeAI)
	if err != nil {
		return nil, err
	}

	conversationUpdates := map[string]any{
		"last_message_id":       message.ID,
		"last_message_at":       now,
		"last_active_at":        now,
		"last_message_summary":  limitText(summary, 255),
		"update_user_id":        operator.UserID,
		"update_user_name":      operator.Username,
		"updated_at":            now,
		"agent_unread_count":    agentUnreadCount,
		"customer_unread_count": customerUnreadCount,
	}
	if err := repositories.ConversationRepository.Updates(ctx.Tx, conversation.ID, conversationUpdates); err != nil {
		return nil, err
	}
	if err := ConversationEventLogService.CreateEvent(ctx,
		conversation.ID,
		enums.IMEventTypeMessageSend,
		enums.IMSenderTypeAI,
		0,
		enums.GetIMSenderTypeLabel(enums.IMSenderTypeAI)+"发送消息",
		"",
	); err != nil {
		return nil, err
	}

	conversation.LastMessageID = message.ID
	conversation.LastMessageAt = now
	conversation.LastActiveAt = now
	conversation.LastMessageSummary = limitText(summary, 255)
	conversation.AgentUnreadCount = int(agentUnreadCount)
	conversation.CustomerUnreadCount = int(customerUnreadCount)
	conversation.UpdatedAt = now
	conversation.UpdateUserID = operator.UserID
	conversation.UpdateUserName = operator.Username
	return message, nil
}

func (s *messageService) SendCustomerMessage(conversationID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, external openidentity.ExternalUser) (*models.Message, error) {
	return s.SendCustomerMessageWithRequestID(conversationID, clientMsgID, messageType, content, payload, external, "")
}

func (s *messageService) SendCustomerMessageWithRequestID(conversationID int64, clientMsgID string, messageType enums.IMMessageType, content, payload string, external openidentity.ExternalUser, requestID string) (*models.Message, error) {
	ext := external
	return s.sendMessage(conversationID, enums.IMSenderTypeCustomer, 0, clientMsgID, messageType, content, payload, nil, &ext, requestID, 0)
}

func (s *messageService) sendMessage(conversationID int64, senderType enums.IMSenderType, reqSenderID int64, clientMsgID string,
	messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal, external *openidentity.ExternalUser, requestID string, workflowRunID int64) (*models.Message, error) {

	if senderType == enums.IMSenderTypeCustomer {
		if external == nil || strings.TrimSpace(external.ExternalID) == "" {
			return nil, errorsx.UnauthorizedI18n("error.e0149")
		}
	} else if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}

	if strs.IsBlank(string(messageType)) {
		messageType = enums.IMMessageTypeText
	}
	conversation, err := s.ValidateConversationSender(conversationID, senderType, operator, external)
	if err != nil {
		return nil, err
	}
	return s.sendValidatedMessage(conversation, senderType, reqSenderID, clientMsgID, messageType, content, payload, operator, external, requestID, workflowRunID)
}

func (s *messageService) sendValidatedMessage(conversation *models.Conversation, senderType enums.IMSenderType, reqSenderID int64, clientMsgID string,
	messageType enums.IMMessageType, content, payload string, operator *dto.AuthPrincipal, external *openidentity.ExternalUser, requestID string, workflowRunID int64) (*models.Message, error) {

	var err error
	var summary string
	content, payload, summary, err = s.normalizeMessageContent(conversation.ID, messageType, content, payload)
	if err != nil {
		return nil, err
	}
	if strs.IsBlank(content) && strs.IsBlank(payload) {
		return nil, errorsx.InvalidParamI18n("error.e0245")
	}

	// 防抖，消息存在就不再发送了
	if strs.IsNotBlank(clientMsgID) {
		if existing := repositories.MessageRepository.GetByClientMsgID(sqls.DB(), conversation.ID, clientMsgID); existing != nil {
			return existing, nil
		}
	}

	var (
		now           = time.Now()
		traceID       = tracex.NormalizeRequestID(requestID)
		auditUserID   = int64(0)
		auditUserName = ""
	)
	if operator != nil {
		auditUserID = operator.UserID
		auditUserName = operator.Username
	}
	if senderType == enums.IMSenderTypeCustomer && external != nil {
		auditUserID = 0
		auditUserName = displayExternalName(external)
	}
	message := &models.Message{
		ConversationID: conversation.ID,
		RequestID:      traceID,
		WorkflowRunID:  workflowRunID,
		ClientMsgID:    clientMsgID,
		SenderType:     senderType,
		SenderID:       reqSenderID,
		MessageType:    messageType,
		Content:        content,
		Payload:        payload,
		SendStatus:     enums.IMMessageStatusSent,
		SentAt:         &now,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   auditUserID,
			CreateUserName: auditUserName,
			UpdatedAt:      now,
			UpdateUserID:   auditUserID,
			UpdateUserName: auditUserName,
		},
	}

	switch senderType {
	case enums.IMSenderTypeAgent:
		if message.SenderID == 0 {
			message.SenderID = operator.UserID
		}
	case enums.IMSenderTypeAI:
		if message.SenderID == 0 {
			message.SenderID = reqSenderID
		}
	default:
		message.SenderID = 0
	}

	err = sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.MessageRepository.Create(ctx.Tx, message); err != nil {
			return err
		}

		// 处理已读、维度
		agentUnreadCount, customerUnreadCount, err := s.handleReadState(ctx, senderType, conversation, operator, message, external)
		if err != nil {
			return err
		}

		conversation.LastMessageID = message.ID
		conversation.LastMessageAt = now
		conversation.LastActiveAt = now
		conversation.LastMessageSummary = limitText(summary, 255)
		conversation.UpdateUserID = int64(0)
		conversation.UpdateUserName = ""
		if operator != nil {
			conversation.UpdateUserID = operator.UserID
			conversation.UpdateUserName = operator.Username
		}
		if senderType == enums.IMSenderTypeCustomer && external != nil {
			conversation.UpdateUserID = 0
			conversation.UpdateUserName = displayExternalName(external)
		}
		conversation.UpdatedAt = now
		conversation.AgentUnreadCount = int(agentUnreadCount)
		conversation.CustomerUnreadCount = int(customerUnreadCount)
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversation.ID, map[string]any{
			"last_message_id":       conversation.LastMessageID,
			"last_message_at":       conversation.LastMessageAt,
			"last_active_at":        conversation.LastActiveAt,
			"last_message_summary":  conversation.LastMessageSummary,
			"update_user_id":        conversation.UpdateUserID,
			"update_user_name":      conversation.UpdateUserName,
			"updated_at":            conversation.UpdatedAt,
			"agent_unread_count":    conversation.AgentUnreadCount,
			"customer_unread_count": conversation.CustomerUnreadCount,
		}); err != nil {
			return err
		}

		// 记录事件日志
		if err := ConversationEventLogService.CreateEventWithRequestID(ctx, conversation.ID, traceID, enums.IMEventTypeMessageSend, senderType,
			func() int64 {
				if operator != nil {
					return operator.UserID
				}
				return 0
			}(),
			enums.GetIMSenderTypeLabel(senderType)+"发送消息",
			"",
		); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 处理websocket消息
	WsService.PublishMessageCreated(conversation, message)
	WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationUpdated)

	// 企业微信客服消息入队，异步发送
	if enqueueErr := ChannelMessageOutboxService.EnqueueWxWorkKFMessage(conversation, message); enqueueErr != nil {
		slog.Error("enqueue wxwork kf outbox failed",
			"conversation_id", conversation.ID,
			"message_id", message.ID,
			"error", enqueueErr,
		)
	}

	// 客户发送消息，触发AI回复
	if senderType == enums.IMSenderTypeCustomer {
		SalesLeadService.ExtractFromCustomerMessageAsync(*conversation, *message)
		if TriggerAIReplyAsyncHook != nil {
			TriggerAIReplyAsyncHook(*conversation, *message)
		}
	}
	return message, err
}

// handleReadState 根据发送者类型更新会话已读状态，并返回更新后的客服和客户未读消息数。
func (s *messageService) handleReadState(ctx *sqls.TxContext, senderType enums.IMSenderType, conversation *models.Conversation, operator *dto.AuthPrincipal, message *models.Message, external *openidentity.ExternalUser) (agentUnreadCount int64, customerUnreadCount int64, err error) {
	readStateType := senderType
	if senderType == enums.IMSenderTypeAI {
		readStateType = enums.IMSenderTypeAgent
	}
	if readStateType == enums.IMSenderTypeAgent {
		if _, err := ConversationReadStateService.MarkAgentRead(ctx, conversation, operator, message); err != nil {
			return 0, 0, err
		}
	} else {
		if _, err := ConversationReadStateService.MarkCustomerRead(ctx, conversation, external, message); err != nil {
			return 0, 0, err
		}
	}
	agentReadState, customerReadState := ConversationReadStateService.getConversationReadStates(ctx.Tx, conversation.ID)
	if agentUnreadCount, err = ConversationReadStateService.CountUnreadMessages(ctx, conversation.ID, s.readMessageID(agentReadState), enums.IMSenderTypeCustomer); err != nil {
		return 0, 0, err
	}
	if customerUnreadCount, err = ConversationReadStateService.CountUnreadMessages(ctx, conversation.ID, s.readMessageID(customerReadState), enums.IMSenderTypeAgent, enums.IMSenderTypeAI); err != nil {
		return 0, 0, err
	}
	return agentUnreadCount, customerUnreadCount, nil
}

func limitText(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}

func buildMessageSummary(messageType enums.IMMessageType, content string) string {
	content = strings.TrimSpace(content)
	if content != "" {
		return content
	}
	switch messageType {
	case enums.IMMessageTypeImage:
		return "[图片]"
	case enums.IMMessageTypeAttachment:
		return "[附件]"
	case enums.IMMessageTypeHTML:
		return utils.BuildHTMLSummary(content)
	case "":
		return ""
	default:
		return "[" + string(messageType) + "]"
	}
}

func (s *messageService) normalizeMessageContent(conversationID int64, messageType enums.IMMessageType, content, payload string) (string, string, string, error) {
	switch messageType {
	case enums.IMMessageTypeHTML:
		sanitized := utils.SanitizeMessageHTML(content)
		normalized, err := utils.NormalizeMessageHTMLAssets(sanitized)
		if err != nil {
			return "", "", "", errorsx.InvalidParamI18n("error.e0030")
		}
		summary := utils.BuildHTMLSummary(normalized)
		if summary == "" {
			return "", "", "", errorsx.InvalidParamI18n("error.e0245")
		}
		return normalized, "", summary, nil
	case enums.IMMessageTypeImage, enums.IMMessageTypeAttachment:
		assetPayload, err := parseIMMessageAssetPayload(payload)
		if err != nil {
			return "", "", "", err
		}
		asset := AssetService.GetByAssetID(assetPayload.AssetID)
		if err := validateConversationAsset(asset, conversationID, messageType); err != nil {
			return "", "", "", err
		}
		canonicalPayload, err := buildIMMessageAssetPayload(asset)
		if err != nil {
			return "", "", "", err
		}
		summary := "[附件]"
		if messageType == enums.IMMessageTypeImage {
			summary = "[图片]"
		}
		content = strings.TrimSpace(asset.Filename)
		return content, canonicalPayload, summary + s.suffixFilenameForSummary(asset.Filename), nil
	default:
		content = strings.TrimSpace(content)
		if content == "" && strings.TrimSpace(payload) == "" {
			return "", "", "", errorsx.InvalidParamI18n("error.e0245")
		}
		return content, strings.TrimSpace(payload), buildMessageSummary(messageType, content), nil
	}
}

func (s *messageService) ValidateConversationSender(conversationID int64, senderType enums.IMSenderType, operator *dto.AuthPrincipal, external *openidentity.ExternalUser) (*models.Conversation, error) {
	conversation := ConversationService.Get(conversationID)
	if conversation == nil {
		return nil, errorsx.InvalidParamI18n("error.e0116")
	}
	if conversation.Status == enums.IMConversationStatusClosed {
		return nil, errorsx.InvalidParamI18n("error.e0119")
	}
	switch senderType {
	case enums.IMSenderTypeAgent:
		if operator == nil {
			return nil, errorsx.UnauthorizedI18n("error.auth.expired")
		}
		if conversation.Status != enums.IMConversationStatusActive || conversation.CurrentAssigneeID == 0 {
			return nil, errorsx.InvalidParamI18n("error.e0120")
		}
		if conversation.CurrentAssigneeID != operator.UserID {
			return nil, errorsx.ForbiddenI18n("error.e0191")
		}
	case enums.IMSenderTypeAI:
		if operator == nil {
			return nil, errorsx.UnauthorizedI18n("error.auth.expired")
		}
		if conversation.Status != enums.IMConversationStatusAIServing && !s.allowAIMessageOnPendingHandoff(conversation) {
			return nil, errorsx.ForbiddenI18n("error.e0189")
		}
		if conversation.CurrentAssigneeID != 0 {
			return nil, errorsx.ForbiddenI18n("error.e0192")
		}
	case enums.IMSenderTypeCustomer:
		if external == nil || !ConversationService.IsCustomerConversationOwner(conversation, *external) {
			return nil, errorsx.ForbiddenI18n("error.e0222")
		}
	default:
		return nil, errorsx.InvalidParamI18n("error.e0080")
	}
	return conversation, nil
}

func (s *messageService) allowAIMessageOnPendingHandoff(conversation *models.Conversation) bool {
	if conversation == nil {
		return false
	}
	return conversation.Status == enums.IMConversationStatusPending &&
		conversation.HandoffAt != nil &&
		conversation.CurrentAssigneeID == 0
}

func (s *messageService) suffixFilenameForSummary(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	return " " + filename
}

func (s *messageService) readMessageID(state *models.ConversationReadState) int64 {
	if state == nil {
		return 0
	}
	return state.LastReadMessageID
}
