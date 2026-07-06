package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/ai/workflow/dsl"
	workflowregistry "agent-desk/internal/ai/workflow/registry"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/services"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func mustMarshalWorkflowTestConfig(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func TestExecutorRoutesByConditionNodeBranch(t *testing.T) {
	executor := NewExecutor()
	result, err := executor.Execute(context.Background(), Input{
		Definition: conditionalReplyDefinition(),
		UserMessage: models.Message{
			Content: "vip",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if result.ReplyText != "VIP reply" {
		t.Fatalf("unexpected reply: %q", result.ReplyText)
	}
	assertPath(t, result.NodePath, []string{"start_1", "condition_1", "vip_reply", "send_vip", "end_1"})
}

func TestExecutorConditionNodeTraceExplainsMatchedEdge(t *testing.T) {
	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition: conditionalReplyDefinition(),
		UserMessage: models.Message{
			Content: "vip",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}

	trace := findNodeTrace(result.NodeTraces, "condition_1")
	if trace == nil {
		t.Fatalf("expected condition node trace, got %#v", result.NodeTraces)
	}
	for _, want := range []string{
		`"selectedEdgeId":"edge_condition_vip"`,
		`"selectedBranchId":"vip"`,
		`"selectedTargetNodeId":"vip_reply"`,
		`"operator":"eq"`,
		`"leftValue":"vip"`,
		`"matched":true`,
	} {
		if !strings.Contains(trace.OutputPreview, want) {
			t.Fatalf("expected condition trace output to contain %s, got %s", want, trace.OutputPreview)
		}
	}
}

func TestExecutorConditionNodeTraceExplainsDefaultEdge(t *testing.T) {
	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition: conditionalReplyDefinition(),
		UserMessage: models.Message{
			Content: "normal",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}

	trace := findNodeTrace(result.NodeTraces, "condition_1")
	if trace == nil {
		t.Fatalf("expected condition node trace, got %#v", result.NodeTraces)
	}
	for _, want := range []string{
		`"selectedEdgeId":"edge_condition_default"`,
		`"selectedBranchId":"default"`,
		`"selectedTargetNodeId":"normal_reply"`,
		`"reason":"no condition branch matched; selected default branch"`,
		`"leftValue":"normal"`,
		`"matched":false`,
	} {
		if !strings.Contains(trace.OutputPreview, want) {
			t.Fatalf("expected condition trace output to contain %s, got %s", want, trace.OutputPreview)
		}
	}
}

func TestExecutorUsesDefaultEdgeWhenConditionDoesNotMatch(t *testing.T) {
	executor := NewExecutor()
	result, err := executor.Execute(context.Background(), Input{
		Definition: conditionalReplyDefinition(),
		UserMessage: models.Message{
			Content: "normal",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if result.ReplyText != "Normal reply" {
		t.Fatalf("unexpected reply: %q", result.ReplyText)
	}
	assertPath(t, result.NodePath, []string{"start_1", "condition_1", "normal_reply", "send_normal", "end_1"})
}

func TestExecutorHandoffToHumanRunsRealDispatchAction(t *testing.T) {
	db := setupWorkflowExecutorHandoffDB(t)
	aiAgent := createWorkflowExecutorHandoffAIAgent(t, db, "1")
	createWorkflowExecutorHandoffTeam(t, db, 1, "售后支持组")
	createWorkflowExecutorHandoffActiveSchedule(t, db, 1)
	createWorkflowExecutorHandoffAgentProfile(t, db, 101, 1)
	conversation := createWorkflowExecutorHandoffConversation(t, db, aiAgent.ID)
	userMessage := createWorkflowExecutorCustomerMessage(t, db, conversation.ID, "需要人工处理")

	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition:   handoffWorkflowDefinition(),
		Conversation: conversation,
		UserMessage:  userMessage,
		AIAgent:      aiAgent,
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if strings.TrimSpace(result.ReplyText) != "" {
		t.Fatalf("expected workflow handoff node to avoid duplicate reply text, got %q", result.ReplyText)
	}
	assertPath(t, result.NodePath, []string{"start_1", "handoff_1", "handoff_route_1", "assigned_end"})

	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusActive {
		t.Fatalf("expected active conversation, got status=%d", current.Status)
	}
	if current.CurrentAssigneeID != 101 || current.CurrentTeamID != 1 {
		t.Fatalf("unexpected assignment: assignee=%d team=%d", current.CurrentAssigneeID, current.CurrentTeamID)
	}
	if current.HandoffAt == nil || current.HandoffReason != "需要人工处理" {
		t.Fatalf("expected handoff metadata, got at=%v reason=%q", current.HandoffAt, current.HandoffReason)
	}

	notice := services.MessageService.FindOne(sqls.NewCnd().Eq("conversation_id", conversation.ID).Eq("sender_type", enums.IMSenderTypeAI).Desc("id"))
	if notice == nil || strings.TrimSpace(notice.Content) == "" {
		t.Fatalf("expected handoff service to send ai notice, got %+v", notice)
	}
}

func TestExecutorResumeSkipsHandoffWhenConfirmationCancelled(t *testing.T) {
	db := setupWorkflowExecutorHandoffDB(t)
	aiAgent := createWorkflowExecutorHandoffAIAgent(t, db, "1")
	createWorkflowExecutorHandoffTeam(t, db, 1, "售后支持组")
	createWorkflowExecutorHandoffActiveSchedule(t, db, 1)
	createWorkflowExecutorHandoffAgentProfile(t, db, 101, 1)
	conversation := createWorkflowExecutorHandoffConversation(t, db, aiAgent.ID)
	userMessage := createWorkflowExecutorCustomerMessage(t, db, conversation.ID, "需要人工处理")
	input := Input{
		Definition:   handoffAfterConfirmationWorkflowDefinition(),
		Conversation: conversation,
		UserMessage:  userMessage,
		AIAgent:      aiAgent,
	}

	interrupted, err := NewExecutor().Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if !interrupted.Interrupted {
		t.Fatalf("expected workflow to interrupt before handoff")
	}

	result, err := NewExecutor().Resume(context.Background(), input, interrupted.CheckPointData, "取消")
	if err != nil {
		t.Fatalf("resume workflow: %v", err)
	}
	if result.Interrupted {
		t.Fatalf("expected cancelled resume to complete")
	}
	assertPath(t, result.NodePath, []string{"handoff_1", "end_1"})

	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusAIServing {
		t.Fatalf("expected conversation to remain ai serving, got status=%d", current.Status)
	}
	if current.CurrentAssigneeID != 0 || current.CurrentTeamID != 0 || current.HandoffAt != nil {
		t.Fatalf("expected no handoff side effect, got assignee=%d team=%d handoffAt=%v", current.CurrentAssigneeID, current.CurrentTeamID, current.HandoffAt)
	}
	if count := services.MessageService.Count(sqls.NewCnd().Eq("conversation_id", conversation.ID).Eq("sender_type", enums.IMSenderTypeAI)); count != 0 {
		t.Fatalf("expected no handoff notice message, got %d", count)
	}
}

func TestExecutorAnalyzeConversationOutputsBranchVariables(t *testing.T) {
	db := setupWorkflowExecutorHandoffDB(t)
	aiAgent := createWorkflowExecutorHandoffAIAgent(t, db, "1")
	conversation := createWorkflowExecutorHandoffConversation(t, db, aiAgent.ID)
	userMessage := createWorkflowExecutorCustomerMessage(t, db, conversation.ID, "你们重复扣费了，我要投诉并转人工")

	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition:   analyzeConversationWorkflowDefinition(),
		Conversation: conversation,
		UserMessage:  userMessage,
		AIAgent:      aiAgent,
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	assertPath(t, result.NodePath, []string{"start_1", "analyze_1", "analyze_route_1", "handoff_end"})
}

func TestExecutorPrepareTicketDraftOutputsDraftVariable(t *testing.T) {
	db := setupWorkflowExecutorHandoffDB(t)
	aiAgent := createWorkflowExecutorHandoffAIAgent(t, db, "1")
	conversation := createWorkflowExecutorHandoffConversation(t, db, aiAgent.ID)
	userMessage := createWorkflowExecutorCustomerMessage(t, db, conversation.ID, "订单支付失败，请帮我登记工单")

	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition:   prepareTicketDraftWorkflowDefinition(),
		Conversation: conversation,
		UserMessage:  userMessage,
		AIAgent:      aiAgent,
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	assertPath(t, result.NodePath, []string{"start_1", "draft_1", "draft_route_1", "ready_end"})
}

func TestExecutorPolicyFirstWorkflowRoutesGreetingToDirectReply(t *testing.T) {
	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition: policyFirstWorkflowDefinition(),
		UserMessage: models.Message{
			Content: "<p>你好。</p>",
		},
		AIAgent: models.AIAgent{
			KnowledgeIDs:    "1",
			FallbackMessage: "我暂时没有找到足够准确的信息。",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if result.ReplyText != "您好，请问有什么可以帮您？" {
		t.Fatalf("expected greeting reply, got %q", result.ReplyText)
	}
	if result.RetrieverCount != 0 {
		t.Fatalf("expected greeting to skip retrieval, got retriever count %d", result.RetrieverCount)
	}
	assertPath(t, result.NodePath, []string{"start_1", "understanding_1", "policy_1", "policy_route_1", "send_direct_1", "end_1"})

	understandingTrace := findNodeTrace(result.NodeTraces, "understanding_1")
	if understandingTrace == nil || !strings.Contains(understandingTrace.OutputPreview, `"messageIntent":"greeting"`) || !strings.Contains(understandingTrace.OutputPreview, `"answerScope":"direct_reply"`) {
		t.Fatalf("expected understanding trace to audit greeting/direct_reply, got %#v", understandingTrace)
	}
	policyTrace := findNodeTrace(result.NodeTraces, "policy_1")
	if policyTrace == nil || !strings.Contains(policyTrace.OutputPreview, `"action":"direct_reply"`) || !strings.Contains(policyTrace.OutputPreview, `"finalReplySource":"direct_reply"`) {
		t.Fatalf("expected policy trace to audit direct reply, got %#v", policyTrace)
	}
}

func TestExecutorPolicyFirstWorkflowRoutesBusinessQuestionToKnowledge(t *testing.T) {
	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition: policyFirstWorkflowDefinition(),
		UserMessage: models.Message{
			Content: "老人腰不好，床垫是不是越硬越好？预算一万五推荐哪种？",
		},
		AIAgent: models.AIAgent{
			KnowledgeIDs:    "1",
			FallbackMessage: "我暂时没有找到足够准确的信息。",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	assertPath(t, result.NodePath, []string{"start_1", "understanding_1", "policy_1", "policy_route_1", "retrieve_end"})
}

func TestUnderstandConversationMessageDoesNotTreatRiskQuestionAsConfirmation(t *testing.T) {
	got := understandConversationMessage("能不能保证治好腰疼？")
	if got.MessageIntent != "business_question" || got.AnswerScope != "needs_knowledge" {
		t.Fatalf("expected risk question to need knowledge, got %#v", got)
	}
}

func TestUnderstandConversationMessageRoutesShortRetailNeedToKnowledge(t *testing.T) {
	tests := []string{
		"我腰疼",
		"太软",
		"老人起夜",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			got := understandConversationMessage(tt)
			if got.MessageIntent != "business_question" || got.AnswerScope != "needs_knowledge" {
				t.Fatalf("expected retail need to need knowledge, got %#v", got)
			}
		})
	}
}

func TestWorkflowClarificationReplyUsesRetailLanguage(t *testing.T) {
	got := decideWorkflowReplyPolicy(models.AIAgent{}, workflowReplyPolicyInput{
		MessageIntent: "ambiguous_question",
		AnswerScope:   "needs_clarification",
	})
	if !strings.Contains(got.ReplyText, "睡眠困扰") || strings.Contains(got.ReplyText, "报错信息") {
		t.Fatalf("expected retail clarification reply, got %q", got.ReplyText)
	}
}

func TestUnderstandConversationMessageRoutesIdentityQuestionToDirectReply(t *testing.T) {
	got := understandConversationMessage("你是谁")
	if got.MessageIntent != "identity" || got.AnswerScope != "direct_reply" {
		t.Fatalf("expected identity direct reply, got %#v", got)
	}
}

func TestWorkflowIdentityReplyAnswersRole(t *testing.T) {
	got := decideWorkflowReplyPolicy(models.AIAgent{Name: "慕小眠"}, workflowReplyPolicyInput{
		MessageIntent: "identity",
		AnswerScope:   "direct_reply",
	})
	if !strings.Contains(got.ReplyText, "慕小眠") || !strings.Contains(got.ReplyText, "慕斯寝具") || !strings.Contains(got.ReplyText, "睡眠顾问") {
		t.Fatalf("expected concrete identity reply, got %q", got.ReplyText)
	}
}

func TestUnderstandConversationMessageStillRecognizesShortConfirmation(t *testing.T) {
	got := understandConversationMessage("好的")
	if got.MessageIntent != "confirmation" || got.AnswerScope != "direct_reply" {
		t.Fatalf("expected confirmation, got %#v", got)
	}
}

func TestExecutorLLMReplyUsesAgentFallbackWhenDeclaredKnowledgeIsEmpty(t *testing.T) {
	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition: emptyKnowledgeReplyDefinition(),
		UserMessage: models.Message{
			Content: "产品功能",
		},
		AIAgent: models.AIAgent{
			KnowledgeIDs:    "1",
			FallbackMode:    enums.AIAgentFallbackModeNoAnswer,
			FallbackMessage: "我暂时没有找到足够准确的信息。你可以补充更具体的问题，我再继续帮你查。",
			SystemPrompt:    "不要编造事实。",
		},
		AIConfig: models.AIConfig{
			ModelName: "should-not-be-called",
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if result.ReplyText != "我暂时没有找到足够准确的信息。你可以补充更具体的问题，我再继续帮你查。" {
		t.Fatalf("expected fallback reply, got %q", result.ReplyText)
	}
	assertPath(t, result.NodePath, []string{"start_1", "reply_1", "send_1", "end_1"})
}

func TestExecutorAnswerabilityAllowsDigitalStoreRuntimeContext(t *testing.T) {
	db := setupWorkflowExecutorDigitalStoreRuntimeDB(t)
	rawConfig := `{"brandName":"慕斯寝具","industry":"家居寝具","storeName":"徐汇体验店","aiManagerName":"慕小眠","initialized":true}`
	if err := db.Create(&models.SystemConfig{
		ConfigKey:   "digital_store.profile",
		ConfigValue: rawConfig,
		GroupCode:   "digital_store",
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create digital store config: %v", err)
	}
	state := &runState{
		vars: map[string]map[string]any{
			"retrieve_1": {"items": []map[string]any{}},
		},
	}
	node := dsl.Node{
		ID:   "answerability_1",
		Type: workflowregistry.NodeTypeAnswerabilityGate,
		Inputs: map[string]dsl.VariableSelector{
			"knowledgeItems": {NodeID: "retrieve_1", Field: "items"},
		},
	}

	if err := NewExecutor().executeAnswerabilityGate(state, node); err != nil {
		t.Fatalf("executeAnswerabilityGate() error = %v", err)
	}
	if got := state.vars["answerability_1"]["answerability"]; got != "answerable" {
		t.Fatalf("expected answerable with digital store runtime context, got %#v", got)
	}
	if got := state.vars["answerability_1"]["reason"]; got != "digital store runtime context is available" {
		t.Fatalf("unexpected reason: %#v", got)
	}
}

func TestExecutorHumanConfirmInterruptsWithCheckpoint(t *testing.T) {
	result, err := NewExecutor().Execute(context.Background(), Input{
		Definition: humanConfirmWorkflowDefinition(),
		Conversation: models.Conversation{
			ID: 11,
		},
		UserMessage: models.Message{
			ID:      22,
			Content: "创建工单",
		},
		AIAgent: models.AIAgent{
			ID: 33,
		},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if !result.Interrupted {
		t.Fatalf("expected workflow to interrupt")
	}
	if result.CheckPointID == "" {
		t.Fatalf("expected checkpoint id")
	}
	if len(result.Interrupts) != 1 {
		t.Fatalf("expected one interrupt, got %#v", result.Interrupts)
	}
	if result.Interrupts[0].Type != "human_confirm" || result.Interrupts[0].ID != "confirm_1" {
		t.Fatalf("unexpected interrupt summary: %#v", result.Interrupts[0])
	}
	if !strings.Contains(result.Interrupts[0].InfoPreview, "请确认创建工单") {
		t.Fatalf("expected confirmation prompt, got %q", result.Interrupts[0].InfoPreview)
	}
	assertPath(t, result.NodePath, []string{"start_1", "prompt_1", "confirm_1"})
}

func TestExecutorResumeHumanConfirmContinuesWithConfirmedVariable(t *testing.T) {
	executor := NewExecutor()
	input := Input{
		Definition: humanConfirmWorkflowDefinition(),
		Conversation: models.Conversation{
			ID: 11,
		},
		UserMessage: models.Message{
			ID:      22,
			Content: "创建工单",
		},
		AIAgent: models.AIAgent{
			ID: 33,
		},
	}
	interrupted, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	result, err := executor.Resume(context.Background(), input, interrupted.CheckPointData, "确认")
	if err != nil {
		t.Fatalf("resume workflow: %v", err)
	}
	if result.Interrupted {
		t.Fatalf("expected workflow resume to complete")
	}
	assertPath(t, result.NodePath, []string{"confirm_route_1", "end_1"})
}

func TestExecutorResumeCreatesTicketAfterHumanConfirmation(t *testing.T) {
	db := setupWorkflowExecutorHandoffDB(t)
	aiAgent := createWorkflowExecutorHandoffAIAgent(t, db, "1")
	conversation := createWorkflowExecutorHandoffConversation(t, db, aiAgent.ID)
	userMessage := createWorkflowExecutorCustomerMessage(t, db, conversation.ID, "订单支付失败，请帮我登记工单")
	executor := NewExecutor()

	interrupted, err := executor.Execute(context.Background(), Input{
		Definition:   createTicketWorkflowDefinition(),
		Conversation: conversation,
		UserMessage:  userMessage,
		AIAgent:      aiAgent,
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if !interrupted.Interrupted {
		t.Fatalf("expected workflow to interrupt before creating ticket")
	}

	result, err := executor.Resume(context.Background(), Input{
		Definition:   createTicketWorkflowDefinition(),
		Conversation: conversation,
		UserMessage:  userMessage,
		AIAgent:      aiAgent,
	}, interrupted.CheckPointData, "确认")
	if err != nil {
		t.Fatalf("resume workflow: %v", err)
	}
	if result.Interrupted {
		t.Fatalf("expected workflow to complete")
	}
	assertPath(t, result.NodePath, []string{"confirm_route_1", "create_ticket_1", "end_1"})

	var ticket models.Ticket
	if err := db.First(&ticket, "conversation_id = ?", conversation.ID).Error; err != nil {
		t.Fatalf("expected created ticket: %v", err)
	}
	if ticket.Title == "" || !strings.Contains(ticket.Description, "订单支付失败") {
		t.Fatalf("unexpected ticket: %+v", ticket)
	}

	trace := findNodeTrace(result.NodeTraces, "create_ticket_1")
	if trace == nil || !strings.Contains(trace.OutputPreview, "工单已创建") {
		t.Fatalf("expected create_ticket output to include customer-visible result message, got %#v", trace)
	}
}

func findNodeTrace(items []NodeTrace, nodeID string) *NodeTrace {
	for i := range items {
		if items[i].NodeID == nodeID {
			return &items[i]
		}
	}
	return nil
}

func emptyKnowledgeReplyDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "reply_1", Type: workflowregistry.NodeTypeLLMReply, Name: "Reply", Inputs: map[string]dsl.VariableSelector{
				"userMessage":    {NodeID: "start_1", Field: "userMessage"},
				"knowledgeItems": {NodeID: "missing_retrieve", Field: "items"},
			}},
			{ID: "send_1", Type: workflowregistry.NodeTypeSendReply, Name: "Send", Inputs: map[string]dsl.VariableSelector{
				"replyText": {NodeID: "reply_1", Field: "replyText"},
			}},
			{ID: "end_1", Type: workflowregistry.NodeTypeEnd, Name: "End"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_reply", Source: "start_1", Target: "reply_1"},
			{ID: "edge_reply_send", Source: "reply_1", Target: "send_1"},
			{ID: "edge_send_end", Source: "send_1", Target: "end_1"},
		},
	}
}

func policyFirstWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "understanding_1", Type: workflowregistry.NodeTypeConversationUnderstanding, Name: "Understanding", Inputs: map[string]dsl.VariableSelector{
				"userMessage": {NodeID: "start_1", Field: "userMessage"},
			}},
			{ID: "policy_1", Type: workflowregistry.NodeTypeReplyPolicy, Name: "Policy", Inputs: map[string]dsl.VariableSelector{
				"userMessage":    {NodeID: "start_1", Field: "userMessage"},
				"messageIntent":  {NodeID: "understanding_1", Field: "messageIntent"},
				"answerScope":    {NodeID: "understanding_1", Field: "answerScope"},
				"riskSignals":    {NodeID: "understanding_1", Field: "riskSignals"},
				"knowledgeItems": {NodeID: "retrieve_1", Field: "items"},
			}},
			{ID: "policy_route_1", Type: workflowregistry.NodeTypeCondition, Name: "Policy Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "direct", Name: "Direct", TargetNodeID: "send_direct_1", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "policy_1", Field: "action"},
					Operator: "eq",
					Right:    "direct_reply",
				}},
				{ID: "knowledge", Name: "Knowledge", TargetNodeID: "retrieve_end", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "policy_1", Field: "action"},
					Operator: "eq",
					Right:    "retrieve_knowledge",
				}},
				{ID: "default", Name: "Default", TargetNodeID: "end_1", Default: true},
			}})},
			{ID: "send_direct_1", Type: workflowregistry.NodeTypeSendReply, Name: "Send Direct", Inputs: map[string]dsl.VariableSelector{
				"replyText": {NodeID: "policy_1", Field: "replyText"},
			}},
			{ID: "retrieve_end", Type: workflowregistry.NodeTypeEnd, Name: "Retrieve"},
			{ID: "end_1", Type: workflowregistry.NodeTypeEnd, Name: "End"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_understanding", Source: "start_1", Target: "understanding_1"},
			{ID: "edge_understanding_policy", Source: "understanding_1", Target: "policy_1"},
			{ID: "edge_policy_route", Source: "policy_1", Target: "policy_route_1"},
			{ID: "edge_policy_direct", Source: "policy_route_1", Target: "send_direct_1"},
			{ID: "edge_policy_knowledge", Source: "policy_route_1", Target: "retrieve_end"},
			{ID: "edge_policy_default", Source: "policy_route_1", Target: "end_1"},
			{ID: "edge_send_direct_end", Source: "send_direct_1", Target: "end_1"},
		},
	}
}

func conditionalReplyDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "condition_1", Type: workflowregistry.NodeTypeCondition, Name: "Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "vip", Name: "VIP", TargetNodeID: "vip_reply", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "start_1", Field: "userMessage"},
					Operator: "eq",
					Right:    "vip",
				}},
				{ID: "default", Name: "Default", TargetNodeID: "normal_reply", Default: true},
			}})},
			{ID: "vip_reply", Type: workflowregistry.NodeTypeLLMReply, Name: "VIP", Config: []byte(`{"staticReply":"VIP reply"}`)},
			{ID: "normal_reply", Type: workflowregistry.NodeTypeLLMReply, Name: "Normal", Config: []byte(`{"staticReply":"Normal reply"}`)},
			{ID: "send_vip", Type: workflowregistry.NodeTypeSendReply, Name: "Send VIP", Inputs: map[string]dsl.VariableSelector{
				"replyText": {NodeID: "vip_reply", Field: "replyText"},
			}},
			{ID: "send_normal", Type: workflowregistry.NodeTypeSendReply, Name: "Send Normal", Inputs: map[string]dsl.VariableSelector{
				"replyText": {NodeID: "normal_reply", Field: "replyText"},
			}},
			{ID: "end_1", Type: workflowregistry.NodeTypeEnd, Name: "End"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_condition", Source: "start_1", Target: "condition_1"},
			{ID: "edge_condition_vip", Source: "condition_1", Target: "vip_reply"},
			{ID: "edge_condition_default", Source: "condition_1", Target: "normal_reply"},
			{ID: "edge_vip_send", Source: "vip_reply", Target: "send_vip"},
			{ID: "edge_normal_send", Source: "normal_reply", Target: "send_normal"},
			{ID: "edge_send_vip_end", Source: "send_vip", Target: "end_1"},
			{ID: "edge_send_normal_end", Source: "send_normal", Target: "end_1"},
		},
	}
}

func createTicketWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "draft_1", Type: workflowregistry.NodeTypePrepareTicketDraft, Name: "Draft", Inputs: map[string]dsl.VariableSelector{
				"issue": {NodeID: "start_1", Field: "userMessage"},
			}},
			{ID: "prompt_1", Type: workflowregistry.NodeTypeLLMReply, Name: "Prompt", Config: []byte(`{"staticReply":"请确认创建工单"}`)},
			{ID: "confirm_1", Type: workflowregistry.NodeTypeHumanConfirm, Name: "Confirm", Inputs: map[string]dsl.VariableSelector{
				"prompt": {NodeID: "prompt_1", Field: "replyText"},
			}},
			{ID: "confirm_route_1", Type: workflowregistry.NodeTypeCondition, Name: "Confirm Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "yes", Name: "Yes", TargetNodeID: "create_ticket_1", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "confirm_1", Field: "confirmed"},
					Operator: "is_true",
				}},
				{ID: "default", Name: "Cancel", TargetNodeID: "cancel_end", Default: true},
			}})},
			{ID: "create_ticket_1", Type: workflowregistry.NodeTypeCreateTicket, Name: "Create Ticket", Inputs: map[string]dsl.VariableSelector{
				"ticketDraft": {NodeID: "draft_1", Field: "ticketDraft"},
				"confirmed":   {NodeID: "confirm_1", Field: "confirmed"},
			}},
			{ID: "end_1", Type: workflowregistry.NodeTypeEnd, Name: "End"},
			{ID: "cancel_end", Type: workflowregistry.NodeTypeEnd, Name: "Cancel"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_draft", Source: "start_1", Target: "draft_1"},
			{ID: "edge_draft_prompt", Source: "draft_1", Target: "prompt_1"},
			{ID: "edge_prompt_confirm", Source: "prompt_1", Target: "confirm_1"},
			{ID: "edge_confirm_route", Source: "confirm_1", Target: "confirm_route_1"},
			{ID: "edge_confirm_create", Source: "confirm_route_1", Target: "create_ticket_1"},
			{ID: "edge_confirm_cancel", Source: "confirm_route_1", Target: "cancel_end"},
			{ID: "edge_create_end", Source: "create_ticket_1", Target: "end_1"},
		},
	}
}

func humanConfirmWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "prompt_1", Type: workflowregistry.NodeTypeLLMReply, Name: "Prompt", Config: []byte(`{"staticReply":"请确认创建工单"}`)},
			{ID: "confirm_1", Type: workflowregistry.NodeTypeHumanConfirm, Name: "Confirm", Inputs: map[string]dsl.VariableSelector{
				"prompt": {NodeID: "prompt_1", Field: "replyText"},
			}},
			{ID: "confirm_route_1", Type: workflowregistry.NodeTypeCondition, Name: "Confirm Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "yes", Name: "Yes", TargetNodeID: "end_1", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "confirm_1", Field: "confirmed"},
					Operator: "is_true",
				}},
				{ID: "default", Name: "Cancel", TargetNodeID: "cancel_end", Default: true},
			}})},
			{ID: "end_1", Type: workflowregistry.NodeTypeEnd, Name: "End"},
			{ID: "cancel_end", Type: workflowregistry.NodeTypeEnd, Name: "Cancel"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_prompt", Source: "start_1", Target: "prompt_1"},
			{ID: "edge_prompt_confirm", Source: "prompt_1", Target: "confirm_1"},
			{ID: "edge_confirm_route", Source: "confirm_1", Target: "confirm_route_1"},
			{ID: "edge_confirm_yes", Source: "confirm_route_1", Target: "end_1"},
			{ID: "edge_confirm_cancel", Source: "confirm_route_1", Target: "cancel_end"},
		},
	}
}

func prepareTicketDraftWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "draft_1", Type: workflowregistry.NodeTypePrepareTicketDraft, Name: "Draft", Inputs: map[string]dsl.VariableSelector{
				"issue": {NodeID: "start_1", Field: "userMessage"},
			}},
			{ID: "draft_route_1", Type: workflowregistry.NodeTypeCondition, Name: "Draft Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "ready", Name: "Ready", TargetNodeID: "ready_end", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "draft_1", Field: "ticketDraft"},
					Operator: "exists",
				}},
				{ID: "default", Name: "Default", TargetNodeID: "default_end", Default: true},
			}})},
			{ID: "ready_end", Type: workflowregistry.NodeTypeEnd, Name: "Ready"},
			{ID: "default_end", Type: workflowregistry.NodeTypeEnd, Name: "Default"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_draft", Source: "start_1", Target: "draft_1"},
			{ID: "edge_draft_route", Source: "draft_1", Target: "draft_route_1"},
			{ID: "edge_draft_ready", Source: "draft_route_1", Target: "ready_end"},
			{ID: "edge_draft_default", Source: "draft_route_1", Target: "default_end"},
		},
	}
}

func analyzeConversationWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "analyze_1", Type: workflowregistry.NodeTypeAnalyzeConversation, Name: "Analyze", Inputs: map[string]dsl.VariableSelector{
				"userMessage": {NodeID: "start_1", Field: "userMessage"},
			}},
			{ID: "analyze_route_1", Type: workflowregistry.NodeTypeCondition, Name: "Analyze Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "handoff", Name: "Handoff", TargetNodeID: "handoff_end", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "analyze_1", Field: "needHumanHandoff"},
					Operator: "is_true",
				}},
				{ID: "default", Name: "Default", TargetNodeID: "default_end", Default: true},
			}})},
			{ID: "handoff_end", Type: workflowregistry.NodeTypeEnd, Name: "Handoff"},
			{ID: "default_end", Type: workflowregistry.NodeTypeEnd, Name: "Default"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_analyze", Source: "start_1", Target: "analyze_1"},
			{ID: "edge_analyze_route", Source: "analyze_1", Target: "analyze_route_1"},
			{ID: "edge_analyze_handoff", Source: "analyze_route_1", Target: "handoff_end"},
			{ID: "edge_analyze_default", Source: "analyze_route_1", Target: "default_end"},
		},
	}
}

func handoffWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "handoff_1", Type: workflowregistry.NodeTypeHandoffToHuman, Name: "Handoff", Inputs: map[string]dsl.VariableSelector{
				"reason": {NodeID: "start_1", Field: "userMessage"},
			}},
			{ID: "handoff_route_1", Type: workflowregistry.NodeTypeCondition, Name: "Handoff Route", Config: mustMarshalWorkflowTestConfig(dsl.ConditionConfig{Branches: []dsl.ConditionBranch{
				{ID: "assigned", Name: "Assigned", TargetNodeID: "assigned_end", Condition: &dsl.Condition{
					Left:     &dsl.VariableSelector{NodeID: "handoff_1", Field: "decision"},
					Operator: "eq",
					Right:    string(services.HandoffDecisionAssigned),
				}},
				{ID: "default", Name: "Default", TargetNodeID: "default_end", Default: true},
			}})},
			{ID: "assigned_end", Type: workflowregistry.NodeTypeEnd, Name: "Assigned"},
			{ID: "default_end", Type: workflowregistry.NodeTypeEnd, Name: "Default"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_handoff", Source: "start_1", Target: "handoff_1"},
			{ID: "edge_handoff_route", Source: "handoff_1", Target: "handoff_route_1"},
			{ID: "edge_handoff_assigned", Source: "handoff_route_1", Target: "assigned_end"},
			{ID: "edge_handoff_default", Source: "handoff_route_1", Target: "default_end"},
		},
	}
}

