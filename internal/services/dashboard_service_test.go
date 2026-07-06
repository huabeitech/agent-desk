package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func TestDashboardTextUsesEnglishLocale(t *testing.T) {
	t.Parallel()

	if got := dashboardText(i18nx.LocaleEnUS, "alert.pendingLongWait.title"); got != "Queued conversations are piling up" {
		t.Fatalf("dashboardText() = %q", got)
	}
	if got := dashboardText(i18nx.LocaleZhCN, "alert.pendingLongWait.title"); got != "待接入会话堆积" {
		t.Fatalf("dashboardText() = %q", got)
	}
}

func TestConversationStatusLabelUsesEnglishLocale(t *testing.T) {
	t.Parallel()

	if got := conversationStatusLabel(enums.IMConversationStatusPending, i18nx.LocaleEnUS); got != "Queued" {
		t.Fatalf("conversationStatusLabel() = %q", got)
	}
	if got := conversationStatusLabel(enums.IMConversationStatusPending, i18nx.LocaleZhCN); got != "待接入" {
		t.Fatalf("conversationStatusLabel() = %q", got)
	}
}

func TestBuildTopKnowledgeQuestions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.KnowledgeRetrieveLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.Local)
	logs := []models.KnowledgeRetrieveLog{
		{Question: "老人腰不好，床垫是不是越硬越好？", AnswerStatus: 1, CreatedAt: now},
		{Question: " 老人腰不好，床垫是不是越硬越好？ ", AnswerStatus: 2, CreatedAt: now.Add(time.Minute)},
		{Question: "周末有什么活动", AnswerStatus: 3, CreatedAt: now.Add(2 * time.Minute)},
		{Question: "昨天的问题", AnswerStatus: 2, CreatedAt: now.AddDate(0, 0, -1)},
	}
	for i := range logs {
		if err := db.Create(&logs[i]).Error; err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	all := buildTopKnowledgeQuestions(db, startOfDay(now), startOfDay(now).AddDate(0, 0, 1), nil)
	if len(all) == 0 || all[0].Name != "老人腰不好，床垫是不是越硬越好" || all[0].Count != 2 {
		t.Fatalf("unexpected top questions: %#v", all)
	}
	unanswered := buildTopKnowledgeQuestions(db, startOfDay(now), startOfDay(now).AddDate(0, 0, 1), []int{2, 3, 4})
	if len(unanswered) != 2 {
		t.Fatalf("unexpected unanswered questions: %#v", unanswered)
	}
}

