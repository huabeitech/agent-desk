package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/repositories"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var DashboardService = newDashboardService()

func newDashboardService() *dashboardService {
	return &dashboardService{}
}

type dashboardService struct {
}

const dashboardDailyReportLastSentConfigKey = "dashboard.daily_report.last_sent_date"

func (s *dashboardService) GetOverview(rangeValue string, locale string) response.DashboardOverviewResponse {
	locale = i18nx.NormalizeLocale(locale)
	now := time.Now()
	normalizedRange, trendDays := normalizeDashboardRange(rangeValue)
	todayStart := startOfDay(now)
	trendStart := todayStart.AddDate(0, 0, -(trendDays - 1))
	db := sqls.DB()

	conversationTodayCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("created_at >= ?", todayStart)
	})
	processingConversationCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status IN ?", []enums.IMConversationStatus{
			enums.IMConversationStatusAIServing,
			enums.IMConversationStatusActive,
		})
	})
	pendingConversationCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", enums.IMConversationStatusPending)
	})

	agentProfiles := repositories.DashboardRepository.ListEnabledAgentProfiles(db)
	agentTeams := repositories.DashboardRepository.ListEnabledAgentTeams(db)
	activeSchedules := repositories.DashboardRepository.ListActiveTeamSchedules(db, now, now)
	activeConversations := repositories.DashboardRepository.ListConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status IN ?", []enums.IMConversationStatus{
			enums.IMConversationStatusAIServing,
			enums.IMConversationStatusPending,
			enums.IMConversationStatusActive,
		})
	})

	onlineAgents, busyAgents, offlineAgents, teamLoads := s.buildAgentStats(now, agentTeams, agentProfiles, activeSchedules, activeConversations)

	enabledAIAgentCount := repositories.DashboardRepository.CountAIAgents(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", enums.StatusOk)
	})
	enabledChannelCount := repositories.DashboardRepository.CountChannels(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", enums.StatusOk)
	})
	knowledgeRetrieveCount := repositories.DashboardRepository.CountKnowledgeRetrieveLogs(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("created_at >= ?", todayStart)
	})
	knowledgeRetrieveFailCount := repositories.DashboardRepository.CountKnowledgeRetrieveLogs(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("created_at >= ? AND answer_status IN ?", todayStart, []int{2, 3, 4})
	})
	skillRunFailCount := repositories.DashboardRepository.CountSkillRunLogs(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("created_at >= ? AND error_message <> ''", todayStart)
	})
	aiHandoffCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("handoff_at >= ?", todayStart)
	})
	digitalStoreStats := s.buildDigitalStoreStats(db, todayStart, trendStart, now, conversationTodayCount, aiHandoffCount, locale)

	enabledAIAgents := repositories.DashboardRepository.ListAIAgents(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", enums.StatusOk)
	})
	alerts := s.buildAlerts(now, db, enabledAIAgents, agentTeams, activeSchedules, locale)

	return response.DashboardOverviewResponse{
		Range:       normalizedRange,
		GeneratedAt: now.Format("2006-01-02 15:04:05"),
		Summary: response.DashboardSummaryResponse{
			TodayNewConversations:        conversationTodayCount,
			ProcessingConversations:      processingConversationCount,
			PendingDispatchConversations: pendingConversationCount,
			OnlineAgents:                 onlineAgents,
			AIServiceRate:                calcAIServiceRate(activeConversations),
		},
		ConversationStats: response.DashboardSectionStatsResponse{
			StatusDistribution: buildConversationStatusDistribution(db, locale),
			Trend:              buildConversationTrend(db, trendStart),
		},
		AgentStats: response.DashboardAgentStatsResponse{
			OnlineAgents:  onlineAgents,
			BusyAgents:    busyAgents,
			OfflineAgents: offlineAgents,
			TeamLoads:     teamLoads,
		},
		AIStats: response.DashboardAIStatsResponse{
			EnabledAIAgents:                 enabledAIAgentCount,
			EnabledChannels:                 enabledChannelCount,
			TodayKnowledgeRetrieves:         knowledgeRetrieveCount,
			TodayKnowledgeRetrieveFailCount: knowledgeRetrieveFailCount,
			TodayKnowledgeRetrieveFailRate:  calcRate(knowledgeRetrieveFailCount, knowledgeRetrieveCount),
			TodaySkillRunFailCount:          skillRunFailCount,
			TodayAIHandoffCount:             aiHandoffCount,
		},
		DigitalStoreStats: digitalStoreStats,
		Alerts:            alerts,
		QuickLinks:        buildDashboardQuickLinks(locale),
	}
}

func (s *dashboardService) GetDailyBusinessReport(dateValue string, locale string) response.DashboardDailyBusinessReportResponse {
	locale = i18nx.NormalizeLocale(locale)
	now := time.Now()
	reportDate, dayStart, dayEnd := resolveReportDay(dateValue, now)
	db := sqls.DB()
	validLeadStatuses := []enums.SalesLeadStatus{
		enums.SalesLeadStatusNew,
		enums.SalesLeadStatusFollowing,
		enums.SalesLeadStatusVisited,
		enums.SalesLeadStatusConverted,
	}

	conversationCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("created_at >= ? AND created_at < ?", dayStart, dayEnd)
	})

	var aiReplyCount int64
	db.Model(&models.Message{}).
		Where("created_at >= ? AND created_at < ? AND sender_type = ?", dayStart, dayEnd, enums.IMSenderTypeAI).
		Count(&aiReplyCount)

	handoffCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("handoff_at >= ? AND handoff_at < ?", dayStart, dayEnd)
	})

	var leadCount int64
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ? AND status IN ?", dayStart, dayEnd, validLeadStatuses).
		Count(&leadCount)

	var highIntentCount int64
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ? AND status IN ? AND intent_level = ?", dayStart, dayEnd, validLeadStatuses, enums.SalesLeadIntentHigh).
		Count(&highIntentCount)

	var appointmentCount int64
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ? AND status IN ? AND buying_stage IN ?", dayStart, dayEnd, validLeadStatuses, []enums.SalesLeadStage{
			enums.SalesLeadStageAppointment,
			enums.SalesLeadStageReadyToBuy,
		}).
		Count(&appointmentCount)

	var convertedCount int64
	db.Model(&models.SalesLead{}).
		Where("updated_at >= ? AND updated_at < ? AND status = ?", dayStart, dayEnd, enums.SalesLeadStatusConverted).
		Count(&convertedCount)

	unresolvedCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status <> ? AND last_active_at >= ? AND last_active_at < ?", enums.IMConversationStatusClosed, dayStart, dayEnd)
	})

	var activeProductCount int64
	db.Model(&models.Product{}).
		Where("status = ?", enums.StatusOk).
		Count(&activeProductCount)

	var activePromotionCount int64
	db.Model(&models.Promotion{}).
		Where("status = ? AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", enums.StatusOk, dayEnd, dayStart).
		Count(&activePromotionCount)

	highIntentLeads := s.listReportHighIntentLeads(db, dayStart, dayEnd, validLeadStatuses)
	unassignedPriorityLeadCount := s.countReportUnassignedPriorityLeads(db, dayEnd)
	overdueFollowUpCount, todayFollowUpCount, unscheduledHotLeads := s.countReportFollowUpRisks(db, dayStart, dayEnd)
	overdueAppointmentCount, todayAppointmentCount, unscheduledAppointmentCount := s.countReportAppointmentRisks(db, dayStart, dayEnd)
	pendingAfterSalesTicketCount, todayAfterSalesTicketCount, todayHandledAfterSalesTicketCount := s.countReportAfterSalesTicketRisks(db, dayStart, dayEnd)
	aiFeedbackCount, aiFeedbackLikeCount, aiFeedbackNegativeCount := s.countReportAIFeedbacks(db, dayStart, dayEnd)
	priorityFollowUps := s.listReportPriorityFollowUps(db, dayStart, dayEnd)
	afterSalesTickets := s.listReportAfterSalesTickets(db)
	recentNegativeAIFeedbacks := s.listReportRecentNegativeAIFeedbacks(db, dayStart, dayEnd)
	pendingFAQDraftCount, pendingFAQDrafts := s.listReportPendingFAQDrafts(db)

	report := response.DashboardDailyBusinessReportResponse{
		ReportDate:                        reportDate,
		ConversationCount:                 conversationCount,
		AIReplyCount:                      aiReplyCount,
		HandoffCount:                      handoffCount,
		LeadCount:                         leadCount,
		LeadConversionRate:                calcRate(leadCount, conversationCount),
		HighIntentCount:                   highIntentCount,
		AppointmentCount:                  appointmentCount,
		ConvertedCount:                    convertedCount,
		UnresolvedCount:                   unresolvedCount,
		UnassignedPriorityLeadCount:       unassignedPriorityLeadCount,
		OverdueFollowUpCount:              overdueFollowUpCount,
		TodayFollowUpCount:                todayFollowUpCount,
		UnscheduledHotLeads:               unscheduledHotLeads,
		OverdueAppointmentCount:           overdueAppointmentCount,
		TodayAppointmentCount:             todayAppointmentCount,
		UnscheduledAppointmentCount:       unscheduledAppointmentCount,
		PendingAfterSalesTicketCount:      pendingAfterSalesTicketCount,
		TodayAfterSalesTicketCount:        todayAfterSalesTicketCount,
		TodayHandledAfterSalesTicketCount: todayHandledAfterSalesTicketCount,
		AIFeedbackCount:                   aiFeedbackCount,
		AIFeedbackLikeCount:               aiFeedbackLikeCount,
		AIFeedbackNegativeCount:           aiFeedbackNegativeCount,
		AIFeedbackNegativeRate:            calcRate(aiFeedbackNegativeCount, aiFeedbackCount),
		ActiveProductCount:                activeProductCount,
		ActivePromotionCount:              activePromotionCount,
		TopLeadProducts:                   buildTopLeadProducts(db, dayStart, validLeadStatuses),
		TopQuestions:                      buildTopKnowledgeQuestions(db, dayStart, dayEnd, nil),
		UnansweredQuestions:               buildTopKnowledgeQuestions(db, dayStart, dayEnd, []int{2, 3, 4}),
		TopAIFeedbackReasons:              buildTopAIFeedbackReasons(db, dayStart, dayEnd),
		RecentNegativeAIFeedbacks:         recentNegativeAIFeedbacks,
		PendingFAQDraftCount:              pendingFAQDraftCount,
		PendingFAQDrafts:                  pendingFAQDrafts,
		HighIntentLeads:                   highIntentLeads,
		PriorityFollowUps:                 priorityFollowUps,
		AfterSalesTickets:                 afterSalesTickets,
	}
	report.Summary = buildDailyBusinessReportSummary(locale, report)
	report.Highlights = buildDailyBusinessReportHighlights(locale, report)
	report.FollowUpSuggestions = buildDailyBusinessReportFollowUps(locale, report)
	report.KnowledgeSuggestions = buildDailyBusinessReportKnowledgeSuggestions(locale, report)
	return report
}

func (s *dashboardService) SendDailyBusinessReportWebhook(dateValue string, locale string, operatorID int64) (response.DashboardDailyBusinessReportPushResponse, error) {
	locale = i18nx.NormalizeLocale(locale)
	report := s.GetDailyBusinessReport(dateValue, locale)
	title := fmt.Sprintf("AI 数字店长经营日报 %s", report.ReportDate)
	body := buildDailyBusinessReportWebhookText(report)
	cfg := config.Current().Notify.DailyReport
	ret := response.DashboardDailyBusinessReportPushResponse{
		ReportDate:       report.ReportDate,
		GeneratedAt:      time.Now().Format("2006-01-02 15:04:05"),
		WebhookEnabled:   WebhookNotifyService.Enabled(),
		DailyEnabled:     cfg.Enabled,
		Title:            title,
		Message:          "日报已生成。",
		WebhookEventType: "daily_business_report",
	}
	if !WebhookNotifyService.Enabled() {
		ret.Message = "外部 Webhook 未启用，日报未发送。"
		return ret, nil
	}
	if err := WebhookNotifyService.SendText(ret.WebhookEventType, title, body, map[string]any{
		"reportDate":                        report.ReportDate,
		"conversationCount":                 report.ConversationCount,
		"leadCount":                         report.LeadCount,
		"convertedCount":                    report.ConvertedCount,
		"leadConversionRate":                report.LeadConversionRate,
		"overdueFollowUpCount":              report.OverdueFollowUpCount,
		"todayFollowUpCount":                report.TodayFollowUpCount,
		"unassignedPriorityLeadCount":       report.UnassignedPriorityLeadCount,
		"pendingAfterSalesTicketCount":      report.PendingAfterSalesTicketCount,
		"todayAfterSalesTicketCount":        report.TodayAfterSalesTicketCount,
		"todayHandledAfterSalesTicketCount": report.TodayHandledAfterSalesTicketCount,
		"aiFeedbackNegativeCount":           report.AIFeedbackNegativeCount,
		"pendingFaqDraftCount":              report.PendingFAQDraftCount,
		"operatorId":                        operatorID,
	}); err != nil {
		ret.Message = "日报发送失败。"
		return ret, err
	}
	ret.Sent = true
	ret.Message = "日报已发送到外部 Webhook。"
	return ret, nil
}

