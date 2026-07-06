package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/ai"
	"agent-desk/internal/ai/runtime/graphs"
	"agent-desk/internal/ai/runtime/internal/impl/retrievers"
	"agent-desk/internal/ai/workflow/dsl"
	workflowregistry "agent-desk/internal/ai/workflow/registry"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/services"
)

const maxWorkflowSteps = 128

var workflowHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)

type Input struct {
	Definition   dsl.Definition
	Conversation models.Conversation
	UserMessage  models.Message
	AIAgent      models.AIAgent
	AIConfig     models.AIConfig
}

type Result struct {
	Status           string
	ReplyText        string
	NodePath         []string
	NodeTraces       []NodeTrace
	PromptTokens     int
	CompletionTokens int
	RetrieverCount   int
	TraceData        string
	CheckPointID     string
	CheckPointData   string
	Interrupted      bool
	Interrupts       []InterruptSummary
}

type NodeTrace struct {
	NodeID        string
	NodeType      string
	Status        string
	InputPreview  string
	OutputPreview string
	ErrorMessage  string
	DurationMS    int
}

type InterruptSummary struct {
	Type        string
	ID          string
	InfoPreview string
}

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

type runState struct {
	input           Input
	nodesByID       map[string]dsl.Node
	outgoing        map[string][]dsl.Edge
	vars            map[string]map[string]any
	branchDecisions map[string]branchDecision
	result          Result
}

type workflowCheckPoint struct {
	Definition    dsl.Definition            `json:"definition"`
	ConfirmNodeID string                    `json:"confirmNodeId"`
	Vars          map[string]map[string]any `json:"vars"`
}

type branchDecision struct {
	SelectedEdgeID       string                `json:"selectedEdgeId,omitempty"`
	SelectedBranchID     string                `json:"selectedBranchId,omitempty"`
	SelectedBranchName   string                `json:"selectedBranchName,omitempty"`
	SelectedTargetNodeID string                `json:"selectedTargetNodeId,omitempty"`
	Reason               string                `json:"reason"`
	Evaluations          []conditionEvaluation `json:"evaluations,omitempty"`
}

type conditionEvaluation struct {
	EdgeID       string `json:"edgeId"`
	BranchID     string `json:"branchId,omitempty"`
	BranchName   string `json:"branchName,omitempty"`
	TargetNodeID string `json:"targetNodeId"`
	SourceNodeID string `json:"sourceNodeId,omitempty"`
	SourceField  string `json:"sourceField,omitempty"`
	Operator     string `json:"operator,omitempty"`
	LeftValue    any    `json:"leftValue,omitempty"`
	RightValue   any    `json:"rightValue,omitempty"`
	Matched      bool   `json:"matched"`
}

func (e *Executor) Execute(ctx context.Context, input Input) (*Result, error) {
	state := newRunState(input)
	currentID := strings.TrimSpace(input.Definition.EntryNodeID)
	if currentID == "" {
		return nil, fmt.Errorf("workflow entry node is required")
	}
	return e.executeFrom(ctx, state, currentID)
}

func (e *Executor) Resume(ctx context.Context, input Input, checkPointData string, resumeText string) (*Result, error) {
	var checkpoint workflowCheckPoint
	if err := json.Unmarshal([]byte(strings.TrimSpace(checkPointData)), &checkpoint); err != nil {
		return nil, fmt.Errorf("invalid workflow checkpoint: %w", err)
	}
	if len(checkpoint.Definition.Nodes) > 0 {
		input.Definition = checkpoint.Definition
	}
	state := newRunState(input)
	state.vars = checkpoint.Vars
	if state.vars == nil {
		state.vars = make(map[string]map[string]any)
	}
	confirmNodeID := strings.TrimSpace(checkpoint.ConfirmNodeID)
	if confirmNodeID == "" {
		return nil, fmt.Errorf("workflow checkpoint confirm node is required")
	}
	decision := graphs.ParseConfirmationDecision(resumeText)
	if decision == "" {
		node, ok := state.nodesByID[confirmNodeID]
		if !ok {
			return nil, fmt.Errorf("workflow node does not exist: %s", confirmNodeID)
		}
		if err := e.executeHumanConfirm(state, node); err != nil {
			return nil, err
		}
		state.result.Status = "interrupted"
		return &state.result, nil
	}
	state.setNodeVars(confirmNodeID, map[string]any{
		"confirmed":    decision == graphs.ConfirmationDecisionConfirm,
		"responseText": strings.TrimSpace(resumeText),
	})
	nextID, ok, err := state.nextNodeID(confirmNodeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		state.result.Status = "completed"
		return &state.result, nil
	}
	return e.executeFrom(ctx, state, nextID)
}