func TestDailyBusinessReportIncludesPriorityFollowUps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.Message{},
		&models.SalesLead{},
		&models.Product{},
		&models.Promotion{},
		&models.KnowledgeRetrieveLog{},
		&models.KnowledgeFeedback{},
		&models.KnowledgeFAQ{},
		&models.User{},
		&models.Ticket{},
		&models.TicketProgress{},
		&models.SystemConfig{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	reportDay := time.Date(2026, 7, 6, 9, 0, 0, 0, time.Local)
	dayStart := startOfDay(reportDay)
	overdue := dayStart.Add(-2 * time.Hour)
	today := dayStart.Add(10 * time.Hour)
	future := dayStart.AddDate(0, 0, 2)
	appointmentOverdue := dayStart.Add(-4 * time.Hour)
	appointmentToday := dayStart.Add(15 * time.Hour)
	ticketHandledAt := dayStart.Add(5 * time.Hour)
	owner := models.User{Username: "顾问A", Status: enums.StatusOk}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	leads := []models.SalesLead{
		{CustomerName: "逾期客户", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, OwnerUserID: owner.ID, NextFollowUpAt: &overdue, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(-24 * time.Hour)}},
		{CustomerName: "今日客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, NextFollowUpAt: &today, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour)}},
		{CustomerName: "未排计划高意向", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(2 * time.Hour)}},
		{CustomerName: "未来客户", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, NextFollowUpAt: &future, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(3 * time.Hour)}},
		{CustomerName: "逾期预约客户", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentOverdue, AppointmentStore: "徐汇店", AuditFields: models.AuditFields{CreatedAt: dayStart.Add(-24 * time.Hour)}},
		{CustomerName: "今日预约客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentToday, AppointmentStore: "静安店", AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour)}},
		{CustomerName: "未定预约客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentTimeText: "周末方便", AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour)}},
		{CustomerName: "已转化预约客户", Status: enums.SalesLeadStatusConverted, IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentToday, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour), UpdatedAt: dayStart.Add(16 * time.Hour)}},
	}
	for i := range leads {
		if err := db.Create(&leads[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}
	tickets := []models.Ticket{
		{
			TicketNo:          "TK202607060001",
			Title:             "售后/投诉风险待处理",
			Description:       "客户反馈床垫异响，要求售后处理。",
			Source:            enums.TicketSourceConversation,
			Status:            enums.TicketStatusPending,
			CurrentAssigneeID: owner.ID,
			ConversationID:    101,
			AuditFields:       models.AuditFields{CreatedAt: dayStart.Add(3 * time.Hour), UpdatedAt: dayStart.Add(4 * time.Hour)},
		},
		{
			TicketNo:    "TK202607050001",
			Title:       "退款进度跟进",
			Description: "客户追问退款处理进度。",
			Source:      enums.TicketSourceConversation,
			Status:      enums.TicketStatusInProgress,
			AuditFields: models.AuditFields{CreatedAt: dayStart.AddDate(0, 0, -1), UpdatedAt: dayStart.Add(2 * time.Hour)},
		},
		{
			TicketNo:    "TK202607060002",
			Title:       "已处理投诉",
			Description: "客户投诉已处理完成。",
			Source:      enums.TicketSourceConversation,
			Status:      enums.TicketStatusDone,
			HandledAt:   &ticketHandledAt,
			AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour), UpdatedAt: dayStart.Add(time.Hour)},
		},
	}
	for i := range tickets {
		if err := db.Create(&tickets[i]).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
	}
	progress := []models.TicketProgress{
		{TicketID: tickets[0].ID, Content: "已电话联系客户，安排师傅明天上门检查。", AuthorID: owner.ID, CreatedAt: dayStart.Add(4*time.Hour + 30*time.Minute)},
		{TicketID: tickets[1].ID, Content: "已同步退款处理进度，等待财务确认。", AuthorID: owner.ID, CreatedAt: dayStart.Add(2*time.Hour + 30*time.Minute)},
	}
	for i := range progress {
		if err := db.Create(&progress[i]).Error; err != nil {
			t.Fatalf("create ticket progress: %v", err)
		}
	}
	feedbackLogs := []models.KnowledgeRetrieveLog{
		{KnowledgeBaseID: 66, Question: "老人腰不好怎么选床垫？", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), ModelName: "test-model", CreatedAt: dayStart.Add(10 * time.Hour)},
		{KnowledgeBaseID: 66, Question: "周末活动能叠加吗？", AnswerStatus: int(enums.KnowledgeAnswerStatusFallback), ModelName: "test-model", CreatedAt: dayStart.Add(11 * time.Hour)},
		{KnowledgeBaseID: 66, Question: "保修政策是什么？", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), ModelName: "test-model", CreatedAt: dayStart.Add(12 * time.Hour)},
	}
	for i := range feedbackLogs {
		if err := db.Create(&feedbackLogs[i]).Error; err != nil {
			t.Fatalf("create feedback retrieve log: %v", err)
		}
	}
	feedbacks := []models.KnowledgeFeedback{
		{RetrieveLogID: feedbackLogs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeLike), FeedbackReason: "回答清楚", CreatedAt: dayStart.Add(9 * time.Hour)},
		{RetrieveLogID: feedbackLogs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeDislike), FeedbackReason: "推荐不准确", CreatedAt: dayStart.Add(10 * time.Hour)},
		{RetrieveLogID: feedbackLogs[1].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeNotHelpful), FeedbackReason: "推荐不准确", CreatedAt: dayStart.Add(11 * time.Hour)},
		{RetrieveLogID: feedbackLogs[2].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeWrongCitation), FeedbackReason: "", CreatedAt: dayStart.Add(12 * time.Hour)},
		{RetrieveLogID: feedbackLogs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeDislike), FeedbackReason: "昨天反馈", CreatedAt: dayStart.AddDate(0, 0, -1)},
	}
	for i := range feedbacks {
		if err := db.Create(&feedbacks[i]).Error; err != nil {
			t.Fatalf("create feedback: %v", err)
		}
	}
	faqDrafts := []models.KnowledgeFAQ{
		{
			KnowledgeBaseID: 66,
			Question:        "保修政策是什么？",
			Answer:          "待补充标准答案",
			Status:          enums.StatusDisabled,
			Remark:          "由知识检索日志生成的待确认 FAQ 草稿\n来源检索日志：3",
			AuditFields:     models.AuditFields{CreatedAt: dayStart.Add(13 * time.Hour), UpdatedAt: dayStart.Add(13 * time.Hour)},
		},
		{
			KnowledgeBaseID: 66,
			Question:        "周末活动能叠加吗？",
			Answer:          "待补充标准答案",
			Status:          enums.StatusDisabled,
			Remark:          "由知识检索日志生成的待确认 FAQ 草稿\n来源检索日志：2",
			AuditFields:     models.AuditFields{CreatedAt: dayStart.Add(12*time.Hour + 30*time.Minute), UpdatedAt: dayStart.Add(12*time.Hour + 30*time.Minute)},
		},
		{
			KnowledgeBaseID: 66,
			Question:        "已经启用的草稿不再待确认",
			Answer:          "已确认答案",
			Status:          enums.StatusOk,
			Remark:          "由知识检索日志生成的待确认 FAQ 草稿\n来源检索日志：999",
			AuditFields:     models.AuditFields{CreatedAt: dayStart.Add(11 * time.Hour), UpdatedAt: dayStart.Add(11 * time.Hour)},
		},
		{
			KnowledgeBaseID: 66,
			Question:        "普通停用 FAQ 不进入日报草稿提醒",
			Answer:          "停用",
			Status:          enums.StatusDisabled,
			Remark:          "人工停用",
			AuditFields:     models.AuditFields{CreatedAt: dayStart.Add(10 * time.Hour), UpdatedAt: dayStart.Add(10 * time.Hour)},
		},
	}
	for i := range faqDrafts {
		if err := db.Create(&faqDrafts[i]).Error; err != nil {
			t.Fatalf("create faq draft: %v", err)
		}
	}

	report := DashboardService.GetDailyBusinessReport("2026-07-06", i18nx.LocaleZhCN)
	if report.OverdueFollowUpCount != 1 || report.TodayFollowUpCount != 1 || report.UnscheduledHotLeads != 3 {
		t.Fatalf("unexpected follow-up counts: %#v", report)
	}
	if report.OverdueAppointmentCount != 1 || report.TodayAppointmentCount != 1 || report.UnscheduledAppointmentCount != 1 {
		t.Fatalf("unexpected appointment counts: %#v", report)
	}
	if report.ConvertedCount != 1 {
		t.Fatalf("unexpected converted count: %#v", report)
	}
	if report.UnassignedPriorityLeadCount != 6 {
		t.Fatalf("unexpected unassigned priority lead count: %#v", report)
	}
	if report.PendingAfterSalesTicketCount != 2 || report.TodayAfterSalesTicketCount != 2 || report.TodayHandledAfterSalesTicketCount != 1 {
		t.Fatalf("unexpected after-sales ticket counts: %#v", report)
	}
	if len(report.AfterSalesTickets) != 2 || report.AfterSalesTickets[0].TicketNo != "TK202607060001" || report.AfterSalesTickets[0].CurrentAssigneeName != "顾问A" {
		t.Fatalf("unexpected after-sales tickets: %#v", report.AfterSalesTickets)
	}
	if !strings.Contains(report.AfterSalesTickets[0].LatestProgress, "安排师傅明天上门检查") || report.AfterSalesTickets[0].LatestProgressAt == "" {
		t.Fatalf("unexpected after-sales ticket progress: %#v", report.AfterSalesTickets[0])
	}
	if report.AIFeedbackCount != 4 || report.AIFeedbackLikeCount != 1 || report.AIFeedbackNegativeCount != 3 || report.AIFeedbackNegativeRate != 75 {
		t.Fatalf("unexpected ai feedback stats: %#v", report)
	}
	if len(report.TopAIFeedbackReasons) != 2 || report.TopAIFeedbackReasons[0].Name != "推荐不准确" || report.TopAIFeedbackReasons[0].Count != 2 {
		t.Fatalf("unexpected ai feedback reasons: %#v", report.TopAIFeedbackReasons)
	}
	if len(report.RecentNegativeAIFeedbacks) != 3 {
		t.Fatalf("unexpected recent negative ai feedbacks: %#v", report.RecentNegativeAIFeedbacks)
	}
	if report.PendingFAQDraftCount != 2 || len(report.PendingFAQDrafts) != 2 {
		t.Fatalf("unexpected pending faq drafts: count=%d drafts=%#v", report.PendingFAQDraftCount, report.PendingFAQDrafts)
	}
	if report.PendingFAQDrafts[0].Question != "保修政策是什么？" || report.PendingFAQDrafts[0].KnowledgeBaseID != 66 {
		t.Fatalf("unexpected first pending faq draft: %#v", report.PendingFAQDrafts[0])
	}
	if report.RecentNegativeAIFeedbacks[0].RetrieveLogID != feedbackLogs[2].ID ||
		report.RecentNegativeAIFeedbacks[0].KnowledgeBaseID != 66 ||
		report.RecentNegativeAIFeedbacks[0].Question != "保修政策是什么？" ||
		report.RecentNegativeAIFeedbacks[0].FeedbackTypeName != "引用错误" {
		t.Fatalf("unexpected first negative ai feedback: %#v", report.RecentNegativeAIFeedbacks[0])
	}
	if len(report.PriorityFollowUps) != 5 {
		t.Fatalf("expected 5 priority follow-ups, got %#v", report.PriorityFollowUps)
	}
	if report.PriorityFollowUps[0].CustomerName != "逾期客户" || report.PriorityFollowUps[0].FollowUpState != "overdue" || report.PriorityFollowUps[0].OwnerUserName != "顾问A" {
		t.Fatalf("unexpected first priority follow-up: %#v", report.PriorityFollowUps[0])
	}
	if !strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "已逾期未跟进") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "还没设置下次跟进时间") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "预约已逾期未到店") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "预约到店") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "预约意向还没确认") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "重点线索未分配负责人") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "售后/投诉工单未处理") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "今日新增 2 个售后/投诉工单") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "今日已处理 1 个售后/投诉工单") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "AI 回答负反馈") ||
		!strings.Contains(strings.Join(report.FollowUpSuggestions, "\n"), "FAQ 草稿待确认") {
		t.Fatalf("follow-up suggestions missing priority guidance: %#v", report.FollowUpSuggestions)
	}
	if !strings.Contains(report.Summary, "成交线索 1 条") ||
		!strings.Contains(strings.Join(report.Highlights, "\n"), "已标记成交 1 条线索") ||
		!strings.Contains(strings.Join(report.Highlights, "\n"), "还有 6 条重点线索未分配负责人") ||
		!strings.Contains(strings.Join(report.Highlights, "\n"), "还有 2 个售后/投诉工单未处理，今日已处理 1 个") ||
		!strings.Contains(strings.Join(report.Highlights, "\n"), "负反馈率 75.0%") {
		t.Fatalf("report missing converted guidance: summary=%s highlights=%#v", report.Summary, report.Highlights)
	}
	if !strings.Contains(strings.Join(report.KnowledgeSuggestions, "\n"), "推荐不准确") ||
		!strings.Contains(strings.Join(report.KnowledgeSuggestions, "\n"), "FAQ 草稿待确认") {
		t.Fatalf("knowledge suggestions missing ai feedback guidance: %#v", report.KnowledgeSuggestions)
	}
}

