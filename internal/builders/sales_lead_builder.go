package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"agent-desk/internal/services"
	"fmt"
	"strings"
	"time"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

const salesLeadMessageSummaryLimit = 120

var salesLeadBuilderDB = func() *gorm.DB {
	return sqls.DB()
}

func BuildSalesLead(item *models.SalesLead) *response.SalesLeadResponse {
	if item == nil {
		return nil
	}
	autoTagDetails := buildSalesLeadAutoTagDetails(item)
	ret := &response.SalesLeadResponse{
		ID:                  item.ID,
		CustomerID:          item.CustomerID,
		ConversationID:      item.ConversationID,
		CustomerName:        item.CustomerName,
		Phone:               item.Phone,
		WeChat:              item.WeChat,
		City:                item.City,
		AddressHint:         item.AddressHint,
		BudgetMin:           item.BudgetMin,
		BudgetMax:           item.BudgetMax,
		InterestedProducts:  item.InterestedProducts,
		DemandSummary:       item.DemandSummary,
		IntentLevel:         item.IntentLevel,
		BuyingStage:         item.BuyingStage,
		AppointmentAt:       utils.FormatTimePtr(item.AppointmentAt),
		AppointmentTimeText: item.AppointmentTimeText,
		AppointmentStore:    item.AppointmentStore,
		AppointmentPeople:   item.AppointmentPeople,
		AppointmentRemark:   item.AppointmentRemark,
		SourceChannel:       item.SourceChannel,
		OwnerUserID:         item.OwnerUserID,
		Status:              item.Status,
		NextFollowUpAt:      utils.FormatTimePtr(item.NextFollowUpAt),
		LastMessageID:       item.LastMessageID,
		LastMessageSummary:  buildSalesLeadConversationSummary(item),
		LastCustomerMessage: buildSalesLeadLastCustomerMessage(item),
		MergeKey:            item.MergeKey,
		MergeReason:         item.MergeReason,
		MergedAt:            utils.FormatTimePtr(item.MergedAt),
		Remark:              item.Remark,
		AutoTags:            salesLeadAutoTagLabels(autoTagDetails),
		AutoTagDetails:      autoTagDetails,
		CreatedAt:           utils.FormatTime(item.CreatedAt),
		UpdatedAt:           utils.FormatTime(item.UpdatedAt),
	}
	if item.CustomerID > 0 {
		ret.Customer = BuildCustomer(services.CustomerService.Get(item.CustomerID))
	}
	if item.OwnerUserID > 0 {
		owner := services.UserService.Get(item.OwnerUserID)
		if owner != nil {
			ret.OwnerUserName = owner.Username
		}
	}
	return ret
}

func buildSalesLeadConversationSummary(item *models.SalesLead) string {
	if item == nil || item.ConversationID <= 0 {
		return ""
	}
	db := salesLeadBuilderDB()
	if db == nil {
		return ""
	}
	conversation := repositories.ConversationRepository.Get(db, item.ConversationID)
	if conversation == nil {
		return ""
	}
	return limitSalesLeadSummary(conversation.LastMessageSummary, salesLeadMessageSummaryLimit)
}

func buildSalesLeadLastCustomerMessage(item *models.SalesLead) string {
	if item == nil || item.LastMessageID <= 0 {
		return ""
	}
	db := salesLeadBuilderDB()
	if db == nil {
		return ""
	}
	message := repositories.MessageRepository.Get(db, item.LastMessageID)
	if message == nil || message.SenderType != enums.IMSenderTypeCustomer {
		return ""
	}
	return limitSalesLeadSummary(message.Content, salesLeadMessageSummaryLimit)
}