func (e *Executor) executeFrom(ctx context.Context, state *runState, currentID string) (*Result, error) {
	for step := 0; step < maxWorkflowSteps; step++ {
		node, ok := state.nodesByID[currentID]
		if !ok {
			err := fmt.Errorf("workflow node does not exist: %s", currentID)
			state.result.Status = "error"
			return &state.result, err
		}
		state.result.NodePath = append(state.result.NodePath, node.ID)
		trace := NodeTrace{
			NodeID:       node.ID,
			NodeType:     node.Type,
			Status:       "running",
			InputPreview: workflowPreviewJSON(state.nodeInputPreview(node)),
		}
		startedAt := time.Now()
		if err := e.executeNode(ctx, state, node); err != nil {
			trace.Status = "failed"
			trace.ErrorMessage = err.Error()
			trace.DurationMS = int(time.Since(startedAt).Milliseconds())
			state.result.NodeTraces = append(state.result.NodeTraces, trace)
			state.result.Status = "error"
			return &state.result, err
		}
		trace.DurationMS = int(time.Since(startedAt).Milliseconds())
		if state.result.Interrupted {
			trace.OutputPreview = workflowPreviewJSON(state.nodeOutputPreview(node.ID))
			trace.Status = "interrupted"
			state.result.NodeTraces = append(state.result.NodeTraces, trace)
			state.result.Status = "interrupted"
			return &state.result, nil
		}
		if node.Type == workflowregistry.NodeTypeEnd {
			trace.OutputPreview = workflowPreviewJSON(state.nodeOutputPreview(node.ID))
			trace.Status = "completed"
			state.result.NodeTraces = append(state.result.NodeTraces, trace)
			state.result.Status = "completed"
			return &state.result, nil
		}
		nextID, ok, err := state.nextNodeID(node.ID)
		if err != nil {
			trace.OutputPreview = workflowPreviewJSON(state.nodeOutputPreview(node.ID))
			trace.Status = "failed"
			trace.ErrorMessage = err.Error()
			trace.DurationMS = int(time.Since(startedAt).Milliseconds())
			state.result.NodeTraces = append(state.result.NodeTraces, trace)
			state.result.Status = "error"
			return &state.result, err
		}
		trace.OutputPreview = workflowPreviewJSON(state.nodeOutputPreview(node.ID))
		trace.Status = "completed"
		state.result.NodeTraces = append(state.result.NodeTraces, trace)
		if !ok {
			state.result.Status = "completed"
			return &state.result, nil
		}
		currentID = nextID
	}
	err := fmt.Errorf("workflow exceeded max steps")
	state.result.Status = "error"
	return &state.result, err
}

func newRunState(input Input) *runState {
	state := &runState{
		input:           input,
		nodesByID:       make(map[string]dsl.Node, len(input.Definition.Nodes)),
		outgoing:        make(map[string][]dsl.Edge),
		vars:            make(map[string]map[string]any),
		branchDecisions: make(map[string]branchDecision),
		result: Result{
			Status:     "started",
			NodePath:   make([]string, 0),
			NodeTraces: make([]NodeTrace, 0),
		},
	}
	for _, node := range input.Definition.Nodes {
		node.ID = strings.TrimSpace(node.ID)
		node.Type = strings.TrimSpace(node.Type)
		if node.ID != "" {
			state.nodesByID[node.ID] = node
		}
	}
	for _, edge := range input.Definition.Edges {
		state.outgoing[edge.Source] = append(state.outgoing[edge.Source], edge)
	}
	return state
}

func (e *Executor) executeNode(ctx context.Context, state *runState, node dsl.Node) error {
	switch node.Type {
	case workflowregistry.NodeTypeStart:
		state.setNodeVars(node.ID, map[string]any{
			"conversationId":    state.input.Conversation.ID,
			"messageId":         state.input.UserMessage.ID,
			"aiAgentId":         state.input.AIAgent.ID,
			"userMessage":       strings.TrimSpace(state.input.UserMessage.Content),
			"knowledgeBaseIds":  utils.SplitInt64s(state.input.AIAgent.KnowledgeIDs),
			"conversationState": state.input.Conversation.Status,
		})
	case workflowregistry.NodeTypeConversationUnderstanding:
		return e.executeConversationUnderstanding(state, node)
	case workflowregistry.NodeTypeReplyPolicy:
		return e.executeReplyPolicy(state, node)
	case workflowregistry.NodeTypeKnowledgeRetrieve:
		return e.executeKnowledgeRetrieve(ctx, state, node)
	case workflowregistry.NodeTypeAnswerabilityGate:
		return e.executeAnswerabilityGate(state, node)
	case workflowregistry.NodeTypeCondition:
		state.setNodeVars(node.ID, map[string]any{"matched": true})
	case workflowregistry.NodeTypeAnalyzeConversation:
		return e.executeAnalyzeConversation(ctx, state, node)
	case workflowregistry.NodeTypePrepareTicketDraft:
		return e.executePrepareTicketDraft(ctx, state, node)
	case workflowregistry.NodeTypeHumanConfirm:
		return e.executeHumanConfirm(state, node)
	case workflowregistry.NodeTypeCreateTicket:
		return e.executeCreateTicket(state, node)
	case workflowregistry.NodeTypeLLMReply:
		return e.executeLLMReply(ctx, state, node)
	case workflowregistry.NodeTypeSendReply:
		replyText := strings.TrimSpace(toString(state.resolveInput(node, "replyText")))
		state.result.ReplyText = replyText
		state.setNodeVars(node.ID, map[string]any{
			"sent":           replyText != "",
			"replyMessageId": int64(0),
		})
	case workflowregistry.NodeTypeHandoffToHuman:
		return e.executeHandoffToHuman(state, node)
	case workflowregistry.NodeTypeEnd:
		state.setNodeVars(node.ID, map[string]any{"status": "completed"})
	default:
		return fmt.Errorf("unsupported workflow node type: %s", node.Type)
	}
	return nil
}

