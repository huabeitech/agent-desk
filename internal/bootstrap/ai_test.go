package bootstrap

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func setupTestDBForAI(t *testing.T) *gorm.DB {
	dbName := fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	if err := db.AutoMigrate(models.AIConfig{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func TestInitAI(t *testing.T) {
	db := setupTestDBForAI(t)

	cfg := &config.Config{
		AI: config.AIConfig{
			Provider:           "openai",
			BaseURL:            "https://api.deepseek.com/v1",
			APIKey:             "sk-test-deepseek-key",
			LLMModel:           "deepseek-chat",
			EmbeddingModel:     "text-embedding-3-small",
			EmbeddingDimension: 1536,
		},
	}

	if err := InitAI(cfg); err != nil {
		t.Fatalf("InitAI failed: %v", err)
	}

	llm := repositories.AIConfigRepository.GetEnabled(db, enums.AIModelTypeLLM)
	if llm == nil {
		t.Fatalf("expected enabled LLM config, got nil")
	}
	if llm.BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("LLM BaseURL = %q, want https://api.deepseek.com/v1", llm.BaseURL)
	}
	if llm.ModelName != "deepseek-chat" {
		t.Errorf("LLM ModelName = %q, want deepseek-chat", llm.ModelName)
	}
	if llm.APIKey != "sk-test-deepseek-key" {
		t.Errorf("LLM APIKey = %q, want sk-test-deepseek-key", llm.APIKey)
	}

	embedding := repositories.AIConfigRepository.GetEnabled(db, enums.AIModelTypeEmbedding)
	if embedding == nil {
		t.Fatalf("expected enabled Embedding config, got nil")
	}
	if embedding.ModelName != "text-embedding-3-small" {
		t.Errorf("Embedding ModelName = %q, want text-embedding-3-small", embedding.ModelName)
	}
	if embedding.Dimension != 1536 {
		t.Errorf("Embedding Dimension = %d, want 1536", embedding.Dimension)
	}
}

func TestInitAI_DOSAI(t *testing.T) {
	db := setupTestDBForAI(t)

	cfg := &config.Config{
		AI: config.AIConfig{
			Provider:           "openai",
			BaseURL:            "https://api.dos.ai/v1",
			APIKey:             "dos_sk_test_key",
			LLMModel:           "dos-ai",
			EmbeddingModel:     "qwen3-embedding-4b",
			EmbeddingDimension: 2560,
		},
	}

	if err := InitAI(cfg); err != nil {
		t.Fatalf("InitAI failed: %v", err)
	}

	llm := repositories.AIConfigRepository.GetEnabled(db, enums.AIModelTypeLLM)
	if llm == nil {
		t.Fatalf("expected enabled LLM config, got nil")
	}
	if llm.BaseURL != "https://api.dos.ai/v1" {
		t.Errorf("LLM BaseURL = %q, want https://api.dos.ai/v1", llm.BaseURL)
	}
	if llm.ModelName != "dos-ai" {
		t.Errorf("LLM ModelName = %q, want dos-ai", llm.ModelName)
	}

	embedding := repositories.AIConfigRepository.GetEnabled(db, enums.AIModelTypeEmbedding)
	if embedding == nil {
		t.Fatalf("expected enabled Embedding config, got nil")
	}
	if embedding.ModelName != "qwen3-embedding-4b" {
		t.Errorf("Embedding ModelName = %q, want qwen3-embedding-4b", embedding.ModelName)
	}
	if embedding.Dimension != 2560 {
		t.Errorf("Embedding Dimension = %d, want 2560", embedding.Dimension)
	}
}
