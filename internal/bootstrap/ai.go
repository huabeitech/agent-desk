package bootstrap

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"log/slog"
	"strings"
	"time"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// InitAI bootstraps or syncs default AI configurations from environment/config file.
// Supports any OpenAI-compatible provider (OpenAI, DeepSeek, OpenRouter, Azure, Ollama, LiteLLM, vLLM, etc.).
func InitAI(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	apiKey := strings.TrimSpace(cfg.AI.APIKey)
	if apiKey == "" {
		return nil
	}

	provider := enums.AIProvider(strings.TrimSpace(cfg.AI.Provider))
	if provider == "" {
		provider = enums.AIProviderOpenAI
	}

	baseURL := strings.TrimSpace(cfg.AI.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	llmModel := strings.TrimSpace(cfg.AI.LLMModel)
	if llmModel == "" {
		llmModel = "gpt-4o-mini"
	}

	embeddingModel := strings.TrimSpace(cfg.AI.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}

	dimension := cfg.AI.EmbeddingDimension
	if dimension <= 0 {
		dimension = 1536
	}

	timeoutMS := cfg.AI.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30000
	}

	maxRetryCount := cfg.AI.MaxRetryCount
	if maxRetryCount < 0 {
		maxRetryCount = 1
	}

	db := sqls.DB()
	if db == nil {
		return nil
	}

	// 1. Ensure LLM config
	llmItem := models.AIConfig{
		Name:             "Default LLM",
		Provider:         provider,
		BaseURL:          baseURL,
		APIKey:           apiKey,
		ModelType:        enums.AIModelTypeLLM,
		ModelName:        llmModel,
		Dimension:        0,
		MaxContextTokens: 128000,
		MaxOutputTokens:  4096,
		TimeoutMS:        timeoutMS,
		MaxRetryCount:    maxRetryCount,
		Status:           enums.StatusOk,
		SortNo:           10,
		Remark:           "Auto-configured from environment variables",
	}
	if err := upsertBootstrapAIConfig(db, llmItem); err != nil {
		slog.Error("failed to bootstrap LLM AI config", "error", err)
	} else {
		slog.Info("bootstrapped LLM AI config", "model", llmModel, "baseUrl", baseURL)
	}

	// 2. Ensure Embedding config
	embeddingItem := models.AIConfig{
		Name:             "Default Embedding",
		Provider:         provider,
		BaseURL:          baseURL,
		APIKey:           apiKey,
		ModelType:        enums.AIModelTypeEmbedding,
		ModelName:        embeddingModel,
		Dimension:        dimension,
		MaxContextTokens: 8191,
		MaxOutputTokens:  0,
		TimeoutMS:        timeoutMS,
		MaxRetryCount:    maxRetryCount,
		Status:           enums.StatusOk,
		SortNo:           20,
		Remark:           "Auto-configured from environment variables",
	}
	if err := upsertBootstrapAIConfig(db, embeddingItem); err != nil {
		slog.Error("failed to bootstrap Embedding AI config", "error", err)
	} else {
		slog.Info("bootstrapped Embedding AI config", "model", embeddingModel, "dimension", dimension, "baseUrl", baseURL)
	}

	return nil
}

func upsertBootstrapAIConfig(db *gorm.DB, item models.AIConfig) error {
	now := time.Now()
	// Find if there's any config with the same model type and name
	existing := repositories.AIConfigRepository.FindOne(db, sqls.NewCnd().
		Eq("model_type", item.ModelType).
		Eq("name", item.Name))

	if existing == nil {
		// Also check if there is an active config of this model type
		existing = repositories.AIConfigRepository.GetEnabled(db, item.ModelType)
	}

	if existing == nil {
		item.AuditFields = models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   constants.SystemAuditUserID,
			CreateUserName: constants.SystemAuditUserName,
			UpdatedAt:      now,
			UpdateUserID:   constants.SystemAuditUserID,
			UpdateUserName: constants.SystemAuditUserName,
		}
		return repositories.AIConfigRepository.Create(db, &item)
	}

	// If existing config exists, update connection & model details to match .env
	return repositories.AIConfigRepository.Updates(db, existing.ID, map[string]any{
		"provider":         item.Provider,
		"base_url":         item.BaseURL,
		"api_key":          item.APIKey,
		"model_name":       item.ModelName,
		"dimension":        item.Dimension,
		"timeout_ms":       item.TimeoutMS,
		"max_retry_count":  item.MaxRetryCount,
		"status":           enums.StatusOk,
		"update_user_id":   constants.SystemAuditUserID,
		"update_user_name": constants.SystemAuditUserName,
		"updated_at":       now,
	})
}