func TestAIQualityReportBuildsTodosAndRiskSamples(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.KnowledgeRetrieveLog{},
		&models.KnowledgeFeedback{},
		&models.KnowledgeFAQ{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	now := time.Now()
	logs := []models.KnowledgeRetrieveLog{
		{KnowledgeBaseID: 88, Question: "床垫保修多久？", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), HitCount: 2, TopScore: 0.91, ModelName: "test-model", CreatedAt: now.Add(-2 * time.Hour)},
		{KnowledgeBaseID: 88, Question: "活动能不能叠加？", AnswerStatus: int(enums.KnowledgeAnswerStatusNoAnswer), HitCount: 0, TopScore: 0.12, ModelName: "test-model", CreatedAt: now.Add(-90 * time.Minute)},
		{KnowledgeBaseID: 88, Question: "能保证治好腰疼吗？", AnswerStatus: int(enums.KnowledgeAnswerStatusBlocked), HitCount: 1, TopScore: 0.52, ModelName: "test-model", CreatedAt: now.Add(-80 * time.Minute)},
		{KnowledgeBaseID: 88, Question: "周末到店礼是什么？", AnswerStatus: int(enums.KnowledgeAnswerStatusFallback), HitCount: 1, TopScore: 0.44, ModelName: "test-model", CreatedAt: now.Add(-70 * time.Minute)},
		{KnowledgeBaseID: 88, Question: "旧问题", AnswerStatus: int(enums.KnowledgeAnswerStatusNoAnswer), HitCount: 0, TopScore: 0.1, ModelName: "test-model", CreatedAt: now.AddDate(0, 0, -40)},
	}
	for i := range logs {
		if err := db.Create(&logs[i]).Error; err != nil {
			t.Fatalf("create retrieve log: %v", err)
		}
	}
	feedbacks := []models.KnowledgeFeedback{
		{RetrieveLogID: logs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeLike), FeedbackReason: "清楚", CreatedAt: now.Add(-2 * time.Hour)},
		{RetrieveLogID: logs[1].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeDislike), FeedbackReason: "没有回答活动", CreatedAt: now.Add(-80 * time.Minute)},
		{RetrieveLogID: logs[2].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeWrongCitation), FeedbackReason: "引用错误", CreatedAt: now.Add(-70 * time.Minute)},
	}
	for i := range feedbacks {
		if err := db.Create(&feedbacks[i]).Error; err != nil {
			t.Fatalf("create feedback: %v", err)
		}
	}
	if err := db.Create(&models.KnowledgeFAQ{
		KnowledgeBaseID: 88,
		Question:        "活动能不能叠加？",
		Answer:          "待补充",
		Status:          enums.StatusDisabled,
		Remark:          "由知识检索日志生成的待确认 FAQ 草稿\n来源检索日志：2",
		AuditFields:     models.AuditFields{CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
	}).Error; err != nil {
		t.Fatalf("create faq draft: %v", err)
	}

	report := DashboardService.GetAIQualityReport("30d", i18nx.LocaleZhCN)
	if report.RetrieveTotal != 4 || report.RetrieveHitTotal != 3 || report.RetrieveHitRate != 75 {
		t.Fatalf("unexpected retrieve stats: %#v", report)
	}
	if report.NoAnswerCount != 1 || report.FallbackCount != 1 || report.BlockedCount != 1 || report.RiskAnswerCount != 3 {
		t.Fatalf("unexpected risk answer stats: %#v", report)
	}
	if report.FeedbackCount != 3 || report.NegativeFeedbackCount != 2 || report.NegativeFeedbackRate != 66.7 {
		t.Fatalf("unexpected feedback stats: %#v", report)
	}
	if report.PendingFAQDraftCount != 1 || len(report.PendingFAQDrafts) != 1 {
		t.Fatalf("unexpected faq draft stats: %#v", report)
	}
	if report.TodoTotal != 5 || len(report.Todos) != 5 {
		t.Fatalf("unexpected quality todos: %#v", report.Todos)
	}
	if len(report.RecentRiskAnswerSamples) != 3 || report.RecentRiskAnswerSamples[0].ActionHref == "" {
		t.Fatalf("unexpected risk samples: %#v", report.RecentRiskAnswerSamples)
	}
	if len(report.UnansweredQuestions) == 0 || report.UnansweredQuestions[0].Count == 0 {
		t.Fatalf("expected unanswered questions: %#v", report.UnansweredQuestions)
	}
	foundPendingQuestion := false
	for _, item := range report.PendingQuestionGroups {
		if item.Question == "活动能不能叠加" {
			foundPendingQuestion = true
			if item.Count != 2 || item.NoAnswerCount != 1 || item.NegativeFeedbackCount != 1 || item.ActionHref == "" {
				t.Fatalf("unexpected pending question group: %#v", item)
			}
			break
		}
	}
	if !foundPendingQuestion {
		t.Fatalf("expected pending question groups to include activity question: %#v", report.PendingQuestionGroups)
	}
	if !strings.Contains(strings.Join(report.KnowledgeSuggestions, "\n"), "负反馈") {
		t.Fatalf("expected ai quality suggestions: %#v", report.KnowledgeSuggestions)
	}
}