func (e *Executor) executeConversationUnderstanding(state *runState, node dsl.Node) error {
	rawMessage := strings.TrimSpace(toString(state.resolveInput(node, "userMessage")))
	if rawMessage == "" {
		rawMessage = state.input.UserMessage.Content
	}
	understanding := understandConversationMessage(rawMessage)
	state.setNodeVars(node.ID, map[string]any{
		"normalizedMessage": understanding.NormalizedMessage,
		"messageIntent":     understanding.MessageIntent,
		"answerScope":       understanding.AnswerScope,
		"confidence":        understanding.Confidence,
		"riskSignals":       understanding.RiskSignals,
		"reason":            understanding.Reason,
	})
	return nil
}

func (e *Executor) executeReplyPolicy(state *runState, node dsl.Node) error {
	intent := strings.TrimSpace(toString(state.resolveInput(node, "messageIntent")))
	scope := strings.TrimSpace(toString(state.resolveInput(node, "answerScope")))
	userMessage := normalizeWorkflowUserMessage(toString(state.resolveInput(node, "userMessage")))
	decision := decideWorkflowReplyPolicy(state.input.AIAgent, workflowReplyPolicyInput{
		MessageIntent: intent,
		AnswerScope:   scope,
		UserMessage:   userMessage,
		Answerability: strings.TrimSpace(toString(state.resolveInput(node, "answerability"))),
	})
	state.setNodeVars(node.ID, map[string]any{
		"action":           decision.Action,
		"replyText":        decision.ReplyText,
		"reason":           decision.Reason,
		"requiresFlow":     decision.RequiresFlow,
		"targetFlow":       decision.TargetFlow,
		"finalReplySource": decision.FinalReplySource,
	})
	return nil
}

func (e *Executor) executeCreateTicket(state *runState, node dsl.Node) error {
	confirmed := truthy(state.resolveInput(node, "confirmed"))
	if !confirmed {
		state.setNodeVars(node.ID, map[string]any{
			"ticketId": int64(0),
			"created":  false,
		})
		return nil
	}
	draft := asMap(state.resolveInput(node, "ticketDraft"))
	title := strings.TrimSpace(toString(draft["title"]))
	description := strings.TrimSpace(toString(draft["description"]))
	item, err := services.TicketService.CreateFromConversation(request.CreateTicketFromConversationRequest{
		ConversationID: state.input.Conversation.ID,
		Title:          title,
		Description:    description,
	}, workflowAIPrincipal(state.input.AIAgent))
	if err != nil {
		return err
	}
	state.setNodeVars(node.ID, map[string]any{
		"ticketId": item.ID,
		"ticketNo": item.TicketNo,
		"created":  true,
		"message":  buildTicketCreatedMessage(item),
	})
	return nil
}

func buildTicketCreatedMessage(item *models.Ticket) string {
	if item == nil {
		return "工单已创建。"
	}
	ticketNo := strings.TrimSpace(item.TicketNo)
	if ticketNo == "" {
		return fmt.Sprintf("工单已创建，工单 ID：%d。", item.ID)
	}
	return "工单已创建，工单号：" + ticketNo + "。"
}

func workflowAIPrincipal(aiAgent models.AIAgent) *dto.AuthPrincipal {
	username := strings.TrimSpace(aiAgent.Name)
	if username == "" {
		username = "AI"
	}
	return &dto.AuthPrincipal{
		UserID:   0,
		Username: username,
		Nickname: username,
	}
}

type workflowConversationUnderstanding struct {
	NormalizedMessage string
	MessageIntent     string
	AnswerScope       string
	Confidence        float64
	RiskSignals       []string
	Reason            string
}

type workflowReplyPolicyInput struct {
	MessageIntent string
	AnswerScope   string
	UserMessage   string
	Answerability string
}

type workflowReplyPolicyDecision struct {
	Action           string
	ReplyText        string
	Reason           string
	RequiresFlow     bool
	TargetFlow       string
	FinalReplySource string
}