func limitSalesLeadSummary(value string, max int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if max <= 0 || value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func buildSalesLeadAutoTags(item *models.SalesLead) []string {
	return salesLeadAutoTagLabels(buildSalesLeadAutoTagDetails(item))
}

func salesLeadAutoTagLabels(details []response.SalesLeadAutoTag) []string {
	labels := make([]string, 0, len(details))
	for _, detail := range details {
		labels = append(labels, detail.Label)
	}
	return labels
}

func buildSalesLeadAutoTagDetails(item *models.SalesLead) []response.SalesLeadAutoTag {
	if item == nil {
		return nil
	}
	tags := make([]response.SalesLeadAutoTag, 0, 8)
	add := func(label string, level string, reason string, actionLabel string, actionURL string) {
		label = strings.TrimSpace(label)
		if label == "" {
			return
		}
		for _, existing := range tags {
			if existing.Label == label {
				return
			}
		}
		tags = append(tags, response.SalesLeadAutoTag{
			Label:       label,
			Level:       level,
			Reason:      reason,
			ActionLabel: actionLabel,
			ActionURL:   actionURL,
		})
	}
	detailURL := "/dashboard/sales-leads"
	if item.ID > 0 {
		detailURL = "/dashboard/sales-leads?leadId=" + fmt.Sprint(item.ID)
	}
	if item.Status == enums.SalesLeadStatusConverted {
		add("已成交", "success", "线索状态已标记为成交。", "沉淀成交话术", detailURL)
	}
	if item.Status == enums.SalesLeadStatusVisited {
		add("已到店", "success", "客户已到店或已完成到店标记。", "补充到店结果", detailURL)
	}
	if item.IntentLevel == enums.SalesLeadIntentHigh {
		add("高意向", "hot", "AI 或顾问判断客户购买意向较强。", "优先跟进", detailURL)
	}
	if item.BuyingStage == enums.SalesLeadStageReadyToBuy {
		add("准成交", "hot", "客户已进入准成交阶段。", "确认报价与下单障碍", detailURL)
	}
	if item.BuyingStage == enums.SalesLeadStageAppointment || item.AppointmentAt != nil || item.AppointmentTimeText != "" || item.AppointmentStore != "" {
		add("已预约", "info", "客户已有预约阶段、预约时间或预约门店信息。", "发送到店提醒", detailURL)
	}
	if item.BuyingStage == enums.SalesLeadStageAfterSales {
		add("售后风险", "warning", "客户诉求进入售后或投诉相关阶段。", "转售后处理", detailURL)
	}
	if item.OwnerUserID == 0 && (item.Status == enums.SalesLeadStatusNew || item.Status == enums.SalesLeadStatusFollowing) {
		add("未分配", "warning", "当前线索还没有负责人。", "认领或分配顾问", "/dashboard/sales-leads?owner=unassigned")
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	if item.NextFollowUpAt != nil && (item.Status == enums.SalesLeadStatusNew || item.Status == enums.SalesLeadStatusFollowing) {
		if item.NextFollowUpAt.Before(todayStart) {
			add("逾期跟进", "danger", "下次跟进时间早于今天。", "立即联系客户", detailURL)
		} else if item.NextFollowUpAt.Before(tomorrowStart) {
			add("今日跟进", "info", "下次跟进时间在今天内。", "按计划跟进", detailURL)
		}
	}
	if item.Phone == "" && item.WeChat == "" {
		add("待补联系方式", "warning", "手机号和微信都为空，后续触达风险高。", "补齐联系方式", detailURL)
	} else {
		add("已留联系方式", "info", "客户已留下手机号或微信。", "安排跟进", detailURL)
	}
	if item.BudgetMax > 0 || item.BudgetMin > 0 {
		add("有预算", "info", "线索已抽取到预算区间。", "按预算推荐产品", detailURL)
	}
	if item.BudgetMax >= 20000 || item.BudgetMin >= 20000 {
		add("高预算", "hot", "预算上限或下限达到 20000 元以上。", "推荐高客单方案", detailURL)
	}
	if item.SourceChannel != "" {
		add("渠道:"+item.SourceChannel, "info", "线索带有来源渠道标识。", "复盘渠道效果", "/dashboard")
	}
	if len(tags) > 8 {
		tags = tags[:8]
	}
	return tags
}

func BuildSalesLeadList(list []models.SalesLead) []response.SalesLeadResponse {
	results := make([]response.SalesLeadResponse, 0, len(list))
	for i := range list {
		if item := BuildSalesLead(&list[i]); item != nil {
			results = append(results, *item)
		}
	}
	return results
}

func BuildLeadFollowUp(item *models.LeadFollowUp) *response.LeadFollowUpResponse {
	if item == nil {
		return nil
	}
	return &response.LeadFollowUpResponse{
		ID:             item.ID,
		LeadID:         item.LeadID,
		OperatorID:     item.OperatorID,
		OperatorName:   item.OperatorName,
		Content:        item.Content,
		NextAction:     item.NextAction,
		NextFollowUpAt: utils.FormatTimePtr(item.NextFollowUpAt),
		CreatedAt:      utils.FormatTime(item.CreatedAt),
	}
}

func BuildLeadFollowUps(list []models.LeadFollowUp) []response.LeadFollowUpResponse {
	results := make([]response.LeadFollowUpResponse, 0, len(list))
	for i := range list {
		if item := BuildLeadFollowUp(&list[i]); item != nil {
			results = append(results, *item)
		}
	}
	return results
}