func (s *dashboardService) SendScheduledDailyBusinessReportWebhook(dateValue string, locale string) (response.DashboardDailyBusinessReportPushResponse, error) {
	reportDate, _, _ := resolveReportDay(dateValue, time.Now())
	dailyCfg := config.Current().Notify.DailyReport
	if !dailyCfg.AllowDuplicate && s.wasScheduledDailyBusinessReportSent(reportDate) {
		return response.DashboardDailyBusinessReportPushResponse{
			ReportDate:       reportDate,
			GeneratedAt:      time.Now().Format("2006-01-02 15:04:05"),
			WebhookEnabled:   WebhookNotifyService.Enabled(),
			DailyEnabled:     dailyCfg.Enabled,
			Sent:             false,
			Title:            fmt.Sprintf("AI 数字店长经营日报 %s", reportDate),
			Message:          "定时日报已发送过，已跳过重复推送。",
			WebhookEventType: "daily_business_report",
		}, nil
	}
	resp, err := s.SendDailyBusinessReportWebhook(reportDate, locale, 0)
	if err != nil {
		return resp, err
	}
	if resp.Sent && !dailyCfg.AllowDuplicate {
		if err := s.markScheduledDailyBusinessReportSent(resp.ReportDate); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

func (s *dashboardService) wasScheduledDailyBusinessReportSent(reportDate string) bool {
	item := repositories.SystemConfigRepository.Take(sqls.DB(), "config_key = ?", dashboardDailyReportLastSentConfigKey)
	return item != nil && strings.TrimSpace(item.ConfigValue) == strings.TrimSpace(reportDate)
}

func (s *dashboardService) markScheduledDailyBusinessReportSent(reportDate string) error {
	now := time.Now()
	reportDate = strings.TrimSpace(reportDate)
	item := repositories.SystemConfigRepository.Take(sqls.DB(), "config_key = ?", dashboardDailyReportLastSentConfigKey)
	if item == nil {
		return repositories.SystemConfigRepository.Create(sqls.DB(), &models.SystemConfig{
			ConfigKey:   dashboardDailyReportLastSentConfigKey,
			ConfigValue: reportDate,
			GroupCode:   "dashboard",
			Title:       "最近一次定时经营日报日期",
			Description: "用于避免定时任务重复推送同一天的老板经营日报。",
			Status:      enums.StatusOk,
			AuditFields: models.AuditFields{CreatedAt: now, UpdatedAt: now, CreateUserName: "system", UpdateUserName: "system"},
		})
	}
	return repositories.SystemConfigRepository.Updates(sqls.DB(), item.ID, map[string]any{
		"config_value":     reportDate,
		"updated_at":       now,
		"update_user_name": "system",
	})
}

func (s *dashboardService) GetAIQualityReport(rangeValue string, locale string) response.DashboardAIQualityReportResponse {
	locale = i18nx.NormalizeLocale(locale)
	now := time.Now()
	normalizedRange, days := normalizeDashboardRange(rangeValue)
	dayEnd := startOfDay(now).AddDate(0, 0, 1)
	dayStart := dayEnd.AddDate(0, 0, -days)
	db := sqls.DB()

	var retrieveTotal int64
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Count(&retrieveTotal)

	var retrieveHitTotal int64
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("hit_count > 0").
		Count(&retrieveHitTotal)

	var noAnswerCount int64
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("answer_status = ?", int(enums.KnowledgeAnswerStatusNoAnswer)).
		Count(&noAnswerCount)

	var fallbackCount int64
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("answer_status = ?", int(enums.KnowledgeAnswerStatusFallback)).
		Count(&fallbackCount)

	var blockedCount int64
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("answer_status = ?", int(enums.KnowledgeAnswerStatusBlocked)).
		Count(&blockedCount)

	feedbackCount, _, negativeFeedbackCount := s.countReportAIFeedbacks(db, dayStart, dayEnd)
	pendingFAQDraftCount, pendingFAQDrafts := s.listReportPendingFAQDrafts(db)
	recentNegativeFeedbacks := s.listReportRecentNegativeAIFeedbacks(db, dayStart, dayEnd)
	topQuestions := buildTopKnowledgeQuestions(db, dayStart, dayEnd, nil)
	unansweredQuestions := buildTopKnowledgeQuestions(db, dayStart, dayEnd, []int{
		int(enums.KnowledgeAnswerStatusNoAnswer),
		int(enums.KnowledgeAnswerStatusFallback),
		int(enums.KnowledgeAnswerStatusBlocked),
	})
	pendingQuestionGroups := s.listPendingQuestionGroups(db, dayStart, dayEnd)
	topNegativeReasons := buildTopAIFeedbackReasons(db, dayStart, dayEnd)
	recentRiskAnswerSamples := s.listRecentRiskAnswerSamples(db, dayStart, dayEnd)
	riskAnswerCount := noAnswerCount + fallbackCount + blockedCount
	todos := buildAIQualityTodos(noAnswerCount, fallbackCount, blockedCount, negativeFeedbackCount, pendingFAQDraftCount)

	report := response.DashboardAIQualityReportResponse{
		Range:                   normalizedRange,
		GeneratedAt:             now.Format("2006-01-02 15:04:05"),
		StartDate:               dayStart.Format("2006-01-02"),
		EndDate:                 dayEnd.Add(-time.Second).Format("2006-01-02"),
		RetrieveTotal:           retrieveTotal,
		RetrieveHitTotal:        retrieveHitTotal,
		RetrieveHitRate:         calcRate(retrieveHitTotal, retrieveTotal),
		NoAnswerCount:           noAnswerCount,
		FallbackCount:           fallbackCount,
		BlockedCount:            blockedCount,
		RiskAnswerCount:         riskAnswerCount,
		NegativeFeedbackCount:   negativeFeedbackCount,
		FeedbackCount:           feedbackCount,
		NegativeFeedbackRate:    calcRate(negativeFeedbackCount, feedbackCount),
		PendingFAQDraftCount:    pendingFAQDraftCount,
		TodoTotal:               int64(len(todos)),
		Todos:                   todos,
		TopQuestions:            topQuestions,
		UnansweredQuestions:     unansweredQuestions,
		TopNegativeReasons:      topNegativeReasons,
		PendingQuestionGroups:   pendingQuestionGroups,
		RecentNegativeFeedbacks: recentNegativeFeedbacks,
		PendingFAQDrafts:        pendingFAQDrafts,
		RecentRiskAnswerSamples: recentRiskAnswerSamples,
	}
	report.KnowledgeSuggestions = buildAIQualityKnowledgeSuggestions(locale, report)
	return report
}

func (s *dashboardService) GetSalesFunnelReport(rangeValue string, locale string) response.DashboardSalesFunnelReportResponse {
	locale = i18nx.NormalizeLocale(locale)
	now := time.Now()
	normalizedRange, days := normalizeDashboardRange(rangeValue)
	dayEnd := startOfDay(now).AddDate(0, 0, 1)
	dayStart := dayEnd.AddDate(0, 0, -days)
	todayStart := startOfDay(now)
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	db := sqls.DB()

	var conversationTotal int64
	db.Model(&models.Conversation{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Count(&conversationTotal)

	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("status <> ?", enums.SalesLeadStatusClosed).
		Find(&leads)

	leadTotal := int64(len(leads))
	highIntentTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.IntentLevel == enums.SalesLeadIntentHigh
	})
	appointmentTotal := countSalesFunnelLeads(leads, salesFunnelLeadHasAppointment)
	visitedTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.Status == enums.SalesLeadStatusVisited || item.Status == enums.SalesLeadStatusConverted
	})
	readyToBuyTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.BuyingStage == enums.SalesLeadStageReadyToBuy || item.Status == enums.SalesLeadStatusConverted
	})
	convertedTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.Status == enums.SalesLeadStatusConverted
	})
	invalidTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.Status == enums.SalesLeadStatusInvalid
	})
	invalidReasons := buildSalesFunnelInvalidReasons(leads, 5)
	unassignedTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.OwnerUserID == 0 && (item.Status == enums.SalesLeadStatusNew || item.Status == enums.SalesLeadStatusFollowing)
	})
	overdueFollowUpTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return (item.Status == enums.SalesLeadStatusNew || item.Status == enums.SalesLeadStatusFollowing) &&
			item.NextFollowUpAt != nil && item.NextFollowUpAt.Before(todayStart)
	})
	steps := buildSalesFunnelSteps(conversationTotal, leadTotal, highIntentTotal, appointmentTotal, visitedTotal, readyToBuyTotal, convertedTotal)
	advisorStats := buildAdvisorEfficiencyStats(db, leads, todayStart, tomorrowStart)

	report := response.DashboardSalesFunnelReportResponse{
		Range:                normalizedRange,
		GeneratedAt:          now.Format("2006-01-02 15:04:05"),
		StartDate:            dayStart.Format("2006-01-02"),
		EndDate:              dayEnd.Add(-time.Second).Format("2006-01-02"),
		ConversationTotal:    conversationTotal,
		LeadTotal:            leadTotal,
		LeadConversionRate:   calcRate(leadTotal, conversationTotal),
		ClosedConversionRate: calcRate(convertedTotal, leadTotal),
		AppointmentTotal:     appointmentTotal,
		VisitedTotal:         visitedTotal,
		ConvertedTotal:       convertedTotal,
		InvalidTotal:         invalidTotal,
		UnassignedTotal:      unassignedTotal,
		OverdueFollowUpTotal: overdueFollowUpTotal,
		InvalidReasons:       invalidReasons,
		Steps:                steps,
		AdvisorStats:         advisorStats,
	}
	report.Suggestions = buildSalesFunnelSuggestions(locale, report)
	return report
}

func (s *dashboardService) GetBusinessTrendReport(rangeValue string, locale string) response.DashboardBusinessTrendReportResponse {
	locale = i18nx.NormalizeLocale(locale)
	now := time.Now()
	normalizedRange, days := normalizeDashboardRange(rangeValue)
	dayEnd := startOfDay(now).AddDate(0, 0, 1)
	dayStart := dayEnd.AddDate(0, 0, -days)
	todayStart := startOfDay(now)
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	db := sqls.DB()

	var conversationTotal int64
	db.Model(&models.Conversation{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Count(&conversationTotal)

	var handoffTotal int64
	db.Model(&models.Conversation{}).
		Where("handoff_at >= ? AND handoff_at < ?", dayStart, dayEnd).
		Count(&handoffTotal)

	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("status <> ?", enums.SalesLeadStatusClosed).
		Find(&leads)

	leadTotal := int64(len(leads))
	highIntentTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.IntentLevel == enums.SalesLeadIntentHigh
	})
	appointmentTotal := countSalesFunnelLeads(leads, salesFunnelLeadHasAppointment)
	visitedTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.Status == enums.SalesLeadStatusVisited || item.Status == enums.SalesLeadStatusConverted
	})
	convertedTotal := countSalesFunnelLeads(leads, func(item models.SalesLead) bool {
		return item.Status == enums.SalesLeadStatusConverted
	})
	_, _, negativeFeedbackTotal := s.countReportAIFeedbacks(db, dayStart, dayEnd)
	pendingFAQDraftCount, _ := s.listReportPendingFAQDrafts(db)
	validLeadStatuses := []enums.SalesLeadStatus{
		enums.SalesLeadStatusNew,
		enums.SalesLeadStatusFollowing,
		enums.SalesLeadStatusVisited,
		enums.SalesLeadStatusConverted,
	}

	report := response.DashboardBusinessTrendReportResponse{
		Range:                  normalizedRange,
		GeneratedAt:            now.Format("2006-01-02 15:04:05"),
		StartDate:              dayStart.Format("2006-01-02"),
		EndDate:                dayEnd.Add(-time.Second).Format("2006-01-02"),
		ConversationTotal:      conversationTotal,
		LeadTotal:              leadTotal,
		LeadConversionRate:     calcRate(leadTotal, conversationTotal),
		HighIntentTotal:        highIntentTotal,
		AppointmentTotal:       appointmentTotal,
		VisitedTotal:           visitedTotal,
		ConvertedTotal:         convertedTotal,
		HandoffTotal:           handoffTotal,
		NegativeFeedbackTotal:  negativeFeedbackTotal,
		PendingFAQDraftCount:   pendingFAQDraftCount,
		Series:                 buildBusinessTrendSeries(db, dayStart, days),
		TopProducts:            buildTopLeadProductsInRange(db, dayStart, dayEnd, validLeadStatuses),
		TopChannels:            buildTopLeadChannels(db, dayStart, dayEnd),
		TopQuestions:           buildTopKnowledgeQuestions(db, dayStart, dayEnd, nil),
		TopUnansweredQuestions: buildTopKnowledgeQuestions(db, dayStart, dayEnd, []int{2, 3, 4}),
		TopNegativeReasons:     buildTopAIFeedbackReasons(db, dayStart, dayEnd),
		AdvisorStats:           buildAdvisorEfficiencyStats(db, leads, todayStart, tomorrowStart),
	}
	report.Suggestions = buildBusinessTrendSuggestions(locale, report)
	report.ReportMarkdown = buildBusinessTrendReportMarkdown(report)
	return report
}

func (s *dashboardService) GetABTestReport(rangeValue string, locale string) response.DashboardABTestReportResponse {
	locale = i18nx.NormalizeLocale(locale)
	now := time.Now()
	normalizedRange, days := normalizeDashboardRange(rangeValue)
	dayEnd := startOfDay(now).AddDate(0, 0, 1)
	dayStart := dayEnd.AddDate(0, 0, -days)
	db := sqls.DB()

	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("status <> ?", enums.SalesLeadStatusClosed).
		Find(&leads)

	feedbackTotal, _, negativeFeedbackTotal := s.countReportAIFeedbacks(db, dayStart, dayEnd)
	variants := buildABTestVariants(leads)
	report := response.DashboardABTestReportResponse{
		Range:                 normalizedRange,
		GeneratedAt:           now.Format("2006-01-02 15:04:05"),
		StartDate:             dayStart.Format("2006-01-02"),
		EndDate:               dayEnd.Add(-time.Second).Format("2006-01-02"),
		VariantTotal:          int64(len(variants)),
		LeadTotal:             int64(len(leads)),
		FeedbackTotal:         feedbackTotal,
		NegativeFeedbackTotal: negativeFeedbackTotal,
		NegativeFeedbackRate:  calcRate(negativeFeedbackTotal, feedbackTotal),
		Variants:              variants,
	}
	report.Suggestions = buildABTestSuggestions(locale, report)
	return report
}

func countSalesFunnelLeads(leads []models.SalesLead, match func(models.SalesLead) bool) int64 {
	var count int64
	for _, item := range leads {
		if match(item) {
			count++
		}
	}
	return count
}

func salesFunnelLeadHasAppointment(item models.SalesLead) bool {
	return item.BuyingStage == enums.SalesLeadStageAppointment ||
		item.BuyingStage == enums.SalesLeadStageReadyToBuy ||
		item.Status == enums.SalesLeadStatusVisited ||
		item.Status == enums.SalesLeadStatusConverted ||
		item.AppointmentAt != nil ||
		strings.TrimSpace(item.AppointmentTimeText) != "" ||
		strings.TrimSpace(item.AppointmentStore) != ""
}