func understandConversationMessage(rawMessage string) workflowConversationUnderstanding {
	message := normalizeWorkflowUserMessage(rawMessage)
	ret := workflowConversationUnderstanding{
		NormalizedMessage: message,
		MessageIntent:     "unknown",
		AnswerScope:       "needs_clarification",
		Confidence:        0.5,
		Reason:            "message intent is unclear",
	}
	if message == "" {
		ret.MessageIntent = "unknown"
		ret.AnswerScope = "needs_clarification"
		ret.Confidence = 0.9
		ret.Reason = "empty message"
		return ret
	}
	lower := strings.ToLower(message)
	switch {
	case isGreetingMessage(lower):
		ret.MessageIntent = "greeting"
		ret.AnswerScope = "direct_reply"
		ret.Confidence = 0.98
		ret.Reason = "matched greeting phrase"
	case containsAnyWorkflowText(lower, "谢谢", "感谢", "多谢", "辛苦了", "thank"):
		ret.MessageIntent = "thanks"
		ret.AnswerScope = "direct_reply"
		ret.Confidence = 0.95
		ret.Reason = "matched thanks phrase"
	case containsAnyWorkflowText(lower, "再见", "拜拜", "不用了", "没事了", "结束"):
		ret.MessageIntent = "end_conversation"
		ret.AnswerScope = "direct_reply"
		ret.Confidence = 0.9
		ret.Reason = "matched ending phrase"
	case containsAnyWorkflowText(lower, "人工", "转人工", "真人", "客服"):
		ret.MessageIntent = "handoff_request"
		ret.AnswerScope = "needs_handoff"
		ret.Confidence = 0.95
		ret.RiskSignals = append(ret.RiskSignals, "handoff_requested")
		ret.Reason = "matched handoff phrase"
	case containsAnyWorkflowText(lower, "投诉", "举报", "差评", "曝光", "起诉", "律师", "12315"):
		ret.MessageIntent = "complaint"
		ret.AnswerScope = "needs_handoff"
		ret.Confidence = 0.92
		ret.RiskSignals = append(ret.RiskSignals, "complaint_escalation")
		ret.Reason = "matched complaint phrase"
	case containsAnyWorkflowText(lower, "工单", "报障", "售后", "登记问题", "记录问题"):
		ret.MessageIntent = "ticket_request"
		ret.AnswerScope = "needs_ticket"
		ret.Confidence = 0.9
		ret.RiskSignals = append(ret.RiskSignals, "ticket_expected")
		ret.Reason = "matched ticket phrase"
	case isKnowledgeSeekingWorkflowQuestion(lower):
		ret.MessageIntent = "business_question"
		ret.AnswerScope = "needs_knowledge"
		ret.Confidence = 0.86
		ret.Reason = "matched knowledge seeking question"
	case isConfirmationWorkflowMessage(lower):
		ret.MessageIntent = "confirmation"
		ret.AnswerScope = "direct_reply"
		ret.Confidence = 0.8
		ret.Reason = "matched confirmation phrase"
	case isAmbiguousWorkflowQuestion(lower):
		ret.MessageIntent = "ambiguous_question"
		ret.AnswerScope = "needs_clarification"
		ret.Confidence = 0.82
		ret.Reason = "message lacks a concrete business object"
	default:
		ret.MessageIntent = "business_question"
		ret.AnswerScope = "needs_knowledge"
		ret.Confidence = 0.7
		ret.Reason = "default business question policy"
	}
	return ret
}

func decideWorkflowReplyPolicy(aiAgent models.AIAgent, input workflowReplyPolicyInput) workflowReplyPolicyDecision {
	intent := strings.TrimSpace(input.MessageIntent)
	scope := strings.TrimSpace(input.AnswerScope)
	if answerability := strings.TrimSpace(input.Answerability); answerability != "" && answerability != "answerable" {
		return workflowReplyPolicyDecision{
			Action:           "knowledge_fallback",
			ReplyText:        workflowKnowledgeFallbackReply(aiAgent),
			Reason:           "knowledge is not sufficient for business answer",
			FinalReplySource: "knowledge_fallback",
		}
	}
	switch {
	case intent == "greeting":
		return workflowReplyPolicyDecision{Action: "direct_reply", ReplyText: "您好，请问有什么可以帮您？", Reason: "greeting can be answered directly", FinalReplySource: "direct_reply"}
	case intent == "thanks":
		return workflowReplyPolicyDecision{Action: "direct_reply", ReplyText: "不客气，如有其他问题可以继续告诉我。", Reason: "thanks can be answered directly", FinalReplySource: "direct_reply"}
	case intent == "end_conversation":
		return workflowReplyPolicyDecision{Action: "end_conversation", ReplyText: "好的，如后续还有问题可以随时联系。", Reason: "conversation ending phrase", FinalReplySource: "direct_reply"}
	case intent == "confirmation":
		return workflowReplyPolicyDecision{Action: "direct_reply", ReplyText: "好的，请继续补充需要处理的问题。", Reason: "confirmation without pending interrupt", FinalReplySource: "direct_reply"}
	case intent == "handoff_request" || scope == "needs_handoff":
		return workflowReplyPolicyDecision{Action: "handoff_to_human", Reason: "user requested human support or risk requires handoff", RequiresFlow: true, TargetFlow: "handoff_to_human", FinalReplySource: "handoff_notice"}
	case intent == "ticket_request" || scope == "needs_ticket":
		return workflowReplyPolicyDecision{Action: "prepare_ticket", Reason: "user requested ticket handling", RequiresFlow: true, TargetFlow: "prepare_ticket", FinalReplySource: "ticket_result"}
	case intent == "ambiguous_question" || scope == "needs_clarification":
		return workflowReplyPolicyDecision{Action: "clarify", ReplyText: "请补充具体的产品、场景、报错信息或你希望处理的结果，我再继续帮你确认。", Reason: "message needs clarification", FinalReplySource: "clarification"}
	case scope == "needs_knowledge":
		return workflowReplyPolicyDecision{Action: "retrieve_knowledge", Reason: "business question should be answered with knowledge evidence", RequiresFlow: true, TargetFlow: "knowledge", FinalReplySource: "knowledge_answer"}
	default:
		return workflowReplyPolicyDecision{Action: "clarify", ReplyText: "请补充更具体的问题，我再继续帮你处理。", Reason: "fallback to clarification for unclear policy input", FinalReplySource: "clarification"}
	}
}

