package ai_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"agent-desk/internal/ai"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func setupDBWithAIConfig(t *testing.T, llmModel, embeddingModel, baseURL, apiKey string, dimension int) *gorm.DB {
	dbName := fmt.Sprintf("file:memdb_live_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	if err := db.AutoMigrate(models.AIConfig{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}
	sqls.SetDB(db)

	now := time.Now()
	// Insert LLM
	_ = repositories.AIConfigRepository.Create(db, &models.AIConfig{
		Name:             "DOS LLM",
		Provider:         enums.AIProviderOpenAI,
		BaseURL:          baseURL,
		APIKey:           apiKey,
		ModelType:        enums.AIModelTypeLLM,
		ModelName:        llmModel,
		Dimension:        0,
		MaxContextTokens: 128000,
		MaxOutputTokens:  4096,
		TimeoutMS:        30000,
		MaxRetryCount:    1,
		Status:           enums.StatusOk,
		SortNo:           10,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	})

	// Insert Embedding
	_ = repositories.AIConfigRepository.Create(db, &models.AIConfig{
		Name:             "DOS Embedding",
		Provider:         enums.AIProviderOpenAI,
		BaseURL:          baseURL,
		APIKey:           apiKey,
		ModelType:        enums.AIModelTypeEmbedding,
		ModelName:        embeddingModel,
		Dimension:        dimension,
		MaxContextTokens: 8191,
		MaxOutputTokens:  0,
		TimeoutMS:        30000,
		MaxRetryCount:    1,
		Status:           enums.StatusOk,
		SortNo:           20,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	})

	return db
}

func TestDOSAILiveIntegration(t *testing.T) {
	cfg, err := config.Load("../../config/config.yaml")
	if err != nil || cfg == nil || cfg.AI.APIKey == "" {
		cfg, _ = config.Load("")
	}

	apiKey := ""
	if cfg != nil {
		apiKey = cfg.AI.APIKey
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("skipping live test: OPENAI_API_KEY / AI_API_KEY is empty")
	}

	baseURL := cfg.AI.BaseURL
	if baseURL == "" {
		baseURL = "https://api.dos.ai/v1"
	}
	llmModel := cfg.AI.LLMModel
	if llmModel == "" {
		llmModel = "dos-ai"
	}
	embeddingModel := cfg.AI.EmbeddingModel
	if embeddingModel == "" {
		embeddingModel = "qwen3-embedding-4b"
	}
	dimension := cfg.AI.EmbeddingDimension
	if dimension <= 0 {
		dimension = 2560
	}

	t.Logf("Testing Live DOS.AI with BaseURL=%s, LLM=%s, Embedding=%s, Dimension=%d", baseURL, llmModel, embeddingModel, dimension)

	setupDBWithAIConfig(t, llmModel, embeddingModel, baseURL, apiKey, dimension)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Test LLM Chat
	t.Run("LLM Chat Completion", func(t *testing.T) {
		result, err := ai.LLM.Chat(ctx, "You are Crove Desk AI Assistant. Reply in 1 concise sentence.", "Xin chào, bạn là ai và hỗ trợ được gì cho khách hàng?")
		if err != nil {
			t.Fatalf("LLM Chat failed: %v", err)
		}
		t.Logf("[SUCCESS] LLM Response: %s (Tokens used: prompt=%d, completion=%d)", result.Content, result.PromptTokens, result.CompletionTokens)
		if result.Content == "" {
			t.Errorf("expected non-empty response content")
		}
	})

	// 2. Test Embedding Generation
	t.Run("Embedding Generation", func(t *testing.T) {
		sampleText := "Crove Desk là giải pháp Customer Support & HelpDesk thông minh thuộc hệ sinh thái Crove Business OS."
		result, err := ai.Embedding.GenerateEmbedding(ctx, sampleText)
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}
		t.Logf("[SUCCESS] Embedding generated: %d dimensions (Tokens used=%d, ModelName=%s)", len(result.Vector), result.TokensUsed, result.ModelName)
		if len(result.Vector) == 0 {
			t.Fatalf("expected non-empty embedding vector")
		}
	})
}