func buildSalesFunnelSteps(conversations, leads, highIntent, appointment, visited, readyToBuy, converted int64) []response.DashboardSalesFunnelStep {
	items := []struct {
		key        string
		label      string
		count      int64
		actionHref string
	}{
		{key: "consultation", label: "咨询", count: conversations, actionHref: "/dashboard/conversations"},
		{key: "lead", label: "留资", count: leads, actionHref: "/dashboard/sales-leads"},
		{key: "high_intent", label: "高意向", count: highIntent, actionHref: "/dashboard/sales-leads?intent=high"},
		{key: "appointment", label: "预约", count: appointment, actionHref: "/dashboard/sales-leads?appointmentStatus=upcoming"},
		{key: "visited", label: "到店", count: visited, actionHref: "/dashboard/sales-leads?status=visited"},
		{key: "ready_to_buy", label: "准成交", count: readyToBuy, actionHref: "/dashboard/sales-leads?taskView=high_intent"},
		{key: "converted", label: "成交", count: converted, actionHref: "/dashboard/sales-leads?status=converted"},
	}
	ret := make([]response.DashboardSalesFunnelStep, 0, len(items))
	base := conversations
	for i, item := range items {
		previous := item.count
		if i > 0 {
			previous = items[i-1].count
		}
		dropOff := previous - item.count
		if dropOff < 0 {
			dropOff = 0
		}
		ret = append(ret, response.DashboardSalesFunnelStep{
			Key:          item.key,
			Label:        item.label,
			Count:        item.count,
			Rate:         calcRate(item.count, base),
			DropOffCount: dropOff,
			DropOffRate:  calcRate(dropOff, previous),
			ActionHref:   item.actionHref,
		})
	}
	return ret
}

func buildAdvisorEfficiencyStats(db *gorm.DB, leads []models.SalesLead, todayStart, tomorrowStart time.Time) []response.DashboardAdvisorEfficiency {
	type counter struct {
		response.DashboardAdvisorEfficiency
		firstFollowUpTotalMinutes int64
		firstFollowUpLeadCount    int64
		invalidReasonCounts       map[string]int64
	}
	counters := map[int64]*counter{}
	leadIDs := make([]int64, 0, len(leads))
	leadByID := map[int64]models.SalesLead{}
	ownerIDs := make([]int64, 0)
	ownerSeen := map[int64]bool{}
	for _, lead := range leads {
		leadIDs = append(leadIDs, lead.ID)
		leadByID[lead.ID] = lead
		ownerID := lead.OwnerUserID
		if _, ok := counters[ownerID]; !ok {
			counters[ownerID] = &counter{
				DashboardAdvisorEfficiency: response.DashboardAdvisorEfficiency{OwnerUserID: ownerID},
				invalidReasonCounts:        map[string]int64{},
			}
		}
		if ownerID > 0 && !ownerSeen[ownerID] {
			ownerSeen[ownerID] = true
			ownerIDs = append(ownerIDs, ownerID)
		}
		current := counters[ownerID]
		current.AssignedLeadCount++
		if lead.Status == enums.SalesLeadStatusConverted {
			current.ConvertedLeadCount++
		}
		if lead.Status == enums.SalesLeadStatusInvalid {
			current.InvalidLeadCount++
			current.invalidReasonCounts[inferSalesFunnelInvalidReason(lead)]++
		}
		if (lead.Status == enums.SalesLeadStatusNew || lead.Status == enums.SalesLeadStatusFollowing) && lead.NextFollowUpAt != nil {
			if lead.NextFollowUpAt.Before(todayStart) {
				current.OverdueFollowUpCount++
			} else if lead.NextFollowUpAt.Before(tomorrowStart) {
				current.TodayFollowUpCount++
			}
		}
	}
	ownerNames := dashboardOwnerNameMap(db, ownerIDs)
	followUps := make([]models.LeadFollowUp, 0)
	if len(leadIDs) > 0 {
		db.Model(&models.LeadFollowUp{}).
			Where("lead_id IN ?", leadIDs).
			Order("created_at ASC, id ASC").
			Find(&followUps)
	}
	firstFollowUpSeen := map[int64]bool{}
	for _, followUp := range followUps {
		lead := leadByID[followUp.LeadID]
		ownerID := lead.OwnerUserID
		current := counters[ownerID]
		if current == nil {
			continue
		}
		current.FollowUpCount++
		if firstFollowUpSeen[followUp.LeadID] || followUp.CreatedAt.Before(lead.CreatedAt) {
			continue
		}
		firstFollowUpSeen[followUp.LeadID] = true
		current.firstFollowUpLeadCount++
		current.firstFollowUpTotalMinutes += int64(followUp.CreatedAt.Sub(lead.CreatedAt).Minutes())
	}
	ret := make([]response.DashboardAdvisorEfficiency, 0, len(counters))
	for ownerID, item := range counters {
		if ownerID == 0 {
			item.OwnerUserName = "未分配"
		} else {
			item.OwnerUserName = ownerNames[ownerID]
			if item.OwnerUserName == "" {
				item.OwnerUserName = fmt.Sprintf("用户 #%d", ownerID)
			}
		}
		item.ConversionRate = calcRate(item.ConvertedLeadCount, item.AssignedLeadCount)
		item.InvalidRate = calcRate(item.InvalidLeadCount, item.AssignedLeadCount)
		if item.firstFollowUpLeadCount > 0 {
			item.AverageFirstFollowUpMinutes = item.firstFollowUpTotalMinutes / item.firstFollowUpLeadCount
		}
		item.InvalidReasons = dashboardTopItems(item.invalidReasonCounts, 3)
		ret = append(ret, item.DashboardAdvisorEfficiency)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].OwnerUserID == 0 {
			return false
		}
		if ret[j].OwnerUserID == 0 {
			return true
		}
		if ret[i].ConvertedLeadCount == ret[j].ConvertedLeadCount {
			if ret[i].OverdueFollowUpCount == ret[j].OverdueFollowUpCount {
				return ret[i].AssignedLeadCount > ret[j].AssignedLeadCount
			}
			return ret[i].OverdueFollowUpCount < ret[j].OverdueFollowUpCount
		}
		return ret[i].ConvertedLeadCount > ret[j].ConvertedLeadCount
	})
	if len(ret) > 8 {
		ret = ret[:8]
	}
	return ret
}

func buildSalesFunnelInvalidReasons(leads []models.SalesLead, limit int) []response.DashboardTopItemResponse {
	counts := map[string]int64{}
	for _, lead := range leads {
		if lead.Status != enums.SalesLeadStatusInvalid {
			continue
		}
		counts[inferSalesFunnelInvalidReason(lead)]++
	}
	return dashboardTopItems(counts, limit)
}

func inferSalesFunnelInvalidReason(lead models.SalesLead) string {
	text := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		lead.Remark,
		lead.DemandSummary,
		lead.MergeReason,
		lead.SourceChannel,
	}, " ")))
	switch {
	case containsAnyLeadText(text, "预算", "budget", "太贵", "贵", "价格", "没钱", "超预算"):
		return "预算不匹配"
	case containsAnyLeadText(text, "联系不上", "空号", "停机", "无人接", "不接", "电话错误", "号码错误", "手机号错"):
		return "联系不上"
	case containsAnyLeadText(text, "重复", "duplicate", "已存在", "归并", "merge"):
		return "重复线索"
	case containsAnyLeadText(text, "售后", "投诉", "退款", "退货", "维修", "质保", "保修", "换货"):
		return "售后/投诉"
	case containsAnyLeadText(text, "不需要", "暂不", "不考虑", "已购买", "买过", "只是看看", "无需求", "没需求"):
		return "暂无需求"
	case containsAnyLeadText(text, "同行", "广告", "刷单", "测试", "无效流量", "垃圾", "机器人"):
		return "渠道质量问题"
	default:
		return "其他原因"
	}
}

func dashboardOwnerNameMap(db *gorm.DB, ownerIDs []int64) map[int64]string {
	ret := map[int64]string{}
	if len(ownerIDs) == 0 {
		return ret
	}
	var users []models.User
	db.Model(&models.User{}).
		Where("id IN ?", ownerIDs).
		Find(&users)
	for _, user := range users {
		name := strings.TrimSpace(user.Nickname)
		if name == "" {
			name = strings.TrimSpace(user.Username)
		}
		ret[user.ID] = name
	}
	return ret
}

func buildBusinessTrendSeries(db *gorm.DB, dayStart time.Time, days int) []response.DashboardBusinessTrendItem {
	if days <= 0 {
		return nil
	}
	series := make([]response.DashboardBusinessTrendItem, 0, days)
	for i := 0; i < days; i++ {
		currentStart := dayStart.AddDate(0, 0, i)
		currentEnd := currentStart.AddDate(0, 0, 1)
		item := response.DashboardBusinessTrendItem{Date: currentStart.Format("2006-01-02")}
		db.Model(&models.Conversation{}).
			Where("created_at >= ? AND created_at < ?", currentStart, currentEnd).
			Count(&item.ConversationCount)
		db.Model(&models.Conversation{}).
			Where("handoff_at >= ? AND handoff_at < ?", currentStart, currentEnd).
			Count(&item.HandoffCount)
		db.Model(&models.SalesLead{}).
			Where("created_at >= ? AND created_at < ? AND status <> ?", currentStart, currentEnd, enums.SalesLeadStatusClosed).
			Count(&item.LeadCount)
		db.Model(&models.SalesLead{}).
			Where("created_at >= ? AND created_at < ? AND status <> ? AND intent_level = ?", currentStart, currentEnd, enums.SalesLeadStatusClosed, enums.SalesLeadIntentHigh).
			Count(&item.HighIntentCount)
		db.Model(&models.SalesLead{}).
			Where("created_at >= ? AND created_at < ? AND status <> ? AND (buying_stage IN ? OR appointment_at IS NOT NULL OR appointment_time_text <> '' OR appointment_store <> '')",
				currentStart, currentEnd, enums.SalesLeadStatusClosed, []enums.SalesLeadStage{
					enums.SalesLeadStageAppointment,
					enums.SalesLeadStageReadyToBuy,
				}).
			Count(&item.AppointmentCount)
		db.Model(&models.SalesLead{}).
			Where("updated_at >= ? AND updated_at < ? AND status IN ?", currentStart, currentEnd, []enums.SalesLeadStatus{
				enums.SalesLeadStatusVisited,
				enums.SalesLeadStatusConverted,
			}).
			Count(&item.VisitedCount)
		db.Model(&models.SalesLead{}).
			Where("updated_at >= ? AND updated_at < ? AND status = ?", currentStart, currentEnd, enums.SalesLeadStatusConverted).
			Count(&item.ConvertedCount)
		db.Model(&models.KnowledgeFeedback{}).
			Where("created_at >= ? AND created_at < ? AND feedback_type <> ?", currentStart, currentEnd, int(enums.KnowledgeFeedbackTypeLike)).
			Count(&item.NegativeFeedbackCount)
		series = append(series, item)
	}
	return series
}

func buildTopLeadProductsInRange(db *gorm.DB, dayStart, dayEnd time.Time, validStatuses []enums.SalesLeadStatus) []response.DashboardTopItemResponse {
	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Select("interested_products").
		Where("created_at >= ? AND created_at < ? AND status IN ? AND interested_products <> ''", dayStart, dayEnd, validStatuses).
		Find(&leads)

	productCounts := make(map[string]int64)
	for _, lead := range leads {
		for _, name := range splitLeadProductNames(lead.InterestedProducts) {
			productCounts[name]++
		}
	}
	return dashboardTopItems(productCounts, 5)
}

func buildTopLeadChannels(db *gorm.DB, dayStart, dayEnd time.Time) []response.DashboardTopItemResponse {
	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Select("source_channel").
		Where("created_at >= ? AND created_at < ? AND status <> ?", dayStart, dayEnd, enums.SalesLeadStatusClosed).
		Find(&leads)

	counts := make(map[string]int64)
	for _, lead := range leads {
		name := strings.TrimSpace(lead.SourceChannel)
		if name == "" {
			name = "未标记渠道"
		}
		counts[name]++
	}
	return dashboardTopItems(counts, 5)
}

