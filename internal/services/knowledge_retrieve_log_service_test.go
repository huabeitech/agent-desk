package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func TestKnowledgeRetrieveLogServiceCreateFeedback(t *testing.T) {
	setupKnowledgeRetrieveLogServiceTestDB(t)
	logItem := createKnowledgeRetrieveLogServiceTestLog(t)

	feedback, err := KnowledgeRetrieveLogService.CreateFeedback(request.CreateKnowledgeFeedbackRequest{
		RetrieveLogID:  logItem.ID,
		FeedbackType:   int(enums.KnowledgeFeedbackTypeWrongCitation),
		FeedbackReason: " 引用内容不匹配 ",
		Remark:         " 需要补充门店售后政策 ",
	}, &dto.AuthPrincipal{UserID: 42, Username: "quality"})
	if err != nil {
		t.Fatalf("CreateFeedback() error = %v", err)
	}

	if feedback.ID == 0 {
		t.Fatal("feedback id was not assigned")
	}
	if feedback.UserID != 42 {
		t.Fatalf("feedback.UserID = %d, want 42", feedback.UserID)
	}
	if feedback.FeedbackReason != "引用内容不匹配" {
		t.Fatalf("feedback reason was not trimmed: %q", feedback.FeedbackReason)
	}
	if feedback.Remark != "需要补充门店售后政策" {
		t.Fatalf("feedback remark was not trimmed: %q", feedback.Remark)
	}

	list := KnowledgeRetrieveLogService.FindFeedbacksByRetrieveLogID(logItem.ID)
	if len(list) != 1 || list[0].ID != feedback.ID {
		t.Fatalf("FindFeedbacksByRetrieveLogID() = %+v, want created feedback", list)
	}

	_, err = KnowledgeRetrieveLogService.CreateFeedback(request.CreateKnowledgeFeedbackRequest{
		RetrieveLogID:  logItem.ID,
		FeedbackType:   int(enums.KnowledgeFeedbackTypeLike),
		FeedbackReason: "后续回答清楚",
	}, &dto.AuthPrincipal{UserID: 43, Username: "quality2"})
	if err != nil {
		t.Fatalf("CreateFeedback() like error = %v", err)
	}

	summary := KnowledgeRetrieveLogService.FindFeedbackSummariesByRetrieveLogIDs([]int64{logItem.ID})[logItem.ID]
	if summary.FeedbackCount != 2 {
		t.Fatalf("summary.FeedbackCount = %d, want 2", summary.FeedbackCount)
	}
	if summary.NegativeFeedbackCount != 1 {
		t.Fatalf("summary.NegativeFeedbackCount = %d, want 1", summary.NegativeFeedbackCount)
	}
	if summary.LatestFeedbackType != int(enums.KnowledgeFeedbackTypeLike) {
		t.Fatalf("summary.LatestFeedbackType = %d, want like", summary.LatestFeedbackType)
	}
	if summary.LatestFeedbackReason != "后续回答清楚" {
		t.Fatalf("summary.LatestFeedbackReason = %q, want latest reason", summary.LatestFeedbackReason)
	}
}

func TestKnowledgeRetrieveLogServiceCreateFeedbackRejectsInvalidInput(t *testing.T) {
	setupKnowledgeRetrieveLogServiceTestDB(t)
	logItem := createKnowledgeRetrieveLogServiceTestLog(t)
	operator := &dto.AuthPrincipal{UserID: 7, Username: "admin"}

	if _, err := KnowledgeRetrieveLogService.CreateFeedback(request.CreateKnowledgeFeedbackRequest{
		RetrieveLogID: logItem.ID,
		FeedbackType:  999,
	}, operator); err == nil {
		t.Fatal("CreateFeedback() error is nil, want invalid feedback type error")
	}

	if _, err := KnowledgeRetrieveLogService.CreateFeedback(request.CreateKnowledgeFeedbackRequest{
		RetrieveLogID: 99999,
		FeedbackType:  int(enums.KnowledgeFeedbackTypeLike),
	}, operator); err == nil {
		t.Fatal("CreateFeedback() error is nil, want invalid retrieve log id error")
	}

	if _, err := KnowledgeRetrieveLogService.CreateFeedback(request.CreateKnowledgeFeedbackRequest{
		RetrieveLogID: logItem.ID,
		FeedbackType:  int(enums.KnowledgeFeedbackTypeLike),
	}, nil); err == nil {
		t.Fatal("CreateFeedback() error is nil, want permission error")
	}
}