func normalizeWorkflowUserMessage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = workflowHTMLTagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func isGreetingMessage(value string) bool {
	trimmed := strings.Trim(value, " 	\r\n。.!！?？~～")
	return containsAnyWorkflowText(trimmed, "你好", "您好", "在吗", "在不在") || trimmed == "hello" || trimmed == "hi"
}

func isAmbiguousWorkflowQuestion(value string) bool {
	trimmed := strings.Trim(value, " 	\r\n。.!！?？~～")
	return containsAnyWorkflowText(trimmed, "怎么弄", "怎么办", "怎么处理", "帮我看看", "有问题") || len([]rune(trimmed)) <= 3
}

func isKnowledgeSeekingWorkflowQuestion(value string) bool {
	trimmed := strings.Trim(value, " 	\r\n。.!！?？~～")
	if trimmed == "" {
		return false
	}
	return containsAnyWorkflowText(trimmed,
		"?", "？", "吗", "呢", "么", "是不是", "是否", "能不能", "能否", "可不可以", "为什么", "怎么", "如何", "哪", "哪个", "哪种", "什么",
		"推荐", "适合", "价格", "多少钱", "多少", "预算", "型号", "产品", "功能", "区别", "对比", "活动", "优惠",
	)
}

func isConfirmationWorkflowMessage(value string) bool {
	trimmed := strings.Trim(value, " 	\r\n。.!！?？~～")
	switch trimmed {
	case "确认", "可以", "可以的", "好的", "好", "是的", "嗯", "嗯嗯", "行", "取消":
		return true
	}
	return (strings.HasPrefix(trimmed, "确认") || strings.HasPrefix(trimmed, "取消")) && len([]rune(trimmed)) <= 8
}

func containsAnyWorkflowText(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func (e *Executor) executeHumanConfirm(state *runState, node dsl.Node) error {
	prompt := strings.TrimSpace(toString(state.resolveInput(node, "prompt")))
	if prompt == "" {
		prompt = "请确认是否继续。"
	}
	infoPreview, err := json.Marshal(map[string]string{"message": prompt})
	if err != nil {
		return err
	}
	state.result.Interrupted = true
	state.result.CheckPointID = buildWorkflowCheckPointID(state.input, node.ID)
	checkpoint, err := json.Marshal(workflowCheckPoint{
		Definition:    state.input.Definition,
		ConfirmNodeID: node.ID,
		Vars:          state.vars,
	})
	if err != nil {
		return err
	}
	state.result.CheckPointData = string(checkpoint)
	state.result.Interrupts = []InterruptSummary{
		{
			Type:        workflowregistry.NodeTypeHumanConfirm,
			ID:          node.ID,
			InfoPreview: string(infoPreview),
		},
	}
	return nil
}

func buildWorkflowCheckPointID(input Input, nodeID string) string {
	return fmt.Sprintf("workflow:%d:%d:%s", input.Conversation.ID, input.UserMessage.ID, strings.TrimSpace(nodeID))
}

func (e *Executor) executePrepareTicketDraft(ctx context.Context, state *runState, node dsl.Node) error {
	issue := strings.TrimSpace(toString(state.resolveInput(node, "issue")))
	input := graphs.PrepareTicketDraftInput{
		Issue: issue,
	}
	if title := strings.TrimSpace(readStringConfig(node.Config, "title")); title != "" {
		input.Title = title
	}
	if description := strings.TrimSpace(readStringConfig(node.Config, "description")); description != "" {
		input.Description = description
	}
	if impact := strings.TrimSpace(readStringConfig(node.Config, "impact")); impact != "" {
		input.Impact = impact
	}
	if expectedOutcome := strings.TrimSpace(readStringConfig(node.Config, "expectedOutcome")); expectedOutcome != "" {
		input.ExpectedOutcome = expectedOutcome
	}
	if currentAttempt := strings.TrimSpace(readStringConfig(node.Config, "currentAttempt")); currentAttempt != "" {
		input.CurrentAttempt = currentAttempt
	}
	args, err := json.Marshal(input)
	if err != nil {
		return err
	}
	raw, err := graphs.NewPrepareTicketDraftGraph(state.input.Conversation).Run(ctx, string(args))
	if err != nil {
		return err
	}
	var result graphs.PrepareTicketDraftResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return err
	}
	state.setNodeVars(node.ID, map[string]any{
		"ticketDraft": map[string]any{
			"ready":             result.Ready,
			"title":             strings.TrimSpace(result.Title),
			"description":       strings.TrimSpace(result.Description),
			"missingFields":     result.MissingFields,
			"followUpQuestions": result.FollowUpQuestions,
			"conversationFacts": result.ConversationFacts,
		},
	})
	return nil
}