func TestSalesFunnelReportIncludesAdvisorEfficiency(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.SalesLead{},
		&models.LeadFollowUp{},
		&models.User{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	now := time.Now()
	dayStart := startOfDay(now)
	advisorA := models.User{Username: "advisor-a", Nickname: "顾问A", Status: enums.StatusOk}
	advisorB := models.User{Username: "advisor-b", Nickname: "顾问B", Status: enums.StatusOk}
	if err := db.Create(&advisorA).Error; err != nil {
		t.Fatalf("create advisor a: %v", err)
	}
	if err := db.Create(&advisorB).Error; err != nil {
		t.Fatalf("create advisor b: %v", err)
	}
	for i := 0; i < 5; i++ {
		item := models.Conversation{
			CustomerName: fmt.Sprintf("客户%d", i+1),
			Status:       enums.IMConversationStatusClosed,
			AuditFields:  models.AuditFields{CreatedAt: dayStart.Add(time.Duration(i) * time.Hour), UpdatedAt: dayStart.Add(time.Duration(i) * time.Hour)},
		}
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("create conversation: %v", err)
		}
	}
	appointmentAt := dayStart.Add(28 * time.Hour)
	overdueFollowUp := dayStart.Add(-2 * time.Hour)
	leads := []models.SalesLead{
		{CustomerName: "成交客户", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageReadyToBuy, Status: enums.SalesLeadStatusConverted, OwnerUserID: advisorA.ID, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(30 * time.Minute), UpdatedAt: dayStart.Add(6 * time.Hour)}},
		{CustomerName: "逾期预约客户", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentAt, Status: enums.SalesLeadStatusFollowing, OwnerUserID: advisorA.ID, NextFollowUpAt: &overdueFollowUp, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(1 * time.Hour), UpdatedAt: dayStart.Add(2 * time.Hour)}},
		{CustomerName: "已到店客户", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentAt, Status: enums.SalesLeadStatusVisited, OwnerUserID: advisorA.ID, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(90 * time.Minute), UpdatedAt: dayStart.Add(5 * time.Hour)}},
		{CustomerName: "无效客户", IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageConsulting, Status: enums.SalesLeadStatusInvalid, OwnerUserID: advisorB.ID, Remark: "客户预算太低，价格超预算", AuditFields: models.AuditFields{CreatedAt: dayStart.Add(2 * time.Hour), UpdatedAt: dayStart.Add(4 * time.Hour)}},
		{CustomerName: "准成交客户", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageReadyToBuy, Status: enums.SalesLeadStatusFollowing, OwnerUserID: advisorB.ID, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(3 * time.Hour), UpdatedAt: dayStart.Add(4 * time.Hour)}},
		{CustomerName: "未分配客户", IntentLevel: enums.SalesLeadIntentLow, BuyingStage: enums.SalesLeadStageConsulting, Status: enums.SalesLeadStatusNew, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(4 * time.Hour), UpdatedAt: dayStart.Add(4 * time.Hour)}},
		{CustomerName: "过期老线索", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageReadyToBuy, Status: enums.SalesLeadStatusConverted, OwnerUserID: advisorA.ID, AuditFields: models.AuditFields{CreatedAt: dayStart.AddDate(0, 0, -40), UpdatedAt: dayStart.AddDate(0, 0, -39)}},
	}
	for i := range leads {
		if err := db.Create(&leads[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}
	followUps := []models.LeadFollowUp{
		{LeadID: leads[0].ID, OperatorID: advisorA.ID, OperatorName: "顾问A", Content: "首跟进", CreatedAt: leads[0].CreatedAt.Add(30 * time.Minute)},
		{LeadID: leads[1].ID, OperatorID: advisorA.ID, OperatorName: "顾问A", Content: "首跟进", CreatedAt: leads[1].CreatedAt.Add(90 * time.Minute)},
		{LeadID: leads[2].ID, OperatorID: advisorB.ID, OperatorName: "顾问B", Content: "首跟进", CreatedAt: leads[2].CreatedAt.Add(120 * time.Minute)},
	}
	for i := range followUps {
		if err := db.Create(&followUps[i]).Error; err != nil {
			t.Fatalf("create follow-up: %v", err)
		}
	}

	report := DashboardService.GetSalesFunnelReport("30d", i18nx.LocaleZhCN)
	if report.ConversationTotal != 5 || report.LeadTotal != 6 || report.LeadConversionRate != 120 || report.ClosedConversionRate != 16.7 {
		t.Fatalf("unexpected funnel totals: %#v", report)
	}
	if report.AppointmentTotal != 4 || report.VisitedTotal != 2 || report.ConvertedTotal != 1 || report.InvalidTotal != 1 || report.UnassignedTotal != 1 || report.OverdueFollowUpTotal != 1 {
		t.Fatalf("unexpected funnel risk totals: %#v", report)
	}
	if len(report.InvalidReasons) != 1 || report.InvalidReasons[0].Name != "预算不匹配" || report.InvalidReasons[0].Count != 1 {
		t.Fatalf("unexpected invalid reasons: %#v", report.InvalidReasons)
	}
	if len(report.Steps) != 7 || report.Steps[0].Key != "consultation" || report.Steps[4].Key != "visited" || report.Steps[6].Key != "converted" {
		t.Fatalf("unexpected funnel steps: %#v", report.Steps)
	}
	if len(report.AdvisorStats) != 3 {
		t.Fatalf("unexpected advisor stats: %#v", report.AdvisorStats)
	}
	if report.AdvisorStats[0].OwnerUserName != "顾问A" ||
		report.AdvisorStats[0].AssignedLeadCount != 3 ||
		report.AdvisorStats[0].ConvertedLeadCount != 1 ||
		report.AdvisorStats[0].OverdueFollowUpCount != 1 ||
		report.AdvisorStats[0].AverageFirstFollowUpMinutes != 80 {
		t.Fatalf("unexpected first advisor stats: %#v", report.AdvisorStats[0])
	}
	var advisorBStats response.DashboardAdvisorEfficiency
	for _, item := range report.AdvisorStats {
		if item.OwnerUserID == advisorB.ID {
			advisorBStats = item
			break
		}
	}
	if advisorBStats.OwnerUserID == 0 || len(advisorBStats.InvalidReasons) != 1 || advisorBStats.InvalidReasons[0].Name != "预算不匹配" {
		t.Fatalf("unexpected advisor invalid reasons: %#v", advisorBStats)
	}
	if !strings.Contains(strings.Join(report.Suggestions, "\n"), "未分配") ||
		!strings.Contains(strings.Join(report.Suggestions, "\n"), "逾期") {
		t.Fatalf("expected funnel suggestions: %#v", report.Suggestions)
	}
}

func TestBusinessTrendReportBuildsSeriesAndTopItems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.SalesLead{},
		&models.LeadFollowUp{},
		&models.User{},
		&models.KnowledgeRetrieveLog{},
		&models.KnowledgeFeedback{},
		&models.KnowledgeFAQ{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	now := time.Now()
	dayStart := startOfDay(now)
	yesterday := dayStart.AddDate(0, 0, -1)
	handoffAt := dayStart.Add(2 * time.Hour)
	advisor := models.User{Username: "trend-advisor", Nickname: "趋势顾问", Status: enums.StatusOk}
	if err := db.Create(&advisor).Error; err != nil {
		t.Fatalf("create advisor: %v", err)
	}
	conversations := []models.Conversation{
		{CustomerName: "今日咨询1", Status: enums.IMConversationStatusClosed, HandoffAt: &handoffAt, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour), UpdatedAt: dayStart.Add(time.Hour)}},
		{CustomerName: "今日咨询2", Status: enums.IMConversationStatusActive, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(2 * time.Hour), UpdatedAt: dayStart.Add(2 * time.Hour)}},
		{CustomerName: "昨日咨询", Status: enums.IMConversationStatusClosed, AuditFields: models.AuditFields{CreatedAt: yesterday.Add(time.Hour), UpdatedAt: yesterday.Add(time.Hour)}},
	}
	for i := range conversations {
		if err := db.Create(&conversations[i]).Error; err != nil {
			t.Fatalf("create conversation: %v", err)
		}
	}
	appointmentAt := dayStart.Add(28 * time.Hour)
	leads := []models.SalesLead{
		{CustomerName: "成交客户", InterestedProducts: "智能床垫, 护脊枕", SourceChannel: "官网", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageReadyToBuy, Status: enums.SalesLeadStatusConverted, OwnerUserID: advisor.ID, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour), UpdatedAt: dayStart.Add(3 * time.Hour)}},
		{CustomerName: "预约客户", InterestedProducts: "智能床垫", SourceChannel: "企微", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentAt, Status: enums.SalesLeadStatusFollowing, OwnerUserID: advisor.ID, AuditFields: models.AuditFields{CreatedAt: yesterday.Add(2 * time.Hour), UpdatedAt: yesterday.Add(3 * time.Hour)}},
		{CustomerName: "普通客户", InterestedProducts: "儿童床垫", SourceChannel: "官网", IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageConsulting, Status: enums.SalesLeadStatusNew, AuditFields: models.AuditFields{CreatedAt: yesterday.Add(3 * time.Hour), UpdatedAt: yesterday.Add(3 * time.Hour)}},
		{CustomerName: "关闭客户", InterestedProducts: "不计入", SourceChannel: "官网", IntentLevel: enums.SalesLeadIntentLow, Status: enums.SalesLeadStatusClosed, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(4 * time.Hour), UpdatedAt: dayStart.Add(4 * time.Hour)}},
	}
	for i := range leads {
		if err := db.Create(&leads[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}
	if err := db.Create(&models.LeadFollowUp{
		LeadID:       leads[0].ID,
		OperatorID:   advisor.ID,
		OperatorName: "趋势顾问",
		Content:      "首跟进",
		CreatedAt:    leads[0].CreatedAt.Add(45 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("create follow-up: %v", err)
	}
	logs := []models.KnowledgeRetrieveLog{
		{KnowledgeBaseID: 1, Question: "智能床垫适合老人吗？", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: dayStart.Add(time.Hour)},
		{KnowledgeBaseID: 1, Question: "周末活动能叠加吗？", AnswerStatus: int(enums.KnowledgeAnswerStatusFallback), CreatedAt: dayStart.Add(2 * time.Hour)},
	}
	for i := range logs {
		if err := db.Create(&logs[i]).Error; err != nil {
			t.Fatalf("create retrieve log: %v", err)
		}
	}
	feedbacks := []models.KnowledgeFeedback{
		{RetrieveLogID: logs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeLike), FeedbackReason: "清楚", CreatedAt: dayStart.Add(time.Hour)},
		{RetrieveLogID: logs[1].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeDislike), FeedbackReason: "活动说错", CreatedAt: dayStart.Add(2 * time.Hour)},
	}
	for i := range feedbacks {
		if err := db.Create(&feedbacks[i]).Error; err != nil {
			t.Fatalf("create feedback: %v", err)
		}
	}
	if err := db.Create(&models.KnowledgeFAQ{
		KnowledgeBaseID: 1,
		Question:        "周末活动能叠加吗？",
		Answer:          "待确认",
		Status:          enums.StatusDisabled,
		Remark:          "由知识检索日志生成的待确认 FAQ 草稿\n来源检索日志：2",
		AuditFields:     models.AuditFields{CreatedAt: dayStart.Add(3 * time.Hour), UpdatedAt: dayStart.Add(3 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("create faq draft: %v", err)
	}

	report := DashboardService.GetBusinessTrendReport("7d", i18nx.LocaleZhCN)
	if report.ConversationTotal != 3 || report.LeadTotal != 3 || report.VisitedTotal != 1 || report.ConvertedTotal != 1 || report.HandoffTotal != 1 {
		t.Fatalf("unexpected trend totals: %#v", report)
	}
	if report.LeadConversionRate != 100 || report.HighIntentTotal != 2 || report.AppointmentTotal != 2 {
		t.Fatalf("unexpected trend conversion: %#v", report)
	}
	if report.NegativeFeedbackTotal != 1 || report.PendingFAQDraftCount != 1 {
		t.Fatalf("unexpected quality totals: %#v", report)
	}
	if len(report.Series) != 7 || report.Series[len(report.Series)-1].Date != dayStart.Format("2006-01-02") {
		t.Fatalf("unexpected trend series: %#v", report.Series)
	}
	todayPoint := report.Series[len(report.Series)-1]
	if todayPoint.ConversationCount != 2 || todayPoint.LeadCount != 1 || todayPoint.VisitedCount != 1 || todayPoint.ConvertedCount != 1 || todayPoint.HandoffCount != 1 || todayPoint.NegativeFeedbackCount != 1 {
		t.Fatalf("unexpected today trend point: %#v", todayPoint)
	}
	if len(report.TopProducts) == 0 || report.TopProducts[0].Name != "智能床垫" || report.TopProducts[0].Count != 2 {
		t.Fatalf("unexpected top products: %#v", report.TopProducts)
	}
	if len(report.TopChannels) == 0 || report.TopChannels[0].Name != "官网" {
		t.Fatalf("unexpected top channels: %#v", report.TopChannels)
	}
	if len(report.TopUnansweredQuestions) == 0 || report.TopUnansweredQuestions[0].Name != "周末活动能叠加吗" {
		t.Fatalf("unexpected unanswered questions: %#v", report.TopUnansweredQuestions)
	}
	if len(report.AdvisorStats) == 0 || report.AdvisorStats[0].OwnerUserName != "趋势顾问" {
		t.Fatalf("unexpected advisor stats: %#v", report.AdvisorStats)
	}
	if !strings.Contains(strings.Join(report.Suggestions, "\n"), "负反馈") {
		t.Fatalf("expected trend suggestions: %#v", report.Suggestions)
	}
	if !strings.Contains(report.ReportMarkdown, "周度经营趋势复盘") ||
		!strings.Contains(report.ReportMarkdown, "智能床垫") ||
		!strings.Contains(report.ReportMarkdown, "官网") ||
		!strings.Contains(report.ReportMarkdown, "趋势顾问") ||
		!strings.Contains(report.ReportMarkdown, "AI 负反馈") {
		t.Fatalf("unexpected trend report markdown: %s", report.ReportMarkdown)
	}
}

func TestABTestReportComparesSourceChannelVariants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.SalesLead{}, &models.KnowledgeFeedback{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	now := time.Now()
	dayStart := startOfDay(now)
	appointmentAt := dayStart.Add(28 * time.Hour)
	leads := []models.SalesLead{
		{CustomerName: "A成交", SourceChannel: "opening_a", InterestedProducts: "智能床垫", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageReadyToBuy, Status: enums.SalesLeadStatusConverted, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(time.Hour), UpdatedAt: dayStart.Add(2 * time.Hour)}},
		{CustomerName: "A预约", SourceChannel: "opening_a", InterestedProducts: "智能床垫", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &appointmentAt, Status: enums.SalesLeadStatusFollowing, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(2 * time.Hour), UpdatedAt: dayStart.Add(3 * time.Hour)}},
		{CustomerName: "B普通", SourceChannel: "opening_b", InterestedProducts: "儿童床垫", IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageConsulting, Status: enums.SalesLeadStatusFollowing, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(3 * time.Hour), UpdatedAt: dayStart.Add(4 * time.Hour)}},
		{CustomerName: "B无效", SourceChannel: "opening_b", InterestedProducts: "儿童床垫", IntentLevel: enums.SalesLeadIntentLow, BuyingStage: enums.SalesLeadStageConsulting, Status: enums.SalesLeadStatusInvalid, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(4 * time.Hour), UpdatedAt: dayStart.Add(5 * time.Hour)}},
		{CustomerName: "未标记", InterestedProducts: "护脊枕", IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageConsulting, Status: enums.SalesLeadStatusNew, AuditFields: models.AuditFields{CreatedAt: dayStart.Add(5 * time.Hour), UpdatedAt: dayStart.Add(6 * time.Hour)}},
	}
	for i := range leads {
		if err := db.Create(&leads[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}
	feedbacks := []models.KnowledgeFeedback{
		{RetrieveLogID: 1001, FeedbackType: int(enums.KnowledgeFeedbackTypeLike), FeedbackReason: "清楚", CreatedAt: dayStart.Add(time.Hour)},
		{RetrieveLogID: 1002, FeedbackType: int(enums.KnowledgeFeedbackTypeDislike), FeedbackReason: "推荐不准确", CreatedAt: dayStart.Add(2 * time.Hour)},
		{RetrieveLogID: 1003, FeedbackType: int(enums.KnowledgeFeedbackTypeNotHelpful), FeedbackReason: "没解决问题", CreatedAt: dayStart.Add(3 * time.Hour)},
	}
	for i := range feedbacks {
		if err := db.Create(&feedbacks[i]).Error; err != nil {
			t.Fatalf("create feedback: %v", err)
		}
	}

	report := DashboardService.GetABTestReport("7d", i18nx.LocaleZhCN)
	if report.LeadTotal != 5 || report.VariantTotal != 3 || len(report.Variants) != 3 {
		t.Fatalf("unexpected ab report totals: %#v", report)
	}
	if report.FeedbackTotal != 3 || report.NegativeFeedbackTotal != 2 || report.NegativeFeedbackRate != 66.7 {
		t.Fatalf("unexpected ab feedback guardrail: %#v", report)
	}
	if report.Variants[0].VariantCode != "opening_a" ||
		report.Variants[0].LeadCount != 2 ||
		report.Variants[0].HighIntentRate != 100 ||
		report.Variants[0].AppointmentRate != 100 ||
		report.Variants[0].VisitedCount != 1 ||
		report.Variants[0].VisitRate != 50 ||
		report.Variants[0].ConversionRate != 50 ||
		report.Variants[0].TopProduct != "智能床垫" {
		t.Fatalf("unexpected top variant: %#v", report.Variants[0])
	}
	var openingB = report.Variants[0]
	foundOpeningB := false
	for _, item := range report.Variants {
		if item.VariantCode == "opening_b" {
			openingB = item
			foundOpeningB = true
			break
		}
	}
	if !foundOpeningB || openingB.InvalidRate != 50 || openingB.QualityRiskLevel != "medium" || openingB.QualityRiskReason == "" {
		t.Fatalf("unexpected risk variant: %#v", openingB)
	}
	if !strings.Contains(strings.Join(report.Suggestions, "\n"), "opening a") {
		t.Fatalf("expected ab suggestions to mention best variant: %#v", report.Suggestions)
	}
	if !strings.Contains(strings.Join(report.Suggestions, "\n"), "AI 负反馈 2 条") {
		t.Fatalf("expected ab suggestions to mention quality guardrail: %#v", report.Suggestions)
	}
}

func TestSendDailyBusinessReportWebhook(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.Message{},
		&models.SalesLead{},
		&models.Product{},
		&models.Promotion{},
		&models.KnowledgeRetrieveLog{},
		&models.KnowledgeFeedback{},
		&models.KnowledgeFAQ{},
		&models.User{},
		&models.Ticket{},
		&models.TicketProgress{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	reportDay := time.Date(2026, 7, 6, 9, 0, 0, 0, time.Local)
	if err := db.Create(&models.Conversation{
		CustomerName: "日报客户",
		Status:       enums.IMConversationStatusClosed,
		AuditFields:  models.AuditFields{CreatedAt: reportDay, UpdatedAt: reportDay},
	}).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := db.Create(&models.SalesLead{
		CustomerName: "成交客户",
		Phone:        "13800000000",
		IntentLevel:  enums.SalesLeadIntentHigh,
		BuyingStage:  enums.SalesLeadStageReadyToBuy,
		Status:       enums.SalesLeadStatusConverted,
		AuditFields:  models.AuditFields{CreatedAt: reportDay, UpdatedAt: reportDay.Add(time.Hour)},
	}).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode webhook payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config.SetCurrent(&config.Config{
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     server.URL,
				Format:  "generic",
			},
			DailyReport: config.DailyReportNotifyConfig{
				Enabled: true,
				Cron:    "0 9 * * *",
			},
		},
	})
	t.Cleanup(func() {
		config.SetCurrent(&config.Config{})
	})

	resp, err := DashboardService.SendDailyBusinessReportWebhook("2026-07-06", i18nx.LocaleZhCN, 99)
	if err != nil {
		t.Fatalf("SendDailyBusinessReportWebhook() error = %v", err)
	}
	if !resp.Sent || !resp.WebhookEnabled || resp.WebhookEventType != "daily_business_report" {
		t.Fatalf("unexpected push response: %#v", resp)
	}
	if got["eventType"] != "daily_business_report" || !strings.Contains(got["title"].(string), "2026-07-06") {
		t.Fatalf("unexpected webhook payload: %#v", got)
	}
	if !strings.Contains(got["text"].(string), "成交客户") && !strings.Contains(got["text"].(string), "成交") {
		t.Fatalf("webhook text should include daily business content: %s", got["text"])
	}
	metadata := got["metadata"].(map[string]any)
	if metadata["operatorId"].(float64) != 99 || metadata["convertedCount"].(float64) != 1 {
		t.Fatalf("unexpected webhook metadata: %#v", metadata)
	}
}

func TestScheduledDailyBusinessReportSkipsDuplicateReportDate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.Message{},
		&models.SalesLead{},
		&models.Product{},
		&models.Promotion{},
		&models.KnowledgeRetrieveLog{},
		&models.KnowledgeFeedback{},
		&models.KnowledgeFAQ{},
		&models.User{},
		&models.Ticket{},
		&models.TicketProgress{},
		&models.SystemConfig{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	sendCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config.SetCurrent(&config.Config{
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     server.URL,
				Format:  "generic",
			},
			DailyReport: config.DailyReportNotifyConfig{
				Enabled:        true,
				Cron:           "0 9 * * *",
				AllowDuplicate: false,
			},
		},
	})
	t.Cleanup(func() {
		config.SetCurrent(&config.Config{})
	})

	first, err := DashboardService.SendScheduledDailyBusinessReportWebhook("2026-07-06", i18nx.LocaleZhCN)
	if err != nil {
		t.Fatalf("first scheduled daily report error = %v", err)
	}
	if !first.Sent || sendCount != 1 {
		t.Fatalf("expected first scheduled report sent once, resp=%#v count=%d", first, sendCount)
	}
	second, err := DashboardService.SendScheduledDailyBusinessReportWebhook("2026-07-06", i18nx.LocaleZhCN)
	if err != nil {
		t.Fatalf("second scheduled daily report error = %v", err)
	}
	if second.Sent || sendCount != 1 || !strings.Contains(second.Message, "已跳过重复") {
		t.Fatalf("expected duplicate scheduled report skipped, resp=%#v count=%d", second, sendCount)
	}

	cfg := config.Current()
	cfg.Notify.DailyReport.AllowDuplicate = true
	config.SetCurrent(&cfg)
	third, err := DashboardService.SendScheduledDailyBusinessReportWebhook("2026-07-06", i18nx.LocaleZhCN)
	if err != nil {
		t.Fatalf("allow duplicate scheduled daily report error = %v", err)
	}
	if !third.Sent || sendCount != 2 {
		t.Fatalf("expected allow duplicate report sent, resp=%#v count=%d", third, sendCount)
	}
}
