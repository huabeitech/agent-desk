package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"agent-desk/internal/events"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/eventbus"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"slices"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var ConversationService = newConversationService()

func newConversationService() *conversationService {
	return &conversationService{}
}

type conversationService struct {
}

func (s *conversationService) Get(id int64) *models.Conversation {
	if id <= 0 {
		return nil
	}
	return repositories.ConversationRepository.Get(sqls.DB(), id)
}

func (s *conversationService) Find(cnd *sqls.Cnd) []models.Conversation {
	return repositories.ConversationRepository.Find(sqls.DB(), cnd)
}

func (s *conversationService) FindOne(cnd *sqls.Cnd) *models.Conversation {
	return repositories.ConversationRepository.FindOne(sqls.DB(), cnd)
}

func (s *conversationService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Conversation, paging *sqls.Paging) {
	return repositories.ConversationRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *conversationService) ListConversations(userID int64, filter request.AgentConversationFilter, keyword string, paging *sqls.Paging) ([]models.Conversation, *sqls.Paging, error) {
	cnd := sqls.NewCnd().Page(paging.Page, paging.Limit)

	if strs.IsNotBlank(keyword) {
		keyword = strings.TrimSpace(keyword)
		keywordLike := "%" + keyword + "%"
		cnd.Where("customer_name LIKE ? OR last_message_summary LIKE ?", keywordLike, keywordLike)
	}

	switch filter {
	case request.AgentConversationFilterAIServing:
		cnd.Eq("current_assignee_id", 0).Eq("status", enums.IMConversationStatusAIServing).Desc("last_active_at").Desc("id")
	case request.AgentConversationFilterMine:
		cnd.Eq("current_assignee_id", userID).Desc("last_active_at").Desc("id")
	case request.AgentConversationFilterActive:
		cnd.Eq("current_assignee_id", userID).Eq("status", enums.IMConversationStatusActive).Desc("last_active_at").Desc("id")
	case request.AgentConversationFilterPending:
		cnd.Eq("current_assignee_id", 0).Eq("status", enums.IMConversationStatusPending).Asc("last_active_at").Desc("id")
	case request.AgentConversationFilterClosed:
		cnd.Eq("current_assignee_id", userID).Eq("status", enums.IMConversationStatusClosed).Desc("last_active_at").Desc("id")
	default:
		return nil, nil, errorsx.InvalidParamI18n("error.e0121")
	}

	list, paging := repositories.ConversationRepository.FindPageByCnd(sqls.DB(), cnd)
	return list, paging, nil
}

func (s *conversationService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.ConversationRepository.Updates(sqls.DB(), id, columns)
}

func (s *conversationService) getLatestNotFinishedByCustomerID(db *gorm.DB, customerID int64) *models.Conversation {
	if customerID <= 0 {
		return nil
	}
	cnd := sqls.NewCnd()
	cnd.Eq("customer_id", customerID)
	cnd.In("status", []enums.IMConversationStatus{
		enums.IMConversationStatusAIServing,
		enums.IMConversationStatusPending,
		enums.IMConversationStatusActive,
	})
	cnd.Desc("id")
	return repositories.ConversationRepository.FindOne(db, cnd)
}

func (s *conversationService) Create(externalUser openidentity.ExternalUser, channelID, aiAgentID int64) (*models.Conversation, error) {
	aiAgent := AIAgentService.Get(aiAgentID)
	if aiAgent == nil || aiAgent.Status != enums.StatusOk {
		return nil, errorsx.InvalidParamI18n("error.e0002")
	}

	var conversation *models.Conversation
	var welcomeMessage *models.Message
	created := false
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		customerID, err := CustomerService.EnsureExternalCustomer(ctx, externalUser)
		if err != nil {
			return err
		}
		customerName := s.getCustomerName(ctx.Tx, customerID)
		if existing := s.getLatestNotFinishedByCustomerID(ctx.Tx, customerID); existing != nil {
			conversation = existing
			if customerName != "" && existing.CustomerName != customerName {
				if err := repositories.ConversationRepository.Updates(ctx.Tx, existing.ID, map[string]any{
					"customer_name": customerName,
					"updated_at":    time.Now(),
				}); err != nil {
					return err
				}
				conversation.CustomerName = customerName
			}
			return nil
		}
		created = true
		now := time.Now()
		conversation = &models.Conversation{
			AIAgentID:         aiAgentID,
			ChannelID:         channelID,
			CustomerID:        customerID,
			CustomerName:      customerName,
			Status:            s.resolveInitialStatus(aiAgent.ServiceMode),
			ServiceMode:       aiAgent.ServiceMode,
			Priority:          0,
			CurrentAssigneeID: 0,
			CurrentTeamID:     0,
			LastMessageAt:     now,
			LastActiveAt:      now,
			AuditFields:       utils.BuildAuditFields(nil),
		}
		if err := ctx.Tx.Create(conversation).Error; err != nil {
			return err
		}
		if err := ConversationParticipantService.CreateCustomerParticipant(ctx, conversation.ID, externalUser); err != nil {
			return err
		}
		if err := ConversationEventLogService.CreateEvent(ctx, conversation.ID, enums.IMEventTypeCreate, enums.IMSenderTypeCustomer, 0, "用户创建会话", ""); err != nil {
			return err
		}
		welcomeMessage, err = MessageService.createAIWelcomeMessage(ctx, conversation, aiAgent, now)
		return err
	}); err != nil {
		return nil, err
	}
	if conversation == nil {
		return nil, errorsx.BusinessErrorI18n(1, "error.conversation.createFailed")
	}
	if !created {
		return conversation, nil
	}

	// 推送会话创建事件
	WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationCreated)
	if welcomeMessage != nil {
		if updatedConversation := s.Get(conversation.ID); updatedConversation != nil {
			WsService.PublishMessageCreated(updatedConversation, welcomeMessage)
			WsService.PublishConversationChanged(updatedConversation, enums.IMRealtimeEventConversationUpdated)
		}
	}

	if aiAgent.ServiceMode == enums.IMConversationServiceModeHumanOnly {
		if _, err := ConversationHumanDispatchService.ApplyHumanOnlyCreate(conversation.ID, *aiAgent); err != nil {
			return nil, err
		}
	}
	return s.Get(conversation.ID), nil
}