func (e *Executor) executeAnalyzeConversation(ctx context.Context, state *runState, node dsl.Node) error {
	userMessage := strings.TrimSpace(toString(state.resolveInput(node, "userMessage")))
	input := graphs.AnalyzeConversationInput{
		ObservedIssue: userMessage,
	}
	if strings.TrimSpace(readStringConfig(node.Config, "goal")) != "" {
		input.Goal = strings.TrimSpace(readStringConfig(node.Config, "goal"))
	}
	if readBoolConfig(node.Config, "needTicket") {
		input.NeedTicket = true
	}
	if readBoolConfig(node.Config, "needHumanHandoff") {
		input.NeedHumanHandoff = true
	}
	if readBoolConfig(node.Config, "needQualityCheck") {
		input.NeedQualityCheck = true
	}
	if strings.TrimSpace(readStringConfig(node.Config, "additionalContext")) != "" {
		input.AdditionalContext = strings.TrimSpace(readStringConfig(node.Config, "additionalContext"))
	}
	args, err := json.Marshal(input)
	if err != nil {
		return err
	}
	raw, err := graphs.NewAnalyzeConversationGraph(state.input.Conversation).Run(ctx, string(args))
	if err != nil {
		return err
	}
	var result graphs.AnalyzeConversationResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return err
	}
	nextAction := strings.TrimSpace(result.RecommendedNextAction)
	state.setNodeVars(node.ID, map[string]any{
		"intent":           strings.TrimSpace(result.UserIntent),
		"riskLevel":        strings.TrimSpace(result.RiskLevel),
		"needTicket":       nextAction == "prepare_ticket",
		"needHumanHandoff": nextAction == "handoff_to_human",
	})
	return nil
}

func (e *Executor) executeHandoffToHuman(state *runState, node dsl.Node) error {
	if _, hasConfirmedInput := node.Inputs["confirmed"]; hasConfirmedInput && !truthy(state.resolveInput(node, "confirmed")) {
		state.setNodeVars(node.ID, map[string]any{
			"handoffId":  int64(0),
			"reason":     strings.TrimSpace(toString(state.resolveInput(node, "reason"))),
			"decision":   "cancelled",
			"teamId":     int64(0),
			"assigneeId": int64(0),
			"message":    "",
			"skipped":    true,
		})
		return nil
	}
	reason := strings.TrimSpace(toString(state.resolveInput(node, "reason")))
	result, err := services.ConversationHumanDispatchService.HandoffByAIWithRequestID(
		state.input.Conversation.ID,
		state.input.AIAgent,
		reason,
		strings.TrimSpace(state.input.UserMessage.RequestID),
	)
	if err != nil {
		return err
	}
	output := map[string]any{
		"handoffId":  int64(0),
		"reason":     reason,
		"decision":   "",
		"teamId":     int64(0),
		"assigneeId": int64(0),
		"message":    "",
	}
	if result != nil {
		output["decision"] = string(result.Decision)
		output["teamId"] = result.TeamID
		output["assigneeId"] = result.AssigneeID
		output["message"] = strings.TrimSpace(result.Message)
	}
	state.setNodeVars(node.ID, output)
	return nil
}

func (e *Executor) executeKnowledgeRetrieve(ctx context.Context, state *runState, node dsl.Node) error {
	query := strings.TrimSpace(toString(state.resolveInput(node, "query")))
	retriever := retrievers.NewKnowledgeRetriever(state.input.AIAgent)
	result, err := retriever.RetrieveContext(ctx, query)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(result.ContextResults))
	for _, item := range result.ContextResults {
		items = append(items, map[string]any{
			"knowledgeBaseId": item.KnowledgeBaseID,
			"documentId":      item.DocumentID,
			"chunkId":         item.ChunkID,
			"content":         item.Content,
			"score":           item.Score,
		})
	}
	state.result.RetrieverCount = len(result.Hits)
	state.setNodeVars(node.ID, map[string]any{
		"items":   items,
		"summary": result.ContextText,
	})
	return nil
}

func (e *Executor) executeAnswerabilityGate(state *runState, node dsl.Node) error {
	items := state.resolveInput(node, "knowledgeItems")
	answerability := "unanswerable"
	reason := "no retrieved knowledge items"
	if hasItems(items) {
		answerability = "answerable"
		reason = "retrieved knowledge items are available"
	} else if strings.TrimSpace(services.DigitalStoreProfileService.BuildRuntimeInstruction()) != "" {
		answerability = "answerable"
		reason = "digital store runtime context is available"
	}
	state.setNodeVars(node.ID, map[string]any{
		"answerability": answerability,
		"reason":        reason,
	})
	return nil
}

