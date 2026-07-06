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
	"agent-desk/internal/repositories"
	svc "agent-desk/internal/services"

	"github.com/mlogclub/simple/sqls"
)

func (s *aiReplyService) resolveReplyTimeout(aiAgent models.AIAgent) time.Duration {
	if aiAgent.ReplyTimeoutSeconds <= 0 {
		return time.Duration(defaultAIReplyAsyncTimeoutSeconds) * time.Second
	}
	if aiAgent.ReplyTimeoutSeconds > maxAIReplyAsyncTimeoutSeconds {
		return time.Duration(maxAIReplyAsyncTimeoutSeconds) * time.Second
	}
	return time.Duration(aiAgent.ReplyTimeoutSeconds) * time.Second
}

func (s *aiReplyService) TriggerReplyAsync(conversation models.Conversation, message models.Message) {
	go func() {
		aiAgent := svc.AIAgentService.Get(conversation.AIAgentID)
		if aiAgent == nil || aiAgent.Status != enums.StatusOk {
			return
		}
		startedAt := time.Now()
		timeout := s.resolveReplyTimeout(*aiAgent)
		ctx, cancel := context.WithTimeout(tracex.ContextWithRequestID(context.Background(), message.RequestID), timeout)
		defer cancel()
		if err := s.TriggerReply(ctx, conversation, message, *aiAgent); err != nil {
			slog.Error("failed to trigger ai reply",
				"requestId", message.RequestID,
				"message_id", message.ID,
				"timeout_ms", timeout.Milliseconds(),
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
				"error", err)
			if fallbackErr := s.sendFailureFallback(conversation, message, *aiAgent); fallbackErr != nil {
				slog.Error("failed to send ai failure fallback",
					"requestId", message.RequestID,
					"message_id", message.ID,
					"error", fallbackErr)
			}
		}
	}()
}

func (s *aiReplyService) TriggerReply(ctx context.Context, conversation models.Conversation, message models.Message, aiAgent models.AIAgent) (retErr error) {
	var summary *applicationruntime.Summary
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
		if s.shouldSendHandoffHoldingReply(conversation, message, aiAgent) {
			return s.sendHandoffHoldingReply(conversation, message, aiAgent)
		}
		return nil
	}
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
	}
	return nil
}

func (s *aiReplyService) sendFailureFallback(conversation models.Conversation, message models.Message, aiAgent models.AIAgent) error {
	if hasReplyAfterMessage(conversation.ID, message.ID) {
		return nil
	}
	replyText := buildAIReplyFailureFallback(message.Content, aiAgent.FallbackMessage)
	_, err := s.commit.SendAIReply(replyCommitInput{
		Conversation: conversation,
		Message:      message,
		AIAgent:      aiAgent,
		ReplyText:    replyText,
		ClientPrefix: "ai_fallback",
	})
	return err
}

func (s *aiReplyService) shouldSendHandoffHoldingReply(conversation models.Conversation, message models.Message, aiAgent models.AIAgent) bool {
	if message.SenderType != enums.IMSenderTypeCustomer || strings.TrimSpace(message.Content) == "" {
		return false
	}
	if aiAgent.ServiceMode == enums.IMConversationServiceModeHumanOnly {
		return false
	}
	current := repositories.ConversationRepository.Get(sqls.DB(), conversation.ID)
	if current == nil {
		current = &conversation
	}
	if current.Status == enums.IMConversationStatusClosed {
		return false
	}
	if current.CurrentAssigneeID > 0 && conversationHasAgentReplyAfterHandoff(current.ID, current.HandoffAt) {
		return false
	}
	if current.HandoffAt == nil && current.Status != enums.IMConversationStatusPending && current.Status != enums.IMConversationStatusActive {
		return false
	}
	return containsAny(strings.ToLower(message.Content),
		"人工", "真人", "客服", "售后", "投诉", "异响", "咯吱", "退", "赔", "上门", "联系", "电话", "手机号", "订单", "购买时间", "没人处理", "没人联系",
		"提供什么", "补充什么", "还要", "材料", "型号", "位置", "视频",
		"怎么跟进", "怎么处理", "接下来", "确认下", "确认一下", "流程",
	)
}

func conversationHasAgentReplyAfterHandoff(conversationID int64, handoffAt *time.Time) bool {
	if conversationID <= 0 || handoffAt == nil {
		return false
	}
	item := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Where("conversation_id = ?", conversationID).
		Where("sender_type = ?", enums.IMSenderTypeAgent).
		Where("created_at >= ?", *handoffAt).
		Asc("id"))
	return item != nil
}

func (s *aiReplyService) sendHandoffHoldingReply(conversation models.Conversation, message models.Message, aiAgent models.AIAgent) error {
	replyText := buildHandoffHoldingReply(message.Content, conversation.ID)
	_, err := s.commit.SendAIReply(replyCommitInput{
		Conversation: conversation,
		Message:      message,
		AIAgent:      aiAgent,
		ReplyText:    replyText,
		ClientPrefix: "ai_handoff_hold",
	})
	return err
}

