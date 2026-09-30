package runtime

import (
	"testing"
	"time"

	applicationruntime "agent-desk/internal/ai/application/runtime"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
)

func TestReplyEligibilityCanReply(t *testing.T) {
	eligibility := newReplyEligibility()
	conversation := newConversationFixture()
	message := newCustomerMessageFixture("hello")
	aiAgent := newAIAgentFixture()

	if !eligibility.CanReply(conversation, message, aiAgent) {
		t.Fatalf("expected customer message to be replyable")
	}

	message.SenderType = enums.IMSenderTypeAgent
	if eligibility.CanReply(conversation, message, aiAgent) {
		t.Fatalf("expected non-customer message to be rejected")
	}

	message = newCustomerMessageFixture("hello")
	conversation.HandoffAt = ptrTime(time.Now())
	if eligibility.CanReply(conversation, message, aiAgent) {
		t.Fatalf("expected handed-off conversation to be rejected")
	}

	conversation = newConversationFixture()
	conversation.CurrentAssigneeID = 1
	if eligibility.CanReply(conversation, message, aiAgent) {
		t.Fatalf("expected assigned conversation to be rejected")
	}

	conversation = newConversationFixture()
	aiAgent.ServiceMode = enums.IMConversationServiceModeHumanOnly
	if eligibility.CanReply(conversation, message, aiAgent) {
		t.Fatalf("expected human-only agent to be rejected")
	}

	aiAgent = newAIAgentFixture()
	message.Content = "   "
	if eligibility.CanReply(conversation, message, aiAgent) {
		t.Fatalf("expected blank message to be rejected")
	}
}

func TestAIAgentRolloutUsesStableConversationBucket(t *testing.T) {
	conversation := models.Conversation{ID: 101, ChannelID: 7}
	agent := models.AIAgent{ID: 9, RolloutPercent: 50}
	first := IsAIAgentRolloutEligible(conversation, agent, &models.Channel{AIAgentRolloutPercent: 100})
	for range 20 {
		if got := IsAIAgentRolloutEligible(conversation, agent, &models.Channel{AIAgentRolloutPercent: 100}); got != first {
			t.Fatalf("rollout bucket changed within one conversation: first=%t got=%t", first, got)
		}
	}
	if normalizedRolloutPercent(0) != 100 || normalizedRolloutPercent(101) != 100 || normalizedRolloutPercent(25) != 25 {
		t.Fatal("unexpected rollout percent normalization")
	}
	if !IsAIAgentRolloutEligible(conversation, models.AIAgent{ID: 9, RolloutPercent: 0}, &models.Channel{}) {
		t.Fatal("legacy zero rollout values must preserve full rollout")
	}
}

func TestResolveReplyTimeout(t *testing.T) {
	service := newAIReplyService()
	aiAgent := newAIAgentFixture()

	if got := service.resolveReplyTimeout(nil, aiAgent); got != 120*time.Second {
		t.Fatalf("expected default timeout, got %v", got)
	}

	aiAgent.ReplyTimeoutSeconds = 30
	if got := service.resolveReplyTimeout(nil, aiAgent); got != 30*time.Second {
		t.Fatalf("expected agent timeout, got %v", got)
	}

	aiAgent.ReplyTimeoutSeconds = 999
	if got := service.resolveReplyTimeout(nil, aiAgent); got != 600*time.Second {
		t.Fatalf("expected clamped timeout, got %v", got)
	}

	// 渠道配置优先于智能体配置
	channel := &models.Channel{AIReplyTimeoutSeconds: 45}
	if got := service.resolveReplyTimeout(channel, aiAgent); got != 45*time.Second {
		t.Fatalf("expected channel timeout to take precedence, got %v", got)
	}
	channel.AIReplyTimeoutSeconds = 0
	if got := service.resolveReplyTimeout(channel, models.AIAgent{}); got != 120*time.Second {
		t.Fatalf("expected default timeout when channel and agent unset, got %v", got)
	}
}

func TestResolveInterruptPrompt(t *testing.T) {
	summary := &applicationruntime.RunResult{
		Interrupts: []applicationruntime.InterruptContextSummary{
			{
				ID:          "interrupt-1",
				Type:        "question",
				InfoPreview: `{"message":"请补充订单号"}`,
			},
		},
	}
	if got := resolveInterruptPrompt(summary); got != "请补充订单号" {
		t.Fatalf("unexpected interrupt prompt: %q", got)
	}

	summary.Interrupts[0].InfoPreview = "直接补充手机号"
	if got := resolveInterruptPrompt(summary); got != "直接补充手机号" {
		t.Fatalf("unexpected raw interrupt prompt: %q", got)
	}

	summary.ReplyText = ""
	summary.Interrupts[0] = applicationruntime.InterruptContextSummary{
		ID:          "system/server_time",
		Type:        "tool_confirmation",
		DisplayName: "获取当前时间",
		InfoPreview: "system/server_time",
	}
	if got := resolveInterruptPrompt(summary); got != "即将执行“获取当前时间”，是否确认继续？" {
		t.Fatalf("tool code leaked into customer prompt: %q", got)
	}

	summary.Interrupts[0].PromptText = "请确认是否更新客户资料。"
	if got := resolveInterruptPrompt(summary); got != "请确认是否更新客户资料。" {
		t.Fatalf("explicit customer prompt was not preferred: %q", got)
	}
}

func newConversationFixture() models.Conversation {
	return models.Conversation{}
}

func newCustomerMessageFixture(content string) models.Message {
	return models.Message{
		SenderType: enums.IMSenderTypeCustomer,
		Content:    content,
	}
}

func newAIAgentFixture() models.AIAgent {
	return models.AIAgent{}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
