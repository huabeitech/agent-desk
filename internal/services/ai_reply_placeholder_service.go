package services

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

// AI 回复占位消息使用确定性 clientMsgID 前缀，保证同一触发消息的占位与超时提示幂等；
// aiReplyPendingPayload 为占位消息的 payload 标记，原地替换前用于守卫，避免覆盖已生成的正式回复。
const (
	aiReplyPlaceholderClientMsgIDPrefix   = "ai_placeholder_"
	aiReplyTimeoutNoticeClientMsgIDPrefix = "ai_timeout_"
	aiReplyPendingPayload                 = `{"aiReplyPending":true}`
)

// aiReplyPrincipal 构建 AI 消息的操作者信息。
func aiReplyPrincipal(aiAgent models.AIAgent) *dto.AuthPrincipal {
	username := "AI"
	if strings.TrimSpace(aiAgent.Name) != "" {
		username = strings.TrimSpace(aiAgent.Name)
	}
	return &dto.AuthPrincipal{
		UserID:   0,
		Username: username,
		Nickname: username,
	}
}

// aiReplyPlaceholderClientMsgID 返回触发消息对应的占位消息 clientMsgID。
func aiReplyPlaceholderClientMsgID(triggerMessageID int64) string {
	return aiReplyPlaceholderClientMsgIDPrefix + strconv.FormatInt(triggerMessageID, 10)
}

// SendAIReplyPlaceholder 在正式 AI 回复生成前，向客户发送一条占位提示消息。
// 通过确定性 clientMsgID 保证幂等；发送失败仅记录日志，不阻断 AI 回复流程。
func (s *messageService) SendAIReplyPlaceholder(conversation *models.Conversation, aiAgent *models.AIAgent, triggerMessage *models.Message, channel *models.Channel, requestID string) {
	if conversation == nil || aiAgent == nil || triggerMessage == nil {
		return
	}
	content := models.DefaultAIReplyPlaceholder
	if channel != nil {
		content = channel.EffectiveAIReplyPlaceholder()
	}
	if _, err := s.SendAIMessageWithRequestIDAndWorkflowRunID(
		conversation.ID,
		aiAgent.ID,
		aiReplyPlaceholderClientMsgID(triggerMessage.ID),
		enums.IMMessageTypeText,
		content,
		aiReplyPendingPayload,
		aiReplyPrincipal(*aiAgent),
		requestID,
		0,
	); err != nil {
		slog.Error("send ai reply placeholder failed",
			"conversation_id", conversation.ID,
			"message_id", triggerMessage.ID,
			"error", err,
		)
	}
}

// TryReplaceAIReplyPlaceholder 尝试将 web 渠道仍处于待回复状态的占位消息原地替换为正式回复。
// 仅 web 渠道支持原地替换；占位消息不存在或已被替换时返回 nil，由调用方回退为新消息。
func (s *messageService) TryReplaceAIReplyPlaceholder(conversation *models.Conversation, triggerMessageID int64, replyText string, workflowRunID int64, aiAgent models.AIAgent) *models.Message {
	if conversation == nil {
		return nil
	}
	channel := ChannelService.Get(conversation.ChannelID)
	if channel == nil || channel.ChannelType != enums.ChannelTypeWeb {
		return nil
	}
	placeholder := repositories.MessageRepository.GetByClientMsgID(sqls.DB(), conversation.ID, aiReplyPlaceholderClientMsgID(triggerMessageID))
	if placeholder == nil || strings.TrimSpace(placeholder.Payload) != aiReplyPendingPayload {
		return nil
	}
	content, _, summary, err := s.normalizeMessageContent(conversation.ID, enums.IMMessageTypeText, replyText, "")
	if err != nil {
		slog.Error("normalize ai reply content failed",
			"conversation_id", conversation.ID,
			"message_id", placeholder.ID,
			"error", err,
		)
		return nil
	}
	return s.replaceAIReplyPlaceholder(conversation, placeholder, content, summary, workflowRunID, aiAgent)
}

