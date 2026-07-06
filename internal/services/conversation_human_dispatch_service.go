package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-desk/internal/events"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/eventbus"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

var ConversationHumanDispatchService = newConversationHumanDispatchService()

var (
	HandoffWaitingMessage  = HandoffWaitingMessageForLocale(i18nx.DefaultLocale)
	HandoffOffHoursMessage = HandoffOffHoursMessageForLocale(i18nx.DefaultLocale)
)

func HandoffWaitingMessageForLocale(locale string) string {
	return i18nx.Getf(locale, "conversation.handoff.waiting")
}

func HandoffOffHoursMessageForLocale(locale string) string {
	return i18nx.Getf(locale, "conversation.handoff.offHours")
}

type HandoffDecisionType string

const (
	HandoffDecisionAssigned   HandoffDecisionType = "assigned"
	HandoffDecisionTeamPool   HandoffDecisionType = "team_pool"
	HandoffDecisionGlobalPool HandoffDecisionType = "global_pool"
	HandoffDecisionOffHours   HandoffDecisionType = "off_hours"
)

type HandoffDecisionResult struct {
	Decision    HandoffDecisionType
	TeamID      int64
	AssigneeID  int64
	Message     string
	ContextText string
}

type conversationHumanDispatchService struct{}

func newConversationHumanDispatchService() *conversationHumanDispatchService {
	return &conversationHumanDispatchService{}
}

func (s *conversationHumanDispatchService) TryOffHoursHandoffByAI(conversationID int64, aiAgent models.AIAgent, reason string) (bool, error) {
	return s.TryOffHoursHandoffByAIWithRequestID(conversationID, aiAgent, reason, "")
}

func (s *conversationHumanDispatchService) TryOffHoursHandoffByAIWithRequestID(conversationID int64, aiAgent models.AIAgent, reason string, requestID string) (bool, error) {
	conversation := ConversationService.Get(conversationID)
	if conversation == nil {
		return false, errorsx.InvalidParamI18n("error.e0116")
	}
	teamIDs := orderedPositiveIDs(aiAgent.TeamIDs)
	activeTeamIDs := ConversationDispatchService.findActiveScheduleTeamIDs(teamIDs, time.Now())
	if len(activeTeamIDs) > 0 {
		return false, nil
	}
	if err := s.createEventWithRequestID(conversationID, requestID, enums.IMEventTypeTransfer, enums.IMSenderTypeAI, aiAgent.ID, "转人工失败：非服务时间", strings.TrimSpace(reason)); err != nil {
		return true, err
	}
	if err := s.sendAITextWithRequestID(conversationID, aiAgent.ID, HandoffOffHoursMessage, requestID); err != nil {
		return true, err
	}
	if err := s.ensureOffHoursFollowUpLead(conversation, aiAgent, reason); err != nil {
		return true, err
	}
	return true, nil
}

func (s *conversationHumanDispatchService) HandoffByAI(conversationID int64, aiAgent models.AIAgent, reason string) (*HandoffDecisionResult, error) {
	return s.HandoffByAIWithRequestID(conversationID, aiAgent, reason, "")
}

func (s *conversationHumanDispatchService) HandoffByAIWithRequestID(conversationID int64, aiAgent models.AIAgent, reason string, requestID string) (*HandoffDecisionResult, error) {
	conversation := ConversationService.Get(conversationID)
	if conversation == nil {
		return nil, errorsx.InvalidParamI18n("error.e0116")
	}
	teamIDs := orderedPositiveIDs(aiAgent.TeamIDs)
	activeTeamIDs := ConversationDispatchService.findActiveScheduleTeamIDs(teamIDs, time.Now())
	if len(activeTeamIDs) == 0 {
		if _, err := s.TryOffHoursHandoffByAIWithRequestID(conversationID, aiAgent, reason, requestID); err != nil {
			return nil, err
		}
		return &HandoffDecisionResult{Decision: HandoffDecisionOffHours, Message: HandoffOffHoursMessage}, nil
	}

	if err := s.markHandoff(conversationID, aiAgent, reason, requestID); err != nil {
		return nil, err
	}
	return s.dispatchAfterHandoffWithRequestID(conversationID, aiAgent.ID, activeTeamIDs, strings.TrimSpace(reason), true, requestID)
}

func (s *conversationHumanDispatchService) ApplyHumanOnlyCreate(conversationID int64, aiAgent models.AIAgent) (*HandoffDecisionResult, error) {
	teamIDs := orderedPositiveIDs(aiAgent.TeamIDs)
	activeTeamIDs := ConversationDispatchService.findActiveScheduleTeamIDs(teamIDs, time.Now())
	if len(activeTeamIDs) == 0 {
		if err := s.moveToGlobalPool(conversationID, aiAgent.Name); err != nil {
			return nil, err
		}
		if err := s.sendAIText(conversationID, aiAgent.ID, HandoffWaitingMessage); err != nil {
			return nil, err
		}
		return &HandoffDecisionResult{Decision: HandoffDecisionGlobalPool, Message: HandoffWaitingMessage}, nil
	}
	return s.dispatchAfterHandoff(conversationID, aiAgent.ID, activeTeamIDs, "仅人工模式新会话", false)
}