func dashboardTopItems(counts map[string]int64, limit int) []response.DashboardTopItemResponse {
	items := make([]response.DashboardTopItemResponse, 0, len(counts))
	for name, count := range counts {
		items = append(items, response.DashboardTopItemResponse{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func buildABTestVariants(leads []models.SalesLead) []response.DashboardABTestVariantResult {
	type counter struct {
		response.DashboardABTestVariantResult
		productCounts map[string]int64
	}
	counters := map[string]*counter{}
	for _, lead := range leads {
		code := strings.TrimSpace(lead.SourceChannel)
		if code == "" {
			code = "untracked"
		}
		current := counters[code]
		if current == nil {
			current = &counter{
				DashboardABTestVariantResult: response.DashboardABTestVariantResult{
					VariantCode: code,
					VariantName: abVariantDisplayName(code),
				},
				productCounts: map[string]int64{},
			}
			counters[code] = current
		}
		current.LeadCount++
		if lead.IntentLevel == enums.SalesLeadIntentHigh {
			current.HighIntentCount++
		}
		if salesFunnelLeadHasAppointment(lead) {
			current.AppointmentCount++
		}
		if lead.Status == enums.SalesLeadStatusVisited || lead.Status == enums.SalesLeadStatusConverted {
			current.VisitedCount++
		}
		if lead.Status == enums.SalesLeadStatusConverted {
			current.ConvertedCount++
		}
		if lead.Status == enums.SalesLeadStatusInvalid {
			current.InvalidCount++
		}
		for _, product := range splitLeadProductNames(lead.InterestedProducts) {
			current.productCounts[product]++
		}
	}
	ret := make([]response.DashboardABTestVariantResult, 0, len(counters))
	for _, item := range counters {
		item.HighIntentRate = calcRate(item.HighIntentCount, item.LeadCount)
		item.AppointmentRate = calcRate(item.AppointmentCount, item.LeadCount)
		item.VisitRate = calcRate(item.VisitedCount, item.LeadCount)
		item.ConversionRate = calcRate(item.ConvertedCount, item.LeadCount)
		item.InvalidRate = calcRate(item.InvalidCount, item.LeadCount)
		if topProducts := dashboardTopItems(item.productCounts, 1); len(topProducts) > 0 {
			item.TopProduct = topProducts[0].Name
		}
		item.QualityRiskLevel, item.QualityRiskReason = buildABVariantQualityRisk(item.DashboardABTestVariantResult)
		item.RecommendedAction = buildABVariantRecommendedAction(item.DashboardABTestVariantResult)
		ret = append(ret, item.DashboardABTestVariantResult)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].ConvertedCount == ret[j].ConvertedCount {
			if ret[i].AppointmentCount == ret[j].AppointmentCount {
				return ret[i].LeadCount > ret[j].LeadCount
			}
			return ret[i].AppointmentCount > ret[j].AppointmentCount
		}
		return ret[i].ConvertedCount > ret[j].ConvertedCount
	})
	if len(ret) > 8 {
		ret = ret[:8]
	}
	return ret
}

func abVariantDisplayName(code string) string {
	code = strings.TrimSpace(code)
	if code == "" || code == "untracked" {
		return "未标记版本"
	}
	replacer := strings.NewReplacer("_", " ", "-", " ", "ab:", "", "AB:", "")
	return strings.TrimSpace(replacer.Replace(code))
}

func buildABVariantQualityRisk(item response.DashboardABTestVariantResult) (string, string) {
	if item.LeadCount < 5 && item.InvalidRate >= 50 {
		return "medium", "样本偏少但无效率已偏高，先复查入口人群和留资判断。"
	}
	if item.LeadCount < 5 {
		return "sample_low", "样本不足，先继续观察，避免过早判断话术优劣。"
	}
	if item.InvalidRate >= 30 {
		return "high", "无效率偏高，可能吸引了不匹配人群或承诺口径过强。"
	}
	if item.AppointmentRate >= 30 && item.VisitRate < 15 {
		return "medium", "预约到店断层明显，需要加强到店确认、路线和利益点提醒。"
	}
	if item.HighIntentRate >= 40 && item.AppointmentRate < 20 {
		return "medium", "高意向未有效转预约，留资后的邀约承接需要优化。"
	}
	if item.ConversionRate >= 15 || item.AppointmentRate >= 35 {
		return "low", "转化表现较稳，可在继续观察质量反馈的前提下扩大使用。"
	}
	return "neutral", "暂无明显风险，建议继续和头部版本做周度对比。"
}

func buildABVariantRecommendedAction(item response.DashboardABTestVariantResult) string {
	if item.LeadCount < 5 {
		return "样本偏少，继续积累后再判断。"
	}
	if item.QualityRiskLevel == "high" {
		return "质量风险偏高，先暂停放量，复盘口径、人群和无效线索来源。"
	}
	if item.AppointmentRate >= 30 && item.VisitRate < 15 {
		return "预约意向不错，但到店承接偏弱，优化到店前确认和路线提醒。"
	}
	if item.ConversionRate >= 15 || item.AppointmentRate >= 35 {
		return "表现较好，可扩大投放或复制话术。"
	}
	if item.HighIntentRate >= 40 && item.AppointmentRate < 20 {
		return "能吸引高意向，但预约承接偏弱，优化预约引导。"
	}
	if item.InvalidRate >= 30 {
		return "无效率偏高，检查入口人群、承诺口径和留资判断。"
	}
	return "表现中性，建议和头部版本对比开场白与权益表达。"
}

func buildSalesFunnelSuggestions(locale string, report response.DashboardSalesFunnelReportResponse) []string {
	suggestions := make([]string, 0, 5)
	if report.ConversationTotal > 0 && report.LeadConversionRate < 20 {
		suggestions = append(suggestions, fmt.Sprintf("咨询到留资转化率 %.1f%%，建议优化开场白、优惠权益和留资引导。", report.LeadConversionRate))
	}
	if report.LeadTotal > 0 && report.ClosedConversionRate < 10 {
		suggestions = append(suggestions, fmt.Sprintf("留资到成交转化率 %.1f%%，建议复盘高意向线索跟进节奏和报价确认。", report.ClosedConversionRate))
	}
	if report.AppointmentTotal > 0 && calcRate(report.VisitedTotal, report.AppointmentTotal) < 50 {
		suggestions = append(suggestions, fmt.Sprintf("预约到店率 %.1f%%，建议提前一天确认时间、门店和体验产品，并及时把已到店客户标记出来。", calcRate(report.VisitedTotal, report.AppointmentTotal)))
	}
	if report.UnassignedTotal > 0 {
		suggestions = append(suggestions, fmt.Sprintf("还有 %d 条线索未分配，建议使用线索页认领或调整自动分配规则。", report.UnassignedTotal))
	}
	if report.OverdueFollowUpTotal > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 条线索已逾期未跟进，优先处理会直接影响预约和成交。", report.OverdueFollowUpTotal))
	}
	if report.InvalidTotal > 0 && report.InvalidTotal >= report.ConvertedTotal {
		suggestions = append(suggestions, fmt.Sprintf("无效线索 %d 条，已接近或超过成交数，建议检查渠道和 AI 留资判断口径。", report.InvalidTotal))
	}
	if len(report.Steps) >= 4 {
		maxDrop := report.Steps[1]
		for _, step := range report.Steps[2:] {
			if step.DropOffCount > maxDrop.DropOffCount {
				maxDrop = step
			}
		}
		if maxDrop.DropOffCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("当前最大流失发生在「%s」前一环节，流失 %d 个，建议针对该环节做话术和跟进 SOP。", maxDrop.Label, maxDrop.DropOffCount))
		}
	}
	if len(suggestions) == 0 {
		if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
			suggestions = append(suggestions, "The sales funnel looks healthy for the selected period. Keep monitoring unassigned and overdue leads.")
		} else {
			suggestions = append(suggestions, "当前周期线索漏斗较健康，继续关注未分配和逾期跟进线索即可。")
		}
	}
	return suggestions
}

func buildBusinessTrendSuggestions(locale string, report response.DashboardBusinessTrendReportResponse) []string {
	suggestions := make([]string, 0, 6)
	if report.ConversationTotal == 0 {
		suggestions = append(suggestions, "当前周期暂无咨询数据，建议先检查渠道入口、嵌入脚本和门店投放链接是否正常。")
	}
	if report.ConversationTotal > 0 && report.LeadConversionRate < 20 {
		suggestions = append(suggestions, fmt.Sprintf("咨询到留资转化率 %.1f%%，建议优化欢迎语、优惠权益和留资时机。", report.LeadConversionRate))
	}
	if report.LeadTotal > 0 && calcRate(report.AppointmentTotal, report.LeadTotal) < 25 {
		suggestions = append(suggestions, fmt.Sprintf("留资到预约占比 %.1f%%，建议让 AI 更早询问到店时间、门店位置和预算区间。", calcRate(report.AppointmentTotal, report.LeadTotal)))
	}
	if report.AppointmentTotal > 0 && calcRate(report.VisitedTotal, report.AppointmentTotal) < 50 {
		suggestions = append(suggestions, fmt.Sprintf("预约到店率 %.1f%%，建议把到店前确认、路线提醒和体验产品准备加入顾问 SOP。", calcRate(report.VisitedTotal, report.AppointmentTotal)))
	}
	if report.NegativeFeedbackTotal > 0 {
		suggestions = append(suggestions, fmt.Sprintf("周期内有 %d 条 AI 负反馈，优先查看负反馈原因和未解决问题，补齐 FAQ 后再复测。", report.NegativeFeedbackTotal))
	}
	if report.PendingFAQDraftCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("还有 %d 条 FAQ 草稿待确认，建议运营每天固定清一次，避免重复兜底。", report.PendingFAQDraftCount))
	}
	if len(report.TopProducts) == 0 && report.LeadTotal > 0 {
		suggestions = append(suggestions, "线索里缺少意向产品，建议检查留资抽取和产品推荐话术，否则后续很难判断哪个产品最有效。")
	}
	if len(report.TopChannels) > 0 && report.TopChannels[0].Name == "未标记渠道" && report.TopChannels[0].Count == report.LeadTotal {
		suggestions = append(suggestions, "当前线索没有来源渠道，建议为网页、企微、广告落地页分别传入渠道标识。")
	}
	if len(suggestions) == 0 {
		if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
			suggestions = append(suggestions, "The selected period looks stable. Keep reviewing top products, channels, questions, and advisor follow-up quality every week.")
		} else {
			suggestions = append(suggestions, "当前周期经营趋势较稳定，建议每周继续复盘热门产品、渠道、问题和顾问跟进质量。")
		}
	}
	return suggestions
}

func buildBusinessTrendReportMarkdown(report response.DashboardBusinessTrendReportResponse) string {
	var builder strings.Builder
	title := "经营趋势复盘"
	if report.Range == "30d" {
		title = "月度经营趋势复盘"
	} else if report.Range == "7d" {
		title = "周度经营趋势复盘"
	}
	builder.WriteString(fmt.Sprintf("# %s（%s 至 %s）\n\n", title, report.StartDate, report.EndDate))
	builder.WriteString("## 核心指标\n")
	builder.WriteString(fmt.Sprintf("- 咨询：%d\n", report.ConversationTotal))
	builder.WriteString(fmt.Sprintf("- 留资：%d，咨询到留资率：%.1f%%\n", report.LeadTotal, report.LeadConversionRate))
	builder.WriteString(fmt.Sprintf("- 高意向：%d，预约：%d，到店：%d，成交：%d\n", report.HighIntentTotal, report.AppointmentTotal, report.VisitedTotal, report.ConvertedTotal))
	builder.WriteString(fmt.Sprintf("- 转人工：%d，AI 负反馈：%d，待确认 FAQ 草稿：%d\n\n", report.HandoffTotal, report.NegativeFeedbackTotal, report.PendingFAQDraftCount))

	builder.WriteString("## 产品与渠道\n")
	builder.WriteString(markdownTopItems("热门产品", report.TopProducts, 5))
	builder.WriteString(markdownTopItems("来源渠道", report.TopChannels, 5))
	builder.WriteString("\n")

	builder.WriteString("## 问题与知识库\n")
	builder.WriteString(markdownTopItems("高频问题", report.TopQuestions, 5))
	builder.WriteString(markdownTopItems("未解决问题", report.TopUnansweredQuestions, 5))
	builder.WriteString(markdownTopItems("负反馈原因", report.TopNegativeReasons, 5))
	builder.WriteString("\n")

	builder.WriteString("## 顾问跟进\n")
	if len(report.AdvisorStats) == 0 {
		builder.WriteString("- 暂无顾问跟进数据\n")
	} else {
		for _, advisor := range report.AdvisorStats {
			builder.WriteString(fmt.Sprintf("- %s：线索 %d，跟进 %d，逾期 %d，成交 %d，无效 %d，转化率 %.1f%%，平均首跟进 %d 分钟\n",
				advisor.OwnerUserName,
				advisor.AssignedLeadCount,
				advisor.FollowUpCount,
				advisor.OverdueFollowUpCount,
				advisor.ConvertedLeadCount,
				advisor.InvalidLeadCount,
				advisor.ConversionRate,
				advisor.AverageFirstFollowUpMinutes,
			))
		}
	}
	builder.WriteString("\n## 行动建议\n")
	if len(report.Suggestions) == 0 {
		builder.WriteString("- 暂无行动建议\n")
	} else {
		for _, suggestion := range report.Suggestions {
			builder.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
	}
	return strings.TrimSpace(builder.String())
}

func markdownTopItems(title string, items []response.DashboardTopItemResponse, limit int) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("### %s\n", title))
	if len(items) == 0 {
		builder.WriteString("- 暂无数据\n")
		return builder.String()
	}
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	for _, item := range items[:limit] {
		builder.WriteString(fmt.Sprintf("- %s：%d\n", item.Name, item.Count))
	}
	return builder.String()
}

func buildABTestSuggestions(locale string, report response.DashboardABTestReportResponse) []string {
	suggestions := make([]string, 0, 4)
	if report.LeadTotal == 0 {
		suggestions = append(suggestions, "当前周期暂无线索样本。要做 A/B 对比，先让不同入口传入不同 sourceChannel，例如 opening_a、opening_b。")
		return suggestions
	}
	if report.VariantTotal <= 1 {
		suggestions = append(suggestions, "当前只有一个话术/渠道版本，建议至少准备两个入口标识，才能对比开场白、留资引导或预约话术。")
	}
	if len(report.Variants) > 0 {
		best := report.Variants[0]
		suggestions = append(suggestions, fmt.Sprintf("当前表现最好的是「%s」：留资 %d 条，预约率 %.1f%%，到店率 %.1f%%，成交率 %.1f%%。", best.VariantName, best.LeadCount, best.AppointmentRate, best.VisitRate, best.ConversionRate))
	}
	if report.NegativeFeedbackTotal > 0 {
		suggestions = append(suggestions, fmt.Sprintf("周期内 AI 负反馈 %d 条，负反馈率 %.1f%%。A/B 放量前先复核高风险回答，避免把有争议的话术继续扩大。", report.NegativeFeedbackTotal, report.NegativeFeedbackRate))
	}
	if len(report.Variants) >= 2 {
		best := report.Variants[0]
		second := report.Variants[1]
		if best.LeadCount >= 5 && second.LeadCount >= 5 && best.AppointmentRate-second.AppointmentRate >= 10 {
			suggestions = append(suggestions, fmt.Sprintf("「%s」预约率比「%s」高 %.1f 个百分点，可优先复用它的预约引导。", best.VariantName, second.VariantName, best.AppointmentRate-second.AppointmentRate))
		}
	}
	if len(suggestions) == 0 {
		if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
			suggestions = append(suggestions, "Keep at least two tracked variants active and compare appointment, visit, and conversion rates weekly.")
		} else {
			suggestions = append(suggestions, "建议每周固定复盘至少两个话术版本的预约率、到店率和成交率，再决定保留或替换。")
		}
	}
	return suggestions
}

