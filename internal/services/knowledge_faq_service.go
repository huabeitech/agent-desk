package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/ai/rag"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
)

var KnowledgeFAQService = newKnowledgeFAQService()

func newKnowledgeFAQService() *knowledgeFAQService {
	return &knowledgeFAQService{}
}

type knowledgeFAQService struct{}

func (s *knowledgeFAQService) Get(id int64) *models.KnowledgeFAQ {
	return repositories.KnowledgeFAQRepository.Get(sqls.DB(), id)
}

func (s *knowledgeFAQService) FindPageByCnd(cnd *sqls.Cnd) (list []models.KnowledgeFAQ, paging *sqls.Paging) {
	return repositories.KnowledgeFAQRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *knowledgeFAQService) FindPageByParams(queryParams *params.QueryParams) (list []models.KnowledgeFAQ, paging *sqls.Paging) {
	return repositories.KnowledgeFAQRepository.FindPageByParams(sqls.DB(), queryParams)
}

func (s *knowledgeFAQService) Count(cnd *sqls.Cnd) int64 {
	return repositories.KnowledgeFAQRepository.Count(sqls.DB(), cnd)
}

func (s *knowledgeFAQService) CreateKnowledgeFAQ(req request.CreateKnowledgeFAQRequest, operator *dto.AuthPrincipal) (*models.KnowledgeFAQ, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	kb, err := s.requireFAQKnowledgeBase(req.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}
	if _, err := KnowledgeDirectoryService.RequireUsableDirectory(req.KnowledgeBaseID, req.DirectoryID); err != nil {
		return nil, err
	}
	item, err := s.buildKnowledgeFAQModel(req)
	if err != nil {
		return nil, err
	}
	item.Status = kb.Status
	item.IndexStatus = enums.KnowledgeDocumentIndexStatusPending
	item.IndexError = ""
	item.IndexedAt = nil
	item.AuditFields = utils.BuildAuditFields(operator)
	if err := repositories.KnowledgeFAQRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	if err := rag.Index.IndexFAQByID(context.Background(), item.ID); err != nil {
		slog.Error("failed to index created knowledge faq", "faq_id", item.ID, "error", err)
		return item, nil
	}
	return s.Get(item.ID), nil
}

func (s *knowledgeFAQService) CreateDraftFromRetrieveLog(req request.CreateKnowledgeFAQDraftFromRetrieveLogRequest, operator *dto.AuthPrincipal) (*models.KnowledgeFAQ, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	logItem := KnowledgeRetrieveLogService.Get(req.RetrieveLogID)
	if logItem == nil {
		return nil, errorsx.InvalidParamI18n("error.e0241")
	}
	if _, err := s.requireFAQKnowledgeBase(logItem.KnowledgeBaseID); err != nil {
		return nil, err
	}
	question := strings.TrimSpace(logItem.Question)
	if question == "" {
		question = strings.TrimSpace(logItem.RewriteQuestion)
	}
	if question == "" {
		return nil, errorsx.InvalidParamI18n("error.e0340")
	}

	var existing models.KnowledgeFAQ
	if err := sqls.DB().
		Where("knowledge_base_id = ? AND question = ? AND status <> ?", logItem.KnowledgeBaseID, question, enums.StatusDeleted).
		Order("id desc").
		First(&existing).Error; err == nil {
		return &existing, nil
	}

	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		answer = strings.TrimSpace(logItem.Answer)
	}
	if answer == "" {
		answer = "待补充标准答案"
	}
	similarQuestions := []string{}
	rewriteQuestion := strings.TrimSpace(logItem.RewriteQuestion)
	if rewriteQuestion != "" && rewriteQuestion != question {
		similarQuestions = append(similarQuestions, rewriteQuestion)
	}
	similarQuestionsJSON, err := json.Marshal(similarQuestions)
	if err != nil {
		return nil, errorsx.InvalidParamI18n("error.e0280")
	}
	remark := strings.TrimSpace(req.Remark)
	if remark == "" {
		remark = "由知识检索日志生成的待确认 FAQ 草稿"
	}
	if req.RetrieveLogID > 0 {
		remark = strings.TrimSpace(remark + "\n来源检索日志：" + strconv.FormatInt(req.RetrieveLogID, 10))
	}

	item := &models.KnowledgeFAQ{
		KnowledgeBaseID:  logItem.KnowledgeBaseID,
		Question:         question,
		Answer:           answer,
		SimilarQuestions: string(similarQuestionsJSON),
		Status:           enums.StatusDisabled,
		IndexStatus:      enums.KnowledgeDocumentIndexStatusPending,
		IndexError:       "",
		IndexedAt:        nil,
		Remark:           remark,
		AuditFields:      utils.BuildAuditFields(operator),
	}
	if err := repositories.KnowledgeFAQRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return s.Get(item.ID), nil
}

