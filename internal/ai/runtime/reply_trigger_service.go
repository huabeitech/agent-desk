package runtime

import (
	"context"
	"log/slog"
	"strings"
	"time"

	applicationruntime "agent-desk/internal/ai/application/runtime"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/tracex"
	svc "agent-desk/internal/services"
)

// resolveReplyTimeout 解析 AI 回复超时时长：接入渠道配置优先，其次智能体配置，最后使用系统默认。
func (s *aiReplyService) resolveReplyTimeout(channel *models.Channel, aiAgent models.AIAgent) time.Duration {
	seconds := 0
	if channel != nil && channel.AIReplyTimeoutSeconds > 0 {
		seconds = channel.AIReplyTimeoutSeconds
	}
	if seconds <= 0 && aiAgent.ReplyTimeoutSeconds > 0 {
		seconds = aiAgent.ReplyTimeoutSeconds
	}
	if seconds <= 0 {
		seconds = models.DefaultAIReplyTimeoutSeconds
	}
	if seconds > maxAIReplyAsyncTimeoutSeconds {
		seconds = maxAIReplyAsyncTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (s *aiReplyService) TriggerReplyAsync(conversation models.Conversation, message models.Message) {
	go func() {
		aiAgent := svc.AIAgentService.Get(conversation.AIAgentID)
		if aiAgent == nil || aiAgent.Status != enums.StatusOk {
			return
		}
		channel := svc.ChannelService.Get(conversation.ChannelID)
		startedAt := time.Now()
		timeout := s.resolveReplyTimeout(channel, *aiAgent)
		ctx, cancel := context.WithTimeout(tracex.ContextWithRequestID(context.Background(), message.RequestID), timeout)
		defer cancel()
		if err := s.TriggerReply(ctx, conversation, message, *aiAgent); err != nil {
			slog.Error("failed to trigger ai reply",
				"requestId", message.RequestID,
				"message_id", message.ID,
				"timeout_ms", timeout.Milliseconds(),
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
				"error", err)
			svc.MessageService.CompleteAIReplyPlaceholderAsFailed(&conversation, aiAgent, &message, message.RequestID)
		}
	}()
}

func (s *aiReplyService) TriggerReply(ctx context.Context, conversation models.Conversation, message models.Message, aiAgent models.AIAgent) (retErr error) {
	var summary *applicationruntime.RunResult
	replyCtx := aiReplyContext{
		Conversation: conversation,
		Message:      message,
		AIAgent:      aiAgent,
		SummaryRef:   &summary,
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.eligibility != nil && !s.eligibility.CanReply(conversation, message, aiAgent) {
		return nil
	}
	channel := svc.ChannelService.Get(conversation.ChannelID)
	if !IsAIAgentRolloutEligible(conversation, aiAgent, channel) {
		return nil
	}
	// 正式回复生成前先发送占位提示，生成后由回复提交环节按渠道能力原地替换或追加新消息
	svc.MessageService.SendAIReplyPlaceholder(&conversation, &aiAgent, &message, channel, message.RequestID)
	if pendingInterrupt := svc.ConversationInterruptService.FindLatestPendingByConversationID(conversation.ID); pendingInterrupt != nil {
		replyCtx.PendingInterrupt = pendingInterrupt
		return s.resumePendingInterrupt(ctx, replyCtx)
	}
	return s.executeReply(ctx, replyCtx)
}

func (s *aiReplyService) resumePendingInterrupt(ctx context.Context, replyCtx aiReplyContext) error {
	return s.interrupts.ResumePendingInterrupt(ctx, s, replyCtx)
}

func (s *aiReplyService) executeReply(ctx context.Context, replyCtx aiReplyContext) error {
	summary, err := s.executor.Run(ctx, runtimeReplyRunInput{
		Conversation: replyCtx.Conversation,
		Message:      replyCtx.Message,
		AIAgent:      replyCtx.AIAgent,
	})
	replyCtx.setSummary(summary)
	if err != nil {
		return err
	}
	if summary != nil && summary.Interrupted {
		return s.interrupts.HandleInterruptedSummary(s, replyCtx, summary)
	}
	if summary != nil && summary.HandoffRequested {
		if _, err := svc.ConversationHumanDispatchService.HandoffByAIWithRequestID(
			replyCtx.Conversation.ID,
			replyCtx.AIAgent,
			summary.HandoffReason,
			replyCtx.Message.RequestID,
		); err != nil {
			return err
		}
		return nil
	}
	if summary != nil && strings.TrimSpace(summary.ReplyText) != "" {
		_, err := s.commit.CommitAIReply(replyCommitInput{
			Conversation:  replyCtx.Conversation,
			Message:       replyCtx.Message,
			AIAgent:       replyCtx.AIAgent,
			ReplyText:     summary.ReplyText,
			ClientPrefix:  "ai_reply",
			WorkflowRunID: summary.WorkflowRunID,
		})
		if err != nil {
			return err
		}
		return nil
	}
	// 工作流执行完成但未产出回复文本，补发失败提示避免占位消息悬挂
	svc.MessageService.CompleteAIReplyPlaceholderAsFailed(&replyCtx.Conversation, &replyCtx.AIAgent, &replyCtx.Message, replyCtx.Message.RequestID)
	return nil
}