func (s *dashboardService) listReportHighIntentLeads(db *gorm.DB, dayStart, dayEnd time.Time, validStatuses []enums.SalesLeadStatus) []response.DashboardReportLeadResponse {
	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND created_at < ? AND status IN ? AND intent_level = ?", dayStart, dayEnd, validStatuses, enums.SalesLeadIntentHigh).
		Order("created_at desc").
		Limit(10).
		Find(&leads)

	ret := make([]response.DashboardReportLeadResponse, 0, len(leads))
	for _, lead := range leads {
		ret = append(ret, response.DashboardReportLeadResponse{
			ID:                  lead.ID,
			CustomerName:        lead.CustomerName,
			Phone:               lead.Phone,
			WeChat:              lead.WeChat,
			City:                lead.City,
			InterestedProducts:  lead.InterestedProducts,
			DemandSummary:       lead.DemandSummary,
			BuyingStage:         string(lead.BuyingStage),
			AppointmentAt:       formatDashboardTimePtr(lead.AppointmentAt),
			AppointmentTimeText: lead.AppointmentTimeText,
			AppointmentStore:    lead.AppointmentStore,
			AppointmentPeople:   lead.AppointmentPeople,
			Status:              string(lead.Status),
			OwnerUserID:         lead.OwnerUserID,
			OwnerUserName:       dashboardLeadOwnerName(lead.OwnerUserID),
			NextFollowUpAt:      formatDashboardTimePtr(lead.NextFollowUpAt),
			CreatedAt:           lead.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return ret
}

func (s *dashboardService) countReportFollowUpRisks(db *gorm.DB, dayStart, dayEnd time.Time) (overdue int64, today int64, unscheduledHot int64) {
	activeLeadStatuses := []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}
	db.Model(&models.SalesLead{}).
		Where("status IN ?", activeLeadStatuses).
		Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", dayStart).
		Count(&overdue)
	db.Model(&models.SalesLead{}).
		Where("status IN ?", activeLeadStatuses).
		Where("next_follow_up_at >= ? AND next_follow_up_at < ?", dayStart, dayEnd).
		Count(&today)
	db.Model(&models.SalesLead{}).
		Where("status IN ?", activeLeadStatuses).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("next_follow_up_at IS NULL").
		Where("(intent_level = ? OR buying_stage IN ?)", enums.SalesLeadIntentHigh, []enums.SalesLeadStage{enums.SalesLeadStageAppointment, enums.SalesLeadStageReadyToBuy}).
		Count(&unscheduledHot)
	return overdue, today, unscheduledHot
}

func (s *dashboardService) countReportUnassignedPriorityLeads(db *gorm.DB, dayEnd time.Time) int64 {
	activeLeadStatuses := []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}
	var count int64
	db.Model(&models.SalesLead{}).
		Where("status IN ?", activeLeadStatuses).
		Where("owner_user_id = 0").
		Where(
			"intent_level = ? OR buying_stage IN ? OR (next_follow_up_at IS NOT NULL AND next_follow_up_at < ?)",
			enums.SalesLeadIntentHigh,
			[]enums.SalesLeadStage{enums.SalesLeadStageAppointment, enums.SalesLeadStageReadyToBuy, enums.SalesLeadStageAfterSales},
			dayEnd,
		).
		Count(&count)
	return count
}

func (s *dashboardService) countReportAppointmentRisks(db *gorm.DB, dayStart, dayEnd time.Time) (overdue int64, today int64, unscheduled int64) {
	activeLeadStatuses := []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}
	base := func() *gorm.DB {
		return db.Model(&models.SalesLead{}).
			Where("status IN ?", activeLeadStatuses).
			Where("(buying_stage = ? OR appointment_at IS NOT NULL OR appointment_time_text <> '' OR appointment_store <> '')", enums.SalesLeadStageAppointment)
	}
	base().
		Where("appointment_at IS NOT NULL AND appointment_at < ?", dayStart).
		Count(&overdue)
	base().
		Where("appointment_at >= ? AND appointment_at < ?", dayStart, dayEnd).
		Count(&today)
	base().
		Where("appointment_at IS NULL").
		Count(&unscheduled)
	return overdue, today, unscheduled
}

func (s *dashboardService) countReportAfterSalesTicketRisks(db *gorm.DB, dayStart, dayEnd time.Time) (pending int64, today int64, todayHandled int64) {
	openBase := buildDashboardAfterSalesTicketQuery(db)
	allBase := buildDashboardAfterSalesTicketKeywordQuery(db)
	openBase().Count(&pending)
	allBase().
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Count(&today)
	allBase().
		Where("status = ?", enums.TicketStatusDone).
		Where("(handled_at >= ? AND handled_at < ?) OR (handled_at IS NULL AND updated_at >= ? AND updated_at < ?)", dayStart, dayEnd, dayStart, dayEnd).
		Count(&todayHandled)
	return pending, today, todayHandled
}