func (s *knowledgeFAQService) BatchCreateDraftsFromRetrieveLogs(req request.BatchCreateKnowledgeFAQDraftsFromRetrieveLogsRequest, operator *dto.AuthPrincipal) (response.KnowledgeFAQDraftBatchCreateResponse, error) {
	ret := response.KnowledgeFAQDraftBatchCreateResponse{
		DraftIDs: make([]int64, 0),
		Skipped:  make([]response.KnowledgeFAQDraftBatchSkipReason, 0),
	}
	if operator == nil {
		return ret, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if _, err := s.requireFAQKnowledgeBase(req.KnowledgeBaseID); err != nil {
		return ret, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	answerStatuses := req.AnswerStatuses
	if len(answerStatuses) == 0 {
		answerStatuses = []int{
			int(enums.KnowledgeAnswerStatusNoAnswer),
			int(enums.KnowledgeAnswerStatusFallback),
			int(enums.KnowledgeAnswerStatusBlocked),
		}
	}

	db := sqls.DB()
	negativeFeedbackLogIDs := db.Model(&models.KnowledgeFeedback{}).
		Select("retrieve_log_id").
		Where("feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike))
	query := db.Model(&models.KnowledgeRetrieveLog{}).
		Where("knowledge_base_id = ?", req.KnowledgeBaseID).
		Where("(question <> '' OR rewrite_question <> '')")
	if req.IncludeNegativeFeedbacks {
		query = query.Where("(answer_status IN ? OR id IN (?))", answerStatuses, negativeFeedbackLogIDs)
	} else {
		query = query.Where("answer_status IN ?", answerStatuses)
	}
	var logs []models.KnowledgeRetrieveLog
	query.Order("created_at desc, id desc").
		Limit(limit).
		Find(&logs)
	ret.TotalCandidates = int64(len(logs))

	for _, logItem := range logs {
		question := strings.TrimSpace(logItem.Question)
		if question == "" {
			question = strings.TrimSpace(logItem.RewriteQuestion)
		}
		if question == "" {
			ret.SkippedCount++
			ret.Skipped = append(ret.Skipped, response.KnowledgeFAQDraftBatchSkipReason{
				RetrieveLogID: logItem.ID,
				Reason:        "问题为空",
			})
			continue
		}
		var existing models.KnowledgeFAQ
		if err := db.Where("knowledge_base_id = ? AND question = ? AND status <> ?", logItem.KnowledgeBaseID, question, enums.StatusDeleted).
			Order("id desc").
			First(&existing).Error; err == nil {
			ret.ReusedCount++
			ret.DraftIDs = append(ret.DraftIDs, existing.ID)
			continue
		}
		draft, err := s.CreateDraftFromRetrieveLog(request.CreateKnowledgeFAQDraftFromRetrieveLogRequest{
			RetrieveLogID: logItem.ID,
			Answer:        req.Answer,
			Remark:        req.Remark,
		}, operator)
		if err != nil {
			ret.SkippedCount++
			ret.Skipped = append(ret.Skipped, response.KnowledgeFAQDraftBatchSkipReason{
				RetrieveLogID: logItem.ID,
				Question:      question,
				Reason:        err.Error(),
			})
			continue
		}
		ret.CreatedCount++
		ret.DraftIDs = append(ret.DraftIDs, draft.ID)
	}
	return ret, nil
}

func (s *knowledgeFAQService) UpdateKnowledgeFAQ(req request.UpdateKnowledgeFAQRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	current := s.Get(req.ID)
	if current == nil {
		return errorsx.InvalidParamI18n("error.e0025")
	}
	if _, err := s.requireFAQKnowledgeBase(req.KnowledgeBaseID); err != nil {
		return err
	}
	if _, err := KnowledgeDirectoryService.RequireUsableDirectory(req.KnowledgeBaseID, req.DirectoryID); err != nil {
		return err
	}
	item, err := s.buildKnowledgeFAQModel(req.CreateKnowledgeFAQRequest)
	if err != nil {
		return err
	}
	if err := repositories.KnowledgeFAQRepository.Updates(sqls.DB(), req.ID, map[string]any{
		"knowledge_base_id": item.KnowledgeBaseID,
		"directory_id":      item.DirectoryID,
		"question":          item.Question,
		"answer":            item.Answer,
		"similar_questions": item.SimilarQuestions,
		"index_status":      enums.KnowledgeDocumentIndexStatusPending,
		"indexed_at":        nil,
		"index_error":       "",
		"remark":            item.Remark,
		"update_user_id":    operator.UserID,
		"update_user_name":  operator.Username,
		"updated_at":        time.Now(),
	}); err != nil {
		return err
	}
	return rag.Index.IndexFAQByID(context.Background(), req.ID)
}

func (s *knowledgeFAQService) UpdateStatus(id int64, status enums.Status, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	current := s.Get(id)
	if current == nil {
		return errorsx.InvalidParamI18n("error.e0025")
	}
	if status != enums.StatusOk && status != enums.StatusDisabled {
		return errorsx.InvalidParamI18n("error.e0254")
	}
	if current.Status == status {
		return nil
	}
	if err := repositories.KnowledgeFAQRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           status,
		"index_status":     enums.KnowledgeDocumentIndexStatusPending,
		"indexed_at":       nil,
		"index_error":      "",
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	}); err != nil {
		return err
	}
	if status == enums.StatusDisabled {
		return rag.Index.RemoveFAQIndex(context.Background(), id)
	}
	return rag.Index.IndexFAQByID(context.Background(), id)
}

func (s *knowledgeFAQService) DeleteKnowledgeFAQ(id int64) error {
	current := s.Get(id)
	if current == nil {
		return errorsx.InvalidParamI18n("error.e0025")
	}
	if err := repositories.KnowledgeFAQRepository.Delete(sqls.DB(), id); err != nil {
		return err
	}
	return rag.Index.RemoveFAQIndex(context.Background(), id)
}

func (s *knowledgeFAQService) BatchMoveKnowledgeFAQs(req request.BatchMoveKnowledgeFAQRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	ids := uniquePositiveIDs(req.IDs)
	if len(ids) == 0 {
		return errorsx.InvalidParamI18n("error.e0332")
	}
	if _, err := s.requireFAQKnowledgeBase(req.KnowledgeBaseID); err != nil {
		return err
	}
	if _, err := KnowledgeDirectoryService.RequireUsableDirectory(req.KnowledgeBaseID, req.DirectoryID); err != nil {
		return err
	}
	for _, id := range ids {
		current := s.Get(id)
		if current == nil {
			return errorsx.InvalidParamI18n("error.e0025")
		}
		if current.KnowledgeBaseID != req.KnowledgeBaseID {
			return errorsx.InvalidParamI18n("error.e0138")
		}
	}
	now := time.Now()
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		for _, id := range ids {
			if err := repositories.KnowledgeFAQRepository.Updates(ctx.Tx, id, map[string]any{
				"directory_id":     req.DirectoryID,
				"index_status":     enums.KnowledgeDocumentIndexStatusPending,
				"indexed_at":       nil,
				"index_error":      "",
				"update_user_id":   operator.UserID,
				"update_user_name": operator.Username,
				"updated_at":       now,
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	for _, id := range ids {
		if err := rag.Index.IndexFAQByID(context.Background(), id); err != nil {
			slog.Error("failed to reindex moved knowledge faq", "faq_id", id, "error", err)
		}
	}
	return nil
}

func (s *knowledgeFAQService) BatchDeleteKnowledgeFAQs(req request.BatchDeleteKnowledgeFAQRequest) error {
	ids := uniquePositiveIDs(req.IDs)
	if len(ids) == 0 {
		return errorsx.InvalidParamI18n("error.e0330")
	}
	for _, id := range ids {
		if err := s.DeleteKnowledgeFAQ(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *knowledgeFAQService) buildKnowledgeFAQModel(req request.CreateKnowledgeFAQRequest) (*models.KnowledgeFAQ, error) {
	if req.KnowledgeBaseID <= 0 {
		return nil, errorsx.InvalidParamI18n("error.e0283")
	}
	if req.Question == "" {
		return nil, errorsx.InvalidParamI18n("error.e0340")
	}
	if req.Answer == "" {
		return nil, errorsx.InvalidParamI18n("error.e0292")
	}
	similarQuestions, err := json.Marshal(normalizeSimilarQuestions(req.SimilarQuestions))
	if err != nil {
		return nil, errorsx.InvalidParamI18n("error.e0280")
	}
	return &models.KnowledgeFAQ{
		KnowledgeBaseID:  req.KnowledgeBaseID,
		DirectoryID:      req.DirectoryID,
		Question:         req.Question,
		Answer:           req.Answer,
		SimilarQuestions: string(similarQuestions),
		Remark:           req.Remark,
	}, nil
}

func (s *knowledgeFAQService) requireFAQKnowledgeBase(knowledgeBaseID int64) (*models.KnowledgeBase, error) {
	kb := KnowledgeBaseService.Get(knowledgeBaseID)
	if kb == nil {
		return nil, errorsx.InvalidParamI18n("error.e0283")
	}
	if kb.KnowledgeType != "faq" {
		return nil, errorsx.InvalidParamI18n("error.e0199")
	}
	return kb, nil
}

func normalizeSimilarQuestions(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		value := item
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}