// CompleteAIReplyPlaceholderAsFailed 在 AI 处理超时或失败时向客户发送失败提示：
// web 渠道且占位消息仍待回复时原地替换为提示语；否则以新消息补发（确定性 clientMsgID 保证幂等）。
// 已转人工或已关闭的会话不再补发提示。
func (s *messageService) CompleteAIReplyPlaceholderAsFailed(conversation *models.Conversation, aiAgent *models.AIAgent, triggerMessage *models.Message, requestID string) {
	if conversation == nil || aiAgent == nil || triggerMessage == nil {
		return
	}
	if conversation.Status == enums.IMConversationStatusClosed {
		return
	}
	if conversation.CurrentAssigneeID > 0 || conversation.HandoffAt != nil {
		return
	}
	notice := models.DefaultAIReplyTimeoutNotice
	if channel := ChannelService.Get(conversation.ChannelID); channel != nil {
		notice = channel.EffectiveAIReplyTimeoutNotice()
	}
	content, _, summary, err := s.normalizeMessageContent(conversation.ID, enums.IMMessageTypeText, notice, "")
	if err != nil {
		slog.Error("normalize ai timeout notice failed",
			"conversation_id", conversation.ID,
			"message_id", triggerMessage.ID,
			"error", err,
		)
		return
	}

	placeholder := repositories.MessageRepository.GetByClientMsgID(sqls.DB(), conversation.ID, aiReplyPlaceholderClientMsgID(triggerMessage.ID))
	if placeholder != nil {
		if strings.TrimSpace(placeholder.Payload) != aiReplyPendingPayload {
			// 正式回复已生成并替换占位消息，无需再提示失败
			return
		}
		if channel := ChannelService.Get(conversation.ChannelID); channel != nil && channel.ChannelType == enums.ChannelTypeWeb {
			s.replaceAIReplyPlaceholder(conversation, placeholder, content, summary, 0, *aiAgent)
			return
		}
	}

	// 外部渠道无法编辑已发送消息，或占位消息缺失时，以新消息补发失败提示
	if _, err := s.SendAIMessageWithRequestIDAndWorkflowRunID(
		conversation.ID,
		aiAgent.ID,
		aiReplyTimeoutNoticeClientMsgIDPrefix+strconv.FormatInt(triggerMessage.ID, 10),
		enums.IMMessageTypeText,
		content,
		"",
		aiReplyPrincipal(*aiAgent),
		requestID,
		0,
	); err != nil {
		slog.Error("send ai timeout notice failed",
			"conversation_id", conversation.ID,
			"message_id", triggerMessage.ID,
			"error", err,
		)
	}
}

// replaceAIReplyPlaceholder 将占位消息原地更新为指定内容，并同步会话摘要与实时事件。
func (s *messageService) replaceAIReplyPlaceholder(conversation *models.Conversation, placeholder *models.Message, content string, summary string, workflowRunID int64, aiAgent models.AIAgent) *models.Message {
	now := time.Now()
	agentName := strings.TrimSpace(aiAgent.Name)
	// 占位消息内容与会话摘要的更新保持原子性
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.MessageRepository.Updates(ctx.Tx, placeholder.ID, map[string]any{
			"content":          content,
			"payload":          "",
			"workflow_run_id":  workflowRunID,
			"update_user_id":   0,
			"update_user_name": agentName,
			"updated_at":       now,
		}); err != nil {
			return err
		}

		// 占位消息仍是会话最后一条消息时，同步列表摘要
		if conversation.LastMessageID == placeholder.ID {
			summaryText := limitText(summary, 255)
			conversation.LastMessageSummary = summaryText
			conversation.UpdatedAt = now
			return repositories.ConversationRepository.Updates(ctx.Tx, conversation.ID, map[string]any{
				"last_message_summary": summaryText,
				"update_user_id":       0,
				"update_user_name":     agentName,
				"updated_at":           now,
			})
		}
		return nil
	}); err != nil {
		slog.Error("replace ai reply placeholder failed",
			"conversation_id", conversation.ID,
			"message_id", placeholder.ID,
			"error", err,
		)
		return nil
	}
	placeholder.Content = content
	placeholder.Payload = ""
	placeholder.WorkflowRunID = workflowRunID
	placeholder.UpdatedAt = now
	placeholder.UpdateUserID = 0
	placeholder.UpdateUserName = agentName

	WsService.PublishMessageUpdated(conversation, placeholder)
	WsService.PublishConversationChanged(conversation, enums.IMRealtimeEventConversationUpdated)
	return placeholder
}