func (s *conversationService) AssignConversation(req request.AssignConversationRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	targetProfile := AgentProfileService.GetByUserID(req.AssigneeID)
	if targetProfile == nil || targetProfile.Status != enums.StatusOk {
		return errorsx.InvalidParamI18n("error.e0276")
	}
	var assignedEvent events.ConversationAssignedEvent
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		conversation := repositories.ConversationRepository.Get(ctx.Tx, req.ConversationID)
		if conversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if conversation.Status != enums.IMConversationStatusPending {
			return errorsx.InvalidParamI18n("error.e0135")
		}
		now := time.Now()
		if err := ConversationAssignmentService.FinishActiveAssignments(ctx, req.ConversationID, now); err != nil {
			return err
		}
		if err := ConversationAssignmentService.CreateAssignment(ctx, req.ConversationID, conversation.CurrentAssigneeID, req.AssigneeID, enums.IMAssignmentTypeAssign, req.Reason, operator, now); err != nil {
			return err
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, req.ConversationID, map[string]any{
			"current_assignee_id": req.AssigneeID,
			"status":              enums.IMConversationStatusActive,
			"update_user_id":      operator.UserID,
			"update_user_name":    operator.Username,
			"updated_at":          now,
		}); err != nil {
			return err
		}
		if err := ConversationEventLogService.CreateEvent(ctx, req.ConversationID, enums.IMEventTypeAssign, enums.IMSenderTypeAgent, operator.UserID, "会话已分配", s.buildEventPayload(map[string]any{
			"fromStatus":     conversation.Status,
			"toStatus":       enums.IMConversationStatusActive,
			"fromAssigneeId": conversation.CurrentAssigneeID,
			"toAssigneeId":   req.AssigneeID,
			"reason":         strings.TrimSpace(req.Reason),
		})); err != nil {
			return err
		}
		assignedEvent = events.ConversationAssignedEvent{
			ConversationID: req.ConversationID,
			FromUserID:     conversation.CurrentAssigneeID,
			ToUserID:       req.AssigneeID,
			OperatorID:     operator.UserID,
			Reason:         strings.TrimSpace(req.Reason),
			AssignType:     events.ConversationAssignTypeAssign,
		}
		return nil
	}); err != nil {
		return err
	}
	if conversation := s.Get(req.ConversationID); conversation != nil {
		WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationAssigned)
	}
	eventbus.PublishAsync(context.Background(), assignedEvent)
	return nil
}

func (s *conversationService) AutoAssignConversation(conversationID int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}

	conversation := s.Get(conversationID)
	if conversation == nil {
		return errorsx.InvalidParamI18n("error.e0116")
	}
	if conversation.Status != enums.IMConversationStatusPending {
		return errorsx.InvalidParamI18n("error.e0136")
	}
	if conversation.CurrentAssigneeID > 0 {
		return errorsx.InvalidParamI18n("error.e0190")
	}

	aiAgent := AIAgentService.Get(conversation.AIAgentID)
	if aiAgent == nil || aiAgent.Status != enums.StatusOk {
		return errorsx.InvalidParamI18n("error.e0003")
	}
	result, err := ConversationHumanDispatchService.DispatchPendingConversation(conversationID, *aiAgent)
	if err != nil {
		return err
	}
	if result == nil || result.Decision == HandoffDecisionOffHours {
		return errorsx.InvalidParamI18n("error.e0194")
	}
	return nil
}

