package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	ai "agent-desk/internal/ai"
	"agent-desk/internal/models"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
)

// einoAgentLoop is the production model/tool loop. AgentDesk still owns tool
// authorization, business execution, interrupts, idempotency, and auditing.
func einoAgentLoop(
	ctx context.Context,
	config models.AIConfig,
	systemPrompt string,
	userPrompt string,
	definitions []ai.ToolDefinition,
	maxSteps int,
	execute ai.ToolCallExecutor,
) (*ai.ToolLoopResult, error) {
	model, err := newEinoChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	if maxSteps <= 0 {
		maxSteps = 6
	}
	tools := make([]einotool.BaseTool, 0, len(definitions))
	for _, definition := range definitions {
		tool, buildErr := newEinoFunctionTool(definition, execute)
		if buildErr != nil {
			return nil, buildErr
		}
		tools = append(tools, tool)
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: tools},
		MaxStep:          maxSteps,
	})
	if err != nil {
		return nil, fmt.Errorf("create Eino agent loop: %w", err)
	}
	messages := make([]*schema.Message, 0, 2)
	if value := strings.TrimSpace(systemPrompt); value != "" {
		messages = append(messages, schema.SystemMessage(value))
	}
	messages = append(messages, schema.UserMessage(strings.TrimSpace(userPrompt)))
	result, err := agent.Generate(ctx, messages)
	if err != nil {
		slog.Warn("eino agent loop failed, falling back to standard ChatWithTools", "error", err)
		return ai.LLM.ChatWithTools(ctx, config, systemPrompt, userPrompt, definitions, maxSteps, execute)
	}
	if result == nil {
		return nil, fmt.Errorf("Eino agent loop returned no result")
	}
	ret := &ai.ToolLoopResult{ChatCompletionResult: ai.ChatCompletionResult{
		Content:   strings.TrimSpace(result.Content),
		ModelName: config.ModelName,
	}}
	if result.ResponseMeta != nil && result.ResponseMeta.Usage != nil {
		ret.PromptTokens = result.ResponseMeta.Usage.PromptTokens
		ret.CompletionTokens = result.ResponseMeta.Usage.CompletionTokens
	}
	return ret, nil
}

func newEinoChatModel(ctx context.Context, config models.AIConfig) (einomodel.ToolCallingChatModel, error) {
	if strings.TrimSpace(config.APIKey) == "" || strings.TrimSpace(config.BaseURL) == "" || strings.TrimSpace(config.ModelName) == "" {
		return nil, fmt.Errorf("ai config base URL, API key, and model name are required")
	}
	modelConfig := &einoopenai.ChatModelConfig{
		APIKey:  strings.TrimSpace(config.APIKey),
		BaseURL: strings.TrimSpace(config.BaseURL),
		Model:   strings.TrimSpace(config.ModelName),
	}
	if config.TimeoutMS > 0 {
		modelConfig.Timeout = time.Duration(config.TimeoutMS) * time.Millisecond
	}
	if config.MaxOutputTokens > 0 {
		maxTokens := config.MaxOutputTokens
		modelConfig.MaxCompletionTokens = &maxTokens
	}
	if isDashScopeQwenThinkingModel(config) {
		modelConfig.ExtraFields = map[string]any{"enable_thinking": false}
	}
	model, err := einoopenai.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, fmt.Errorf("create Eino OpenAI-compatible model: %w", err)
	}
	return model, nil
}

func isDashScopeQwenThinkingModel(config models.AIConfig) bool {
	baseURL := strings.ToLower(strings.TrimSpace(config.BaseURL))
	modelName := strings.ToLower(strings.TrimSpace(config.ModelName))
	return strings.Contains(baseURL, "dashscope.aliyuncs.com") && strings.HasPrefix(modelName, "qwen3")
}

type einoFunctionTool struct {
	info    *schema.ToolInfo
	execute ai.ToolCallExecutor
}

var _ einotool.InvokableTool = (*einoFunctionTool)(nil)

func newEinoFunctionTool(definition ai.ToolDefinition, execute ai.ToolCallExecutor) (*einoFunctionTool, error) {
	if strings.TrimSpace(definition.Name) == "" || execute == nil {
		return nil, fmt.Errorf("Eino tool name and executor are required")
	}
	info := &schema.ToolInfo{
		Name: strings.TrimSpace(definition.Name),
		Desc: strings.TrimSpace(definition.Description),
	}
	if len(definition.Parameters) > 0 {
		data, err := json.Marshal(definition.Parameters)
		if err != nil {
			return nil, fmt.Errorf("encode Eino tool schema: %w", err)
		}
		var params einojsonschema.Schema
		if err := json.Unmarshal(data, &params); err != nil {
			return nil, fmt.Errorf("decode Eino tool schema: %w", err)
		}
		info.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(&params)
	}
	return &einoFunctionTool{info: info, execute: execute}, nil
}

func (t *einoFunctionTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *einoFunctionTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	result, err := t.execute(ctx, ai.ToolCall{Name: t.info.Name, Arguments: arguments})
	if err == nil {
		return result, nil
	}
	var interrupt *agentLoopInterruptError
	if errors.As(err, &interrupt) {
		return "", err
	}
	observation, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return "", err
	}
	return string(observation), nil
}