func (s *conversationHumanDispatchService) DispatchPendingConversation(conversationID int64, aiAgent models.AIAgent) (*HandoffDecisionResult, error) {
	conversation := ConversationService.Get(conversationID)
	if conversation == nil {
		return nil, errorsx.InvalidParamI18n("error.e0116")
	}
	if conversation.Status != enums.IMConversationStatusPending || conversation.CurrentAssigneeID > 0 {
		return nil, errorsx.InvalidParamI18n("error.e0137")
	}
	activeTeamIDs := ConversationDispatchService.findActiveScheduleTeamIDs(orderedPositiveIDs(aiAgent.TeamIDs), time.Now())
	if len(activeTeamIDs) == 0 {
		return &HandoffDecisionResult{Decision: HandoffDecisionOffHours}, nil
	}
	candidates, _, err := ConversationDispatchService.pickDispatchCandidates(activeTeamIDs, time.Now())
	if err != nil {
		return nil, err
	}
	if len(candidates) > 0 {
		dispatched, err := ConversationDispatchService.tryAssignConversation(conversationID, candidates[0].profile, "自动分配")
		if err != nil {
			return nil, err
		}
		if dispatched != nil {
			WsService.PublishConversationChanged(dispatched, enums.IMRealtimeEventConversationAssigned)
			return &HandoffDecisionResult{
				Decision:   HandoffDecisionAssigned,
				TeamID:     dispatched.CurrentTeamID,
				AssigneeID: dispatched.CurrentAssigneeID,
			}, nil
		}
	}
	teamID := activeTeamIDs[0]
	teamPoolConversation, err := s.moveToTeamPool(conversationID, teamID, "手动触发自动分配")
	if err != nil {
		return nil, err
	}
	if teamPoolConversation != nil {
		WsService.PublishConversationChanged(teamPoolConversation, enums.IMRealtimeEventConversationUpdated)
	}
	return &HandoffDecisionResult{Decision: HandoffDecisionTeamPool, TeamID: teamID}, nil
}

func (s *conversationHumanDispatchService) dispatchAfterHandoff(conversationID, aiAgentID int64, activeTeamIDs []int64, reason string, publishAssignEvent bool) (*HandoffDecisionResult, error) {
	return s.dispatchAfterHandoffWithRequestID(conversationID, aiAgentID, activeTeamIDs, reason, publishAssignEvent, "")
}

func (s *conversationHumanDispatchService) dispatchAfterHandoffWithRequestID(conversationID, aiAgentID int64, activeTeamIDs []int64, reason string, publishAssignEvent bool, requestID string) (*HandoffDecisionResult, error) {
	if err := s.sendAITextWithRequestID(conversationID, aiAgentID, HandoffWaitingMessage, requestID); err != nil {
		return nil, err
	}
	contextText := ConversationService.BuildHandoffContext(ConversationService.Get(conversationID), reason)

	candidates, _, err := ConversationDispatchService.pickDispatchCandidates(activeTeamIDs, time.Now())
	if err != nil {
		return nil, err
	}
	if len(candidates) > 0 {
		dispatched, err := ConversationDispatchService.tryAssignConversation(conversationID, candidates[0].profile, "自动分配")
		if err != nil {
			return nil, err
		}
		if dispatched != nil {
			WsService.PublishConversationChanged(dispatched, enums.IMRealtimeEventConversationAssigned)
			if publishAssignEvent {
				eventbus.PublishAsync(context.Background(), events.ConversationAssignedEvent{
					ConversationID: dispatched.ID,
					ToUserID:       dispatched.CurrentAssigneeID,
					OperatorID:     systemDispatchPrincipal().UserID,
					Reason:         "自动分配",
					AssignType:     events.ConversationAssignTypeAutoAssign,
					ContextText:    contextText,
				})
			}
			return &HandoffDecisionResult{
				Decision:    HandoffDecisionAssigned,
				TeamID:      dispatched.CurrentTeamID,
				AssigneeID:  dispatched.CurrentAssigneeID,
				Message:     HandoffWaitingMessage,
				ContextText: contextText,
			}, nil
		}
	}

	teamID := activeTeamIDs[0]
	teamPoolConversation, err := s.moveToTeamPoolWithRequestID(conversationID, teamID, reason, requestID)
	if err != nil {
		return nil, err
	}
	if teamPoolConversation != nil {
		WsService.PublishConversationChanged(teamPoolConversation, enums.IMRealtimeEventConversationUpdated)
	}
	return &HandoffDecisionResult{Decision: HandoffDecisionTeamPool, TeamID: teamID, Message: HandoffWaitingMessage, ContextText: contextText}, nil
}

