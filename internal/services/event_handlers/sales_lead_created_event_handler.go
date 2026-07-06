package event_handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/events"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/eventbus"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/services"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
)

func init() {
	eventbus.
		Register[events.SalesLeadCreatedEvent]().
		Subscribe(handleSalesLeadCreatedInAppNotification)
	eventbus.
		Register[events.SalesLeadCreatedEvent]().
		Subscribe(handleSalesLeadCreatedWxWorkNotify)
	eventbus.
		Register[events.SalesLeadCreatedEvent]().
		Subscribe(handleSalesLeadCreatedWebhookNotify)
	eventbus.
		Register[events.SalesLeadCreatedEvent]().
		Subscribe(handleSalesLeadCreatedCRMAutoSync)
}

func handleSalesLeadCreatedInAppNotification(ctx context.Context, event events.SalesLeadCreatedEvent) error {
	if event.LeadID <= 0 {
		return nil
	}
	lead := services.SalesLeadService.Get(event.LeadID)
	if lead == nil {
		return nil
	}
	recipients := resolveSalesLeadNotifyRecipients(lead)
	if len(recipients) == 0 {
		return nil
	}
	title := "新销售线索提醒"
	if lead.IntentLevel == enums.SalesLeadIntentHigh || lead.BuyingStage == enums.SalesLeadStageAppointment {
		title = "高意向销售线索提醒"
	}
	content := buildSalesLeadCreatedNotifyBody(lead, event.Reason)
	for _, recipientID := range recipients {
		if _, err := services.NotificationService.CreateAndPush(request.CreateNotificationRequest{
			RecipientUserID:  recipientID,
			Title:            title,
			Content:          content,
			NotificationType: "sales_lead_created",
			BizType:          "sales_lead",
			BizID:            lead.ID,
			ActionURL:        fmt.Sprintf("/dashboard/sales-leads?leadId=%d", lead.ID),
		}); err != nil {
			slog.Error("create sales lead in-app notification failed", "error", err, "leadId", lead.ID, "recipientUserId", recipientID)
		}
	}
	return nil
}

func handleSalesLeadCreatedWxWorkNotify(ctx context.Context, event events.SalesLeadCreatedEvent) error {
	if event.LeadID <= 0 {
		return nil
	}
	lead := services.SalesLeadService.Get(event.LeadID)
	if lead == nil {
		return nil
	}
	title := "新销售线索提醒"
	if lead.IntentLevel == enums.SalesLeadIntentHigh || lead.BuyingStage == enums.SalesLeadStageAppointment {
		title = "高意向销售线索提醒"
	}
	return services.WxWorkNotifyService.SendTextToAssigneeOrDefault(lead.OwnerUserID, title, buildSalesLeadCreatedNotifyBody(lead, event.Reason))
}

func handleSalesLeadCreatedWebhookNotify(ctx context.Context, event events.SalesLeadCreatedEvent) error {
	if event.LeadID <= 0 {
		return nil
	}
	lead := services.SalesLeadService.Get(event.LeadID)
	if lead == nil {
		return nil
	}
	title := "新销售线索提醒"
	if lead.IntentLevel == enums.SalesLeadIntentHigh || lead.BuyingStage == enums.SalesLeadStageAppointment {
		title = "高意向销售线索提醒"
	}
	return services.WebhookNotifyService.SendText("sales_lead_created", title, buildSalesLeadCreatedNotifyBody(lead, event.Reason), map[string]any{
		"leadId":         lead.ID,
		"conversationId": lead.ConversationID,
		"ownerUserId":    lead.OwnerUserID,
		"actionUrl":      fmt.Sprintf("/dashboard/sales-leads?leadId=%d", lead.ID),
	})
}

func handleSalesLeadCreatedCRMAutoSync(ctx context.Context, event events.SalesLeadCreatedEvent) error {
	if event.LeadID <= 0 {
		return nil
	}
	lead := services.SalesLeadService.Get(event.LeadID)
	if lead == nil || !shouldAutoSyncSalesLeadToCRM(lead) {
		return nil
	}
	remark := "AI数字店长自动同步"
	if reason := strings.TrimSpace(event.Reason); reason != "" {
		remark += "：" + reason
	}
	resp, err := services.SalesLeadService.SyncToCRM(request.SyncSalesLeadToCRMRequest{
		ID:     lead.ID,
		Remark: remark,
	}, &dto.AuthPrincipal{Username: "system"})
	if err != nil {
		slog.Error("auto sync sales lead to CRM failed", "error", err, "leadId", lead.ID)
		return nil
	}
	if resp.WebhookEnabled && resp.Sent {
		slog.Info("auto synced sales lead to CRM", "leadId", lead.ID, "eventType", resp.WebhookEventType)
	}
	return nil
}

func shouldAutoSyncSalesLeadToCRM(lead *models.SalesLead) bool {
	if lead == nil {
		return false
	}
	if lead.Status == enums.SalesLeadStatusConverted || lead.Status == enums.SalesLeadStatusVisited {
		return true
	}
	if lead.IntentLevel == enums.SalesLeadIntentHigh {
		return true
	}
	if lead.BuyingStage == enums.SalesLeadStageAppointment || lead.BuyingStage == enums.SalesLeadStageReadyToBuy {
		return true
	}
	return lead.AppointmentAt != nil || strings.TrimSpace(lead.AppointmentTimeText) != "" || strings.TrimSpace(lead.AppointmentStore) != ""
}