func (s *conversationService) TransferConversation(conversationID, toUserID int64, reason string, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if toUserID <= 0 {
		return errorsx.InvalidParamI18n("error.e0278")
	}
	targetProfile := AgentProfileService.GetByUserID(toUserID)
	if targetProfile == nil || targetProfile.Status != enums.StatusOk {
		return errorsx.InvalidParamI18n("error.e0276")
	}
	var assignedEvent events.ConversationAssignedEvent
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		conversation := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if conversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if !s.canTransferConversation(conversation, operator) {
			return errorsx.ForbiddenI18n("error.e0223")
		}
		if conversation.Status != enums.IMConversationStatusActive {
			return errorsx.InvalidParamI18n("error.e0134")
		}
		if conversation.CurrentAssigneeID <= 0 {
			return errorsx.InvalidParamI18n("error.e0193")
		}
		if conversation.CurrentAssigneeID == toUserID {
			return errorsx.InvalidParamI18n("error.e0277")
		}
		now := time.Now()
		if err := ConversationAssignmentService.FinishActiveAssignments(ctx, conversationID, now); err != nil {
			return err
		}
		if err := ConversationAssignmentService.CreateAssignment(ctx, conversationID, conversation.CurrentAssigneeID, toUserID, enums.IMAssignmentTypeTransfer, reason, operator, now); err != nil {
			return err
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"current_assignee_id": toUserID,
			"status":              enums.IMConversationStatusActive,
			"update_user_id":      operator.UserID,
			"update_user_name":    operator.Username,
			"updated_at":          now,
		}); err != nil {
			return err
		}
		if err := ConversationEventLogService.CreateEvent(ctx, conversationID, enums.IMEventTypeTransfer, enums.IMSenderTypeAgent, operator.UserID, "会话已转接", s.buildEventPayload(map[string]any{
			"fromStatus":     conversation.Status,
			"toStatus":       enums.IMConversationStatusActive,
			"fromAssigneeId": conversation.CurrentAssigneeID,
			"toAssigneeId":   toUserID,
			"reason":         strings.TrimSpace(reason),
		})); err != nil {
			return err
		}
		assignedEvent = events.ConversationAssignedEvent{
			ConversationID: conversationID,
			FromUserID:     conversation.CurrentAssigneeID,
			ToUserID:       toUserID,
			OperatorID:     operator.UserID,
			Reason:         strings.TrimSpace(reason),
			AssignType:     events.ConversationAssignTypeTransfer,
		}
		return nil
	}); err != nil {
		return err
	}
	if conversation := s.Get(conversationID); conversation != nil {
		WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationTransferred)
	}
	eventbus.PublishAsync(context.Background(), assignedEvent)
	return nil
}

func (s *conversationService) HandoffByAI(conversationID int64, aiAgent models.AIAgent, reason string) error {
	return s.HandoffByAIWithRequestID(conversationID, aiAgent, reason, "")
}

func (s *conversationService) HandoffByAIWithRequestID(conversationID int64, aiAgent models.AIAgent, reason string, requestID string) error {
	if conversationID <= 0 {
		return errorsx.InvalidParamI18n("error.e0116")
	}
	_, err := ConversationHumanDispatchService.HandoffByAIWithRequestID(conversationID, aiAgent, reason, requestID)
	if err != nil {
		slog.Warn("schedule-aware ai handoff failed",
			"requestId", requestID,
			"conversation_id", conversationID,
			"ai_agent_id", aiAgent.ID,
			"error", err)
	}
	return err
}

func (s *conversationService) TryOffHoursHandoffByAI(conversationID int64, aiAgent models.AIAgent, reason string) (bool, error) {
	return s.TryOffHoursHandoffByAIWithRequestID(conversationID, aiAgent, reason, "")
}

func (s *conversationService) TryOffHoursHandoffByAIWithRequestID(conversationID int64, aiAgent models.AIAgent, reason string, requestID string) (bool, error) {
	if conversationID <= 0 {
		return false, errorsx.InvalidParamI18n("error.e0116")
	}
	handled, err := ConversationHumanDispatchService.TryOffHoursHandoffByAIWithRequestID(conversationID, aiAgent, reason, requestID)
	if err != nil {
		slog.Warn("off-hours ai handoff failed",
			"requestId", requestID,
			"conversation_id", conversationID,
			"ai_agent_id", aiAgent.ID,
			"error", err)
	}
	return handled, err
}

func (s *conversationService) CloseConversation(conversationID int64, closeReason string, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	return s.closeConversation(conversationID, enums.IMSenderTypeAgent, closeReason, operator)
}

func (s *conversationService) CloseCustomerConversation(conversationID int64, externalUser openidentity.ExternalUser) error {
	conversation := s.Get(conversationID)
	if conversation == nil {
		return errorsx.InvalidParamI18n("error.e0116")
	}
	if !s.IsCustomerConversationOwner(conversation, externalUser) {
		return errorsx.ForbiddenI18n("error.e0222")
	}
	return s.closeConversation(conversationID, enums.IMSenderTypeCustomer, "", nil)
}

func (s *conversationService) ResumeAIConversation(conversationID int64, reason string, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "人工处理完成，恢复 AI 接待"
	}
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		conversation := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if conversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if !s.canResumeAIConversation(conversation, operator) {
			return errorsx.ForbiddenI18n("error.e0221")
		}
		now := time.Now()
		if err := ConversationAssignmentService.FinishActiveAssignments(ctx, conversationID, now); err != nil {
			return err
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"status":              enums.IMConversationStatusAIServing,
			"current_team_id":     0,
			"current_assignee_id": 0,
			"update_user_id":      operator.UserID,
			"update_user_name":    operator.Username,
			"updated_at":          now,
		}); err != nil {
			return err
		}
		return ConversationEventLogService.CreateEvent(ctx, conversationID, enums.IMEventTypeTransfer, enums.IMSenderTypeAgent, operator.UserID, "恢复 AI 接待", s.buildEventPayload(map[string]any{
			"fromStatus":     conversation.Status,
			"toStatus":       enums.IMConversationStatusAIServing,
			"fromAssigneeId": conversation.CurrentAssigneeID,
			"toAssigneeId":   int64(0),
			"fromTeamId":     conversation.CurrentTeamID,
			"toTeamId":       int64(0),
			"reason":         reason,
		}))
	}); err != nil {
		return err
	}
	if conversation := s.Get(conversationID); conversation != nil {
		WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationUpdated)
	}
	return nil
}