func (e *Executor) executeLLMReply(ctx context.Context, state *runState, node dsl.Node) error {
	if staticReply := strings.TrimSpace(readStringConfig(node.Config, "staticReply")); staticReply != "" {
		state.setNodeVars(node.ID, map[string]any{"replyText": staticReply})
		return nil
	}
	userPrompt := strings.TrimSpace(toString(state.resolveInput(node, "userMessage")))
	if userPrompt == "" {
		userPrompt = strings.TrimSpace(state.input.UserMessage.Content)
	}
	knowledgeItems := toString(state.resolveInput(node, "knowledgeItems"))
	systemPrompt := strings.TrimSpace(state.input.AIAgent.SystemPrompt)
	runtimeInstruction := services.DigitalStoreProfileService.BuildRuntimeInstruction()
	if runtimeInstruction != "" {
		systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + runtimeInstruction)
	}
	if prompt := strings.TrimSpace(readStringConfig(node.Config, "prompt")); prompt != "" {
		systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + prompt)
	}
	if _, declaresKnowledge := node.Inputs["knowledgeItems"]; declaresKnowledge && len(utils.SplitInt64s(state.input.AIAgent.KnowledgeIDs)) > 0 && !hasItems(state.resolveInput(node, "knowledgeItems")) && runtimeInstruction == "" {
		state.setNodeVars(node.ID, map[string]any{"replyText": workflowKnowledgeFallbackReply(state.input.AIAgent)})
		return nil
	}
	if knowledgeItems != "" {
		userPrompt = userPrompt + "\n\nKnowledge context:\n" + knowledgeItems
	}
	result, err := ai.LLM.ChatWithConfig(ctx, state.input.AIConfig, systemPrompt, userPrompt)
	if err != nil {
		return err
	}
	state.result.PromptTokens += result.PromptTokens
	state.result.CompletionTokens += result.CompletionTokens
	state.setNodeVars(node.ID, map[string]any{"replyText": result.Content})
	return nil
}

func workflowKnowledgeFallbackReply(aiAgent models.AIAgent) string {
	if reply := strings.TrimSpace(aiAgent.FallbackMessage); reply != "" {
		return reply
	}
	if aiAgent.FallbackMode == enums.AIAgentFallbackModeSuggestRetry {
		return "当前知识库里没有找到足够明确的信息，你可以换个更具体的问法再试一次。"
	}
	return "当前知识库暂无明确信息。"
}

func (s *runState) nextNodeID(sourceNodeID string) (string, bool, error) {
	edges := s.outgoing[sourceNodeID]
	if len(edges) == 0 {
		return "", false, nil
	}
	node := s.nodesByID[sourceNodeID]
	if strings.TrimSpace(node.Type) != workflowregistry.NodeTypeCondition {
		return strings.TrimSpace(edges[0].Target), true, nil
	}
	config := dsl.ConditionConfig{}
	if len(node.Config) > 0 {
		if err := json.Unmarshal(node.Config, &config); err != nil {
			return "", false, fmt.Errorf("invalid condition node config: %w", err)
		}
	}
	evaluations := make([]conditionEvaluation, 0)
	for _, branch := range config.Branches {
		if branch.Default {
			continue
		}
		matched, evaluation, err := s.evaluateConditionBranch(sourceNodeID, branch)
		if err != nil {
			return "", false, err
		}
		evaluations = append(evaluations, evaluation)
		if matched {
			targetNodeID := strings.TrimSpace(branch.TargetNodeID)
			s.branchDecisions[sourceNodeID] = branchDecision{
				SelectedEdgeID:       s.edgeIDForTarget(sourceNodeID, targetNodeID),
				SelectedBranchID:     strings.TrimSpace(branch.ID),
				SelectedBranchName:   strings.TrimSpace(branch.Name),
				SelectedTargetNodeID: targetNodeID,
				Reason:               "condition branch matched",
				Evaluations:          evaluations,
			}
			return targetNodeID, true, nil
		}
	}
	for _, branch := range config.Branches {
		if !branch.Default {
			continue
		}
		targetNodeID := strings.TrimSpace(branch.TargetNodeID)
		s.branchDecisions[sourceNodeID] = branchDecision{
			SelectedEdgeID:       s.edgeIDForTarget(sourceNodeID, targetNodeID),
			SelectedBranchID:     strings.TrimSpace(branch.ID),
			SelectedBranchName:   strings.TrimSpace(branch.Name),
			SelectedTargetNodeID: targetNodeID,
			Reason:               "no condition branch matched; selected default branch",
			Evaluations:          evaluations,
		}
		return targetNodeID, true, nil
	}
	s.branchDecisions[sourceNodeID] = branchDecision{
		Reason:      "no condition branch matched and no default branch exists",
		Evaluations: evaluations,
	}
	return "", false, nil
}