func (s *dashboardService) countReportAIFeedbacks(db *gorm.DB, dayStart, dayEnd time.Time) (total int64, likes int64, negative int64) {
	base := func() *gorm.DB {
		return db.Model(&models.KnowledgeFeedback{}).
			Where("created_at >= ? AND created_at < ?", dayStart, dayEnd)
	}
	base().Count(&total)
	base().
		Where("feedback_type = ?", int(enums.KnowledgeFeedbackTypeLike)).
		Count(&likes)
	base().
		Where("feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike)).
		Count(&negative)
	return total, likes, negative
}

func (s *dashboardService) listReportAfterSalesTickets(db *gorm.DB) []response.DashboardReportTicketResponse {
	var tickets []models.Ticket
	buildDashboardAfterSalesTicketQuery(db)().
		Order("updated_at DESC, id DESC").
		Limit(6).
		Find(&tickets)

	progressMap := latestTicketProgressMap(db, tickets)
	ret := make([]response.DashboardReportTicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		progress := progressMap[ticket.ID]
		ret = append(ret, response.DashboardReportTicketResponse{
			ID:                  ticket.ID,
			TicketNo:            ticket.TicketNo,
			Title:               ticket.Title,
			Description:         ticket.Description,
			Status:              string(ticket.Status),
			CurrentAssigneeID:   ticket.CurrentAssigneeID,
			CurrentAssigneeName: dashboardTicketAssigneeName(ticket.CurrentAssigneeID),
			ConversationID:      ticket.ConversationID,
			CustomerID:          ticket.CustomerID,
			LatestProgress:      strings.TrimSpace(progress.Content),
			LatestProgressAt:    formatDashboardTime(progress.CreatedAt),
			HandledAt:           formatDashboardTimePtr(ticket.HandledAt),
			CreatedAt:           ticket.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:           ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return ret
}

func latestTicketProgressMap(db *gorm.DB, tickets []models.Ticket) map[int64]models.TicketProgress {
	ret := map[int64]models.TicketProgress{}
	ticketIDs := make([]int64, 0, len(tickets))
	for _, ticket := range tickets {
		if ticket.ID > 0 {
			ticketIDs = append(ticketIDs, ticket.ID)
		}
	}
	if len(ticketIDs) == 0 {
		return ret
	}
	var progressList []models.TicketProgress
	db.Model(&models.TicketProgress{}).
		Where("ticket_id IN ?", ticketIDs).
		Order("created_at DESC, id DESC").
		Find(&progressList)
	for _, progress := range progressList {
		if _, exists := ret[progress.TicketID]; !exists {
			ret[progress.TicketID] = progress
		}
	}
	return ret
}

func (s *dashboardService) listReportRecentNegativeAIFeedbacks(db *gorm.DB, dayStart, dayEnd time.Time) []response.DashboardAIFeedbackResponse {
	var feedbacks []models.KnowledgeFeedback
	db.Model(&models.KnowledgeFeedback{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike)).
		Order("created_at DESC, id DESC").
		Limit(5).
		Find(&feedbacks)

	logIDs := make([]int64, 0, len(feedbacks))
	for _, item := range feedbacks {
		if item.RetrieveLogID > 0 {
			logIDs = append(logIDs, item.RetrieveLogID)
		}
	}
	logMap := map[int64]models.KnowledgeRetrieveLog{}
	if len(logIDs) > 0 {
		var logs []models.KnowledgeRetrieveLog
		db.Model(&models.KnowledgeRetrieveLog{}).
			Where("id IN ?", logIDs).
			Find(&logs)
		for _, item := range logs {
			logMap[item.ID] = item
		}
	}

	ret := make([]response.DashboardAIFeedbackResponse, 0, len(feedbacks))
	for _, item := range feedbacks {
		feedbackType := enums.KnowledgeFeedbackType(item.FeedbackType)
		logItem := logMap[item.RetrieveLogID]
		ret = append(ret, response.DashboardAIFeedbackResponse{
			ID:               item.ID,
			RetrieveLogID:    item.RetrieveLogID,
			KnowledgeBaseID:  logItem.KnowledgeBaseID,
			FeedbackType:     item.FeedbackType,
			FeedbackTypeName: enums.GetKnowledgeFeedbackTypeLabel(feedbackType),
			FeedbackReason:   strings.TrimSpace(item.FeedbackReason),
			Question:         strings.TrimSpace(logItem.Question),
			AnswerStatus:     logItem.AnswerStatus,
			AnswerStatusName: enums.GetKnowledgeAnswerStatusLabel(enums.KnowledgeAnswerStatus(logItem.AnswerStatus)),
			ModelName:        logItem.ModelName,
			CreatedAt:        item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return ret
}

func (s *dashboardService) listRecentRiskAnswerSamples(db *gorm.DB, dayStart, dayEnd time.Time) []response.DashboardAIRiskAnswerItem {
	var logs []models.KnowledgeRetrieveLog
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("answer_status IN ?", []int{
			int(enums.KnowledgeAnswerStatusNoAnswer),
			int(enums.KnowledgeAnswerStatusFallback),
			int(enums.KnowledgeAnswerStatusBlocked),
		}).
		Order("created_at DESC, id DESC").
		Limit(8).
		Find(&logs)

	ret := make([]response.DashboardAIRiskAnswerItem, 0, len(logs))
	for _, item := range logs {
		status := enums.KnowledgeAnswerStatus(item.AnswerStatus)
		ret = append(ret, response.DashboardAIRiskAnswerItem{
			ID:               item.ID,
			KnowledgeBaseID:  item.KnowledgeBaseID,
			Question:         strings.TrimSpace(item.Question),
			AnswerStatus:     item.AnswerStatus,
			AnswerStatusName: enums.GetKnowledgeAnswerStatusLabel(status),
			HitCount:         item.HitCount,
			TopScore:         fmt.Sprintf("%.4f", item.TopScore),
			ModelName:        item.ModelName,
			CreatedAt:        item.CreatedAt.Format("2006-01-02 15:04:05"),
			ActionHref:       dashboardKnowledgeRetrieveLogHref(item.ID, item.KnowledgeBaseID),
		})
	}
	return ret
}

func (s *dashboardService) listPendingQuestionGroups(db *gorm.DB, dayStart, dayEnd time.Time) []response.DashboardPendingQuestionGroup {
	type aggregate struct {
		question              string
		count                 int64
		noAnswerCount         int64
		fallbackCount         int64
		blockedCount          int64
		negativeFeedbackCount int64
		latestRetrieveLogID   int64
		knowledgeBaseID       int64
		latestAt              time.Time
	}
	groups := map[string]*aggregate{}
	upsert := func(log models.KnowledgeRetrieveLog, countAnswerStatus bool, negativeFeedback bool) {
		question := normalizeDashboardQuestion(log.Question)
		if question == "" {
			question = normalizeDashboardQuestion(log.RewriteQuestion)
		}
		if question == "" {
			question = fmt.Sprintf("检索日志 #%d", log.ID)
		}
		item := groups[question]
		if item == nil {
			item = &aggregate{question: question}
			groups[question] = item
		}
		item.count++
		if countAnswerStatus {
			switch enums.KnowledgeAnswerStatus(log.AnswerStatus) {
			case enums.KnowledgeAnswerStatusNoAnswer:
				item.noAnswerCount++
			case enums.KnowledgeAnswerStatusFallback:
				item.fallbackCount++
			case enums.KnowledgeAnswerStatusBlocked:
				item.blockedCount++
			}
		}
		if negativeFeedback {
			item.negativeFeedbackCount++
		}
		if log.CreatedAt.After(item.latestAt) || item.latestRetrieveLogID == 0 {
			item.latestAt = log.CreatedAt
			item.latestRetrieveLogID = log.ID
			item.knowledgeBaseID = log.KnowledgeBaseID
		}
	}

	var riskLogs []models.KnowledgeRetrieveLog
	db.Model(&models.KnowledgeRetrieveLog{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("answer_status IN ?", []int{
			int(enums.KnowledgeAnswerStatusNoAnswer),
			int(enums.KnowledgeAnswerStatusFallback),
			int(enums.KnowledgeAnswerStatusBlocked),
		}).
		Find(&riskLogs)
	for _, log := range riskLogs {
		upsert(log, true, false)
	}

	var negativeFeedbacks []models.KnowledgeFeedback
	db.Model(&models.KnowledgeFeedback{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike)).
		Find(&negativeFeedbacks)
	logIDs := make([]int64, 0, len(negativeFeedbacks))
	for _, feedback := range negativeFeedbacks {
		if feedback.RetrieveLogID > 0 {
			logIDs = append(logIDs, feedback.RetrieveLogID)
		}
	}
	if len(logIDs) > 0 {
		var logs []models.KnowledgeRetrieveLog
		db.Model(&models.KnowledgeRetrieveLog{}).
			Where("id IN ?", logIDs).
			Find(&logs)
		logMap := make(map[int64]models.KnowledgeRetrieveLog, len(logs))
		for _, log := range logs {
			logMap[log.ID] = log
		}
		for _, feedback := range negativeFeedbacks {
			if log, ok := logMap[feedback.RetrieveLogID]; ok {
				upsert(log, false, true)
			}
		}
	}

	ret := make([]response.DashboardPendingQuestionGroup, 0, len(groups))
	for _, item := range groups {
		ret = append(ret, response.DashboardPendingQuestionGroup{
			Question:              item.question,
			Count:                 item.count,
			NoAnswerCount:         item.noAnswerCount,
			FallbackCount:         item.fallbackCount,
			BlockedCount:          item.blockedCount,
			NegativeFeedbackCount: item.negativeFeedbackCount,
			LatestRetrieveLogID:   item.latestRetrieveLogID,
			KnowledgeBaseID:       item.knowledgeBaseID,
			LatestAt:              formatDashboardTime(item.latestAt),
			ActionHref:            dashboardKnowledgeRetrieveLogHref(item.latestRetrieveLogID, item.knowledgeBaseID),
			ActionLabel:           "查看并生成 FAQ",
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Count == ret[j].Count {
			return ret[i].LatestAt > ret[j].LatestAt
		}
		return ret[i].Count > ret[j].Count
	})
	if len(ret) > 6 {
		ret = ret[:6]
	}
	return ret
}

func (s *dashboardService) listReportPendingFAQDrafts(db *gorm.DB) (int64, []response.DashboardFAQDraftResponse) {
	base := func() *gorm.DB {
		return db.Model(&models.KnowledgeFAQ{}).
			Where("status = ?", enums.StatusDisabled).
			Where("remark LIKE ?", "%来源检索日志%")
	}
	var count int64
	base().Count(&count)

	var drafts []models.KnowledgeFAQ
	base().
		Order("created_at DESC, id DESC").
		Limit(5).
		Find(&drafts)

	ret := make([]response.DashboardFAQDraftResponse, 0, len(drafts))
	for _, item := range drafts {
		ret = append(ret, response.DashboardFAQDraftResponse{
			ID:              item.ID,
			KnowledgeBaseID: item.KnowledgeBaseID,
			Question:        strings.TrimSpace(item.Question),
			Answer:          strings.TrimSpace(item.Answer),
			Remark:          strings.TrimSpace(item.Remark),
			CreatedAt:       item.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       item.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return count, ret
}

func buildAIQualityTodos(noAnswerCount, fallbackCount, blockedCount, negativeFeedbackCount, pendingFAQDraftCount int64) []response.DashboardAIQualityTodoItem {
	todos := make([]response.DashboardAIQualityTodoItem, 0, 5)
	if noAnswerCount > 0 {
		todos = append(todos, response.DashboardAIQualityTodoItem{
			Key:         "no_answer",
			Title:       "补充无答案问题",
			Description: "AI 未能回答的问题应优先转成 FAQ 或补充产品/活动知识。",
			Count:       noAnswerCount,
			Level:       "warning",
			ActionHref:  "/dashboard/knowledge?tab=retrieveLogs&answerStatus=2",
			ActionLabel: "查看无答案",
		})
	}
	if fallbackCount > 0 {
		todos = append(todos, response.DashboardAIQualityTodoItem{
			Key:         "fallback",
			Title:       "复盘兜底回复",
			Description: "兜底回复说明知识命中或答案边界不足，建议检查热门问题和引用来源。",
			Count:       fallbackCount,
			Level:       "warning",
			ActionHref:  "/dashboard/knowledge?tab=retrieveLogs&answerStatus=3",
			ActionLabel: "查看兜底",
		})
	}
	if blockedCount > 0 {
		todos = append(todos, response.DashboardAIQualityTodoItem{
			Key:         "blocked",
			Title:       "检查风控拦截",
			Description: "风控拦截可能来自敏感承诺或行业禁用口径，需要确认话术和转人工规则。",
			Count:       blockedCount,
			Level:       "error",
			ActionHref:  "/dashboard/knowledge?tab=retrieveLogs&answerStatus=4",
			ActionLabel: "查看拦截",
		})
	}
	if negativeFeedbackCount > 0 {
		todos = append(todos, response.DashboardAIQualityTodoItem{
			Key:         "negative_feedback",
			Title:       "处理 AI 负反馈",
			Description: "点踩、无帮助和引用错误应尽快复盘，必要时生成 FAQ 草稿。",
			Count:       negativeFeedbackCount,
			Level:       "warning",
			ActionHref:  "/dashboard/knowledge?tab=retrieveLogs&feedback=negative",
			ActionLabel: "查看负反馈",
		})
	}
	if pendingFAQDraftCount > 0 {
		todos = append(todos, response.DashboardAIQualityTodoItem{
			Key:         "pending_faq_drafts",
			Title:       "确认 FAQ 草稿",
			Description: "由检索日志生成的 FAQ 草稿需要人工确认后才能进入正式知识库。",
			Count:       pendingFAQDraftCount,
			Level:       "info",
			ActionHref:  "/dashboard/knowledge?tab=documents&status=1",
			ActionLabel: "查看草稿",
		})
	}
	return todos
}

func buildAIQualityKnowledgeSuggestions(locale string, report response.DashboardAIQualityReportResponse) []string {
	suggestions := make([]string, 0, 5)
	if report.RiskAnswerCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("近 %s 有 %d 次无答案/兜底/风控回复，先处理未解决问题 Top3。", report.Range, report.RiskAnswerCount))
	}
	if len(report.UnansweredQuestions) > 0 {
		top := report.UnansweredQuestions[0]
		suggestions = append(suggestions, fmt.Sprintf("优先补充「%s」相关知识，近周期出现 %d 次未解决。", top.Name, top.Count))
	}
	if report.NegativeFeedbackCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("近周期 AI 负反馈 %d 条，负反馈率 %.1f%%，建议逐条查看引用和回答边界。", report.NegativeFeedbackCount, report.NegativeFeedbackRate))
	}
	if len(report.TopNegativeReasons) > 0 {
		top := report.TopNegativeReasons[0]
		suggestions = append(suggestions, fmt.Sprintf("负反馈最常见原因是「%s」，出现 %d 次，可针对该类问题补 FAQ 或调整话术。", top.Name, top.Count))
	}
	if report.PendingFAQDraftCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("还有 %d 条 FAQ 草稿待确认，确认后记得重建索引。", report.PendingFAQDraftCount))
	}
	if len(suggestions) == 0 {
		if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
			suggestions = append(suggestions, "AI answer quality looks stable for the selected period. Keep reviewing new negative feedback weekly.")
		} else {
			suggestions = append(suggestions, "当前周期 AI 回答质量稳定，建议每周继续复盘新增负反馈和热门问题。")
		}
	}
	return suggestions
}

func dashboardKnowledgeRetrieveLogHref(retrieveLogID int64, knowledgeBaseID int64) string {
	params := fmt.Sprintf("tab=retrieveLogs&retrieveLogId=%d", retrieveLogID)
	if knowledgeBaseID > 0 {
		params += fmt.Sprintf("&knowledgeBaseId=%d", knowledgeBaseID)
	}
	return "/dashboard/knowledge?" + params
}

func buildTopAIFeedbackReasons(db *gorm.DB, dayStart, dayEnd time.Time) []response.DashboardTopItemResponse {
	var feedbacks []models.KnowledgeFeedback
	db.Model(&models.KnowledgeFeedback{}).
		Select("feedback_type, feedback_reason").
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike)).
		Find(&feedbacks)

	counts := make(map[string]int64)
	for _, item := range feedbacks {
		reason := normalizeDashboardQuestion(item.FeedbackReason)
		if reason == "" {
			reason = enums.GetKnowledgeFeedbackTypeLabel(enums.KnowledgeFeedbackType(item.FeedbackType))
		}
		if reason == "" {
			reason = "其他"
		}
		counts[reason]++
	}
	items := make([]response.DashboardTopItemResponse, 0, len(counts))
	for reason, count := range counts {
		items = append(items, response.DashboardTopItemResponse{Name: reason, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}

func buildDashboardAfterSalesTicketKeywordQuery(db *gorm.DB) func() *gorm.DB {
	keywords := []string{"售后", "投诉", "退款", "退货", "退换", "异响", "差评", "不满意", "质保", "安装"}
	return func() *gorm.DB {
		tx := db.Model(&models.Ticket{}).
			Where("source = ?", enums.TicketSourceConversation)
		keywordTx := db.Where("title LIKE ? OR description LIKE ?", "%"+keywords[0]+"%", "%"+keywords[0]+"%")
		for _, keyword := range keywords[1:] {
			keywordTx = keywordTx.Or("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		return tx.Where(keywordTx)
	}
}

func buildDashboardAfterSalesTicketQuery(db *gorm.DB) func() *gorm.DB {
	base := buildDashboardAfterSalesTicketKeywordQuery(db)
	return func() *gorm.DB {
		return base().Where("status <> ?", enums.TicketStatusDone)
	}
}

func (s *dashboardService) listReportPriorityFollowUps(db *gorm.DB, dayStart, dayEnd time.Time) []response.DashboardReportLeadResponse {
	activeLeadStatuses := []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}
	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Where("status IN ?", activeLeadStatuses).
		Where(`(next_follow_up_at IS NOT NULL AND next_follow_up_at < ?) OR (created_at >= ? AND created_at < ? AND next_follow_up_at IS NULL AND (intent_level = ? OR buying_stage IN ?))`,
			dayEnd,
			dayStart,
			dayEnd,
			enums.SalesLeadIntentHigh,
			[]enums.SalesLeadStage{enums.SalesLeadStageAppointment, enums.SalesLeadStageReadyToBuy},
		).
		Order("CASE WHEN next_follow_up_at IS NULL THEN 1 ELSE 0 END ASC, next_follow_up_at ASC, created_at DESC").
		Limit(8).
		Find(&leads)

	ret := make([]response.DashboardReportLeadResponse, 0, len(leads))
	for _, lead := range leads {
		ret = append(ret, response.DashboardReportLeadResponse{
			ID:                  lead.ID,
			CustomerName:        lead.CustomerName,
			Phone:               lead.Phone,
			WeChat:              lead.WeChat,
			City:                lead.City,
			InterestedProducts:  lead.InterestedProducts,
			DemandSummary:       lead.DemandSummary,
			BuyingStage:         string(lead.BuyingStage),
			AppointmentAt:       formatDashboardTimePtr(lead.AppointmentAt),
			AppointmentTimeText: lead.AppointmentTimeText,
			AppointmentStore:    lead.AppointmentStore,
			AppointmentPeople:   lead.AppointmentPeople,
			Status:              string(lead.Status),
			OwnerUserID:         lead.OwnerUserID,
			OwnerUserName:       dashboardLeadOwnerName(lead.OwnerUserID),
			NextFollowUpAt:      formatDashboardTimePtr(lead.NextFollowUpAt),
			FollowUpState:       dashboardFollowUpState(lead.NextFollowUpAt, dayStart, dayEnd),
			CreatedAt:           lead.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return ret
}

func dashboardTicketAssigneeName(userID int64) string {
	if userID <= 0 {
		return ""
	}
	if user := UserService.Get(userID); user != nil {
		return user.Username
	}
	return ""
}

func dashboardLeadOwnerName(ownerUserID int64) string {
	if ownerUserID <= 0 {
		return ""
	}
	if owner := UserService.Get(ownerUserID); owner != nil {
		return owner.Username
	}
	return ""
}

func dashboardFollowUpState(value *time.Time, dayStart, dayEnd time.Time) string {
	if value == nil {
		return "unscheduled"
	}
	if value.Before(dayStart) {
		return "overdue"
	}
	if value.Before(dayEnd) {
		return "today"
	}
	return "scheduled"
}

func (s *dashboardService) buildDigitalStoreStats(db *gorm.DB, todayStart, rangeStart, now time.Time, conversationTodayCount int64, aiHandoffCount int64, locale string) response.DashboardDigitalStoreResponse {
	validLeadStatuses := []enums.SalesLeadStatus{
		enums.SalesLeadStatusNew,
		enums.SalesLeadStatusFollowing,
		enums.SalesLeadStatusVisited,
		enums.SalesLeadStatusConverted,
	}

	var todayLeads int64
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND status IN ?", todayStart, validLeadStatuses).
		Count(&todayLeads)

	var todayHighIntentLeads int64
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND status IN ? AND intent_level = ?", todayStart, validLeadStatuses, enums.SalesLeadIntentHigh).
		Count(&todayHighIntentLeads)

	var todayAppointmentLeads int64
	db.Model(&models.SalesLead{}).
		Where("created_at >= ? AND status IN ? AND buying_stage = ?", todayStart, validLeadStatuses, enums.SalesLeadStageAppointment).
		Count(&todayAppointmentLeads)

	var todayConvertedLeads int64
	db.Model(&models.SalesLead{}).
		Where("updated_at >= ? AND status = ?", todayStart, enums.SalesLeadStatusConverted).
		Count(&todayConvertedLeads)

	var pendingFollowUpLeads int64
	db.Model(&models.SalesLead{}).
		Where("status IN ?", []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}).
		Count(&pendingFollowUpLeads)

	var activeProducts int64
	db.Model(&models.Product{}).
		Where("status = ?", enums.StatusOk).
		Count(&activeProducts)

	var activePromotions int64
	db.Model(&models.Promotion{}).
		Where("status = ? AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", enums.StatusOk, now, now).
		Count(&activePromotions)

	return response.DashboardDigitalStoreResponse{
		TodayConsultations:    conversationTodayCount,
		TodayLeads:            todayLeads,
		LeadConversionRate:    calcRate(todayLeads, conversationTodayCount),
		TodayHighIntentLeads:  todayHighIntentLeads,
		TodayAppointmentLeads: todayAppointmentLeads,
		TodayConvertedLeads:   todayConvertedLeads,
		PendingFollowUpLeads:  pendingFollowUpLeads,
		ActiveProducts:        activeProducts,
		ActivePromotions:      activePromotions,
		TodayHandoffs:         aiHandoffCount,
		TopLeadProducts:       buildTopLeadProducts(db, rangeStart, validLeadStatuses),
		Summary:               buildDigitalStoreSummary(locale, todayLeads, conversationTodayCount, todayHighIntentLeads, todayAppointmentLeads, activeProducts, activePromotions),
	}
}

func (s *dashboardService) buildAgentStats(now time.Time, teams []models.AgentTeam, profiles []models.AgentProfile, schedules []models.AgentTeamSchedule, conversations []models.Conversation) (int64, int64, int64, []response.DashboardTeamLoadResponse) {
	const onlineWindow = 15 * time.Minute

	scheduledTeamIDs := make(map[int64]bool, len(schedules))
	for _, item := range schedules {
		scheduledTeamIDs[item.TeamID] = true
	}

	type teamCounter struct {
		totalAgents             int64
		onlineAgents            int64
		busyAgents              int64
		offlineAgents           int64
		waitingConversations    int64
		processingConversations int64
		maxConcurrentCapacity   int64
	}

	teamCounters := make(map[int64]*teamCounter, len(teams))
	for _, team := range teams {
		teamCounters[team.ID] = &teamCounter{}
	}

	var onlineAgents int64
	var busyAgents int64
	var offlineAgents int64

	for _, profile := range profiles {
		counter := teamCounters[profile.TeamID]
		if counter == nil {
			counter = &teamCounter{}
			teamCounters[profile.TeamID] = counter
		}
		counter.totalAgents++
		counter.maxConcurrentCapacity += int64(profile.MaxConcurrentCount)
		if profile.LastOnlineAt != nil && now.Sub(*profile.LastOnlineAt) <= onlineWindow {
			counter.onlineAgents++
			onlineAgents++
			if profile.ServiceStatus == enums.ServiceStatusBusy {
				counter.busyAgents++
				busyAgents++
			}
			continue
		}
		counter.offlineAgents++
		offlineAgents++
	}

	for _, item := range conversations {
		if item.CurrentTeamID <= 0 {
			continue
		}
		counter := teamCounters[item.CurrentTeamID]
		if counter == nil {
			counter = &teamCounter{}
			teamCounters[item.CurrentTeamID] = counter
		}
		switch item.Status {
		case enums.IMConversationStatusAIServing:
			counter.processingConversations++
		case enums.IMConversationStatusPending:
			counter.waitingConversations++
		case enums.IMConversationStatusActive:
			counter.processingConversations++
		}
	}

	teamLoads := make([]response.DashboardTeamLoadResponse, 0, len(teams))
	for _, team := range teams {
		counter := teamCounters[team.ID]
		if counter == nil {
			counter = &teamCounter{}
		}
		teamLoads = append(teamLoads, response.DashboardTeamLoadResponse{
			TeamID:                  team.ID,
			TeamName:                team.Name,
			TotalAgents:             counter.totalAgents,
			OnlineAgents:            counter.onlineAgents,
			BusyAgents:              counter.busyAgents,
			OfflineAgents:           counter.offlineAgents,
			WaitingConversations:    counter.waitingConversations,
			ProcessingConversations: counter.processingConversations,
			MaxConcurrentCapacity:   counter.maxConcurrentCapacity,
			LoadRate:                calcRate(counter.processingConversations, counter.maxConcurrentCapacity),
			HasScheduleNow:          scheduledTeamIDs[team.ID],
		})
	}

	sort.Slice(teamLoads, func(i, j int) bool {
		if teamLoads[i].WaitingConversations == teamLoads[j].WaitingConversations {
			if teamLoads[i].LoadRate == teamLoads[j].LoadRate {
				return teamLoads[i].TeamID < teamLoads[j].TeamID
			}
			return teamLoads[i].LoadRate > teamLoads[j].LoadRate
		}
		return teamLoads[i].WaitingConversations > teamLoads[j].WaitingConversations
	})

	return onlineAgents, busyAgents, offlineAgents, teamLoads
}

func (s *dashboardService) buildAlerts(now time.Time, db *gorm.DB, aiAgents []models.AIAgent, teams []models.AgentTeam, schedules []models.AgentTeamSchedule, locale string) []response.DashboardAlertResponse {
	alerts := make([]response.DashboardAlertResponse, 0, 4)
	pendingTimeout := now.Add(-10 * time.Minute)
	activeTimeout := now.Add(-30 * time.Minute)

	pendingLongWaitCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ? AND created_at < ?", enums.IMConversationStatusPending, pendingTimeout)
	})
	if pendingLongWaitCount > 0 {
		alerts = append(alerts, response.DashboardAlertResponse{
			ID:          "pending-long-wait",
			Level:       "warning",
			Title:       dashboardText(locale, "alert.pendingLongWait.title"),
			Description: dashboardText(locale, "alert.pendingLongWait.description"),
			Count:       pendingLongWaitCount,
			Link:        "/dashboard/conversations",
		})
	}

	staleProcessingCount := repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status IN ? AND (last_message_at IS NULL OR last_message_at < ?)", []enums.IMConversationStatus{
			enums.IMConversationStatusAIServing,
			enums.IMConversationStatusActive,
		}, activeTimeout)
	})
	if staleProcessingCount > 0 {
		alerts = append(alerts, response.DashboardAlertResponse{
			ID:          "stale-processing",
			Level:       "warning",
			Title:       dashboardText(locale, "alert.staleProcessing.title"),
			Description: dashboardText(locale, "alert.staleProcessing.description"),
			Count:       staleProcessingCount,
			Link:        "/dashboard/conversations",
		})
	}

	scheduledTeamIDs := make(map[int64]bool, len(schedules))
	for _, item := range schedules {
		scheduledTeamIDs[item.TeamID] = true
	}
	var scheduleMissingCount int64
	for _, team := range teams {
		if !scheduledTeamIDs[team.ID] {
			scheduleMissingCount++
		}
	}
	if scheduleMissingCount > 0 {
		alerts = append(alerts, response.DashboardAlertResponse{
			ID:          "team-no-schedule",
			Level:       "info",
			Title:       dashboardText(locale, "alert.teamNoSchedule.title"),
			Description: dashboardText(locale, "alert.teamNoSchedule.description"),
			Count:       scheduleMissingCount,
			Link:        "/dashboard/agent-team-schedules",
		})
	}

	var aiAgentWithoutKnowledgeCount int64
	for _, item := range aiAgents {
		if strings.TrimSpace(item.KnowledgeIDs) == "" {
			aiAgentWithoutKnowledgeCount++
		}
	}
	if aiAgentWithoutKnowledgeCount > 0 {
		alerts = append(alerts, response.DashboardAlertResponse{
			ID:          "ai-no-knowledge",
			Level:       "info",
			Title:       dashboardText(locale, "alert.aiNoKnowledge.title"),
			Description: dashboardText(locale, "alert.aiNoKnowledge.description"),
			Count:       aiAgentWithoutKnowledgeCount,
			Link:        "/dashboard/ai-agents",
		})
	}

	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Count == alerts[j].Count {
			return alerts[i].ID < alerts[j].ID
		}
		return alerts[i].Count > alerts[j].Count
	})

	return alerts
}

func buildConversationStatusDistribution(db *gorm.DB, locale string) []response.DashboardStatusDistributionItem {
	ret := make([]response.DashboardStatusDistributionItem, 0, len(enums.IMConversationStatusValues))
	for _, status := range enums.IMConversationStatusValues {
		ret = append(ret, response.DashboardStatusDistributionItem{
			Status: int(status),
			Label:  conversationStatusLabel(status, locale),
			Count: repositories.DashboardRepository.CountConversations(db, func(tx *gorm.DB) *gorm.DB {
				return tx.Where("status = ?", status)
			}),
		})
	}
	return ret
}

func buildConversationTrend(db *gorm.DB, start time.Time) []response.DashboardTrendItem {
	created := repositories.DashboardRepository.ListConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Select("created_at").Where("created_at >= ?", start)
	})
	closed := repositories.DashboardRepository.ListConversations(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Select("closed_at").Where("closed_at IS NOT NULL AND closed_at >= ?", start)
	})
	return buildTrendItems(start, created, closed, func(item models.Conversation) *time.Time {
		return &item.CreatedAt
	}, func(item models.Conversation) *time.Time {
		return item.ClosedAt
	})
}