func resolveSalesLeadNotifyRecipients(lead *models.SalesLead) []int64 {
	if lead == nil {
		return nil
	}
	if lead.OwnerUserID > 0 {
		return []int64{lead.OwnerUserID}
	}
	var users []models.User
	if err := sqls.DB().
		Where("status = ?", enums.StatusOk).
		Order("id ASC").
		Limit(20).
		Find(&users).Error; err != nil {
		slog.Error("load sales lead notification recipients failed", "error", err, "leadId", lead.ID)
		return nil
	}
	recipients := make([]int64, 0, len(users))
	for i := range users {
		if users[i].ID > 0 {
			recipients = append(recipients, users[i].ID)
		}
	}
	return recipients
}

func buildSalesLeadCreatedNotifyBody(lead *models.SalesLead, reason string) string {
	if lead == nil {
		return ""
	}
	lines := []string{
		fmt.Sprintf("客户: %s", strs.DefaultIfBlank(lead.CustomerName, "未命名客户")),
		fmt.Sprintf("联系方式: %s", salesLeadContactText(lead)),
		fmt.Sprintf("意向等级: %s", salesLeadIntentNotifyLabel(lead.IntentLevel)),
		fmt.Sprintf("购买阶段: %s", salesLeadStageNotifyLabel(lead.BuyingStage)),
	}
	if products := strings.TrimSpace(lead.InterestedProducts); products != "" {
		lines = append(lines, fmt.Sprintf("意向产品: %s", products))
	}
	if lead.BudgetMin > 0 || lead.BudgetMax > 0 {
		lines = append(lines, fmt.Sprintf("预算: %s", salesLeadBudgetNotifyText(lead)))
	}
	if appointment := salesLeadAppointmentNotifyText(lead); appointment != "" {
		lines = append(lines, fmt.Sprintf("预约: %s", appointment))
	}
	if summary := strings.TrimSpace(lead.DemandSummary); summary != "" {
		lines = append(lines, fmt.Sprintf("需求: %s", summary))
	}
	if strings.TrimSpace(reason) != "" {
		lines = append(lines, fmt.Sprintf("触发原因: %s", strings.TrimSpace(reason)))
	}
	if lead.ConversationID > 0 {
		lines = append(lines, fmt.Sprintf("会话: #%d", lead.ConversationID))
	}
	lines = append(lines, fmt.Sprintf("时间: %s", time.Now().Format(time.DateTime)))
	return strings.Join(lines, "\n")
}

func salesLeadContactText(lead *models.SalesLead) string {
	parts := make([]string, 0, 2)
	if phone := strings.TrimSpace(lead.Phone); phone != "" {
		parts = append(parts, phone)
	}
	if wechat := strings.TrimSpace(lead.WeChat); wechat != "" {
		parts = append(parts, "微信 "+wechat)
	}
	if len(parts) == 0 {
		return "暂无"
	}
	return strings.Join(parts, " / ")
}

func salesLeadBudgetNotifyText(lead *models.SalesLead) string {
	if lead.BudgetMin > 0 && lead.BudgetMax > 0 {
		return fmt.Sprintf("%d-%d元", lead.BudgetMin, lead.BudgetMax)
	}
	if lead.BudgetMax > 0 {
		return fmt.Sprintf("%d元左右", lead.BudgetMax)
	}
	if lead.BudgetMin > 0 {
		return fmt.Sprintf("%d元以上", lead.BudgetMin)
	}
	return "-"
}

func salesLeadAppointmentNotifyText(lead *models.SalesLead) string {
	parts := []string{
		utils.FormatTimePtr(lead.AppointmentAt),
		strings.TrimSpace(lead.AppointmentTimeText),
		strings.TrimSpace(lead.AppointmentStore),
	}
	if lead.AppointmentPeople > 0 {
		parts = append(parts, fmt.Sprintf("%d人", lead.AppointmentPeople))
	}
	if remark := strings.TrimSpace(lead.AppointmentRemark); remark != "" {
		parts = append(parts, remark)
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " / ")
}

func salesLeadIntentNotifyLabel(value enums.SalesLeadIntent) string {
	switch value {
	case enums.SalesLeadIntentHigh:
		return "高意向"
	case enums.SalesLeadIntentMedium:
		return "中意向"
	case enums.SalesLeadIntentLow:
		return "低意向"
	default:
		return "未知"
	}
}

func salesLeadStageNotifyLabel(value enums.SalesLeadStage) string {
	switch value {
	case enums.SalesLeadStageConsulting:
		return "咨询了解"
	case enums.SalesLeadStageComparing:
		return "对比决策"
	case enums.SalesLeadStageAppointment:
		return "预约到店"
	case enums.SalesLeadStageReadyToBuy:
		return "准备购买"
	case enums.SalesLeadStageAfterSales:
		return "售后问题"
	default:
		return "未知"
	}
}