func (s *conversationService) closeConversation(conversationID int64, senderType enums.IMSenderType, closeReason string, operator *dto.AuthPrincipal) error {
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		conversation := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if conversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if conversation.Status == enums.IMConversationStatusClosed {
			return nil
		}
		if conversation.Status != enums.IMConversationStatusAIServing &&
			conversation.Status != enums.IMConversationStatusPending &&
			conversation.Status != enums.IMConversationStatusActive {
			return errorsx.InvalidParamI18n("error.e0197")
		}
		var (
			now          = time.Now()
			eventDesc    = "会话已关闭"
			operatorID   int64
			operatorName string
		)
		closeReason = strings.TrimSpace(closeReason)
		if senderType == enums.IMSenderTypeCustomer {
			eventDesc = "客户关闭会话"
		} else {
			if operator == nil {
				return errorsx.InvalidParamI18n("error.e0226")
			}
			if closeReason == "" {
				return errorsx.InvalidParamI18n("error.e0128")
			}
			if !s.canCloseConversation(conversation, operator) {
				return errorsx.ForbiddenI18n("error.e0221")
			}
			operatorID = operator.UserID
			operatorName = operator.Nickname
		}
		if err := ConversationAssignmentService.FinishActiveAssignments(ctx, conversationID, now); err != nil {
			return err
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"status":           enums.IMConversationStatusClosed,
			"closed_at":        now,
			"closed_by":        operatorID,
			"close_reason":     closeReason,
			"update_user_id":   operatorID,
			"update_user_name": operatorName,
			"updated_at":       now,
		}); err != nil {
			return err
		}
		return ConversationEventLogService.CreateEvent(ctx, conversationID, enums.IMEventTypeClose, senderType, operatorID, eventDesc, s.buildEventPayload(map[string]any{
			"fromStatus":     conversation.Status,
			"toStatus":       enums.IMConversationStatusClosed,
			"fromAssigneeId": conversation.CurrentAssigneeID,
			"toAssigneeId":   conversation.CurrentAssigneeID,
			"closeReason":    closeReason,
		}))
	}); err != nil {
		return err
	}
	if conversation := s.Get(conversationID); conversation != nil {
		WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationClosed)
	}
	return nil
}

// MarkAgentConversationReadToMessage 控制台客服将会话已读推进到指定消息。
func (s *conversationService) MarkAgentConversationReadToMessage(conversationID, messageID int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	conversation := s.Get(conversationID)
	if conversation == nil {
		return errorsx.InvalidParamI18n("error.e0116")
	}
	changed, err := s.markConversationReadWithActor(conversation, messageID, agentConversationReadActor{operator: operator})
	if err != nil {
		return err
	}
	if changed {
		if updated := s.Get(conversationID); updated != nil {
			WsService.PublishConversationChanged(updated, enums.IMRealtimeEventConversationRead)
		}
	}
	return nil
}

// MarkCustomerConversationReadToMessage IM 客户将会话已读推进到指定消息（需为会话归属外部身份）。
func (s *conversationService) MarkCustomerConversationReadToMessage(conversationID, messageID int64, external *openidentity.ExternalUser) error {
	if external == nil || strings.TrimSpace(external.ExternalID) == "" {
		return errorsx.UnauthorizedI18n("error.e0149")
	}
	conversation := s.Get(conversationID)
	if conversation == nil {
		return errorsx.InvalidParamI18n("error.e0116")
	}
	if !s.IsCustomerConversationOwner(conversation, *external) {
		return errorsx.ForbiddenI18n("error.e0222")
	}
	changed, err := s.markConversationReadWithActor(conversation, messageID, customerConversationReadActor{external: external})
	if err != nil {
		return err
	}
	if changed {
		if updated := s.Get(conversationID); updated != nil {
			WsService.PublishConversationChanged(updated, enums.IMRealtimeEventConversationRead)
		}
	}
	return nil
}

func displayExternalName(ext *openidentity.ExternalUser) string {
	if ext == nil {
		return ""
	}
	if n := strings.TrimSpace(ext.ExternalName); n != "" {
		return n
	}
	return strings.TrimSpace(ext.ExternalID)
}

// conversationReadActor 抽象「读者身份」，供 markConversationReadWithActor 共用（包内私有）。
type conversationReadActor interface {
	isAgentSide() bool
	getReadState(conversationID int64) *models.ConversationReadState
	markRead(ctx *sqls.TxContext, conversation *models.Conversation, targetMessage *models.Message) error
	conversationUpdateAudit() (userID int64, userName string)
}

type agentConversationReadActor struct {
	operator *dto.AuthPrincipal
}

func (a agentConversationReadActor) isAgentSide() bool { return true }

func (a agentConversationReadActor) getReadState(conversationID int64) *models.ConversationReadState {
	return ConversationReadStateService.GetByAgentReader(conversationID, a.operator)
}