func buildTrendItems(start time.Time, created []models.Conversation, closed []models.Conversation, createdAt func(models.Conversation) *time.Time, closedAt func(models.Conversation) *time.Time) []response.DashboardTrendItem {
	series := initTrendMap(start, time.Now())
	for _, item := range created {
		if ts := createdAt(item); ts != nil {
			series[ts.Format("2006-01-02")].NewCount++
		}
	}
	for _, item := range closed {
		if ts := closedAt(item); ts != nil {
			series[ts.Format("2006-01-02")].ClosedCount++
		}
	}
	return flattenTrendMap(series)
}

func initTrendMap(start, end time.Time) map[string]*response.DashboardTrendItem {
	series := make(map[string]*response.DashboardTrendItem)
	for current := startOfDay(start); !current.After(end); current = current.AddDate(0, 0, 1) {
		key := current.Format("2006-01-02")
		series[key] = &response.DashboardTrendItem{Date: key}
	}
	return series
}

func flattenTrendMap(series map[string]*response.DashboardTrendItem) []response.DashboardTrendItem {
	keys := make([]string, 0, len(series))
	for key := range series {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ret := make([]response.DashboardTrendItem, 0, len(keys))
	for _, key := range keys {
		ret = append(ret, *series[key])
	}
	return ret
}

func buildDashboardQuickLinks(locale string) []response.DashboardQuickLinkResponse {
	return []response.DashboardQuickLinkResponse{
		{Title: dashboardText(locale, "quick.conversations.title"), Description: dashboardText(locale, "quick.conversations.description"), Link: "/dashboard/conversations"},
		{Title: dashboardText(locale, "quick.agents.title"), Description: dashboardText(locale, "quick.agents.description"), Link: "/dashboard/agents"},
		{Title: dashboardText(locale, "quick.knowledge.title"), Description: dashboardText(locale, "quick.knowledge.description"), Link: "/dashboard/knowledge"},
		{Title: dashboardText(locale, "quick.aiAgents.title"), Description: dashboardText(locale, "quick.aiAgents.description"), Link: "/dashboard/ai-agents"},
		{Title: dashboardText(locale, "quick.channels.title"), Description: dashboardText(locale, "quick.channels.description"), Link: "/dashboard/channels"},
	}
}

func normalizeDashboardRange(value string) (string, int) {
	switch value {
	case "30d":
		return "30d", 30
	case "today":
		return "today", 1
	default:
		return "7d", 7
	}
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func calcRate(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	ratio := float64(numerator) / float64(denominator) * 100
	return float64(int(ratio*10+0.5)) / 10
}

func calcAIServiceRate(conversations []models.Conversation) float64 {
	var aiCount int64
	var total int64
	for _, item := range conversations {
		total++
		if item.ServiceMode == enums.IMConversationServiceModeAIOnly || item.ServiceMode == enums.IMConversationServiceModeAIFirst {
			aiCount++
		}
	}
	return calcRate(aiCount, total)
}

func buildTopLeadProducts(db *gorm.DB, rangeStart time.Time, validStatuses []enums.SalesLeadStatus) []response.DashboardTopItemResponse {
	var leads []models.SalesLead
	db.Model(&models.SalesLead{}).
		Select("interested_products").
		Where("created_at >= ? AND status IN ? AND interested_products <> ''", rangeStart, validStatuses).
		Find(&leads)

	productCounts := make(map[string]int64)
	for _, lead := range leads {
		for _, name := range splitLeadProductNames(lead.InterestedProducts) {
			productCounts[name]++
		}
	}

	items := make([]response.DashboardTopItemResponse, 0, len(productCounts))
	for name, count := range productCounts {
		items = append(items, response.DashboardTopItemResponse{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}

func splitLeadProductNames(value string) []string {
	replacer := strings.NewReplacer(
		"，", ",",
		"、", ",",
		"；", ",",
		";", ",",
		"|", ",",
		"/", ",",
		"\n", ",",
	)
	parts := strings.Split(replacer.Replace(value), ",")
	names := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func buildTopKnowledgeQuestions(db *gorm.DB, dayStart, dayEnd time.Time, answerStatuses []int) []response.DashboardTopItemResponse {
	var logs []models.KnowledgeRetrieveLog
	tx := db.Model(&models.KnowledgeRetrieveLog{}).
		Select("question, rewrite_question").
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Where("(question <> '' OR rewrite_question <> '')")
	if len(answerStatuses) > 0 {
		tx = tx.Where("answer_status IN ?", answerStatuses)
	}
	tx.Find(&logs)

	counts := make(map[string]int64)
	for _, item := range logs {
		question := normalizeDashboardQuestion(item.Question)
		if question == "" {
			question = normalizeDashboardQuestion(item.RewriteQuestion)
		}
		if question == "" {
			continue
		}
		counts[question]++
	}
	items := make([]response.DashboardTopItemResponse, 0, len(counts))
	for question, count := range counts {
		items = append(items, response.DashboardTopItemResponse{Name: question, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}

func normalizeDashboardQuestion(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	value = strings.Trim(value, " \t\r\n。.!！?？")
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) > 80 {
		value = string(runes[:80]) + "..."
	}
	return value
}

func resolveReportDay(value string, now time.Time) (string, time.Time, time.Time) {
	location := now.Location()
	if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), location); err == nil {
		start := startOfDay(parsed)
		return start.Format("2006-01-02"), start, start.AddDate(0, 0, 1)
	}
	start := startOfDay(now)
	return start.Format("2006-01-02"), start, start.AddDate(0, 0, 1)
}

func formatDashboardTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatDashboardTime(*value)
}

func formatDashboardTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func buildDailyBusinessReportSummary(locale string, report response.DashboardDailyBusinessReportResponse) string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		return fmt.Sprintf("%s: the digital store manager handled %d inquiries, sent %d AI replies, captured %d leads, identified %d high-intent customers, and closed %d leads. Lead conversion was %.1f%%.",
			report.ReportDate, report.ConversationCount, report.AIReplyCount, report.LeadCount, report.HighIntentCount, report.ConvertedCount, report.LeadConversionRate)
	}
	return fmt.Sprintf("%s 数字店长复盘：今日承接 %d 个咨询，AI 回复 %d 次，沉淀 %d 条线索，识别高意向客户 %d 条，成交线索 %d 条，留资转化率 %.1f%%。",
		report.ReportDate, report.ConversationCount, report.AIReplyCount, report.LeadCount, report.HighIntentCount, report.ConvertedCount, report.LeadConversionRate)
}

func buildDailyBusinessReportHighlights(locale string, report response.DashboardDailyBusinessReportResponse) []string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		return []string{
			fmt.Sprintf("%d active products and %d live promotions are available for recommendations.", report.ActiveProductCount, report.ActivePromotionCount),
			fmt.Sprintf("%d conversations were handed off to human agents.", report.HandoffCount),
			fmt.Sprintf("%d customers reached appointment or ready-to-buy stages.", report.AppointmentCount),
			fmt.Sprintf("%d priority leads are still unassigned.", report.UnassignedPriorityLeadCount),
			fmt.Sprintf("%d leads were marked converted today.", report.ConvertedCount),
			fmt.Sprintf("%d after-sales or complaint tickets are still open, and %d were handled today.", report.PendingAfterSalesTicketCount, report.TodayHandledAfterSalesTicketCount),
			fmt.Sprintf("%d AI answer feedbacks were recorded, with %.1f%% negative feedback.", report.AIFeedbackCount, report.AIFeedbackNegativeRate),
		}
	}
	return []string{
		fmt.Sprintf("当前可推荐 %d 个在售产品、%d 个有效活动。", report.ActiveProductCount, report.ActivePromotionCount),
		fmt.Sprintf("今日转人工 %d 次，可结合原因判断是否需要补知识库或优化话术。", report.HandoffCount),
		fmt.Sprintf("已有 %d 位客户进入预约/临门购买阶段，适合优先跟进。", report.AppointmentCount),
		fmt.Sprintf("当前还有 %d 条重点线索未分配负责人。", report.UnassignedPriorityLeadCount),
		fmt.Sprintf("今日已标记成交 %d 条线索，可用于复盘顾问转化结果。", report.ConvertedCount),
		fmt.Sprintf("当前还有 %d 个售后/投诉工单未处理，今日已处理 %d 个。", report.PendingAfterSalesTicketCount, report.TodayHandledAfterSalesTicketCount),
		fmt.Sprintf("今日收到 %d 条 AI 回答反馈，负反馈率 %.1f%%。", report.AIFeedbackCount, report.AIFeedbackNegativeRate),
	}
}

func buildDailyBusinessReportFollowUps(locale string, report response.DashboardDailyBusinessReportResponse) []string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		suggestions := []string{
			fmt.Sprintf("Follow up with %d high-intent leads first and confirm appointment times.", report.HighIntentCount),
		}
		if report.OverdueFollowUpCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d follow-ups are overdue; assign an owner and contact them before new leads.", report.OverdueFollowUpCount))
		}
		if report.TodayFollowUpCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d leads are due today; confirm next actions in the sales lead list.", report.TodayFollowUpCount))
		}
		if report.UnscheduledHotLeads > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d high-intent or appointment leads do not have a next follow-up time; schedule them now.", report.UnscheduledHotLeads))
		}
		if report.UnassignedPriorityLeadCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d priority leads are unassigned; claim or assign them before they cool down.", report.UnassignedPriorityLeadCount))
		}
		if report.OverdueAppointmentCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d appointments are overdue; confirm whether customers visited or need a new time.", report.OverdueAppointmentCount))
		}
		if report.TodayAppointmentCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d appointments are scheduled today; remind store advisors before customers arrive.", report.TodayAppointmentCount))
		}
		if report.UnscheduledAppointmentCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d appointment-intent leads still need a confirmed time or store.", report.UnscheduledAppointmentCount))
		}
		if report.PendingAfterSalesTicketCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d after-sales or complaint tickets are still open; assign an owner and update progress before closing the day.", report.PendingAfterSalesTicketCount))
		}
		if report.TodayAfterSalesTicketCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d new after-sales or complaint tickets were created today; check whether they share a product or policy issue.", report.TodayAfterSalesTicketCount))
		}
		if report.TodayHandledAfterSalesTicketCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d after-sales or complaint tickets were handled today; verify progress notes before sending the daily recap.", report.TodayHandledAfterSalesTicketCount))
		}
		if report.AIFeedbackNegativeCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d AI answers received negative feedback; review feedback reasons and update FAQ or product guidance.", report.AIFeedbackNegativeCount))
		}
		if report.PendingFAQDraftCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d FAQ drafts are waiting for review; edit and enable them after confirming the answer.", report.PendingFAQDraftCount))
		}
		if report.UnresolvedCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Review %d unresolved active conversations before closing the day.", report.UnresolvedCount))
		}
		if report.LeadCount == 0 && report.ConversationCount > 0 {
			suggestions = append(suggestions, "Review lead-capture wording because inquiries did not become leads today.")
		}
		if report.ConvertedCount == 0 && report.HighIntentCount > 0 {
			suggestions = append(suggestions, "No converted leads were recorded today; review high-intent follow-ups and mark outcomes promptly.")
		}
		return suggestions
	}

	suggestions := []string{
		fmt.Sprintf("优先跟进 %d 条高意向线索，确认试躺时间、预算和具体门店。", report.HighIntentCount),
	}
	if report.OverdueFollowUpCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 条线索已逾期未跟进，请先分配负责人并联系客户。", report.OverdueFollowUpCount))
	}
	if report.TodayFollowUpCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("今天还有 %d 条线索需要跟进，建议在销售线索页逐条记录下一步动作。", report.TodayFollowUpCount))
	}
	if report.UnscheduledHotLeads > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 条今日高意向/预约线索还没设置下次跟进时间，请立刻补齐。", report.UnscheduledHotLeads))
	}
	if report.UnassignedPriorityLeadCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 条重点线索未分配负责人，请在线索页筛选“未分配”后领取或指派顾问。", report.UnassignedPriorityLeadCount))
	}
	if report.OverdueAppointmentCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 条预约已逾期未到店，请确认客户是否到访或重新约时间。", report.OverdueAppointmentCount))
	}
	if report.TodayAppointmentCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("今天有 %d 条预约到店，请提前提醒门店顾问准备接待。", report.TodayAppointmentCount))
	}
	if report.UnscheduledAppointmentCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 条预约意向还没确认具体时间或门店，请优先补齐。", report.UnscheduledAppointmentCount))
	}
	if report.PendingAfterSalesTicketCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("有 %d 个售后/投诉工单未处理，请分配负责人并在当天更新处理进展。", report.PendingAfterSalesTicketCount))
	}
	if report.TodayAfterSalesTicketCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("今日新增 %d 个售后/投诉工单，建议复盘是否集中在某个产品、安装或质保口径。", report.TodayAfterSalesTicketCount))
	}
	if report.TodayHandledAfterSalesTicketCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("今日已处理 %d 个售后/投诉工单，请确认处理进展已写清楚，方便老板判断是否真正闭环。", report.TodayHandledAfterSalesTicketCount))
	}
	if report.AIFeedbackNegativeCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("今日有 %d 条 AI 回答负反馈，请结合反馈原因补充 FAQ、产品话术或禁用承诺。", report.AIFeedbackNegativeCount))
	}
	if report.PendingFAQDraftCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("还有 %d 条 FAQ 草稿待确认，请编辑标准答案后启用，让修正口径进入知识库。", report.PendingFAQDraftCount))
	}
	if report.UnresolvedCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("复查 %d 个仍未解决的活跃会话，避免客户流失。", report.UnresolvedCount))
	}
	if report.LeadCount == 0 && report.ConversationCount > 0 {
		suggestions = append(suggestions, "今日有咨询但没有留资，建议优化 AI 的留资邀约话术。")
	}
	if report.ConvertedCount == 0 && report.HighIntentCount > 0 {
		suggestions = append(suggestions, "今日还没有记录成交线索，请复查高意向客户跟进结果并及时标记成交或无效。")
	}
	return suggestions
}