func (s *conversationHumanDispatchService) markHandoff(conversationID int64, aiAgent models.AIAgent, reason string, requestID string) error {
	now := time.Now()
	trimmedReason := strings.TrimSpace(reason)
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		conversation := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if conversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"handoff_at":          now,
			"handoff_reason":      trimmedReason,
			"status":              enums.IMConversationStatusPending,
			"current_team_id":     0,
			"current_assignee_id": 0,
			"update_user_id":      0,
			"update_user_name":    aiAgent.Name,
			"updated_at":          now,
		}); err != nil {
			return err
		}
		return ConversationEventLogService.CreateEventWithRequestID(ctx, conversationID, requestID, enums.IMEventTypeTransfer, enums.IMSenderTypeAI, aiAgent.ID, "AI转人工", ConversationService.buildEventPayload(map[string]any{
			"reason":  trimmedReason,
			"context": ConversationService.BuildHandoffContext(conversation, trimmedReason),
		}))
	})
}

func (s *conversationHumanDispatchService) moveToTeamPool(conversationID, teamID int64, reason string) (*models.Conversation, error) {
	return s.moveToTeamPoolWithRequestID(conversationID, teamID, reason, "")
}

func (s *conversationHumanDispatchService) moveToTeamPoolWithRequestID(conversationID, teamID int64, reason string, requestID string) (*models.Conversation, error) {
	now := time.Now()
	var conversation *models.Conversation
	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		current := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if current == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if err := ConversationAssignmentService.FinishActiveAssignments(ctx, conversationID, now); err != nil {
			return err
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"status":              enums.IMConversationStatusPending,
			"current_team_id":     teamID,
			"current_assignee_id": 0,
			"update_user_id":      0,
			"update_user_name":    "system",
			"updated_at":          now,
		}); err != nil {
			return err
		}
		if err := ConversationEventLogService.CreateEventWithRequestID(ctx, conversationID, requestID, enums.IMEventTypeTransfer, enums.IMSenderTypeSystem, 0, "会话进入客服组待接入", ConversationService.buildEventPayload(map[string]any{
			"fromStatus":     current.Status,
			"toStatus":       enums.IMConversationStatusPending,
			"fromAssigneeId": current.CurrentAssigneeID,
			"toAssigneeId":   int64(0),
			"toTeamId":       teamID,
			"reason":         strings.TrimSpace(reason),
			"decision":       string(HandoffDecisionTeamPool),
			"context":        ConversationService.BuildHandoffContext(current, reason),
		})); err != nil {
			return err
		}
		current.Status = enums.IMConversationStatusPending
		current.CurrentTeamID = teamID
		current.CurrentAssigneeID = 0
		current.UpdateUserID = 0
		current.UpdateUserName = "system"
		current.UpdatedAt = now
		conversation = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

func (s *conversationHumanDispatchService) moveToGlobalPool(conversationID int64, operatorName string) error {
	now := time.Now()
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		conversation := repositories.ConversationRepository.Get(ctx.Tx, conversationID)
		if conversation == nil {
			return errorsx.InvalidParamI18n("error.e0116")
		}
		if err := repositories.ConversationRepository.Updates(ctx.Tx, conversationID, map[string]any{
			"status":              enums.IMConversationStatusPending,
			"current_team_id":     0,
			"current_assignee_id": 0,
			"update_user_id":      0,
			"update_user_name":    operatorName,
			"updated_at":          now,
		}); err != nil {
			return err
		}
		return ConversationEventLogService.CreateEvent(ctx, conversationID, enums.IMEventTypeTransfer, enums.IMSenderTypeSystem, 0, "会话进入全局待接入", ConversationService.buildEventPayload(map[string]any{
			"fromStatus": conversation.Status,
			"toStatus":   enums.IMConversationStatusPending,
			"decision":   string(HandoffDecisionGlobalPool),
		}))
	})
}