func (a agentConversationReadActor) markRead(ctx *sqls.TxContext, conversation *models.Conversation, targetMessage *models.Message) error {
	_, err := ConversationReadStateService.MarkAgentRead(ctx, conversation, a.operator, targetMessage)
	return err
}

func (a agentConversationReadActor) conversationUpdateAudit() (int64, string) {
	if a.operator == nil {
		return 0, ""
	}
	return a.operator.UserID, a.operator.Username
}

type customerConversationReadActor struct {
	external *openidentity.ExternalUser
}

func (a customerConversationReadActor) isAgentSide() bool { return false }

func (a customerConversationReadActor) getReadState(conversationID int64) *models.ConversationReadState {
	return ConversationReadStateService.GetByCustomerReader(conversationID, a.external)
}

func (a customerConversationReadActor) markRead(ctx *sqls.TxContext, conversation *models.Conversation, targetMessage *models.Message) error {
	_, err := ConversationReadStateService.MarkCustomerRead(ctx, conversation, a.external, targetMessage)
	return err
}

func (a customerConversationReadActor) conversationUpdateAudit() (int64, string) {
	return 0, displayExternalName(a.external)
}

func (s *conversationService) markConversationReadWithActor(conversation *models.Conversation, messageID int64, actor conversationReadActor) (bool, error) {
	if conversation == nil {
		return false, errorsx.InvalidParamI18n("error.e0116")
	}
	targetMessage, err := MessageService.GetConversationReadTarget(conversation.ID, messageID)
	if err != nil {
		return false, err
	}
	if targetMessage == nil {
		if actor.isAgentSide() && conversation.AgentUnreadCount == 0 {
			return false, nil
		}
		if !actor.isAgentSide() && conversation.CustomerUnreadCount == 0 {
			return false, nil
		}
		now := time.Now()
		updateUserID, updateUserName := actor.conversationUpdateAudit()
		updates := map[string]any{
			"update_user_id":   updateUserID,
			"update_user_name": updateUserName,
			"updated_at":       now,
		}
		if actor.isAgentSide() {
			updates["agent_unread_count"] = 0
		} else {
			updates["customer_unread_count"] = 0
		}
		return true, s.Updates(conversation.ID, updates)
	}

	currentReadState := actor.getReadState(conversation.ID)
	if currentReadState != nil && currentReadState.LastReadMessageID >= targetMessage.ID {
		if actor.isAgentSide() && conversation.AgentUnreadCount == 0 {
			return false, nil
		}
		if !actor.isAgentSide() && conversation.CustomerUnreadCount == 0 {
			return false, nil
		}
	}

	err = sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		currentConversation := repositories.ConversationRepository.Get(ctx.Tx, conversation.ID)
		if currentConversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if err := actor.markRead(ctx, currentConversation, targetMessage); err != nil {
			return err
		}
		agentReadState, customerReadState := ConversationReadStateService.getConversationReadStates(ctx.Tx, currentConversation.ID)
		agentUnreadCount, err := s.countUnreadByState(ctx, currentConversation.ID, agentReadState, enums.IMSenderTypeCustomer)
		if err != nil {
			return err
		}
		customerUnreadCount, err := s.countUnreadByState(ctx, currentConversation.ID, customerReadState, enums.IMSenderTypeAgent, enums.IMSenderTypeAI)
		if err != nil {
			return err
		}
		if actor.isAgentSide() && currentConversation.AgentUnreadCount == agentUnreadCount && currentReadState != nil && currentReadState.LastReadMessageID >= targetMessage.ID {
			return nil
		}
		if !actor.isAgentSide() && currentConversation.CustomerUnreadCount == customerUnreadCount && currentReadState != nil && currentReadState.LastReadMessageID >= targetMessage.ID {
			return nil
		}
		updateUserID, updateUserName := actor.conversationUpdateAudit()
		return repositories.ConversationRepository.Updates(ctx.Tx, currentConversation.ID, map[string]any{
			"agent_unread_count":    agentUnreadCount,
			"customer_unread_count": customerUnreadCount,
			"update_user_id":        updateUserID,
			"update_user_name":      updateUserName,
			"updated_at":            time.Now(),
		})
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *conversationService) countUnreadByState(ctx *sqls.TxContext, conversationID int64, state *models.ConversationReadState, senderTypes ...enums.IMSenderType) (int, error) {
	lastReadMessageID := int64(0)
	if state != nil {
		lastReadMessageID = state.LastReadMessageID
	}
	normalizedSenderTypes := make([]enums.IMSenderType, 0, len(senderTypes))
	for _, senderType := range senderTypes {
		normalizedSenderTypes = append(normalizedSenderTypes, senderType)
	}
	count, err := ConversationReadStateService.CountUnreadMessages(ctx, conversationID, lastReadMessageID, normalizedSenderTypes...)
	return int(count), err
}

func (s *conversationService) IsCustomerConversationOwner(conversation *models.Conversation, externalUser openidentity.ExternalUser) bool {
	if conversation == nil {
		return false
	}
	extID := strings.TrimSpace(externalUser.ExternalID)
	if extID == "" || strings.TrimSpace(string(externalUser.ExternalSource)) == "" || conversation.CustomerID <= 0 {
		return false
	}
	identity := repositories.CustomerIdentityRepository.GetBy(sqls.DB(), externalUser.ExternalSource, extID)
	if identity == nil {
		return false
	}
	return identity.CustomerID == conversation.CustomerID
}