func buildDailyBusinessReportKnowledgeSuggestions(locale string, report response.DashboardDailyBusinessReportResponse) []string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		suggestions := []string{"Review unanswered handoff reasons and add missing FAQ entries."}
		if len(report.UnansweredQuestions) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Add or improve FAQ coverage for: %s.", report.UnansweredQuestions[0].Name))
		}
		if len(report.TopLeadProducts) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Enrich product guidance for %s because it appeared in lead interest.", report.TopLeadProducts[0].Name))
		}
		if report.PendingFAQDraftCount > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Review %d pending FAQ drafts generated from AI feedback before they go stale.", report.PendingFAQDraftCount))
		}
		return suggestions
	}

	suggestions := []string{"复盘转人工原因，把重复问题补成 FAQ 或产品话术。"}
	if len(report.UnansweredQuestions) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("优先补充“%s”的标准答案，今天该问题出现未解决/兜底。", report.UnansweredQuestions[0].Name))
	}
	if len(report.TopLeadProducts) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("客户今天更关注“%s”，建议补充该产品的适用人群、价格区间和到店体验话术。", report.TopLeadProducts[0].Name))
	}
	if len(report.TopAIFeedbackReasons) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("优先复盘 AI 负反馈原因“%s”，把对应口径补进知识库或店长禁用承诺。", report.TopAIFeedbackReasons[0].Name))
	}
	if report.PendingFAQDraftCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("当前有 %d 条 FAQ 草稿待确认，建议当天完成编辑并启用，避免同类问题继续答错。", report.PendingFAQDraftCount))
	}
	return suggestions
}

func buildDailyBusinessReportWebhookText(report response.DashboardDailyBusinessReportResponse) string {
	lines := []string{
		report.Summary,
		"",
		"核心指标",
		fmt.Sprintf("- 咨询：%d，AI回复：%d，转人工：%d", report.ConversationCount, report.AIReplyCount, report.HandoffCount),
		fmt.Sprintf("- 留资：%d，高意向：%d，预约：%d，成交：%d，转化率：%.1f%%", report.LeadCount, report.HighIntentCount, report.AppointmentCount, report.ConvertedCount, report.LeadConversionRate),
		fmt.Sprintf("- 逾期跟进：%d，今日跟进：%d，未排计划高意向：%d，未分配重点线索：%d", report.OverdueFollowUpCount, report.TodayFollowUpCount, report.UnscheduledHotLeads, report.UnassignedPriorityLeadCount),
		fmt.Sprintf("- 逾期预约：%d，今日预约：%d，未定预约：%d", report.OverdueAppointmentCount, report.TodayAppointmentCount, report.UnscheduledAppointmentCount),
		fmt.Sprintf("- 售后/投诉未处理：%d，今日新增：%d，今日已处理：%d", report.PendingAfterSalesTicketCount, report.TodayAfterSalesTicketCount, report.TodayHandledAfterSalesTicketCount),
		fmt.Sprintf("- AI负反馈：%d，负反馈率：%.1f%%，待确认FAQ草稿：%d", report.AIFeedbackNegativeCount, report.AIFeedbackNegativeRate, report.PendingFAQDraftCount),
	}
	appendTopItems := func(title string, items []response.DashboardTopItemResponse) {
		lines = append(lines, "", title)
		if len(items) == 0 {
			lines = append(lines, "- 暂无")
			return
		}
		for _, item := range items {
			lines = append(lines, fmt.Sprintf("- %s（%d次）", item.Name, item.Count))
		}
	}
	appendTopItems("热门咨询问题", report.TopQuestions)
	appendTopItems("未解决问题", report.UnansweredQuestions)
	if len(report.PriorityFollowUps) > 0 {
		lines = append(lines, "", "优先跟进")
		for _, lead := range report.PriorityFollowUps {
			contact := strings.TrimSpace(lead.Phone)
			if contact == "" {
				contact = strings.TrimSpace(lead.WeChat)
			}
			if contact == "" {
				contact = "暂无联系方式"
			}
			lines = append(lines, fmt.Sprintf("- %s / %s / %s / %s", dashboardReportValueOrDash(lead.CustomerName), contact, dashboardReportValueOrDash(lead.OwnerUserName), dashboardReportValueOrDash(lead.NextFollowUpAt)))
		}
	}
	if len(report.AfterSalesTickets) > 0 {
		lines = append(lines, "", "售后/投诉工单进展")
		for _, ticket := range report.AfterSalesTickets {
			owner := dashboardReportValueOrDash(ticket.CurrentAssigneeName)
			progress := dashboardReportValueOrDash(ticket.LatestProgress)
			if ticket.LatestProgressAt != "" {
				progress = progress + "（" + ticket.LatestProgressAt + "）"
			}
			lines = append(lines, fmt.Sprintf("- %s / %s / %s / %s / 最近进展：%s",
				dashboardReportValueOrDash(ticket.TicketNo),
				dashboardReportValueOrDash(enums.GetTicketStatusLabel(enums.TicketStatus(ticket.Status))),
				owner,
				dashboardReportValueOrDash(ticket.Title),
				progress,
			))
		}
	}
	if len(report.FollowUpSuggestions) > 0 {
		lines = append(lines, "", "跟进建议")
		for _, item := range report.FollowUpSuggestions {
			lines = append(lines, "- "+item)
		}
	}
	if len(report.KnowledgeSuggestions) > 0 {
		lines = append(lines, "", "知识库建议")
		for _, item := range report.KnowledgeSuggestions {
			lines = append(lines, "- "+item)
		}
	}
	lines = append(lines, "", "后台入口：/dashboard")
	return strings.Join(lines, "\n")
}

func dashboardReportValueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func buildDigitalStoreSummary(locale string, todayLeads, todayConsultations, todayHighIntentLeads, todayAppointmentLeads, activeProducts, activePromotions int64) string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		if todayConsultations <= 0 {
			return fmt.Sprintf("The store manager is ready with %d active products and %d live promotions.", activeProducts, activePromotions)
		}
		return fmt.Sprintf("Today the store manager handled %d inquiries, captured %d leads, found %d high-intent customers, and moved %d toward appointments.", todayConsultations, todayLeads, todayHighIntentLeads, todayAppointmentLeads)
	}
	if todayConsultations <= 0 {
		return fmt.Sprintf("数字店长已准备好 %d 个在售产品和 %d 个有效活动，可开始承接客户咨询。", activeProducts, activePromotions)
	}
	return fmt.Sprintf("今日数字店长承接 %d 个咨询，沉淀 %d 条线索，其中高意向 %d 条、预约阶段 %d 条。", todayConsultations, todayLeads, todayHighIntentLeads, todayAppointmentLeads)
}

func labelOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func conversationStatusLabel(status enums.IMConversationStatus, locale string) string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		switch status {
		case enums.IMConversationStatusAIServing:
			return "AI active"
		case enums.IMConversationStatusPending:
			return "Queued"
		case enums.IMConversationStatusActive:
			return "In progress"
		case enums.IMConversationStatusClosed:
			return "Closed"
		default:
			return fmt.Sprintf("Status %d", status)
		}
	}
	return labelOrDefault(enums.GetIMConversationStatusLabel(status), fmt.Sprintf("状态 %d", status))
}

func dashboardText(locale string, key string) string {
	if i18nx.NormalizeLocale(locale) == i18nx.LocaleEnUS {
		if value, ok := dashboardEnUS[key]; ok {
			return value
		}
	}
	if value, ok := dashboardZhCN[key]; ok {
		return value
	}
	return key
}

var dashboardZhCN = map[string]string{
	"alert.pendingLongWait.title":       "待接入会话堆积",
	"alert.pendingLongWait.description": "存在超过 10 分钟仍未接入的会话，建议优先处理分配。",
	"alert.staleProcessing.title":       "处理中会话长时间无响应",
	"alert.staleProcessing.description": "部分处理中会话已超过 30 分钟没有最新消息，需要确认跟进状态。",
	"alert.teamNoSchedule.title":        "客服组当前无生效排班",
	"alert.teamNoSchedule.description":  "部分启用中的客服组当前没有生效排班，可能影响自动分配。",
	"alert.aiNoKnowledge.title":         "AI Agent 未绑定知识库",
	"alert.aiNoKnowledge.description":   "部分启用中的 AI Agent 尚未绑定知识库，回答质量可能不稳定。",
	"quick.conversations.title":         "会话管理",
	"quick.conversations.description":   "查看待接入与处理中会话",
	"quick.agents.title":                "客服档案",
	"quick.agents.description":          "查看客服状态与分组配置",
	"quick.knowledge.title":             "知识库",
	"quick.knowledge.description":       "维护文档与查看检索日志",
	"quick.aiAgents.title":              "AI Agent",
	"quick.aiAgents.description":        "配置 AI 接待策略与知识绑定",
	"quick.channels.title":              "接入渠道",
	"quick.channels.description":        "管理接入渠道与默认 Agent",
}

var dashboardEnUS = map[string]string{
	"alert.pendingLongWait.title":       "Queued conversations are piling up",
	"alert.pendingLongWait.description": "Some conversations have been waiting for more than 10 minutes. Prioritize assignment.",
	"alert.staleProcessing.title":       "Active conversations need attention",
	"alert.staleProcessing.description": "Some active conversations have had no new messages for over 30 minutes. Check their follow-up status.",
	"alert.teamNoSchedule.title":        "Agent teams have no active schedule",
	"alert.teamNoSchedule.description":  "Some enabled agent teams do not have an active schedule right now, which may affect automatic assignment.",
	"alert.aiNoKnowledge.title":         "AI Agents are missing knowledge bases",
	"alert.aiNoKnowledge.description":   "Some enabled AI Agents are not linked to a knowledge base yet, which may reduce answer quality.",
	"quick.conversations.title":         "Conversations",
	"quick.conversations.description":   "Review queued and active conversations",
	"quick.agents.title":                "Agents",
	"quick.agents.description":          "Check agent status and team setup",
	"quick.knowledge.title":             "Knowledge base",
	"quick.knowledge.description":       "Manage documents and review retrieval logs",
	"quick.aiAgents.title":              "AI Agents",
	"quick.aiAgents.description":        "Configure AI service policies and knowledge bindings",
	"quick.channels.title":              "Channels",
	"quick.channels.description":        "Manage channels and default agents",
}
