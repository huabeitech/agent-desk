package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"strings"
	"time"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
)

var KnowledgeRetrieveLogService = newKnowledgeRetrieveLogService()

func newKnowledgeRetrieveLogService() *knowledgeRetrieveLogService {
	return &knowledgeRetrieveLogService{}
}

type knowledgeRetrieveLogService struct {
}

type KnowledgeRetrieveLogFeedbackSummary struct {
	FeedbackCount          int64
	NegativeFeedbackCount  int64
	LatestFeedbackType     int
	LatestFeedbackTypeName string
	LatestFeedbackReason   string
}

const (
	KnowledgeRetrieveLogFeedbackStateNegative    = "negative"
	KnowledgeRetrieveLogFeedbackStateHasFeedback = "has_feedback"
	KnowledgeRetrieveLogFeedbackStateNoFeedback  = "no_feedback"
)

func (s *knowledgeRetrieveLogService) Get(id int64) *models.KnowledgeRetrieveLog {
	ret := &models.KnowledgeRetrieveLog{}
	if err := sqls.DB().First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (s *knowledgeRetrieveLogService) ApplyFeedbackStateFilter(cnd *sqls.Cnd, state string) *sqls.Cnd {
	if cnd == nil {
		cnd = sqls.NewCnd()
	}
	switch strings.TrimSpace(state) {
	case KnowledgeRetrieveLogFeedbackStateNegative:
		cnd.Where("EXISTS (SELECT 1 FROM knowledge_feedbacks WHERE knowledge_feedbacks.retrieve_log_id = knowledge_retrieve_logs.id AND knowledge_feedbacks.feedback_type <> ?)", int(enums.KnowledgeFeedbackTypeLike))
	case KnowledgeRetrieveLogFeedbackStateHasFeedback:
		cnd.Where("EXISTS (SELECT 1 FROM knowledge_feedbacks WHERE knowledge_feedbacks.retrieve_log_id = knowledge_retrieve_logs.id)")
	case KnowledgeRetrieveLogFeedbackStateNoFeedback:
		cnd.Where("NOT EXISTS (SELECT 1 FROM knowledge_feedbacks WHERE knowledge_feedbacks.retrieve_log_id = knowledge_retrieve_logs.id)")
	}
	return cnd
}

func (s *knowledgeRetrieveLogService) FindPageByParams(params *params.QueryParams) (list []models.KnowledgeRetrieveLog, paging *sqls.Paging) {
	cnd := &params.Cnd
	cnd.Find(sqls.DB(), &list)
	count := cnd.Count(sqls.DB(), &models.KnowledgeRetrieveLog{})
	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (s *knowledgeRetrieveLogService) FindHitsByRetrieveLogID(retrieveLogID int64) []models.KnowledgeRetrieveHit {
	if retrieveLogID <= 0 {
		return nil
	}
	var list []models.KnowledgeRetrieveHit
	sqls.DB().Where("retrieve_log_id = ?", retrieveLogID).Order("rank_no asc, id asc").Find(&list)
	return list
}

func (s *knowledgeRetrieveLogService) FindFeedbacksByRetrieveLogID(retrieveLogID int64) []models.KnowledgeFeedback {
	if retrieveLogID <= 0 {
		return nil
	}
	var list []models.KnowledgeFeedback
	sqls.DB().Where("retrieve_log_id = ?", retrieveLogID).Order("id desc").Find(&list)
	return list
}

func (s *knowledgeRetrieveLogService) FindFeedbackSummariesByRetrieveLogIDs(retrieveLogIDs []int64) map[int64]KnowledgeRetrieveLogFeedbackSummary {
	ret := make(map[int64]KnowledgeRetrieveLogFeedbackSummary, len(retrieveLogIDs))
	if len(retrieveLogIDs) == 0 {
		return ret
	}

	var feedbacks []models.KnowledgeFeedback
	sqls.DB().Where("retrieve_log_id in ?", retrieveLogIDs).Order("id desc").Find(&feedbacks)
	for _, item := range feedbacks {
		summary := ret[item.RetrieveLogID]
		summary.FeedbackCount++
		if item.FeedbackType != int(enums.KnowledgeFeedbackTypeLike) {
			summary.NegativeFeedbackCount++
		}
		if summary.LatestFeedbackType == 0 {
			summary.LatestFeedbackType = item.FeedbackType
			summary.LatestFeedbackTypeName = enums.GetKnowledgeFeedbackTypeLabel(enums.KnowledgeFeedbackType(item.FeedbackType))
			summary.LatestFeedbackReason = strings.TrimSpace(item.FeedbackReason)
			if summary.LatestFeedbackReason == "" {
				summary.LatestFeedbackReason = strings.TrimSpace(item.Remark)
			}
		}
		ret[item.RetrieveLogID] = summary
	}
	return ret
}

func (s *knowledgeRetrieveLogService) CreateFeedback(req request.CreateKnowledgeFeedbackRequest, operator *dto.AuthPrincipal) (*models.KnowledgeFeedback, error) {
	if operator == nil {
		return nil, errorsx.ForbiddenI18n("error.e0225")
	}
	if req.RetrieveLogID <= 0 || s.Get(req.RetrieveLogID) == nil {
		return nil, errorsx.InvalidParam("retrieveLogId is invalid")
	}
	if !isValidKnowledgeFeedbackType(req.FeedbackType) {
		return nil, errorsx.InvalidParam("feedbackType is invalid")
	}

	item := &models.KnowledgeFeedback{
		RetrieveLogID:  req.RetrieveLogID,
		FeedbackType:   req.FeedbackType,
		FeedbackReason: strings.TrimSpace(req.FeedbackReason),
		UserID:         operator.UserID,
		Remark:         strings.TrimSpace(req.Remark),
		CreatedAt:      time.Now(),
	}
	if err := sqls.DB().Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func isValidKnowledgeFeedbackType(feedbackType int) bool {
	switch enums.KnowledgeFeedbackType(feedbackType) {
	case enums.KnowledgeFeedbackTypeLike,
		enums.KnowledgeFeedbackTypeDislike,
		enums.KnowledgeFeedbackTypeNotHelpful,
		enums.KnowledgeFeedbackTypeWrongCitation,
		enums.KnowledgeFeedbackTypeOther:
		return true
	default:
		return false
	}
}