func (s *conversationService) BuildConversationSummary(conversation *models.Conversation) string {
	if conversation == nil {
		return ""
	}
	if strings.TrimSpace(conversation.LastMessageSummary) != "" {
		return conversation.LastMessageSummary
	}
	return strings.TrimSpace(conversation.CustomerName)
}

func (s *conversationService) BuildHandoffContext(conversation *models.Conversation, reason string) string {
	if conversation == nil {
		return ""
	}
	lines := make([]string, 0, 10)
	if name := strings.TrimSpace(conversation.CustomerName); name != "" {
		lines = append(lines, "客户: "+name)
	}
	if lead := s.findConversationLead(conversation); lead != nil {
		lines = append(lines, buildLeadHandoffLines(lead)...)
	}
	if summary := strings.TrimSpace(s.BuildConversationSummary(conversation)); summary != "" {
		lines = append(lines, "会话摘要: "+summary)
	}
	if recent := s.buildRecentMessageSummary(conversation.ID, 4); recent != "" {
		lines = append(lines, "最近对话:\n"+recent)
	}
	return strings.Join(dedupeNonBlankLines(lines), "\n")
}

func (s *conversationService) BuildFollowUpAdvice(conversationID int64) (response.ConversationFollowUpAdviceResponse, error) {
	ret := response.ConversationFollowUpAdviceResponse{
		ConversationID: conversationID,
		Source:         "conversation",
	}
	conversation := s.Get(conversationID)
	if conversation == nil {
		return ret, errorsx.InvalidParamI18n("error.e0116")
	}
	if lead := s.findConversationLead(conversation); lead != nil {
		followUps := SalesLeadService.FindFollowUps(lead.ID)
		ret.LeadID = lead.ID
		ret.Source = "sales_lead"
		ret.SalesLeadFollowUpAdviceResult = SalesLeadService.BuildFollowUpAdvice(lead, followUps)
		if recent := strings.TrimSpace(s.buildRecentMessageSummary(conversation.ID, 4)); recent != "" && !strings.Contains(ret.CopyText, "最近对话：") {
			ret.CopyText = strings.TrimSpace(ret.CopyText) + "\n最近对话：\n" + recent
		}
		return ret, nil
	}

	ret.SalesLeadFollowUpAdviceResult = s.buildConversationOnlyFollowUpAdvice(conversation)
	return ret, nil
}

func (s *conversationService) buildConversationOnlyFollowUpAdvice(conversation *models.Conversation) response.SalesLeadFollowUpAdviceResult {
	if conversation == nil {
		return response.SalesLeadFollowUpAdviceResult{}
	}
	customerName := strings.TrimSpace(conversation.CustomerName)
	if customerName == "" {
		customerName = fmt.Sprintf("会话#%d客户", conversation.ID)
	}
	summary := strings.TrimSpace(s.BuildConversationSummary(conversation))
	recent := strings.TrimSpace(s.buildRecentMessageSummary(conversation.ID, 6))
	customerSummaryParts := []string{customerName}
	if conversation.CustomerID > 0 {
		customerSummaryParts = append(customerSummaryParts, fmt.Sprintf("客户ID %d", conversation.CustomerID))
	}
	if summary != "" && summary != customerName {
		customerSummaryParts = append(customerSummaryParts, limitText(summary, 160))
	}
	nextAction := "先确认客户核心需求、联系方式、预算区间和是否方便到店体验，再根据需求建立销售线索并安排跟进。"
	scriptName := "您好"
	if customerName != "" && !strings.Contains(customerName, "会话#") {
		scriptName = customerName + "您好"
	}
	script := fmt.Sprintf("%s，我看到您刚才咨询的内容。为了帮您推荐得更准，我想再确认一下使用人、尺寸、软硬偏好、预算以及是否方便到店体验，我会据此给您整理 1-2 个更合适的方案。", scriptName)
	riskHints := []string{"尚未形成销售线索"}
	if recent == "" {
		riskHints = append(riskHints, "最近对话内容不足")
	}
	if conversation.CustomerID <= 0 {
		riskHints = append(riskHints, "未关联 CRM 客户")
	}

	copyLines := []string{
		"【会话跟进摘要】",
		"客户：" + customerName,
		fmt.Sprintf("会话ID：%d", conversation.ID),
	}
	if summary != "" {
		copyLines = append(copyLines, "会话摘要："+limitText(summary, 240))
	}
	if recent != "" {
		copyLines = append(copyLines, "最近对话：\n"+recent)
	}
	copyLines = append(copyLines, "建议下一步："+nextAction, "建议话术："+script)
	if len(riskHints) > 0 {
		copyLines = append(copyLines, "注意事项："+strings.Join(riskHints, "；"))
	}

	return response.SalesLeadFollowUpAdviceResult{
		CustomerSummary: strings.Join(dedupeNonBlankLines(customerSummaryParts), "｜"),
		NextAction:      nextAction,
		Script:          script,
		CopyText:        strings.Join(copyLines, "\n"),
		RiskHints:       riskHints,
	}
}