func buildHandoffHoldingReply(customerContent string, conversationID int64) string {
	content := strings.TrimSpace(customerContent)
	hasPhone := normalizeFallbackPhone(strings.ToLower(content)) != "" || customerHistoryHasPhone(conversationID)
	switch {
	case containsAny(strings.ToLower(content), "退", "赔", "保证", "一定"):
		prefix := "我已收到您关于退换/赔付的诉求"
		if hasPhone {
			prefix = "我已收到您补充的联系方式和退换/赔付诉求"
		}
		return prefix + "，会继续转给人工/售后顾问确认。这里不能先承诺一定退换或赔付，后续需要结合订单条款、产品型号和售后检测结果判断。"
	case containsAny(strings.ToLower(content), "上门", "今天", "什么时候", "多久", "没人处理", "没人联系", "投诉"):
		prefix := "我已收到您的催促和投诉诉求"
		if hasPhone {
			prefix = "我已收到您的联系方式、催促和投诉诉求"
		}
		return prefix + "，会继续转给人工/售后顾问处理。具体联系时间、上门方式和处理结论需要以售后排班及订单信息确认为准；您也可以继续补充订单号、型号、异响位置或视频情况。"
	case containsAny(strings.ToLower(content), "怎么跟进", "怎么处理", "接下来", "确认下", "确认一下", "流程"):
		return "接下来我会把您已补充的联系方式、购买时间和异响诉求继续转给人工/售后顾问；售后会结合订单、型号、异响位置和必要的视频或检测情况确认处理方式。这里不先承诺上门时间、退换或赔付结论，最终以售后确认结果为准。"
	case containsAny(strings.ToLower(content), "提供什么", "补充什么", "还要", "材料"):
		return "可以继续补充购买时间、订单号、产品型号、异响位置，以及是否方便提供一段翻身异响的视频；这些信息会帮助售后顾问更快判断检测方式。处理结论仍以订单信息和售后检测为准。"
	case hasPhone:
		return "我已收到您补充的联系方式和售后信息，会继续转给人工/售后顾问确认。异响原因、是否上门、退换或赔付，都需要结合订单、产品型号、床架/排骨架和检测结果判断，我不会在这里先替结果下结论。"
	default:
		return "我已收到您的补充信息，当前会继续转给人工/售后顾问确认。为了方便后续处理，可以补充手机号、购买时间、订单号、产品型号和异响位置；退换、赔付或上门时间需要以售后确认结果为准。"
	}
}

func hasReplyAfterMessage(conversationID int64, messageID int64) bool {
	if conversationID <= 0 || messageID <= 0 {
		return false
	}
	item := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Where("conversation_id = ?", conversationID).
		Where("id > ?", messageID).
		Where("sender_type <> ?", enums.IMSenderTypeCustomer).
		Asc("id"))
	return item != nil
}

func buildAIReplyFailureFallback(customerContent string, configuredFallback string) string {
	content := strings.ToLower(strings.TrimSpace(customerContent))
	hasPhone := normalizeFallbackPhone(content) != ""
	switch {
	case containsAny(content, "人工", "真人", "客服") && hasPhone:
		return "收到，你留下的联系方式我已经记录。我会按人工/门店顾问跟进处理；涉及最终价格、库存、退换或售后结论，需要顾问结合订单和门店信息确认。"
	case containsAny(content, "治好", "治疗", "腰疼", "腰痛", "医生", "疾病"):
		return "我先给你一个稳妥答复：床垫不能替代医疗诊断或治疗，也不能保证治好腰疼；如果持续疼痛，建议先咨询医生。睡眠支撑方面可以到店试躺，我也可以安排门店顾问进一步确认适合的护脊款式。"
	case containsAny(content, "最低", "便宜", "成交价", "保证退", "退货", "退款", "退换"):
		return "关于最终价格、额外优惠和退换货政策，我不能给出未经门店确认的承诺；这些需要以门店顾问确认和购买合同为准。你可以留下手机号或微信，我会安排顾问跟进确认。"
	case containsAny(content, "库存", "现货", "有货"):
		return "库存和现货会实时变化，我不能直接承诺一定有货；建议留下联系方式或到店前让门店顾问确认规格和库存。"
	case containsAny(content, "售后", "投诉", "异响", "质保", "不满意", "差评"):
		if hasPhone {
			return "你反馈的售后/投诉诉求和联系方式我已经记录，会转给人工顾问继续确认。异响、退换或赔付需要结合订单、产品型号和售后检查/检测结果判断，我不会在这里先替结果下结论。"
		}
		return "你反馈的售后/投诉诉求我已经记录，会转给人工顾问继续确认。异响、退换或赔付需要结合订单、产品型号和售后检查/检测结果判断；可以补充购买时间、型号、异响位置和联系方式。"
	case containsAny(content, "写诗", "闲聊", "聊天"):
		return "可以呀，短短来一句：好睡像云落在肩上，醒来把疲惫放下。不过我更擅长慕斯寝具、床垫、电动床等产品和预约咨询，你想随便看看还是有睡眠困扰？"
	}
	if fallback := strings.TrimSpace(configuredFallback); fallback != "" {
		return fallback
	}
	return "你的问题我已经记录。我会先按门店知识继续为你确认；如果涉及价格、库存、售后或最终承诺，建议留下联系方式让顾问接着跟进。"
}

func containsAny(value string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func normalizeFallbackPhone(value string) string {
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return r < '0' || r > '9'
	}) {
		if len(field) == 11 && strings.HasPrefix(field, "1") {
			return field
		}
	}
	return ""
}
