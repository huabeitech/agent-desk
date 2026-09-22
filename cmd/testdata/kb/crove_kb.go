package kb

import (
	"agent-desk/cmd/testdata/seeds"
	"agent-desk/internal/ai/rag"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

type CroveInitResult struct {
	KnowledgeBaseID int64
	TotalFAQs       int
	CreatedFAQs     int
	UpdatedFAQs     int
}

// InitCroveDeskKB creates or updates the official Crove Desk Knowledge Base and seeds sample FAQs.
func InitCroveDeskKB() (*CroveInitResult, error) {
	faqSeeds := seeds.CroveDeskFAQSeeds()
	result := &CroveInitResult{
		TotalFAQs: len(faqSeeds),
	}

	var kbID int64
	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		kb, err := ensureCroveFAQKnowledgeBase(ctx.Tx)
		if err != nil {
			return err
		}
		kbID = kb.ID
		result.KnowledgeBaseID = kb.ID

		for _, faq := range faqSeeds {
			created, err := upsertCroveKnowledgeFAQ(ctx.Tx, kb.ID, faq)
			if err != nil {
				return err
			}
			if created {
				result.CreatedFAQs++
			} else {
				result.UpdatedFAQs++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Trigger indexing for all FAQs in this Knowledge Base
	faqs := repositories.KnowledgeFAQRepository.Find(sqls.DB(), sqls.NewCnd().Eq("knowledge_base_id", kbID))
	for _, f := range faqs {
		if err := rag.Index.IndexFAQByID(context.Background(), f.ID); err != nil {
			slog.Warn("failed to index FAQ into vector db", "faq_id", f.ID, "question", f.Question, "error", err)
		} else {
			slog.Info("successfully indexed FAQ into vector db", "faq_id", f.ID, "question", f.Question)
		}
	}

	return result, nil
}

func ensureCroveFAQKnowledgeBase(db *gorm.DB) (*models.KnowledgeBase, error) {
	now := time.Now()
	name := "Crove Desk - Customer Support & AI HelpDesk"
	item := repositories.KnowledgeBaseRepository.FindOne(db, sqls.NewCnd().Eq("name", name))
	if item == nil {
		item = &models.KnowledgeBase{
			Name:                  name,
			Description:           "Cơ sở tri thức và tập câu hỏi thường gặp (FAQ) chính thức của Crove Desk, phục vụ giải đáp thông tin sản phẩm, tính năng AI Agent, RAG, MCP và tích hợp Crove Business OS.",
			KnowledgeType:         "faq",
			Status:                enums.StatusOk,
			DefaultTopK:           10,
			DefaultScoreThreshold: 0.5,
			DefaultRerankLimit:    5,
			ChunkProvider:         "structured",
			ChunkTargetTokens:     300,
			ChunkMaxTokens:        400,
			ChunkOverlapTokens:    40,
			AnswerMode:            1,
			SortNo:                10,
			Remark:                "Official Crove Desk FAQ Knowledge Base",
			AuditFields: models.AuditFields{
				CreatedAt:      now,
				CreateUserID:   constants.SystemAuditUserID,
				CreateUserName: constants.SystemAuditUserName,
				UpdatedAt:      now,
				UpdateUserID:   constants.SystemAuditUserID,
				UpdateUserName: constants.SystemAuditUserName,
			},
		}
		if err := repositories.KnowledgeBaseRepository.Create(db, item); err != nil {
			return nil, err
		}
		return item, nil
	}

	_ = repositories.KnowledgeBaseRepository.Updates(db, item.ID, map[string]any{
		"description": "Cơ sở tri thức và tập câu hỏi thường gặp (FAQ) chính thức của Crove Desk, phục vụ giải đáp thông tin sản phẩm, tính năng AI Agent, RAG, MCP và tích hợp Crove Business OS.",
		"status":      enums.StatusOk,
		"updated_at":  now,
	})
	return item, nil
}

func upsertCroveKnowledgeFAQ(db *gorm.DB, knowledgeBaseID int64, seed seeds.CroveFAQItem) (bool, error) {
	now := time.Now()
	similarJSON, err := json.Marshal(seed.SimilarQuestions)
	if err != nil {
		similarJSON = []byte("[]")
	}

	items := repositories.KnowledgeFAQRepository.Find(db, sqls.NewCnd().
		Eq("knowledge_base_id", knowledgeBaseID).
		Eq("question", seed.Question))

	if len(items) == 0 {
		item := &models.KnowledgeFAQ{
			KnowledgeBaseID:  knowledgeBaseID,
			DirectoryID:      0,
			Question:         seed.Question,
			Answer:           seed.Answer,
			SimilarQuestions: string(similarJSON),
			IndexStatus:      enums.KnowledgeDocumentIndexStatusPending,
			Status:           enums.StatusOk,
			Remark:           seed.Remark,
			AuditFields: models.AuditFields{
				CreatedAt:      now,
				CreateUserID:   constants.SystemAuditUserID,
				CreateUserName: constants.SystemAuditUserName,
				UpdatedAt:      now,
				UpdateUserID:   constants.SystemAuditUserID,
				UpdateUserName: constants.SystemAuditUserName,
			},
		}
		if err := repositories.KnowledgeFAQRepository.Create(db, item); err != nil {
			return false, err
		}
		return true, nil
	}

	err = repositories.KnowledgeFAQRepository.Updates(db, items[0].ID, map[string]any{
		"answer":            seed.Answer,
		"similar_questions": string(similarJSON),
		"status":            enums.StatusOk,
		"remark":            seed.Remark,
		"updated_at":        now,
	})
	if err != nil {
		return false, err
	}
	return false, nil
}
