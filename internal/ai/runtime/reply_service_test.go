package runtime

import (
	"strings"
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

func TestResolveReplyTimeout(t *testing.T) {
	service := newAIReplyService()
	aiAgent := newAIAgentFixture()

	if got := service.resolveReplyTimeout(aiAgent); got != 180*time.Second {
		t.Fatalf("expected default timeout, got %v", got)
	}

	aiAgent.ReplyTimeoutSeconds = 30
	if got := service.resolveReplyTimeout(aiAgent); got != 30*time.Second {
		t.Fatalf("expected exact timeout, got %v", got)
	}

	aiAgent.ReplyTimeoutSeconds = 999
	if got := service.resolveReplyTimeout(aiAgent); got != 600*time.Second {
		t.Fatalf("expected clamped timeout, got %v", got)
	}
}

func TestBuildAIReplyFailureFallbackUsesCommercialGuardrails(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAll []string
	}{
		{
			name:    "medical claim",
			input:   "能不能保证治好腰疼？",
			wantAll: []string{"不能", "治疗", "医生", "试躺", "顾问"},
		},
		{
			name:    "final price and return promise",
			input:   "这款最低多少钱？不合适能不能保证退？",
			wantAll: []string{"价格", "退换货", "不能", "顾问", "确认"},
		},
		{
			name:    "inventory",
			input:   "这款今天有没有现货？",
			wantAll: []string{"库存", "现货", "不能", "顾问", "确认"},
		},
		{
			name:    "after sales",
			input:   "我之前买的床垫有异响怎么办？",
			wantAll: []string{"售后", "异响", "人工", "检查", "顾问"},
		},
		{
			name:    "off topic",
			input:   "你会写诗吗？",
			wantAll: []string{"可以", "睡眠", "床垫", "产品"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAIReplyFailureFallback(tt.input, "默认兜底")
			if strings.Contains(got, "响应有点慢") {
				t.Fatalf("fallback should not expose slow-response wording:\n%s", got)
			}
			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Fatalf("fallback missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestBuildAIReplyFailureFallbackAcknowledgesPhoneForHumanHandoff(t *testing.T) {
	got := buildAIReplyFailureFallback("订单找不到，电话是 13900001111，你让人工联系我。", "默认兜底")
	for _, want := range []string{"联系方式", "已经记录", "人工", "顾问"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fallback missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "留下手机号") || strings.Contains(got, "响应有点慢") {
		t.Fatalf("fallback should not ask for phone again or expose slow-response wording:\n%s", got)
	}
}

func TestHandoffHoldingReplyAcknowledgesAfterSalesDetails(t *testing.T) {
	got := buildHandoffHoldingReply("手机号是 13800138000，购买时间是2026年6月15日，能不能退赔？", 0)
	for _, want := range []string{"联系方式", "退换/赔付", "售后", "订单条款", "检测结果"} {
		if !strings.Contains(got, want) {
			t.Fatalf("holding reply missing %q:\n%s", want, got)
		}
	}
	for _, banned := range []string{"会一定退", "会一定赔", "马上上门", "今天联系"} {
		if strings.Contains(got, banned) {
			t.Fatalf("holding reply contains risky wording %q:\n%s", banned, got)
		}
	}
}

func TestShouldSendHandoffHoldingReplyForUnassignedAfterSalesConversation(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	now := time.Now()
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)
	if err := db.Model(&models.Conversation{}).Where("id = ?", conversation.ID).Updates(map[string]any{
		"status":              enums.IMConversationStatusPending,
		"handoff_at":          now,
		"current_assignee_id": int64(0),
	}).Error; err != nil {
		t.Fatalf("update conversation: %v", err)
	}
	conversation.Status = enums.IMConversationStatusPending
	conversation.HandoffAt = &now

	message := newCustomerMessageFixture("我已经给过电话了，没人处理我就投诉，能不能退赔？")
	if !newAIReplyService().shouldSendHandoffHoldingReply(*conversation, message, *aiAgent) {
		t.Fatalf("expected after-sales message in pending handoff conversation to get AI holding reply")
	}

	conversation.CurrentAssigneeID = 99
	if err := db.Model(&models.Conversation{}).Where("id = ?", conversation.ID).Update("current_assignee_id", int64(99)).Error; err != nil {
		t.Fatalf("assign conversation: %v", err)
	}
	if !newAIReplyService().shouldSendHandoffHoldingReply(*conversation, message, *aiAgent) {
		t.Fatalf("expected assigned but unreplied human conversation to keep AI holding reply")
	}
	if err := db.Create(&models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeAgent,
		MessageType:    enums.IMMessageTypeText,
		Content:        "您好，我是人工客服，正在查看。",
		AuditFields: models.AuditFields{
			CreatedAt: now.Add(time.Second),
			UpdatedAt: now.Add(time.Second),
		},
	}).Error; err != nil {
		t.Fatalf("create agent message: %v", err)
	}
	if newAIReplyService().shouldSendHandoffHoldingReply(*conversation, message, *aiAgent) {
		t.Fatalf("expected AI to stay silent after human agent replied")
	}
}

func TestBuildAIReplyFailureFallbackUsesConfiguredDefault(t *testing.T) {
	got := buildAIReplyFailureFallback("我想了解一下", "请留下联系方式，顾问稍后跟进。")
	if got != "请留下联系方式，顾问稍后跟进。" {
		t.Fatalf("unexpected fallback: %q", got)
	}
}

func TestResolveInterruptPrompt(t *testing.T) {
	summary := &applicationruntime.Summary{
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