func (s *runState) evaluateConditionBranch(sourceNodeID string, branch dsl.ConditionBranch) (bool, conditionEvaluation, error) {
	condition := branch.Condition
	targetNodeID := strings.TrimSpace(branch.TargetNodeID)
	evaluation := conditionEvaluation{
		EdgeID:       s.edgeIDForTarget(sourceNodeID, targetNodeID),
		BranchID:     strings.TrimSpace(branch.ID),
		BranchName:   strings.TrimSpace(branch.Name),
		TargetNodeID: targetNodeID,
	}
	if condition == nil {
		evaluation.Matched = true
		return true, evaluation, nil
	}
	left := s.resolveSelector(condition.Left)
	operator := strings.TrimSpace(condition.Operator)
	if condition.Left != nil {
		evaluation.SourceNodeID = strings.TrimSpace(condition.Left.NodeID)
		evaluation.SourceField = strings.TrimSpace(condition.Left.Field)
	}
	evaluation.Operator = operator
	evaluation.LeftValue = left
	evaluation.RightValue = condition.Right
	if operator == "" && strings.TrimSpace(condition.Expression) != "" {
		return false, evaluation, fmt.Errorf("free-form workflow condition expressions are not supported")
	}
	var matched bool
	switch operator {
	case "eq", "equals":
		matched = compareString(left, condition.Right) == 0
	case "neq", "not_equals":
		matched = compareString(left, condition.Right) != 0
	case "contains":
		matched = strings.Contains(toString(left), toString(condition.Right))
	case "exists":
		matched = exists(left)
	case "not_exists":
		matched = !exists(left)
	case "truthy", "is_true":
		matched = truthy(left)
	case "falsy", "is_false":
		matched = !truthy(left)
	case "gt":
		matched = compareNumber(left, condition.Right) > 0
	case "gte":
		matched = compareNumber(left, condition.Right) >= 0
	case "lt":
		matched = compareNumber(left, condition.Right) < 0
	case "lte":
		matched = compareNumber(left, condition.Right) <= 0
	default:
		return false, evaluation, fmt.Errorf("unsupported workflow condition operator: %s", operator)
	}
	evaluation.Matched = matched
	return matched, evaluation, nil
}

func (s *runState) edgeIDForTarget(sourceNodeID string, targetNodeID string) string {
	for _, edge := range s.outgoing[sourceNodeID] {
		if strings.TrimSpace(edge.Target) == targetNodeID {
			return strings.TrimSpace(edge.ID)
		}
	}
	return ""
}

func (s *runState) setNodeVars(nodeID string, values map[string]any) {
	s.vars[nodeID] = values
}

func (s *runState) resolveInput(node dsl.Node, inputName string) any {
	selector, ok := node.Inputs[inputName]
	if !ok {
		return nil
	}
	return s.resolveSelector(&selector)
}

func (s *runState) nodeInputPreview(node dsl.Node) map[string]any {
	inputs := make(map[string]any, len(node.Inputs))
	for name, selector := range node.Inputs {
		inputs[name] = s.resolveSelector(&selector)
	}
	ret := map[string]any{
		"inputs": inputs,
	}
	if len(node.Config) > 0 {
		var cfg any
		if err := json.Unmarshal(node.Config, &cfg); err == nil {
			ret["config"] = cfg
		} else {
			ret["config"] = string(node.Config)
		}
	}
	return ret
}

func (s *runState) nodeOutputPreview(nodeID string) map[string]any {
	ret := map[string]any{
		"outputs": s.vars[nodeID],
	}
	if decision, ok := s.branchDecisions[nodeID]; ok {
		ret["branchDecision"] = decision
	}
	return ret
}

func workflowPreviewJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	const maxPreviewBytes = 2000
	if len(raw) <= maxPreviewBytes {
		return string(raw)
	}
	return string(raw[:maxPreviewBytes])
}

func (s *runState) resolveSelector(selector *dsl.VariableSelector) any {
	if selector == nil {
		return nil
	}
	fields := s.vars[strings.TrimSpace(selector.NodeID)]
	if fields == nil {
		return nil
	}
	return fields[strings.TrimSpace(selector.Field)]
}

func readStringConfig(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ""
	}
	return toString(cfg[key])
}

func readBoolConfig(raw json.RawMessage, key string) bool {
	if len(raw) == 0 {
		return false
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return false
	}
	return truthy(cfg[key])
}

func compareString(left any, right any) int {
	return strings.Compare(toString(left), toString(right))
}

func compareNumber(left any, right any) int {
	leftNum := toFloat(left)
	rightNum := toFloat(right)
	switch {
	case leftNum > rightNum:
		return 1
	case leftNum < rightNum:
		return -1
	default:
		return 0
	}
}

func toString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []map[string]any:
		buf, _ := json.Marshal(v)
		return string(buf)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func toFloat(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}

func asMap(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	case map[string]string:
		ret := make(map[string]any, len(v))
		for key, item := range v {
			ret[key] = item
		}
		return ret
	case string:
		var ret map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(v)), &ret); err == nil {
			return ret
		}
	}
	return map[string]any{}
}

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		return normalized != "" && normalized != "false" && normalized != "0"
	default:
		return !reflect.ValueOf(value).IsZero()
	}
}

func exists(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return true
	}
}

func hasItems(value any) bool {
	if value == nil {
		return false
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() > 0
	default:
		return exists(value)
	}
}