func (s *conversationHumanDispatchService) ensureOffHoursFollowUpLead(conversation *models.Conversation, aiAgent models.AIAgent, reason string) error {
	if conversation == nil || conversation.ID <= 0 {
		return nil
	}
	now := time.Now()
	nextFollowUpAt := nextOffHoursFollowUpTime(now)
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "客户在非服务时间请求人工"
	}
	summary := limitText("非服务时间转人工："+trimmedReason, 500)
	stage := enums.SalesLeadStageConsulting
	if containsAnyLeadText(trimmedReason, "售后", "投诉", "退款", "退货", "换货", "质保", "异响", "故障", "不满意", "差评") {
		stage = enums.SalesLeadStageAfterSales
	}

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		lead := repositories.SalesLeadRepository.FindOne(ctx.Tx, sqls.NewCnd().
			Where("conversation_id = ?", conversation.ID).
			Where("status IN ?", []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}).
			Desc("id"))
		if lead == nil && conversation.CustomerID > 0 {
			lead = repositories.SalesLeadRepository.FindOne(ctx.Tx, sqls.NewCnd().
				Where("customer_id = ?", conversation.CustomerID).
				Where("status IN ?", []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}).
				Desc("id"))
		}
		if lead == nil {
			lead = &models.SalesLead{
				CustomerID:     conversation.CustomerID,
				ConversationID: conversation.ID,
				CustomerName:   strings.TrimSpace(conversation.CustomerName),
				DemandSummary:  summary,
				IntentLevel:    enums.SalesLeadIntentMedium,
				BuyingStage:    stage,
				SourceChannel:  "off_hours_handoff",
				Status:         enums.SalesLeadStatusNew,
				NextFollowUpAt: &nextFollowUpAt,
				Remark:         "非服务时间请求人工，待顾问跟进",
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					UpdatedAt:      now,
					CreateUserName: aiAgent.Name,
					UpdateUserName: aiAgent.Name,
				},
			}
			return repositories.SalesLeadRepository.Create(ctx.Tx, lead)
		}
		updates := map[string]any{
			"conversation_id":   conversation.ID,
			"updated_at":        now,
			"update_user_name":  aiAgent.Name,
			"source_channel":    "off_hours_handoff",
			"last_message_id":   conversation.LastMessageID,
			"next_follow_up_at": &nextFollowUpAt,
		}
		if strings.TrimSpace(lead.CustomerName) == "" && strings.TrimSpace(conversation.CustomerName) != "" {
			updates["customer_name"] = strings.TrimSpace(conversation.CustomerName)
		}
		if strings.TrimSpace(lead.DemandSummary) == "" {
			updates["demand_summary"] = summary
		} else if !strings.Contains(lead.DemandSummary, trimmedReason) {
			updates["demand_summary"] = limitText(lead.DemandSummary+"\n"+summary, 1000)
		}
		if lead.BuyingStage == enums.SalesLeadStageUnknown || lead.BuyingStage == enums.SalesLeadStageConsulting {
			updates["buying_stage"] = stage
		}
		if lead.IntentLevel == enums.SalesLeadIntentUnknown || lead.IntentLevel == enums.SalesLeadIntentLow {
			updates["intent_level"] = enums.SalesLeadIntentMedium
		}
		if lead.Status == enums.SalesLeadStatusNew || lead.Status == "" {
			updates["status"] = enums.SalesLeadStatusNew
		}
		return repositories.SalesLeadRepository.Updates(ctx.Tx, lead.ID, updates)
	})
}

func nextOffHoursFollowUpTime(now time.Time) time.Time {
	target := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
	if now.Before(target) {
		return target
	}
	return target.AddDate(0, 0, 1)
}

func (s *conversationHumanDispatchService) createEvent(conversationID int64, eventType enums.IMEventType, senderType enums.IMSenderType, senderID int64, content, payload string) error {
	return s.createEventWithRequestID(conversationID, "", eventType, senderType, senderID, content, payload)
}

func (s *conversationHumanDispatchService) createEventWithRequestID(conversationID int64, requestID string, eventType enums.IMEventType, senderType enums.IMSenderType, senderID int64, content, payload string) error {
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		return ConversationEventLogService.CreateEventWithRequestID(ctx, conversationID, requestID, eventType, senderType, senderID, content, payload)
	})
}

func (s *conversationHumanDispatchService) sendAIText(conversationID, aiAgentID int64, content string) error {
	return s.sendAITextWithRequestID(conversationID, aiAgentID, content, "")
}

func (s *conversationHumanDispatchService) sendAITextWithRequestID(conversationID, aiAgentID int64, content string, requestID string) error {
	_, err := MessageService.SendAIServiceNoticeWithRequestID(conversationID, aiAgentID, content, requestID)
	return err
}

func orderedPositiveIDs(value string) []int64 {
	return uniquePositiveInt64sFromStrings(strings.Split(value, ","))
}

func uniquePositiveInt64sFromStrings(values []string) []int64 {
	seen := make(map[int64]struct{}, len(values))
	ret := make([]int64, 0, len(values))
	for _, value := range values {
		var id int64
		_, _ = fmt.Sscan(strings.TrimSpace(value), &id)
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ret = append(ret, id)
	}
	return ret
}