func handoffAfterConfirmationWorkflowDefinition() dsl.Definition {
	return dsl.Definition{
		SchemaVersion: 1,
		EntryNodeID:   "start_1",
		Nodes: []dsl.Node{
			{ID: "start_1", Type: workflowregistry.NodeTypeStart, Name: "Start"},
			{ID: "prompt_1", Type: workflowregistry.NodeTypeLLMReply, Name: "Prompt", Config: []byte(`{"staticReply":"请确认转人工"}`)},
			{ID: "confirm_1", Type: workflowregistry.NodeTypeHumanConfirm, Name: "Confirm", Inputs: map[string]dsl.VariableSelector{
				"prompt": {NodeID: "prompt_1", Field: "replyText"},
			}},
			{ID: "handoff_1", Type: workflowregistry.NodeTypeHandoffToHuman, Name: "Handoff", Inputs: map[string]dsl.VariableSelector{
				"reason":    {NodeID: "start_1", Field: "userMessage"},
				"confirmed": {NodeID: "confirm_1", Field: "confirmed"},
			}},
			{ID: "end_1", Type: workflowregistry.NodeTypeEnd, Name: "End"},
		},
		Edges: []dsl.Edge{
			{ID: "edge_start_prompt", Source: "start_1", Target: "prompt_1"},
			{ID: "edge_prompt_confirm", Source: "prompt_1", Target: "confirm_1"},
			{ID: "edge_confirm_handoff", Source: "confirm_1", Target: "handoff_1"},
			{ID: "edge_handoff_end", Source: "handoff_1", Target: "end_1"},
		},
	}
}

func setupWorkflowExecutorHandoffDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.User{},
		&models.Customer{},
		&models.CustomerIdentity{},
		&models.AIAgent{},
		&models.AgentTeam{},
		&models.AgentTeamSchedule{},
		&models.AgentProfile{},
		&models.Channel{},
		&models.Conversation{},
		&models.ConversationAssignment{},
		&models.ConversationEventLog{},
		&models.ConversationReadState{},
		&models.Message{},
		&models.ChannelMessageOutbox{},
		&models.Ticket{},
		&models.TicketNoSequence{},
		&models.TicketTag{},
		&models.TicketProgress{},
	); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	sqls.SetDB(db)
	return db
}

func setupWorkflowExecutorDigitalStoreRuntimeDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.SystemConfig{}, &models.Product{}, &models.Promotion{}); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createWorkflowExecutorHandoffAIAgent(t *testing.T, db *gorm.DB, teamIDs string) models.AIAgent {
	t.Helper()
	item := models.AIAgent{
		Name:        "测试AI",
		ServiceMode: enums.IMConversationServiceModeAIFirst,
		TeamIDs:     teamIDs,
		Status:      enums.StatusOk,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create ai agent error = %v", err)
	}
	return item
}

func createWorkflowExecutorHandoffTeam(t *testing.T, db *gorm.DB, id int64, name string) {
	t.Helper()
	if err := db.Create(&models.AgentTeam{ID: id, Name: name, Status: enums.StatusOk}).Error; err != nil {
		t.Fatalf("create team error = %v", err)
	}
}

func createWorkflowExecutorHandoffActiveSchedule(t *testing.T, db *gorm.DB, teamID int64) {
	t.Helper()
	now := time.Now()
	if err := db.Create(&models.AgentTeamSchedule{
		TeamID:  teamID,
		StartAt: now.Add(-time.Hour),
		EndAt:   now.Add(time.Hour),
		Status:  enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create schedule error = %v", err)
	}
}

func createWorkflowExecutorHandoffAgentProfile(t *testing.T, db *gorm.DB, userID int64, teamID int64) {
	t.Helper()
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "agent",
		Nickname: "客服",
		Status:   enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create user error = %v", err)
	}
	if err := db.Create(&models.AgentProfile{
		UserID:             userID,
		TeamID:             teamID,
		AgentCode:          "A001",
		DisplayName:        "客服",
		ServiceStatus:      enums.ServiceStatusIdle,
		MaxConcurrentCount: 3,
		AutoAssignEnabled:  true,
		Status:             enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create profile error = %v", err)
	}
}

func createWorkflowExecutorHandoffConversation(t *testing.T, db *gorm.DB, aiAgentID int64) models.Conversation {
	t.Helper()
	now := time.Now()
	if err := db.FirstOrCreate(&models.Customer{
		ID:     1,
		Name:   "测试访客",
		Status: enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create customer error = %v", err)
	}
	item := models.Conversation{
		AIAgentID:     aiAgentID,
		ChannelID:     1,
		CustomerID:    1,
		CustomerName:  "测试访客",
		Status:        enums.IMConversationStatusAIServing,
		ServiceMode:   enums.IMConversationServiceModeAIFirst,
		LastMessageAt: now,
		LastActiveAt:  now,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create conversation error = %v", err)
	}
	return item
}

func createWorkflowExecutorCustomerMessage(t *testing.T, db *gorm.DB, conversationID int64, content string) models.Message {
	t.Helper()
	now := time.Now()
	item := models.Message{
		ConversationID: conversationID,
		ClientMsgID:    "customer-message",
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        content,
		SendStatus:     enums.IMMessageStatusSent,
		SentAt:         &now,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create message error = %v", err)
	}
	return item
}

func assertPath(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected path length: got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected path: got %#v want %#v", got, want)
		}
	}
}