func TestKnowledgeRetrieveLogFeedbackStateFilter(t *testing.T) {
	setupKnowledgeRetrieveLogServiceTestDB(t)
	logs := []models.KnowledgeRetrieveLog{
		{KnowledgeBaseID: 1, RequestID: "req-like", Question: "点赞问题", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: time.Now()},
		{KnowledgeBaseID: 1, RequestID: "req-negative", Question: "负反馈问题", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: time.Now()},
		{KnowledgeBaseID: 1, RequestID: "req-none", Question: "无反馈问题", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: time.Now()},
	}
	if err := sqls.DB().Create(&logs).Error; err != nil {
		t.Fatalf("create retrieve logs: %v", err)
	}
	feedbacks := []models.KnowledgeFeedback{
		{RetrieveLogID: logs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeLike), FeedbackReason: "清楚", CreatedAt: time.Now()},
		{RetrieveLogID: logs[1].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeWrongCitation), FeedbackReason: "引用不准", CreatedAt: time.Now()},
	}
	if err := sqls.DB().Create(&feedbacks).Error; err != nil {
		t.Fatalf("create feedbacks: %v", err)
	}

	tests := []struct {
		name  string
		state string
		want  []int64
	}{
		{name: "negative", state: KnowledgeRetrieveLogFeedbackStateNegative, want: []int64{logs[1].ID}},
		{name: "has_feedback", state: KnowledgeRetrieveLogFeedbackStateHasFeedback, want: []int64{logs[0].ID, logs[1].ID}},
		{name: "no_feedback", state: KnowledgeRetrieveLogFeedbackStateNoFeedback, want: []int64{logs[2].ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnd := KnowledgeRetrieveLogService.ApplyFeedbackStateFilter(sqls.NewCnd().Asc("id"), tt.state)
			var got []models.KnowledgeRetrieveLog
			cnd.Find(sqls.DB(), &got)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d logs, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, item := range got {
				if item.ID != tt.want[i] {
					t.Fatalf("got id[%d]=%d, want %d", i, item.ID, tt.want[i])
				}
			}
		})
	}
}

func TestKnowledgeFAQServiceCreateDraftFromRetrieveLog(t *testing.T) {
	setupKnowledgeFAQDraftFromRetrieveLogTestDB(t)
	kb := &models.KnowledgeBase{
		Name:          "FAQ KB",
		KnowledgeType: string(enums.KnowledgeBaseTypeFAQ),
		Status:        enums.StatusOk,
	}
	if err := sqls.DB().Create(kb).Error; err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}
	logItem := &models.KnowledgeRetrieveLog{
		KnowledgeBaseID: kb.ID,
		Question:        "老人腰不好怎么选床垫？",
		RewriteQuestion: "老人腰不好适合什么床垫？",
		Answer:          "建议优先看支撑分区款。",
		AnswerStatus:    int(enums.KnowledgeAnswerStatusNormal),
		CreatedAt:       time.Now(),
	}
	if err := sqls.DB().Create(logItem).Error; err != nil {
		t.Fatalf("create retrieve log: %v", err)
	}

	draft, err := KnowledgeFAQService.CreateDraftFromRetrieveLog(request.CreateKnowledgeFAQDraftFromRetrieveLogRequest{
		RetrieveLogID: logItem.ID,
		Remark:        "负反馈后补充知识",
	}, &dto.AuthPrincipal{UserID: 9, Username: "operator"})
	if err != nil {
		t.Fatalf("CreateDraftFromRetrieveLog() error = %v", err)
	}
	if draft.ID == 0 {
		t.Fatal("draft id was not assigned")
	}
	if draft.Status != enums.StatusDisabled {
		t.Fatalf("draft.Status = %v, want disabled", draft.Status)
	}
	if draft.KnowledgeBaseID != kb.ID || draft.Question != logItem.Question || draft.Answer != logItem.Answer {
		t.Fatalf("unexpected draft content: %#v", draft)
	}
	if draft.IndexStatus != enums.KnowledgeDocumentIndexStatusPending {
		t.Fatalf("draft.IndexStatus = %s, want pending", draft.IndexStatus)
	}
	if draft.UpdateUserName != "operator" || draft.CreateUserName != "operator" {
		t.Fatalf("draft audit fields not set: %#v", draft.AuditFields)
	}
	if draft.Remark == "" || !strings.Contains(draft.Remark, "来源检索日志") {
		t.Fatalf("draft remark missing source log: %q", draft.Remark)
	}
	var similar []string
	if err := json.Unmarshal([]byte(draft.SimilarQuestions), &similar); err != nil {
		t.Fatalf("unmarshal similar questions: %v", err)
	}
	if len(similar) != 1 || similar[0] != logItem.RewriteQuestion {
		t.Fatalf("unexpected similar questions: %#v", similar)
	}

	reused, err := KnowledgeFAQService.CreateDraftFromRetrieveLog(request.CreateKnowledgeFAQDraftFromRetrieveLogRequest{
		RetrieveLogID: logItem.ID,
	}, &dto.AuthPrincipal{UserID: 10, Username: "operator2"})
	if err != nil {
		t.Fatalf("CreateDraftFromRetrieveLog() reuse error = %v", err)
	}
	if reused.ID != draft.ID {
		t.Fatalf("expected existing draft to be reused, got %d want %d", reused.ID, draft.ID)
	}
}