func (s *conversationService) findConversationLead(conversation *models.Conversation) *models.SalesLead {
	if conversation == nil {
		return nil
	}
	if conversation.ID > 0 {
		if lead := repositories.SalesLeadRepository.FindOne(sqls.DB(), sqls.NewCnd().
			Where("conversation_id = ?", conversation.ID).
			Where("status <> ?", enums.SalesLeadStatusClosed).
			Desc("id")); lead != nil {
			return lead
		}
	}
	if conversation.CustomerID > 0 {
		return repositories.SalesLeadRepository.FindOne(sqls.DB(), sqls.NewCnd().
			Where("customer_id = ?", conversation.CustomerID).
			Where("status <> ?", enums.SalesLeadStatusClosed).
			Desc("id"))
	}
	return nil
}

func buildLeadHandoffLines(lead *models.SalesLead) []string {
	if lead == nil {
		return nil
	}
	lines := make([]string, 0, 8)
	if name := strings.TrimSpace(lead.CustomerName); name != "" {
		lines = append(lines, "线索姓名: "+name)
	}
	if phone := strings.TrimSpace(lead.Phone); phone != "" {
		lines = append(lines, "手机号: "+phone)
	}
	if wechat := strings.TrimSpace(lead.WeChat); wechat != "" {
		lines = append(lines, "微信: "+wechat)
	}
	if city := strings.TrimSpace(lead.City); city != "" {
		lines = append(lines, "城市: "+city)
	}
	if product := strings.TrimSpace(lead.InterestedProducts); product != "" {
		lines = append(lines, "关注产品: "+product)
	}
	if budget := formatLeadBudget(lead.BudgetMin, lead.BudgetMax); budget != "" {
		lines = append(lines, "预算: "+budget)
	}
	if appointment := formatLeadAppointment(lead); appointment != "" {
		lines = append(lines, "预约: "+appointment)
	}
	lines = append(lines, "意向等级: "+salesLeadIntentLabel(lead.IntentLevel))
	lines = append(lines, "购买阶段: "+salesLeadStageLabel(lead.BuyingStage))
	if demand := strings.TrimSpace(lead.DemandSummary); demand != "" {
		lines = append(lines, "需求摘要: "+limitText(demand, 180))
	}
	return lines
}

func formatLeadAppointment(lead *models.SalesLead) string {
	if lead == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if lead.AppointmentAt != nil {
		parts = append(parts, lead.AppointmentAt.Format("2006-01-02 15:04"))
	}
	if text := strings.TrimSpace(lead.AppointmentTimeText); text != "" {
		parts = append(parts, text)
	}
	if store := strings.TrimSpace(lead.AppointmentStore); store != "" {
		parts = append(parts, store)
	}
	if lead.AppointmentPeople > 0 {
		parts = append(parts, fmt.Sprintf("%d人", lead.AppointmentPeople))
	}
	return strings.Join(dedupeNonBlankLines(parts), " / ")
}

func (s *conversationService) buildRecentMessageSummary(conversationID int64, limit int) string {
	if conversationID <= 0 {
		return ""
	}
	if limit <= 0 {
		limit = 4
	}
	messages := repositories.MessageRepository.Find(sqls.DB(), sqls.NewCnd().
		Where("conversation_id = ?", conversationID).
		Where("message_type = ?", enums.IMMessageTypeText).
		Where("recalled_at IS NULL").
		Desc("id").
		Limit(limit))
	if len(messages) == 0 {
		return ""
	}
	slices.Reverse(messages)
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		content := limitText(message.Content, 120)
		if content == "" {
			continue
		}
		lines = append(lines, enums.GetIMSenderTypeLabel(message.SenderType)+": "+content)
	}
	return strings.Join(lines, "\n")
}

func dedupeNonBlankLines(lines []string) []string {
	ret := make([]string, 0, len(lines))
	seen := make(map[string]bool, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		ret = append(ret, line)
	}
	return ret
}

func formatLeadBudget(min, max int64) string {
	switch {
	case min > 0 && max > 0:
		return fmt.Sprintf("%d-%d元", min, max)
	case max > 0:
		return fmt.Sprintf("%d元左右", max)
	case min > 0:
		return fmt.Sprintf("%d元以上", min)
	default:
		return ""
	}
}

func salesLeadIntentLabel(value enums.SalesLeadIntent) string {
	switch value {
	case enums.SalesLeadIntentHigh:
		return "高"
	case enums.SalesLeadIntentMedium:
		return "中"
	case enums.SalesLeadIntentLow:
		return "低"
	default:
		return "未知"
	}
}

func salesLeadStageLabel(value enums.SalesLeadStage) string {
	switch value {
	case enums.SalesLeadStageConsulting:
		return "咨询了解"
	case enums.SalesLeadStageComparing:
		return "对比方案"
	case enums.SalesLeadStageAppointment:
		return "预约到店"
	case enums.SalesLeadStageReadyToBuy:
		return "临门购买"
	case enums.SalesLeadStageAfterSales:
		return "售后咨询"
	default:
		return "未知"
	}
}

func (s *conversationService) getCustomerName(db *gorm.DB, customerID int64) string {
	if customerID <= 0 {
		return ""
	}
	if customer := repositories.CustomerRepository.Get(db, customerID); customer != nil {
		return strings.TrimSpace(customer.Name)
	}
	return ""
}