func TestKnowledgeFAQServiceBatchCreateDraftsFromRetrieveLogs(t *testing.T) {
	setupKnowledgeFAQDraftFromRetrieveLogTestDB(t)
	kb := &models.KnowledgeBase{
		Name:          "FAQ KB",
		KnowledgeType: string(enums.KnowledgeBaseTypeFAQ),
		Status:        enums.StatusOk,
	}
	if err := sqls.DB().Create(kb).Error; err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}
	now := time.Now()
	logs := []models.KnowledgeRetrieveLog{
		{KnowledgeBaseID: kb.ID, Question: "周末活动能叠加吗？", Answer: "", AnswerStatus: int(enums.KnowledgeAnswerStatusFallback), CreatedAt: now.Add(3 * time.Minute)},
		{KnowledgeBaseID: kb.ID, Question: "老人腰不好怎么选？", Answer: "建议看分区支撑。", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: now.Add(2 * time.Minute)},
		{KnowledgeBaseID: kb.ID, Question: "旧问题已有 FAQ", Answer: "已有答案", AnswerStatus: int(enums.KnowledgeAnswerStatusNoAnswer), CreatedAt: now.Add(time.Minute)},
		{KnowledgeBaseID: kb.ID, Question: "正常回答不进候选", Answer: "正常答案", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: now},
	}
	for i := range logs {
		if err := sqls.DB().Create(&logs[i]).Error; err != nil {
			t.Fatalf("create retrieve log: %v", err)
		}
	}
	if err := sqls.DB().Create(&models.KnowledgeFeedback{
		RetrieveLogID:  logs[1].ID,
		FeedbackType:   int(enums.KnowledgeFeedbackTypeDislike),
		FeedbackReason: "不准确",
		CreatedAt:      now.Add(4 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("create feedback: %v", err)
	}
	existing := models.KnowledgeFAQ{
		KnowledgeBaseID: kb.ID,
		Question:        "旧问题已有 FAQ",
		Answer:          "旧答案",
		Status:          enums.StatusDisabled,
	}
	if err := sqls.DB().Create(&existing).Error; err != nil {
		t.Fatalf("create existing faq: %v", err)
	}

	ret, err := KnowledgeFAQService.BatchCreateDraftsFromRetrieveLogs(request.BatchCreateKnowledgeFAQDraftsFromRetrieveLogsRequest{
		KnowledgeBaseID:          kb.ID,
		IncludeNegativeFeedbacks: true,
		Limit:                    20,
		Remark:                   "批量生成待确认 FAQ 草稿",
	}, &dto.AuthPrincipal{UserID: 12, Username: "quality"})
	if err != nil {
		t.Fatalf("BatchCreateDraftsFromRetrieveLogs() error = %v", err)
	}
	if ret.TotalCandidates != 3 || ret.CreatedCount != 2 || ret.ReusedCount != 1 || ret.SkippedCount != 0 {
		t.Fatalf("unexpected batch result: %#v", ret)
	}
	if len(ret.DraftIDs) != 3 {
		t.Fatalf("unexpected draft ids: %#v", ret.DraftIDs)
	}
	var faqCount int64
	sqls.DB().Model(&models.KnowledgeFAQ{}).
		Where("knowledge_base_id = ? AND status <> ?", kb.ID, enums.StatusDeleted).
		Count(&faqCount)
	if faqCount != 3 {
		t.Fatalf("faq count = %d, want 3", faqCount)
	}
}

func TestKnowledgeFAQServiceUpdateStatusRejectsInvalidStatus(t *testing.T) {
	setupKnowledgeFAQDraftFromRetrieveLogTestDB(t)
	faq := &models.KnowledgeFAQ{
		KnowledgeBaseID: 1,
		Question:        "问题",
		Answer:          "答案",
		Status:          enums.StatusDisabled,
	}
	if err := sqls.DB().Create(faq).Error; err != nil {
		t.Fatalf("create faq: %v", err)
	}

	err := KnowledgeFAQService.UpdateStatus(faq.ID, enums.StatusDeleted, &dto.AuthPrincipal{UserID: 1, Username: "admin"})
	if err == nil {
		t.Fatal("UpdateStatus() error is nil, want invalid status error")
	}
}

func setupKnowledgeRetrieveLogServiceTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.KnowledgeRetrieveLog{}, &models.KnowledgeFeedback{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
}

func setupKnowledgeFAQDraftFromRetrieveLogTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.KnowledgeBase{}, &models.KnowledgeRetrieveLog{}, &models.KnowledgeFeedback{}, &models.KnowledgeFAQ{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
}

func createKnowledgeRetrieveLogServiceTestLog(t *testing.T) *models.KnowledgeRetrieveLog {
	t.Helper()
	item := &models.KnowledgeRetrieveLog{
		KnowledgeBaseID: 1,
		Channel:         string(enums.KnowledgeRetrieveChannelIM),
		Scene:           string(enums.KnowledgeRetrieveSceneQA),
		RequestID:       "req-feedback-test",
		Question:        "慕斯床垫怎么选？",
		AnswerStatus:    int(enums.KnowledgeAnswerStatusNormal),
		CreatedAt:       time.Now(),
	}
	if err := sqls.DB().Create(item).Error; err != nil {
		t.Fatalf("create retrieve log: %v", err)
	}
	return item
}