func (s *conversationService) canCloseConversation(conversation *models.Conversation, operator *dto.AuthPrincipal) bool {
	if conversation == nil || operator == nil {
		return false
	}
	if s.isAdmin(operator) {
		return true
	}
	return conversation.Status == enums.IMConversationStatusActive && conversation.CurrentAssigneeID > 0 && conversation.CurrentAssigneeID == operator.UserID
}

func (s *conversationService) canTransferConversation(conversation *models.Conversation, operator *dto.AuthPrincipal) bool {
	if conversation == nil || operator == nil {
		return false
	}
	if s.isAdmin(operator) {
		return true
	}
	return conversation.Status == enums.IMConversationStatusActive &&
		conversation.CurrentAssigneeID > 0 &&
		conversation.CurrentAssigneeID == operator.UserID
}

func (s *conversationService) canResumeAIConversation(conversation *models.Conversation, operator *dto.AuthPrincipal) bool {
	if conversation == nil || operator == nil {
		return false
	}
	if conversation.ServiceMode == enums.IMConversationServiceModeHumanOnly ||
		conversation.ServiceMode == enums.IMConversationServiceModeAIOnly ||
		conversation.Status == enums.IMConversationStatusClosed ||
		conversation.Status == enums.IMConversationStatusAIServing {
		return false
	}
	if s.isAdmin(operator) {
		return true
	}
	return conversation.Status == enums.IMConversationStatusActive &&
		conversation.CurrentAssigneeID > 0 &&
		conversation.CurrentAssigneeID == operator.UserID
}

func (s *conversationService) isAdmin(operator *dto.AuthPrincipal) bool {
	if operator == nil {
		return false
	}
	return slices.Contains(operator.Roles, constants.RoleCodeSuperAdmin) || slices.Contains(operator.Roles, constants.RoleCodeAdmin)
}

func (s *conversationService) buildEventPayload(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

// LinkConversationCustomer 将会话绑定到指定客户。
func (s *conversationService) LinkConversationCustomer(conversationID, customerID int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if conversationID <= 0 || customerID <= 0 {
		return errorsx.InvalidParamI18n("error.e0133")
	}
	cust := CustomerService.Get(customerID)
	if cust == nil || cust.Status == enums.StatusDeleted {
		return errorsx.InvalidParamI18n("error.e0155")
	}
	conv := s.Get(conversationID)
	if conv == nil {
		return errorsx.InvalidParamI18n("error.e0116")
	}
	if conv.Status == enums.IMConversationStatusClosed {
		return errorsx.InvalidParamI18n("error.e0183")
	}
	if !s.canLinkConversationCustomer(conv, operator) {
		return errorsx.ForbiddenI18n("error.e0224")
	}

	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		current := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if current == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		now := time.Now()
		return repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"customer_id":      customerID,
			"customer_name":    strings.TrimSpace(cust.Name),
			"update_user_id":   operator.UserID,
			"update_user_name": operator.Username,
			"updated_at":       now,
		})
	})
	if err != nil {
		return err
	}
	if updated := s.Get(conversationID); updated != nil {
		WsService.PublishConversationChanged(updated, enums.IMRealtimeEventConversationUpdated)
	}
	return nil
}

func (s *conversationService) GetConversationExternalIdentity(conversation *models.Conversation) *models.CustomerIdentity {
	if conversation == nil || conversation.CustomerID <= 0 {
		return nil
	}
	identities := repositories.CustomerIdentityRepository.FindByCustomerID(sqls.DB(), conversation.CustomerID)
	if len(identities) == 0 {
		return nil
	}
	if channel := ChannelService.Get(conversation.ChannelID); channel != nil {
		expected := externalSourceForChannelType(channel.ChannelType)
		if strings.TrimSpace(string(expected)) != "" {
			for i := range identities {
				if identities[i].ExternalSource == expected {
					return &identities[i]
				}
			}
		}
	}
	return &identities[0]
}

func externalSourceForChannelType(channelType string) enums.ExternalSource {
	switch strings.TrimSpace(channelType) {
	case enums.ChannelTypeWxWorkKF:
		return enums.ExternalSourceWxWorkKF
	case enums.ChannelTypeWeb:
		return enums.ExternalSourceGuest
	default:
		return ""
	}
}

func (s *conversationService) canLinkConversationCustomer(conv *models.Conversation, operator *dto.AuthPrincipal) bool {
	if conv == nil || operator == nil {
		return false
	}
	if s.isAdmin(operator) {
		return true
	}
	switch conv.Status {
	case enums.IMConversationStatusAIServing:
		return true
	case enums.IMConversationStatusPending:
		return true
	case enums.IMConversationStatusActive:
		return conv.CurrentAssigneeID == 0 || conv.CurrentAssigneeID == operator.UserID
	default:
		return false
	}
}

func (s *conversationService) resolveInitialStatus(serviceMode enums.IMConversationServiceMode) enums.IMConversationStatus {
	switch serviceMode {
	case enums.IMConversationServiceModeHumanOnly:
		return enums.IMConversationStatusPending
	case enums.IMConversationServiceModeAIOnly, enums.IMConversationServiceModeAIFirst:
		return enums.IMConversationStatusAIServing
	default:
		return enums.IMConversationStatusAIServing
	}
}
