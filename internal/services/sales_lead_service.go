package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/events"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/eventbus"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var SalesLeadService = newSalesLeadService()

func newSalesLeadService() *salesLeadService {
	return &salesLeadService{}
}

type salesLeadService struct {
}

type extractedLeadInfo struct {
	CustomerName         string
	CustomerNameExplicit bool
	Phone                string
	WeChat               string
	City                 string
	AddressHint          string
	BudgetMin            int64
	BudgetMax            int64
	InterestedProducts   string
	DemandSummary        string
	IntentLevel          enums.SalesLeadIntent
	BuyingStage          enums.SalesLeadStage
	AppointmentAt        *time.Time
	AppointmentTimeText  string
	AppointmentStore     string
	AppointmentPeople    int
	AppointmentRemark    string
	HasSignal            bool
}

type salesLeadMatch struct {
	Lead        *models.SalesLead
	MergeKey    string
	MergeReason string
}

var (
	leadMobilePattern      = regexp.MustCompile(`1[3-9]\d{9}`)
	leadWeChatPattern      = regexp.MustCompile(`(?:微信|微信号|wx|VX|v信|加我)[号：:\s]*([a-zA-Z][-_a-zA-Z0-9]{5,19})`)
	leadArabicWanPattern   = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*万`)
	leadArabicYuanPattern  = regexp.MustCompile(`(\d{4,7})\s*(?:元|块|左右|以内|上下)?`)
	leadChineseWanPattern  = regexp.MustCompile(`([一二两三四五六七八九十])万([一二两三四五六七八九十])?`)
	leadSurnamePattern     = regexp.MustCompile(`(?:我姓|姓)([\p{Han}])`)
	leadNamePattern        = regexp.MustCompile(`(?:我叫|我是)([\p{Han}]{2,4})(?:，|。|,|\.|\s|$)`)
	leadHonorificPattern   = regexp.MustCompile(`([\p{Han}]{1,3})(先生|女士|小姐|老师)`)
	leadCityPattern        = regexp.MustCompile(`(?:我在|人在|位于|城市是|城市：|城市:)([\p{Han}]{2,8})`)
	leadAddressHintPattern = regexp.MustCompile(`(?:小区|地址|门店附近|附近|住在)[：:\s]*([\p{Han}A-Za-z0-9\-—_·]{2,30})`)
	leadDatePattern        = regexp.MustCompile(`(?:(20\d{2})[年\-/.])?(\d{1,2})[月\-/.](\d{1,2})[日号]?`)
	leadTimePattern        = regexp.MustCompile(`(?:(上午|中午|下午|晚上|晚间|傍晚|早上|周末|本周末|这周末|明天|后天|今天|周一|周二|周三|周四|周五|周六|周日|星期一|星期二|星期三|星期四|星期五|星期六|星期日)[\p{Han}A-Za-z0-9:：点半左右前后\-— ]{0,12})`)
	leadPeoplePattern      = regexp.MustCompile(`(\d+|[一二两三四五六七八九十])\s*(?:个)?(?:人|位|大人|成人)`)
	leadStorePattern       = regexp.MustCompile(`(?:到|去|在|约|预约)([\p{Han}A-Za-z·\-—]{2,20}(?:店|门店|旗舰店|体验店|商场|广场))`)
)

func (s *salesLeadService) Get(id int64) *models.SalesLead {
	return repositories.SalesLeadRepository.Get(sqls.DB(), id)
}

func (s *salesLeadService) FindFollowUps(leadID int64) []models.LeadFollowUp {
	if leadID <= 0 {
		return nil
	}
	return repositories.LeadFollowUpRepository.Find(sqls.DB(), sqls.NewCnd().Where("lead_id = ?", leadID).Desc("id"))
}

func (s *salesLeadService) List(req request.SalesLeadListRequest) (list []models.SalesLead, paging *sqls.Paging) {
	tx := s.buildListQuery(req)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		slog.Error("sales lead list count failed", "error", err)
	}
	if err := tx.Order("id DESC").Offset(req.Offset()).Limit(req.GetLimit()).Find(&list).Error; err != nil {
		slog.Error("sales lead list scan failed", "error", err)
	}
	return list, &sqls.Paging{Page: req.GetPage(), Limit: req.GetLimit(), Total: total}
}

func (s *salesLeadService) Export(req request.SalesLeadListRequest) []models.SalesLead {
	var list []models.SalesLead
	if err := s.buildListQuery(req).Order("id DESC").Limit(5000).Find(&list).Error; err != nil {
		slog.Error("sales lead export scan failed", "error", err)
	}
	return list
}

func (s *salesLeadService) buildListQuery(req request.SalesLeadListRequest) *gorm.DB {
	tx := sqls.DB().Model(&models.SalesLead{}).Where("status <> ?", enums.SalesLeadStatusClosed)
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		pat := "%" + kw + "%"
		tx = tx.Where(`customer_name LIKE ? OR phone LIKE ? OR we_chat LIKE ? OR city LIKE ? OR demand_summary LIKE ? OR interested_products LIKE ?`, pat, pat, pat, pat, pat, pat)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		tx = tx.Where("status = ?", status)
	}
	if intent := strings.TrimSpace(req.Intent); intent != "" {
		tx = tx.Where("intent_level = ?", intent)
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	tx = applySalesLeadTaskView(tx, strings.TrimSpace(req.TaskView), todayStart, tomorrowStart)
	switch strings.TrimSpace(req.FollowUpStatus) {
	case "overdue":
		tx = tx.Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", todayStart)
	case "today":
		tx = tx.Where("next_follow_up_at >= ? AND next_follow_up_at < ?", todayStart, tomorrowStart)
	case "scheduled":
		tx = tx.Where("next_follow_up_at >= ?", tomorrowStart)
	case "none":
		tx = tx.Where("next_follow_up_at IS NULL")
	}
	switch strings.TrimSpace(req.AppointmentStatus) {
	case "overdue":
		tx = tx.Where("appointment_at IS NOT NULL AND appointment_at < ?", todayStart)
	case "today":
		tx = tx.Where("appointment_at >= ? AND appointment_at < ?", todayStart, tomorrowStart)
	case "upcoming":
		tx = tx.Where("appointment_at >= ?", tomorrowStart)
	case "unscheduled":
		tx = tx.Where("appointment_at IS NULL").
			Where("(buying_stage = ? OR appointment_time_text <> '' OR appointment_store <> '')", enums.SalesLeadStageAppointment)
	case "all":
		tx = tx.Where("(buying_stage = ? OR appointment_at IS NOT NULL OR appointment_time_text <> '' OR appointment_store <> '')", enums.SalesLeadStageAppointment)
	}
	if req.OwnerUserID != nil {
		if *req.OwnerUserID > 0 {
			tx = tx.Where("owner_user_id = ?", *req.OwnerUserID)
		} else if *req.OwnerUserID == -1 {
			tx = tx.Where("owner_user_id = 0")
		}
	}
	return tx
}

func applySalesLeadTaskView(tx *gorm.DB, taskView string, todayStart, tomorrowStart time.Time) *gorm.DB {
	activeLeadStatuses := []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}
	switch taskView {
	case "today":
		return tx.Where("status IN ?", activeLeadStatuses).
			Where("next_follow_up_at >= ? AND next_follow_up_at < ?", todayStart, tomorrowStart)
	case "overdue":
		return tx.Where("status IN ?", activeLeadStatuses).
			Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", todayStart)
	case "high_intent":
		return tx.Where("status IN ?", activeLeadStatuses).
			Where("intent_level = ?", enums.SalesLeadIntentHigh)
	case "appointment":
		return tx.Where("status IN ?", activeLeadStatuses).
			Where("(buying_stage = ? OR appointment_at IS NOT NULL OR appointment_time_text <> '' OR appointment_store <> '')", enums.SalesLeadStageAppointment)
	case "after_sales":
		return tx.Where("status IN ?", activeLeadStatuses).
			Where("buying_stage = ?", enums.SalesLeadStageAfterSales)
	default:
		return tx
	}
}

func (s *salesLeadService) Update(req request.UpdateSalesLeadRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := repositories.SalesLeadRepository.Get(sqls.DB(), req.ID)
	if item == nil {
		return errorsx.InvalidParam("sales lead not found")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = string(item.Status)
	}
	if !enums.IsValidSalesLeadStatus(status) {
		return errorsx.InvalidParam("invalid sales lead status")
	}
	updates := map[string]any{
		"customer_name":         strings.TrimSpace(req.CustomerName),
		"phone":                 normalizeLeadPhone(req.Phone),
		"we_chat":               strings.TrimSpace(req.WeChat),
		"city":                  strings.TrimSpace(req.City),
		"address_hint":          strings.TrimSpace(req.AddressHint),
		"budget_min":            req.BudgetMin,
		"budget_max":            req.BudgetMax,
		"interested_products":   strings.TrimSpace(req.InterestedProducts),
		"demand_summary":        strings.TrimSpace(req.DemandSummary),
		"intent_level":          normalizeLeadIntent(req.IntentLevel, item.IntentLevel),
		"buying_stage":          normalizeLeadStage(req.BuyingStage, item.BuyingStage),
		"appointment_at":        parseLeadTimePtr(req.AppointmentAt),
		"appointment_time_text": strings.TrimSpace(req.AppointmentTimeText),
		"appointment_store":     strings.TrimSpace(req.AppointmentStore),
		"appointment_people":    req.AppointmentPeople,
		"appointment_remark":    strings.TrimSpace(req.AppointmentRemark),
		"owner_user_id":         req.OwnerUserID,
		"status":                enums.SalesLeadStatus(status),
		"remark":                strings.TrimSpace(req.Remark),
		"updated_at":            time.Now(),
		"update_user_id":        operator.UserID,
		"update_user_name":      operator.Username,
	}
	return repositories.SalesLeadRepository.Updates(sqls.DB(), item.ID, updates)
}

func (s *salesLeadService) Assign(req request.AssignSalesLeadRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if repositories.SalesLeadRepository.Get(sqls.DB(), req.ID) == nil {
		return errorsx.InvalidParam("sales lead not found")
	}
	return repositories.SalesLeadRepository.Updates(sqls.DB(), req.ID, map[string]any{
		"owner_user_id":    req.OwnerUserID,
		"status":           enums.SalesLeadStatusFollowing,
		"updated_at":       time.Now(),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	})
}

func (s *salesLeadService) UpdateStatus(req request.UpdateSalesLeadStatusRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := repositories.SalesLeadRepository.Get(sqls.DB(), req.ID)
	if item == nil {
		return errorsx.InvalidParam("sales lead not found")
	}
	status := strings.TrimSpace(req.Status)
	if !enums.IsValidSalesLeadStatus(status) {
		return errorsx.InvalidParam("invalid sales lead status")
	}
	updates := map[string]any{
		"status":           enums.SalesLeadStatus(status),
		"updated_at":       time.Now(),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}
	if remark := strings.TrimSpace(req.Remark); remark != "" {
		if item.Remark == "" {
			updates["remark"] = remark
		} else if !strings.Contains(item.Remark, remark) {
			updates["remark"] = limitText(item.Remark+"\n"+remark, 1000)
		}
	}
	return repositories.SalesLeadRepository.Updates(sqls.DB(), item.ID, updates)
}

func (s *salesLeadService) SyncToCRM(req request.SyncSalesLeadToCRMRequest, operator *dto.AuthPrincipal) (response.SalesLeadCRMSyncResponse, error) {
	if operator == nil {
		return response.SalesLeadCRMSyncResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	lead := repositories.SalesLeadRepository.Get(sqls.DB(), req.ID)
	if lead == nil {
		return response.SalesLeadCRMSyncResponse{}, errorsx.InvalidParam("sales lead not found")
	}
	title := fmt.Sprintf("销售线索 CRM 同步 #%d", lead.ID)
	body := buildSalesLeadCRMWebhookText(lead, req.Remark)
	ret := response.SalesLeadCRMSyncResponse{
		LeadID:           lead.ID,
		GeneratedAt:      time.Now().Format(time.DateTime),
		WebhookEnabled:   WebhookNotifyService.Enabled(),
		Title:            title,
		Message:          "CRM 同步请求已生成。",
		WebhookEventType: "sales_lead_crm_sync",
	}
	if !WebhookNotifyService.Enabled() {
		ret.Message = "外部 Webhook 未启用，线索未同步到 CRM。"
		return ret, nil
	}
	if err := WebhookNotifyService.SendText(ret.WebhookEventType, title, body, buildSalesLeadCRMWebhookMetadata(lead, operator, req.Remark)); err != nil {
		ret.Message = "线索同步到 CRM 失败。"
		return ret, err
	}
	ret.Sent = true
	ret.Message = "线索已同步到 CRM Webhook。"
	return ret, nil
}

func (s *salesLeadService) ClaimUnassigned(req request.ClaimUnassignedSalesLeadsRequest, operator *dto.AuthPrincipal) (response.ClaimUnassignedSalesLeadsResponse, error) {
	if operator == nil {
		return response.ClaimUnassignedSalesLeadsResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if operator.UserID <= 0 {
		return response.ClaimUnassignedSalesLeadsResponse{}, errorsx.InvalidParam("invalid operator")
	}
	listReq := req.ToListRequest()
	var leads []models.SalesLead
	if err := s.buildListQuery(listReq).
		Where("status IN ?", []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}).
		Order("id DESC").
		Limit(req.GetLimit()).
		Find(&leads).Error; err != nil {
		return response.ClaimUnassignedSalesLeadsResponse{}, err
	}
	ids := make([]int64, 0, len(leads))
	for _, lead := range leads {
		ids = append(ids, lead.ID)
	}
	if len(ids) == 0 {
		return response.ClaimUnassignedSalesLeadsResponse{
			LeadIDs: []int64{},
			Message: "当前没有可领取的未分配线索",
		}, nil
	}
	updateResult := sqls.DB().Model(&models.SalesLead{}).
		Where("id IN ?", ids).
		Where("owner_user_id = 0").
		Updates(map[string]any{
			"owner_user_id":    operator.UserID,
			"status":           enums.SalesLeadStatusFollowing,
			"updated_at":       time.Now(),
			"update_user_id":   operator.UserID,
			"update_user_name": operator.Username,
		})
	if err := updateResult.Error; err != nil {
		return response.ClaimUnassignedSalesLeadsResponse{}, err
	}
	return response.ClaimUnassignedSalesLeadsResponse{
		ClaimedCount: updateResult.RowsAffected,
		LeadIDs:      ids,
		Message:      fmt.Sprintf("已领取 %d 条未分配线索", updateResult.RowsAffected),
	}, nil
}

func (s *salesLeadService) CreateFollowUp(req request.CreateLeadFollowUpRequest, operator *dto.AuthPrincipal) (*models.LeadFollowUp, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	lead := repositories.SalesLeadRepository.Get(sqls.DB(), req.LeadID)
	if lead == nil {
		return nil, errorsx.InvalidParam("sales lead not found")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, errorsx.InvalidParam("follow-up content is required")
	}
	var nextAt *time.Time
	if raw := strings.TrimSpace(req.NextFollowUpAt); raw != "" {
		parsed, err := time.ParseInLocation(time.DateTime, raw, time.Local)
		if err != nil {
			return nil, errorsx.InvalidParam("invalid next follow-up time")
		}
		nextAt = &parsed
	}
	item := &models.LeadFollowUp{
		LeadID:         lead.ID,
		OperatorID:     operator.UserID,
		OperatorName:   operator.Username,
		Content:        content,
		NextAction:     strings.TrimSpace(req.NextAction),
		NextFollowUpAt: nextAt,
		CreatedAt:      time.Now(),
	}
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.LeadFollowUpRepository.Create(ctx.Tx, item); err != nil {
			return err
		}
		return repositories.SalesLeadRepository.Updates(ctx.Tx, lead.ID, map[string]any{
			"status":            enums.SalesLeadStatusFollowing,
			"next_follow_up_at": nextAt,
			"updated_at":        time.Now(),
			"update_user_id":    operator.UserID,
			"update_user_name":  operator.Username,
		})
	}); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *salesLeadService) BuildFollowUpAdvice(lead *models.SalesLead, followUps []models.LeadFollowUp) response.SalesLeadFollowUpAdviceResult {
	if lead == nil {
		return response.SalesLeadFollowUpAdviceResult{}
	}
	customerName := strings.TrimSpace(lead.CustomerName)
	if customerName == "" {
		customerName = "客户"
	}
	contact := lead.Phone
	if contact == "" {
		contact = lead.WeChat
	}
	if contact == "" {
		contact = "暂无联系方式"
	}

	summaryParts := []string{customerName, contact}
	if lead.City != "" {
		summaryParts = append(summaryParts, lead.City)
	}
	if budget := formatSalesLeadAdviceBudget(lead.BudgetMin, lead.BudgetMax); budget != "" {
		summaryParts = append(summaryParts, "预算"+budget)
	}
	if lead.InterestedProducts != "" {
		summaryParts = append(summaryParts, "关注"+lead.InterestedProducts)
	}
	if lead.DemandSummary != "" {
		summaryParts = append(summaryParts, limitText(lead.DemandSummary, 120))
	}

	nextAction := buildLeadNextAction(lead)
	script := buildLeadFollowUpScript(customerName, lead, nextAction)
	riskHints := buildLeadRiskHints(lead, followUps)
	copyLines := []string{
		"【客户跟进摘要】",
		"客户：" + customerName,
		"联系方式：" + contact,
		"阶段：" + string(lead.BuyingStage) + " / 意向：" + string(lead.IntentLevel),
	}
	if budget := formatSalesLeadAdviceBudget(lead.BudgetMin, lead.BudgetMax); budget != "" {
		copyLines = append(copyLines, "预算："+budget)
	}
	if lead.InterestedProducts != "" {
		copyLines = append(copyLines, "意向产品："+lead.InterestedProducts)
	}
	if lead.DemandSummary != "" {
		copyLines = append(copyLines, "需求："+limitText(lead.DemandSummary, 240))
	}
	if appointment := buildLeadAppointmentText(lead); appointment != "" {
		copyLines = append(copyLines, "预约："+appointment)
	}
	if len(followUps) > 0 {
		latest := followUps[0]
		copyLines = append(copyLines, "最近跟进："+limitText(latest.Content, 180))
		if latest.NextAction != "" {
			copyLines = append(copyLines, "上次下一步："+latest.NextAction)
		}
	}
	copyLines = append(copyLines, "建议下一步："+nextAction, "建议话术："+script)
	if len(riskHints) > 0 {
		copyLines = append(copyLines, "注意事项："+strings.Join(riskHints, "；"))
	}

	return response.SalesLeadFollowUpAdviceResult{
		CustomerSummary: strings.Join(summaryParts, "｜"),
		NextAction:      nextAction,
		Script:          script,
		CopyText:        strings.Join(copyLines, "\n"),
		RiskHints:       riskHints,
	}
}

func formatSalesLeadAdviceBudget(minValue, maxValue int64) string {
	if minValue > 0 && maxValue > 0 {
		if minValue == maxValue {
			return fmt.Sprintf("%d 元左右", minValue)
		}
		return fmt.Sprintf("%d-%d 元", minValue, maxValue)
	}
	if minValue > 0 {
		return fmt.Sprintf("%d 元以上", minValue)
	}
	if maxValue > 0 {
		return fmt.Sprintf("%d 元以内", maxValue)
	}
	return ""
}

func buildLeadAppointmentText(lead *models.SalesLead) string {
	if lead == nil {
		return ""
	}
	parts := []string{}
	if lead.AppointmentAt != nil {
		parts = append(parts, lead.AppointmentAt.Format(time.DateTime))
	}
	if lead.AppointmentTimeText != "" {
		parts = append(parts, lead.AppointmentTimeText)
	}
	if lead.AppointmentStore != "" {
		parts = append(parts, lead.AppointmentStore)
	}
	if lead.AppointmentPeople > 0 {
		parts = append(parts, fmt.Sprintf("%d人", lead.AppointmentPeople))
	}
	if lead.AppointmentRemark != "" {
		parts = append(parts, lead.AppointmentRemark)
	}
	return strings.Join(parts, " / ")
}

func buildSalesLeadCRMWebhookText(lead *models.SalesLead, remark string) string {
	if lead == nil {
		return ""
	}
	lines := []string{
		fmt.Sprintf("线索ID: %d", lead.ID),
		fmt.Sprintf("客户: %s", strs.DefaultIfBlank(strings.TrimSpace(lead.CustomerName), "未命名客户")),
		fmt.Sprintf("联系方式: %s", salesLeadContactTextForService(lead)),
		fmt.Sprintf("城市: %s", strs.DefaultIfBlank(strings.TrimSpace(lead.City), "-")),
		fmt.Sprintf("意向: %s", string(lead.IntentLevel)),
		fmt.Sprintf("阶段: %s", string(lead.BuyingStage)),
		fmt.Sprintf("状态: %s", string(lead.Status)),
	}
	if lead.InterestedProducts != "" {
		lines = append(lines, fmt.Sprintf("意向产品: %s", strings.TrimSpace(lead.InterestedProducts)))
	}
	if lead.BudgetMin > 0 || lead.BudgetMax > 0 {
		lines = append(lines, fmt.Sprintf("预算: %s", buildLeadBudgetTextForCRM(lead)))
	}
	if appointment := buildLeadAppointmentText(lead); appointment != "" {
		lines = append(lines, fmt.Sprintf("预约: %s", appointment))
	}
	if lead.DemandSummary != "" {
		lines = append(lines, fmt.Sprintf("需求: %s", strings.TrimSpace(lead.DemandSummary)))
	}
	if lead.SourceChannel != "" {
		lines = append(lines, fmt.Sprintf("来源渠道: %s", strings.TrimSpace(lead.SourceChannel)))
	}
	if remark = strings.TrimSpace(remark); remark != "" {
		lines = append(lines, fmt.Sprintf("同步备注: %s", remark))
	}
	lines = append(lines, fmt.Sprintf("后台链接: /dashboard/sales-leads?leadId=%d", lead.ID))
	return strings.Join(lines, "\n")
}

func buildSalesLeadCRMWebhookMetadata(lead *models.SalesLead, operator *dto.AuthPrincipal, remark string) map[string]any {
	metadata := map[string]any{
		"leadId":              lead.ID,
		"customerId":          lead.CustomerID,
		"conversationId":      lead.ConversationID,
		"customerName":        strings.TrimSpace(lead.CustomerName),
		"phone":               strings.TrimSpace(lead.Phone),
		"wechat":              strings.TrimSpace(lead.WeChat),
		"city":                strings.TrimSpace(lead.City),
		"addressHint":         strings.TrimSpace(lead.AddressHint),
		"budgetMin":           lead.BudgetMin,
		"budgetMax":           lead.BudgetMax,
		"interestedProducts":  strings.TrimSpace(lead.InterestedProducts),
		"demandSummary":       strings.TrimSpace(lead.DemandSummary),
		"intentLevel":         string(lead.IntentLevel),
		"buyingStage":         string(lead.BuyingStage),
		"appointmentAt":       utils.FormatTimePtr(lead.AppointmentAt),
		"appointmentTimeText": strings.TrimSpace(lead.AppointmentTimeText),
		"appointmentStore":    strings.TrimSpace(lead.AppointmentStore),
		"appointmentPeople":   lead.AppointmentPeople,
		"appointmentRemark":   strings.TrimSpace(lead.AppointmentRemark),
		"sourceChannel":       strings.TrimSpace(lead.SourceChannel),
		"ownerUserId":         lead.OwnerUserID,
		"status":              string(lead.Status),
		"nextFollowUpAt":      utils.FormatTimePtr(lead.NextFollowUpAt),
		"autoTags":            buildSalesLeadCRMAutoTags(lead),
		"autoTagDetails":      buildSalesLeadCRMAutoTagDetails(lead),
		"actionUrl":           fmt.Sprintf("/dashboard/sales-leads?leadId=%d", lead.ID),
		"remark":              strings.TrimSpace(remark),
		"operatorId":          operator.UserID,
		"operatorName":        operator.Username,
	}
	return metadata
}

func buildSalesLeadCRMAutoTags(lead *models.SalesLead) []string {
	tags := make([]string, 0, 8)
	add := func(label string) {
		if strings.TrimSpace(label) == "" {
			return
		}
		for _, existing := range tags {
			if existing == label {
				return
			}
		}
		tags = append(tags, label)
	}
	if lead.Status == enums.SalesLeadStatusConverted {
		add("已成交")
	}
	if lead.Status == enums.SalesLeadStatusVisited {
		add("已到店")
	}
	if lead.IntentLevel == enums.SalesLeadIntentHigh {
		add("高意向")
	}
	if lead.BuyingStage == enums.SalesLeadStageReadyToBuy {
		add("准成交")
	}
	if lead.BuyingStage == enums.SalesLeadStageAppointment || lead.AppointmentAt != nil || lead.AppointmentTimeText != "" || lead.AppointmentStore != "" {
		add("已预约")
	}
	if lead.BuyingStage == enums.SalesLeadStageAfterSales {
		add("售后风险")
	}
	if lead.Phone == "" && lead.WeChat == "" {
		add("待补联系方式")
	} else {
		add("已留联系方式")
	}
	if lead.BudgetMin > 0 || lead.BudgetMax > 0 {
		add("有预算")
	}
	if lead.BudgetMax >= 20000 || lead.BudgetMin >= 20000 {
		add("高预算")
	}
	if lead.SourceChannel != "" {
		add("渠道:" + strings.TrimSpace(lead.SourceChannel))
	}
	return tags
}

func buildSalesLeadCRMAutoTagDetails(lead *models.SalesLead) []map[string]string {
	labels := buildSalesLeadCRMAutoTags(lead)
	ret := make([]map[string]string, 0, len(labels))
	for _, label := range labels {
		ret = append(ret, map[string]string{
			"label":       label,
			"reason":      salesLeadCRMAutoTagReason(label),
			"actionLabel": salesLeadCRMAutoTagAction(label),
		})
	}
	return ret
}

func salesLeadCRMAutoTagReason(label string) string {
	switch {
	case label == "已成交":
		return "线索状态已成交。"
	case label == "已到店":
		return "客户已完成到店标记。"
	case label == "高意向":
		return "客户购买意向较强。"
	case label == "准成交":
		return "客户进入准成交阶段。"
	case label == "已预约":
		return "客户已有预约信息。"
	case label == "售后风险":
		return "客户诉求涉及售后或投诉。"
	case label == "待补联系方式":
		return "手机号和微信都为空。"
	case label == "已留联系方式":
		return "客户已留下手机号或微信。"
	case label == "有预算":
		return "线索已抽取到预算信息。"
	case label == "高预算":
		return "预算达到高客单阈值。"
	case strings.HasPrefix(label, "渠道:"):
		return "线索带有来源渠道标识。"
	default:
		return "系统根据线索状态自动生成。"
	}
}

func salesLeadCRMAutoTagAction(label string) string {
	switch {
	case label == "高意向":
		return "优先跟进"
	case label == "准成交":
		return "确认报价与下单障碍"
	case label == "已预约":
		return "发送到店提醒"
	case label == "售后风险":
		return "转售后处理"
	case label == "待补联系方式":
		return "补齐联系方式"
	case label == "高预算":
		return "推荐高客单方案"
	case strings.HasPrefix(label, "渠道:"):
		return "复盘渠道效果"
	default:
		return "查看线索详情"
	}
}

func buildLeadBudgetTextForCRM(lead *models.SalesLead) string {
	if lead.BudgetMin > 0 && lead.BudgetMax > 0 {
		return fmt.Sprintf("%d-%d 元", lead.BudgetMin, lead.BudgetMax)
	}
	if lead.BudgetMin > 0 {
		return fmt.Sprintf("%d 元以上", lead.BudgetMin)
	}
	if lead.BudgetMax > 0 {
		return fmt.Sprintf("%d 元左右", lead.BudgetMax)
	}
	return "-"
}

func salesLeadContactTextForService(lead *models.SalesLead) string {
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

func buildLeadNextAction(lead *models.SalesLead) string {
	if lead == nil {
		return "补充客户需求并确认下一步。"
	}
	if lead.BuyingStage == enums.SalesLeadStageAfterSales {
		return "先安抚客户并确认售后问题、订单信息、购买门店和可联系时间，必要时同步售后负责人。"
	}
	if lead.AppointmentAt != nil {
		return "确认预约时间、到店门店、同行人数和重点体验产品，并提醒顾问提前准备接待。"
	}
	if lead.BuyingStage == enums.SalesLeadStageAppointment || lead.AppointmentTimeText != "" || lead.AppointmentStore != "" {
		return "把预约意向补成明确到店时间、门店和人数，并发送到店提醒。"
	}
	if lead.IntentLevel == enums.SalesLeadIntentHigh || lead.BuyingStage == enums.SalesLeadStageReadyToBuy {
		return "尽快电话或微信联系，确认预算、尺寸、使用人和到店时间，推动预约或人工报价。"
	}
	if lead.Phone == "" && lead.WeChat == "" {
		return "先补齐手机号或微信，再确认预算、使用场景和是否方便到店体验。"
	}
	if lead.InterestedProducts == "" {
		return "追问使用人、软硬偏好、尺寸和预算，再匹配 1-2 个主推产品。"
	}
	return "围绕客户关注产品确认使用场景、预算和试躺时间，记录下次跟进计划。"
}

func buildLeadFollowUpScript(customerName string, lead *models.SalesLead, nextAction string) string {
	name := strings.TrimSpace(customerName)
	if name == "" || name == "客户" {
		name = "您好"
	} else {
		name = name + "您好"
	}
	product := strings.TrimSpace(lead.InterestedProducts)
	if product == "" {
		product = "适合您的产品"
	}
	need := strings.TrimSpace(lead.DemandSummary)
	if need == "" {
		need = "您前面咨询的需求"
	}
	if lead.BuyingStage == enums.SalesLeadStageAfterSales {
		return fmt.Sprintf("%s，我看到您反馈了售后问题。为了尽快帮您处理，我先和您确认一下购买门店、订单信息、具体问题表现和方便联系的时间。", name)
	}
	if lead.AppointmentAt != nil || lead.BuyingStage == enums.SalesLeadStageAppointment {
		appointment := buildLeadAppointmentText(lead)
		if appointment == "" {
			appointment = "您方便的到店时间"
		}
		return fmt.Sprintf("%s，我这边看到您想了解%s，也提到%s。我们先帮您预留%s的体验安排，到店重点试%s，您看这个时间是否方便？", name, product, need, appointment, product)
	}
	if lead.IntentLevel == enums.SalesLeadIntentHigh || lead.BuyingStage == enums.SalesLeadStageReadyToBuy {
		return fmt.Sprintf("%s，我看您对%s比较感兴趣，也提到%s。我先帮您按预算和使用场景筛一下合适方案，再确认是否需要预约到店试躺或让门店顾问给您报价。", name, product, need)
	}
	return fmt.Sprintf("%s，我看到您前面咨询了%s。为了推荐更准，我想再确认一下使用人、尺寸、预算和软硬偏好，然后给您整理两套更适合的方案。", name, need)
}

func buildLeadRiskHints(lead *models.SalesLead, followUps []models.LeadFollowUp) []string {
	hints := []string{}
	if lead == nil {
		return hints
	}
	if lead.Phone == "" && lead.WeChat == "" {
		hints = append(hints, "缺少联系方式")
	}
	if lead.OwnerUserID == 0 {
		hints = append(hints, "未分配负责人")
	}
	if lead.IntentLevel == enums.SalesLeadIntentHigh && lead.NextFollowUpAt == nil {
		hints = append(hints, "高意向但未设置下次跟进")
	}
	if lead.BuyingStage == enums.SalesLeadStageAppointment && lead.AppointmentAt == nil && lead.AppointmentTimeText == "" {
		hints = append(hints, "预约意向未确认具体时间")
	}
	if lead.BuyingStage == enums.SalesLeadStageAfterSales {
		hints = append(hints, "售后/投诉场景需优先处理")
	}
	if len(followUps) == 0 && lead.Status == enums.SalesLeadStatusFollowing {
		hints = append(hints, "跟进中但暂无跟进记录")
	}
	return hints
}

func (s *salesLeadService) GetFollowUpReminderSummary(req request.SalesLeadFollowUpReminderRequest) response.SalesLeadFollowUpReminderSummaryResponse {
	now := time.Now()
	todayStart, tomorrowStart := leadDayBounds(now)
	ret := response.SalesLeadFollowUpReminderSummaryResponse{
		GeneratedAt: utils.FormatTime(now),
	}
	_ = s.buildFollowUpReminderQuery(req).
		Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", todayStart).
		Count(&ret.OverdueCount).Error
	_ = s.buildFollowUpReminderQuery(req).
		Where("next_follow_up_at >= ? AND next_follow_up_at < ?", todayStart, tomorrowStart).
		Count(&ret.TodayCount).Error
	ret.DueCount = ret.OverdueCount + ret.TodayCount
	_ = s.buildFollowUpReminderQuery(req).
		Where("owner_user_id = 0").
		Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", tomorrowStart).
		Count(&ret.UnassignedDueCount).Error
	_ = s.buildFollowUpReminderQuery(req).
		Where("next_follow_up_at IS NULL").
		Count(&ret.MissingScheduleCount).Error

	var preview []models.SalesLead
	if err := s.buildFollowUpReminderQuery(req).
		Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", tomorrowStart).
		Order("next_follow_up_at ASC, id ASC").
		Limit(req.GetLimit()).
		Find(&preview).Error; err != nil {
		slog.Error("load sales lead follow-up reminder preview failed", "error", err)
	}
	ret.PreviewLeads = s.buildFollowUpReminderPreview(preview, todayStart, tomorrowStart)
	ret.Message = buildSalesLeadFollowUpReminderBody(ret)
	return ret
}

func (s *salesLeadService) SendFollowUpReminder(req request.SalesLeadFollowUpReminderRequest, operator *dto.AuthPrincipal) (response.SalesLeadFollowUpReminderSummaryResponse, error) {
	if operator == nil {
		return response.SalesLeadFollowUpReminderSummaryResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	summary := s.GetFollowUpReminderSummary(req)
	if summary.DueCount == 0 && summary.MissingScheduleCount == 0 {
		return summary, nil
	}
	recipients := s.resolveFollowUpReminderRecipients(req, operator.UserID)
	title := "销售线索跟进提醒"
	for _, recipientID := range recipients {
		if _, err := NotificationService.CreateAndPush(request.CreateNotificationRequest{
			RecipientUserID:  recipientID,
			Title:            title,
			Content:          summary.Message,
			NotificationType: "sales_lead_follow_up_reminder",
			BizType:          "sales_lead",
			BizID:            0,
			ActionURL:        "/dashboard/sales-leads?followUpStatus=today",
		}); err != nil {
			slog.Error("create sales lead follow-up reminder notification failed", "error", err, "recipientUserId", recipientID)
		} else {
			summary.NotificationSent = true
		}
	}
	if err := WebhookNotifyService.SendText("sales_lead_follow_up_reminder", title, summary.Message, map[string]any{
		"overdueCount":         summary.OverdueCount,
		"todayCount":           summary.TodayCount,
		"dueCount":             summary.DueCount,
		"unassignedDueCount":   summary.UnassignedDueCount,
		"missingScheduleCount": summary.MissingScheduleCount,
		"operatorId":           operator.UserID,
	}); err != nil {
		slog.Error("send sales lead follow-up reminder webhook failed", "error", err)
	} else if WebhookNotifyService.Enabled() {
		summary.NotificationSent = true
	}
	return summary, nil
}

func (s *salesLeadService) GetAppointmentSummary(req request.SalesLeadAppointmentSummaryRequest) response.SalesLeadAppointmentSummaryResponse {
	now := time.Now()
	todayStart, tomorrowStart := leadDayBounds(now)
	windowEnd := todayStart.AddDate(0, 0, req.GetDays()+1)
	ret := response.SalesLeadAppointmentSummaryResponse{
		GeneratedAt: utils.FormatTime(now),
		Days:        req.GetDays(),
	}
	_ = s.buildAppointmentQuery(req).
		Where("appointment_at IS NOT NULL AND appointment_at < ?", todayStart).
		Count(&ret.OverdueCount).Error
	_ = s.buildAppointmentQuery(req).
		Where("appointment_at >= ? AND appointment_at < ?", todayStart, tomorrowStart).
		Count(&ret.TodayCount).Error
	_ = s.buildAppointmentQuery(req).
		Where("appointment_at >= ? AND appointment_at < ?", tomorrowStart, windowEnd).
		Count(&ret.UpcomingCount).Error
	_ = s.buildAppointmentQuery(req).
		Where("appointment_at IS NULL").
		Count(&ret.UnscheduledCount).Error
	_ = s.buildAppointmentQuery(req).
		Where("owner_user_id = 0").
		Count(&ret.UnassignedCount).Error

	var preview []models.SalesLead
	if err := s.buildAppointmentQuery(req).
		Order("CASE WHEN appointment_at IS NULL THEN 1 ELSE 0 END ASC").
		Order("appointment_at ASC").
		Order("id DESC").
		Limit(req.GetLimit()).
		Find(&preview).Error; err != nil {
		slog.Error("load sales lead appointment preview failed", "error", err)
	}
	ret.PreviewAppointments = s.buildAppointmentPreview(preview, todayStart, tomorrowStart)
	ret.Message = buildSalesLeadAppointmentSummaryMessage(ret)
	return ret
}

func (s *salesLeadService) SendAppointmentReminder(req request.SalesLeadAppointmentSummaryRequest, operator *dto.AuthPrincipal) (response.SalesLeadAppointmentSummaryResponse, error) {
	if operator == nil {
		return response.SalesLeadAppointmentSummaryResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	summary := s.GetAppointmentSummary(req)
	if summary.OverdueCount == 0 && summary.TodayCount == 0 && summary.UnscheduledCount == 0 {
		return summary, nil
	}
	recipients := s.resolveAppointmentReminderRecipients(req, operator.UserID)
	title := "销售线索预约提醒"
	for _, recipientID := range recipients {
		if _, err := NotificationService.CreateAndPush(request.CreateNotificationRequest{
			RecipientUserID:  recipientID,
			Title:            title,
			Content:          summary.Message,
			NotificationType: "sales_lead_appointment_reminder",
			BizType:          "sales_lead",
			BizID:            0,
			ActionURL:        "/dashboard/sales-leads",
		}); err != nil {
			slog.Error("create sales lead appointment reminder notification failed", "error", err, "recipientUserId", recipientID)
		} else {
			summary.NotificationSent = true
		}
	}
	if err := WebhookNotifyService.SendText("sales_lead_appointment_reminder", title, summary.Message, map[string]any{
		"overdueCount":     summary.OverdueCount,
		"todayCount":       summary.TodayCount,
		"upcomingCount":    summary.UpcomingCount,
		"unscheduledCount": summary.UnscheduledCount,
		"unassignedCount":  summary.UnassignedCount,
		"operatorId":       operator.UserID,
	}); err != nil {
		slog.Error("send sales lead appointment reminder webhook failed", "error", err)
	} else if WebhookNotifyService.Enabled() {
		summary.NotificationSent = true
	}
	return summary, nil
}

func (s *salesLeadService) buildFollowUpReminderQuery(req request.SalesLeadFollowUpReminderRequest) *gorm.DB {
	tx := sqls.DB().Model(&models.SalesLead{}).
		Where("status IN ?", []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}).
		Where("status <> ?", enums.SalesLeadStatusClosed)
	if req.OwnerUserID != nil && *req.OwnerUserID > 0 {
		tx = tx.Where("owner_user_id = ?", *req.OwnerUserID)
	}
	return tx
}

func (s *salesLeadService) buildAppointmentQuery(req request.SalesLeadAppointmentSummaryRequest) *gorm.DB {
	tx := sqls.DB().Model(&models.SalesLead{}).
		Where("status IN ?", []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}).
		Where("(buying_stage = ? OR appointment_at IS NOT NULL OR appointment_time_text <> '' OR appointment_store <> '')", enums.SalesLeadStageAppointment)
	if req.OwnerUserID != nil && *req.OwnerUserID > 0 {
		tx = tx.Where("owner_user_id = ?", *req.OwnerUserID)
	}
	return tx
}

func (s *salesLeadService) buildFollowUpReminderPreview(list []models.SalesLead, todayStart, tomorrowStart time.Time) []response.SalesLeadFollowUpReminderLeadResponse {
	ret := make([]response.SalesLeadFollowUpReminderLeadResponse, 0, len(list))
	for i := range list {
		item := list[i]
		ownerName := ""
		if owner := UserService.Get(item.OwnerUserID); owner != nil {
			ownerName = owner.Username
		}
		ret = append(ret, response.SalesLeadFollowUpReminderLeadResponse{
			ID:             item.ID,
			CustomerName:   item.CustomerName,
			Phone:          item.Phone,
			WeChat:         item.WeChat,
			IntentLevel:    item.IntentLevel,
			Status:         item.Status,
			OwnerUserID:    item.OwnerUserID,
			OwnerUserName:  ownerName,
			NextFollowUpAt: utils.FormatTimePtr(item.NextFollowUpAt),
			FollowUpState:  leadFollowUpState(item.NextFollowUpAt, todayStart, tomorrowStart),
			DemandSummary:  limitText(item.DemandSummary, 120),
			ActionURL:      fmt.Sprintf("/dashboard/sales-leads?leadId=%d", item.ID),
		})
	}
	return ret
}

func (s *salesLeadService) buildAppointmentPreview(list []models.SalesLead, todayStart, tomorrowStart time.Time) []response.SalesLeadAppointmentItemResponse {
	ret := make([]response.SalesLeadAppointmentItemResponse, 0, len(list))
	for i := range list {
		item := list[i]
		ownerName := ""
		if owner := UserService.Get(item.OwnerUserID); owner != nil {
			ownerName = owner.Username
		}
		ret = append(ret, response.SalesLeadAppointmentItemResponse{
			ID:                  item.ID,
			CustomerName:        item.CustomerName,
			Phone:               item.Phone,
			WeChat:              item.WeChat,
			IntentLevel:         item.IntentLevel,
			Status:              item.Status,
			OwnerUserID:         item.OwnerUserID,
			OwnerUserName:       ownerName,
			AppointmentAt:       utils.FormatTimePtr(item.AppointmentAt),
			AppointmentTimeText: item.AppointmentTimeText,
			AppointmentStore:    item.AppointmentStore,
			AppointmentPeople:   item.AppointmentPeople,
			DemandSummary:       limitText(item.DemandSummary, 120),
			AppointmentState:    leadAppointmentState(item.AppointmentAt, todayStart, tomorrowStart),
			ActionURL:           fmt.Sprintf("/dashboard/sales-leads?leadId=%d", item.ID),
		})
	}
	return ret
}

func (s *salesLeadService) resolveAppointmentReminderRecipients(req request.SalesLeadAppointmentSummaryRequest, operatorID int64) []int64 {
	recipients := map[int64]struct{}{}
	if operatorID > 0 {
		recipients[operatorID] = struct{}{}
	}
	_, tomorrowStart := leadDayBounds(time.Now())
	var ownerIDs []int64
	if err := s.buildAppointmentQuery(req).
		Where("owner_user_id > 0").
		Where("(appointment_at IS NULL OR appointment_at < ?)", tomorrowStart).
		Distinct("owner_user_id").
		Pluck("owner_user_id", &ownerIDs).Error; err != nil {
		slog.Error("load appointment reminder recipients failed", "error", err)
	}
	for _, ownerID := range ownerIDs {
		if ownerID > 0 {
			recipients[ownerID] = struct{}{}
		}
	}
	ret := make([]int64, 0, len(recipients))
	for id := range recipients {
		ret = append(ret, id)
	}
	return ret
}

func (s *salesLeadService) resolveFollowUpReminderRecipients(req request.SalesLeadFollowUpReminderRequest, operatorID int64) []int64 {
	recipients := map[int64]struct{}{}
	if operatorID > 0 {
		recipients[operatorID] = struct{}{}
	}
	var ownerIDs []int64
	now := time.Now()
	_, tomorrowStart := leadDayBounds(now)
	if err := s.buildFollowUpReminderQuery(req).
		Where("owner_user_id > 0").
		Where("next_follow_up_at IS NOT NULL AND next_follow_up_at < ?", tomorrowStart).
		Distinct("owner_user_id").
		Pluck("owner_user_id", &ownerIDs).Error; err != nil {
		slog.Error("load sales lead follow-up reminder owners failed", "error", err)
	}
	for _, ownerID := range ownerIDs {
		if ownerID > 0 {
			recipients[ownerID] = struct{}{}
		}
	}
	ret := make([]int64, 0, len(recipients))
	for id := range recipients {
		ret = append(ret, id)
	}
	return ret
}

func (s *salesLeadService) ExtractFromCustomerMessageAsync(conversation models.Conversation, message models.Message) {
	go func() {
		if err := s.ExtractFromCustomerMessage(conversation, message); err != nil {
			slog.Error("extract sales lead from customer message failed",
				"conversationId", conversation.ID,
				"messageId", message.ID,
				"error", err,
			)
		}
	}()
}

func (s *salesLeadService) ExtractFromCustomerMessage(conversation models.Conversation, message models.Message) error {
	if conversation.ID <= 0 || message.ID <= 0 || message.SenderType != enums.IMSenderTypeCustomer || message.MessageType != enums.IMMessageTypeText {
		return nil
	}
	info := extractLeadInfo(message.Content)
	if !info.HasSignal {
		return nil
	}
	if info.CustomerName == "" {
		info.CustomerName = strings.TrimSpace(conversation.CustomerName)
	}
	if info.DemandSummary == "" {
		info.DemandSummary = limitText(message.Content, 500)
	}
	channelType := ""
	if channel := ChannelService.Get(conversation.ChannelID); channel != nil {
		channelType = channel.ChannelType
	}
	var notifyEvent *events.SalesLeadCreatedEvent
	var shouldEnsureAfterSalesTicket bool
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		customerID, err := s.ensureLeadCustomerProfile(ctx.Tx, &conversation, info)
		if err != nil {
			return err
		}
		if customerID > 0 {
			conversation.CustomerID = customerID
		}
		match := s.findExistingLead(ctx.Tx, conversation.ID, conversation.CustomerID, info.Phone, info.WeChat)
		if match.Lead == nil {
			now := time.Now()
			lead := &models.SalesLead{
				CustomerID:          conversation.CustomerID,
				ConversationID:      conversation.ID,
				CustomerName:        info.CustomerName,
				Phone:               info.Phone,
				WeChat:              info.WeChat,
				City:                info.City,
				AddressHint:         info.AddressHint,
				BudgetMin:           info.BudgetMin,
				BudgetMax:           info.BudgetMax,
				InterestedProducts:  info.InterestedProducts,
				DemandSummary:       info.DemandSummary,
				IntentLevel:         info.IntentLevel,
				BuyingStage:         info.BuyingStage,
				AppointmentAt:       info.AppointmentAt,
				AppointmentTimeText: info.AppointmentTimeText,
				AppointmentStore:    info.AppointmentStore,
				AppointmentPeople:   info.AppointmentPeople,
				AppointmentRemark:   info.AppointmentRemark,
				SourceChannel:       channelType,
				Status:              enums.SalesLeadStatusNew,
				LastMessageID:       message.ID,
				MergeKey:            "new",
				MergeReason:         buildSalesLeadMergeReason("new", 0, conversation.ID, conversation.CustomerID, info.Phone, info.WeChat),
				MergedAt:            &now,
				AuditFields:         utils.BuildAuditFields(nil),
			}
			lead.CreateUserName = "system"
			lead.UpdateUserName = "system"
			if err := repositories.SalesLeadRepository.Create(ctx.Tx, lead); err != nil {
				return err
			}
			if isActionableLeadInfo(info) {
				notifyEvent = &events.SalesLeadCreatedEvent{
					LeadID:         lead.ID,
					ConversationID: conversation.ID,
					Reason:         leadNotifyReasonFromInfo(info),
				}
			}
			shouldEnsureAfterSalesTicket = info.BuyingStage == enums.SalesLeadStageAfterSales
			return nil
		}
		lead := match.Lead
		if shouldNotifySalesLeadUpdate(lead, info) {
			notifyEvent = &events.SalesLeadCreatedEvent{
				LeadID:         lead.ID,
				ConversationID: conversation.ID,
				Reason:         leadNotifyReasonFromInfo(info),
			}
		}
		shouldEnsureAfterSalesTicket = info.BuyingStage == enums.SalesLeadStageAfterSales || lead.BuyingStage == enums.SalesLeadStageAfterSales
		return repositories.SalesLeadRepository.Updates(ctx.Tx, lead.ID, mergeLeadUpdates(lead, info, conversation, message.ID, channelType, match))
	}); err != nil {
		return err
	}
	if shouldEnsureAfterSalesTicket {
		if _, err := TicketService.EnsureAfterSalesTicketFromConversation(conversation, "售后/投诉风险待处理", info.DemandSummary); err != nil {
			slog.Error("ensure after-sales ticket from sales lead failed", "conversationId", conversation.ID, "messageId", message.ID, "error", err)
		}
	}
	if notifyEvent != nil && notifyEvent.LeadID > 0 {
		eventbus.PublishAsync(context.Background(), *notifyEvent)
	}
	return nil
}

func (s *salesLeadService) findExistingLead(db *gorm.DB, conversationID, customerID int64, phone, wechat string) salesLeadMatch {
	if conversationID > 0 {
		if lead := repositories.SalesLeadRepository.FindOne(db, sqls.NewCnd().
			Where("conversation_id = ?", conversationID).
			Where("status <> ?", enums.SalesLeadStatusClosed)); lead != nil {
			return salesLeadMatch{
				Lead:        lead,
				MergeKey:    "conversation",
				MergeReason: buildSalesLeadMergeReason("conversation", lead.ID, conversationID, customerID, phone, wechat),
			}
		}
	}
	phone = normalizeLeadPhone(phone)
	if phone != "" {
		if lead := repositories.SalesLeadRepository.FindOne(db, sqls.NewCnd().
			Where("phone = ?", phone).
			Where("status IN ?", activeSalesLeadStatuses()).
			Desc("id")); lead != nil {
			return salesLeadMatch{
				Lead:        lead,
				MergeKey:    "phone",
				MergeReason: buildSalesLeadMergeReason("phone", lead.ID, conversationID, customerID, phone, wechat),
			}
		}
	}
	wechat = strings.TrimSpace(wechat)
	if wechat != "" {
		if lead := repositories.SalesLeadRepository.FindOne(db, sqls.NewCnd().
			Where("we_chat = ?", wechat).
			Where("status IN ?", activeSalesLeadStatuses()).
			Desc("id")); lead != nil {
			return salesLeadMatch{
				Lead:        lead,
				MergeKey:    "wechat",
				MergeReason: buildSalesLeadMergeReason("wechat", lead.ID, conversationID, customerID, phone, wechat),
			}
		}
	}
	if customerID > 0 {
		if lead := repositories.SalesLeadRepository.FindOne(db, sqls.NewCnd().
			Where("customer_id = ?", customerID).
			Where("status IN ?", activeSalesLeadStatuses()).Desc("id")); lead != nil {
			return salesLeadMatch{
				Lead:        lead,
				MergeKey:    "customer",
				MergeReason: buildSalesLeadMergeReason("customer", lead.ID, conversationID, customerID, phone, wechat),
			}
		}
	}
	return salesLeadMatch{}
}

func activeSalesLeadStatuses() []enums.SalesLeadStatus {
	return []enums.SalesLeadStatus{enums.SalesLeadStatusNew, enums.SalesLeadStatusFollowing}
}

func buildSalesLeadMergeReason(key string, leadID int64, conversationID int64, customerID int64, phone string, wechat string) string {
	switch key {
	case "conversation":
		return fmt.Sprintf("同一会话 #%d 已有未关闭线索，继续补充线索 #%d。", conversationID, leadID)
	case "phone":
		return fmt.Sprintf("手机号 %s 命中活跃线索 #%d，跨会话合并到该线索。", normalizeLeadPhone(phone), leadID)
	case "wechat":
		return fmt.Sprintf("微信 %s 命中活跃线索 #%d，跨会话合并到该线索。", strings.TrimSpace(wechat), leadID)
	case "customer":
		return fmt.Sprintf("客户档案 #%d 已有活跃线索 #%d，复用客户线索继续跟进。", customerID, leadID)
	default:
		parts := []string{"未匹配到同会话、手机号、微信或客户档案的活跃线索，已创建新线索。"}
		if conversationID > 0 {
			parts = append(parts, fmt.Sprintf("来源会话 #%d。", conversationID))
		}
		return strings.Join(parts, "")
	}
}

func (s *salesLeadService) ensureLeadCustomerProfile(db *gorm.DB, conversation *models.Conversation, info extractedLeadInfo) (int64, error) {
	if conversation == nil {
		return 0, nil
	}
	customerID := conversation.CustomerID
	if customerID <= 0 && hasLeadCustomerProfileSignal(info) {
		customerID = s.findCustomerIDByLeadProfile(db, info)
		if customerID <= 0 {
			createdID, err := s.createLeadCustomerProfile(db, conversation, info)
			if err != nil {
				return 0, err
			}
			customerID = createdID
		}
	}
	if customerID <= 0 {
		return 0, nil
	}
	now := time.Now()
	if info.CustomerNameExplicit && strings.TrimSpace(info.CustomerName) != "" {
		if err := repositories.CustomerRepository.Updates(db, customerID, map[string]any{
			"name":             strings.TrimSpace(info.CustomerName),
			"last_active_at":   now,
			"updated_at":       now,
			"update_user_name": "system",
		}); err != nil {
			return 0, err
		}
	} else {
		if err := repositories.CustomerRepository.Updates(db, customerID, map[string]any{
			"last_active_at": now,
			"updated_at":     now,
		}); err != nil {
			return 0, err
		}
	}
	if conversation.ID > 0 && conversation.CustomerID != customerID {
		customerName := strings.TrimSpace(info.CustomerName)
		if customerName == "" {
			if customer := repositories.CustomerRepository.Get(db, customerID); customer != nil {
				customerName = strings.TrimSpace(customer.Name)
			}
		}
		updates := map[string]any{
			"customer_id":      customerID,
			"updated_at":       now,
			"update_user_name": "system",
		}
		if customerName != "" {
			updates["customer_name"] = customerName
			conversation.CustomerName = customerName
		}
		if err := repositories.ConversationRepository.Updates(db, conversation.ID, updates); err != nil {
			return 0, err
		}
		conversation.CustomerID = customerID
	}
	if err := s.upsertCustomerContact(db, customerID, enums.ContactTypeMobile, info.Phone); err != nil {
		return 0, err
	}
	if err := s.upsertCustomerContact(db, customerID, enums.ContactTypeWeChat, info.WeChat); err != nil {
		return 0, err
	}
	return customerID, nil
}

func hasLeadCustomerProfileSignal(info extractedLeadInfo) bool {
	return strings.TrimSpace(info.Phone) != "" ||
		strings.TrimSpace(info.WeChat) != "" ||
		strings.TrimSpace(info.CustomerName) != ""
}

func (s *salesLeadService) findCustomerIDByLeadProfile(db *gorm.DB, info extractedLeadInfo) int64 {
	phone := normalizeLeadPhone(info.Phone)
	if phone != "" {
		if customerID := s.findCustomerIDByContact(db, enums.ContactTypeMobile, phone); customerID > 0 {
			return customerID
		}
		if customer := repositories.CustomerRepository.FindOne(db, sqls.NewCnd().
			Where("primary_mobile = ?", phone).
			Where("status <> ?", enums.StatusDeleted)); customer != nil {
			return customer.ID
		}
		if lead := repositories.SalesLeadRepository.FindOne(db, sqls.NewCnd().
			Where("phone = ?", phone).
			Where("customer_id > 0").
			Where("status IN ?", activeSalesLeadStatuses()).
			Desc("id")); lead != nil {
			return lead.CustomerID
		}
	}
	wechat := strings.TrimSpace(info.WeChat)
	if wechat != "" {
		if customerID := s.findCustomerIDByContact(db, enums.ContactTypeWeChat, wechat); customerID > 0 {
			return customerID
		}
		if lead := repositories.SalesLeadRepository.FindOne(db, sqls.NewCnd().
			Where("we_chat = ?", wechat).
			Where("customer_id > 0").
			Where("status IN ?", activeSalesLeadStatuses()).
			Desc("id")); lead != nil {
			return lead.CustomerID
		}
	}
	return 0
}

func (s *salesLeadService) findCustomerIDByContact(db *gorm.DB, contactType enums.ContactType, contactValue string) int64 {
	contactValue = strings.TrimSpace(contactValue)
	if contactValue == "" {
		return 0
	}
	contact := repositories.CustomerContactRepository.FindOne(db, sqls.NewCnd().
		Where("contact_type = ?", contactType).
		Where("contact_value = ?", contactValue).
		Where("status <> ?", enums.StatusDeleted).
		Desc("id"))
	if contact == nil || contact.CustomerID <= 0 {
		return 0
	}
	customer := repositories.CustomerRepository.Get(db, contact.CustomerID)
	if customer == nil || customer.Status == enums.StatusDeleted {
		return 0
	}
	return contact.CustomerID
}

func (s *salesLeadService) createLeadCustomerProfile(db *gorm.DB, conversation *models.Conversation, info extractedLeadInfo) (int64, error) {
	name := strings.TrimSpace(info.CustomerName)
	if name == "" && conversation != nil {
		name = strings.TrimSpace(conversation.CustomerName)
	}
	if name == "" {
		switch {
		case strings.TrimSpace(info.Phone) != "":
			name = "AI线索客户" + tailString(info.Phone, 4)
		case strings.TrimSpace(info.WeChat) != "":
			name = "AI线索客户" + tailString(info.WeChat, 4)
		default:
			name = "AI线索客户"
		}
	}
	now := time.Now()
	customer := &models.Customer{
		Name:         name,
		LastActiveAt: &now,
		Status:       enums.StatusOk,
		Remark:       "AI 数字店长自动留资创建",
		AuditFields:  utils.BuildAuditFields(nil),
	}
	customer.CreateUserName = "system"
	customer.UpdateUserName = "system"
	if err := repositories.CustomerRepository.Create(db, customer); err != nil {
		return 0, err
	}
	return customer.ID, nil
}

func tailString(value string, size int) string {
	runes := []rune(strings.TrimSpace(value))
	if size <= 0 || len(runes) <= size {
		return string(runes)
	}
	return string(runes[len(runes)-size:])
}

func (s *salesLeadService) upsertCustomerContact(db *gorm.DB, customerID int64, contactType enums.ContactType, value string) error {
	value = strings.TrimSpace(value)
	if customerID <= 0 || value == "" {
		return nil
	}
	existing := repositories.CustomerContactRepository.FindOne(db, sqls.NewCnd().
		Where("customer_id = ?", customerID).
		Where("contact_type = ?", contactType).
		Where("contact_value = ?", value).
		Where("status <> ?", enums.StatusDeleted))
	if existing != nil {
		return nil
	}
	item := &models.CustomerContact{
		CustomerID:   customerID,
		ContactType:  contactType,
		ContactValue: value,
		IsPrimary:    contactType == enums.ContactTypeMobile,
		IsVerified:   false,
		Source:       "ai_lead",
		Status:       enums.StatusOk,
		Remark:       "AI 数字店长自动识别",
		AuditFields:  utils.BuildAuditFields(nil),
	}
	item.CreateUserName = "system"
	item.UpdateUserName = "system"
	if contactType == enums.ContactTypeMobile {
		if err := CustomerContactService.clearPrimaryExcept(db, customerID, 0); err != nil {
			return err
		}
	}
	if err := repositories.CustomerContactRepository.Create(db, item); err != nil {
		return err
	}
	return CustomerContactService.syncCustomerPrimaryFromContacts(db, customerID)
}

func extractLeadInfo(content string) extractedLeadInfo {
	text := strings.TrimSpace(content)
	info := extractedLeadInfo{
		DemandSummary: limitText(text, 500),
		IntentLevel:   enums.SalesLeadIntentUnknown,
		BuyingStage:   enums.SalesLeadStageUnknown,
	}
	if text == "" {
		return info
	}
	info.Phone = normalizeLeadPhone(firstRegexpMatch(leadMobilePattern, text, 0))
	info.WeChat = firstRegexpMatch(leadWeChatPattern, text, 1)
	info.CustomerName = extractLeadName(text)
	info.CustomerNameExplicit = info.CustomerName != ""
	info.City = firstRegexpMatch(leadCityPattern, text, 1)
	info.AddressHint = firstRegexpMatch(leadAddressHintPattern, text, 1)
	info.BudgetMin, info.BudgetMax = extractLeadBudget(text)
	info.InterestedProducts = extractInterestedProducts(text)
	info.BuyingStage = inferLeadBuyingStage(text)
	info.AppointmentAt, info.AppointmentTimeText = extractAppointmentTime(text)
	info.AppointmentStore = extractAppointmentStore(text)
	info.AppointmentPeople = extractAppointmentPeople(text)
	info.AppointmentRemark = extractAppointmentRemark(text)
	info.IntentLevel = inferLeadIntent(text, info)
	info.HasSignal = info.Phone != "" ||
		info.WeChat != "" ||
		info.BudgetMax > 0 ||
		info.CustomerName != "" ||
		info.City != "" ||
		info.InterestedProducts != "" ||
		info.BuyingStage == enums.SalesLeadStageAppointment ||
		info.BuyingStage == enums.SalesLeadStageAfterSales
	return info
}

func mergeLeadUpdates(lead *models.SalesLead, info extractedLeadInfo, conversation models.Conversation, messageID int64, channelType string, match salesLeadMatch) map[string]any {
	updates := map[string]any{
		"last_message_id":  messageID,
		"updated_at":       time.Now(),
		"update_user_name": "system",
	}
	if match.MergeKey != "" {
		now := time.Now()
		updates["merge_key"] = match.MergeKey
		updates["merge_reason"] = match.MergeReason
		updates["merged_at"] = &now
	}
	if conversation.ID > 0 && lead.ConversationID != conversation.ID {
		updates["conversation_id"] = conversation.ID
	}
	if conversation.CustomerID > 0 && lead.CustomerID == 0 {
		updates["customer_id"] = conversation.CustomerID
	}
	putStringUpdate(updates, "customer_name", lead.CustomerName, info.CustomerName)
	putStringUpdate(updates, "phone", lead.Phone, info.Phone)
	putStringUpdate(updates, "we_chat", lead.WeChat, info.WeChat)
	putStringUpdate(updates, "city", lead.City, info.City)
	putStringUpdate(updates, "address_hint", lead.AddressHint, info.AddressHint)
	putStringUpdate(updates, "interested_products", lead.InterestedProducts, info.InterestedProducts)
	putStringUpdate(updates, "appointment_time_text", lead.AppointmentTimeText, info.AppointmentTimeText)
	putStringUpdate(updates, "appointment_store", lead.AppointmentStore, info.AppointmentStore)
	putStringUpdate(updates, "appointment_remark", lead.AppointmentRemark, info.AppointmentRemark)
	putStringUpdate(updates, "source_channel", lead.SourceChannel, channelType)
	if info.DemandSummary != "" && !strings.Contains(lead.DemandSummary, info.DemandSummary) {
		if lead.DemandSummary == "" {
			updates["demand_summary"] = info.DemandSummary
		} else {
			updates["demand_summary"] = limitText(lead.DemandSummary+"\n"+info.DemandSummary, 1000)
		}
	}
	if info.BudgetMin > 0 && lead.BudgetMin == 0 {
		updates["budget_min"] = info.BudgetMin
	}
	if info.BudgetMax > 0 && lead.BudgetMax == 0 {
		updates["budget_max"] = info.BudgetMax
	}
	if lead.IntentLevel != enums.SalesLeadIntentHigh || info.IntentLevel == enums.SalesLeadIntentHigh {
		updates["intent_level"] = info.IntentLevel
	}
	if lead.BuyingStage == enums.SalesLeadStageUnknown || info.BuyingStage != enums.SalesLeadStageUnknown {
		updates["buying_stage"] = info.BuyingStage
	}
	if lead.AppointmentAt == nil && info.AppointmentAt != nil {
		updates["appointment_at"] = info.AppointmentAt
	}
	if lead.AppointmentPeople == 0 && info.AppointmentPeople > 0 {
		updates["appointment_people"] = info.AppointmentPeople
	}
	if lead.Status == enums.SalesLeadStatusNew || lead.Status == "" {
		updates["status"] = enums.SalesLeadStatusNew
	}
	return updates
}

func putStringUpdate(updates map[string]any, key, oldValue, newValue string) {
	newValue = strings.TrimSpace(newValue)
	if newValue != "" && strings.TrimSpace(oldValue) == "" {
		updates[key] = newValue
	}
}

func isActionableLeadInfo(info extractedLeadInfo) bool {
	return strings.TrimSpace(info.Phone) != "" ||
		strings.TrimSpace(info.WeChat) != "" ||
		info.IntentLevel == enums.SalesLeadIntentHigh ||
		info.BuyingStage == enums.SalesLeadStageAppointment
}

func shouldNotifySalesLeadUpdate(lead *models.SalesLead, info extractedLeadInfo) bool {
	if lead == nil {
		return false
	}
	if strings.TrimSpace(lead.Phone) == "" && strings.TrimSpace(info.Phone) != "" {
		return true
	}
	if strings.TrimSpace(lead.WeChat) == "" && strings.TrimSpace(info.WeChat) != "" {
		return true
	}
	if lead.IntentLevel != enums.SalesLeadIntentHigh && info.IntentLevel == enums.SalesLeadIntentHigh {
		return true
	}
	if lead.BuyingStage != enums.SalesLeadStageAppointment && info.BuyingStage == enums.SalesLeadStageAppointment {
		return true
	}
	return false
}

func leadNotifyReasonFromInfo(info extractedLeadInfo) string {
	reasons := make([]string, 0, 4)
	if strings.TrimSpace(info.Phone) != "" || strings.TrimSpace(info.WeChat) != "" {
		reasons = append(reasons, "客户已留联系方式")
	}
	if info.IntentLevel == enums.SalesLeadIntentHigh {
		reasons = append(reasons, "高意向客户")
	}
	if info.BuyingStage == enums.SalesLeadStageAppointment {
		reasons = append(reasons, "预约/到店意向")
	}
	if len(reasons) == 0 {
		return "AI数字店长识别到新销售线索"
	}
	return strings.Join(reasons, "、")
}

func leadDayBounds(now time.Time) (time.Time, time.Time) {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return todayStart, todayStart.AddDate(0, 0, 1)
}

func leadFollowUpState(value *time.Time, todayStart, tomorrowStart time.Time) string {
	if value == nil {
		return "none"
	}
	if value.Before(todayStart) {
		return "overdue"
	}
	if value.Before(tomorrowStart) {
		return "today"
	}
	return "scheduled"
}

func leadAppointmentState(value *time.Time, todayStart, tomorrowStart time.Time) string {
	if value == nil {
		return "unscheduled"
	}
	if value.Before(todayStart) {
		return "overdue"
	}
	if value.Before(tomorrowStart) {
		return "today"
	}
	return "upcoming"
}

func buildSalesLeadFollowUpReminderBody(summary response.SalesLeadFollowUpReminderSummaryResponse) string {
	lines := []string{
		"销售线索跟进提醒",
		fmt.Sprintf("逾期未跟进：%d", summary.OverdueCount),
		fmt.Sprintf("今日待跟进：%d", summary.TodayCount),
		fmt.Sprintf("到期未分配：%d", summary.UnassignedDueCount),
		fmt.Sprintf("未设置下次跟进：%d", summary.MissingScheduleCount),
	}
	if len(summary.PreviewLeads) > 0 {
		lines = append(lines, "重点线索：")
		for _, item := range summary.PreviewLeads {
			customer := strs.DefaultIfBlank(item.CustomerName, fmt.Sprintf("线索 #%d", item.ID))
			contact := strings.TrimSpace(item.Phone)
			if contact == "" {
				contact = strings.TrimSpace(item.WeChat)
			}
			if contact == "" {
				contact = "暂无联系方式"
			}
			lines = append(lines, fmt.Sprintf("- %s / %s / %s / %s", customer, contact, leadFollowUpStateLabel(item.FollowUpState), strs.DefaultIfBlank(item.NextFollowUpAt, "未设置")))
		}
	}
	if summary.DueCount == 0 && summary.MissingScheduleCount == 0 {
		lines = append(lines, "当前没有需要提醒的跟进事项。")
	}
	lines = append(lines, "后台入口：/dashboard/sales-leads?followUpStatus=today")
	return strings.Join(lines, "\n")
}

func buildSalesLeadAppointmentSummaryMessage(summary response.SalesLeadAppointmentSummaryResponse) string {
	lines := []string{
		"销售线索预约看板",
		fmt.Sprintf("逾期未到店：%d", summary.OverdueCount),
		fmt.Sprintf("今日预约：%d", summary.TodayCount),
		fmt.Sprintf("未来%d天预约：%d", summary.Days, summary.UpcomingCount),
		fmt.Sprintf("已表达预约但未定时间：%d", summary.UnscheduledCount),
		fmt.Sprintf("预约线索未分配：%d", summary.UnassignedCount),
	}
	if len(summary.PreviewAppointments) > 0 {
		lines = append(lines, "重点预约：")
		for _, item := range summary.PreviewAppointments {
			customer := strs.DefaultIfBlank(item.CustomerName, fmt.Sprintf("线索 #%d", item.ID))
			contact := strings.TrimSpace(item.Phone)
			if contact == "" {
				contact = strings.TrimSpace(item.WeChat)
			}
			if contact == "" {
				contact = "暂无联系方式"
			}
			appointmentTime := strs.DefaultIfBlank(item.AppointmentAt, item.AppointmentTimeText)
			appointmentTime = strs.DefaultIfBlank(appointmentTime, "未定时间")
			store := strs.DefaultIfBlank(item.AppointmentStore, "未定门店")
			lines = append(lines, fmt.Sprintf("- %s / %s / %s / %s / %s", customer, contact, leadAppointmentStateLabel(item.AppointmentState), appointmentTime, store))
		}
	}
	if summary.OverdueCount == 0 && summary.TodayCount == 0 && summary.UpcomingCount == 0 && summary.UnscheduledCount == 0 {
		lines = append(lines, "当前没有需要处理的预约线索。")
	}
	lines = append(lines, "后台入口：/dashboard/sales-leads")
	return strings.Join(lines, "\n")
}

func leadFollowUpStateLabel(value string) string {
	switch value {
	case "overdue":
		return "已逾期"
	case "today":
		return "今日跟进"
	case "scheduled":
		return "已安排"
	case "none":
		return "未设置"
	default:
		return "-"
	}
}

func leadAppointmentStateLabel(value string) string {
	switch value {
	case "overdue":
		return "逾期未到店"
	case "today":
		return "今日预约"
	case "upcoming":
		return "即将到店"
	case "unscheduled":
		return "未定时间"
	default:
		return "-"
	}
}

func firstRegexpMatch(pattern *regexp.Regexp, text string, group int) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) <= group {
		return ""
	}
	return strings.TrimSpace(match[group])
}

func normalizeLeadPhone(value string) string {
	return strings.TrimSpace(leadMobilePattern.FindString(value))
}

func extractLeadName(text string) string {
	if name := firstRegexpMatch(leadNamePattern, text, 1); name != "" {
		return name
	}
	if name := firstRegexpMatch(leadHonorificPattern, text, 1); name != "" {
		return name + firstRegexpMatch(leadHonorificPattern, text, 2)
	}
	if surname := firstRegexpMatch(leadSurnamePattern, text, 1); surname != "" {
		return surname
	}
	return ""
}

func extractLeadBudget(text string) (int64, int64) {
	if match := leadArabicWanPattern.FindStringSubmatch(text); len(match) > 1 {
		if value, err := strconv.ParseFloat(match[1], 64); err == nil && value > 0 {
			amount := int64(value * 10000)
			return budgetRange(amount)
		}
	}
	if match := leadChineseWanPattern.FindStringSubmatch(text); len(match) > 1 {
		amount := int64(chineseDigitValue(match[1]) * 10000)
		if len(match) > 2 && match[2] != "" {
			amount += int64(chineseDigitValue(match[2]) * 1000)
		}
		if amount > 0 {
			return budgetRange(amount)
		}
	}
	if match := leadArabicYuanPattern.FindStringSubmatch(text); len(match) > 1 {
		if amount, err := strconv.ParseInt(match[1], 10, 64); err == nil && amount > 0 {
			return budgetRange(amount)
		}
	}
	return 0, 0
}

func extractAppointmentTime(text string) (*time.Time, string) {
	text = strings.TrimSpace(text)
	if text == "" || !containsAnyLeadText(text, "预约", "到店", "试躺", "去店", "去门店", "看看", "体验") {
		return nil, ""
	}
	timeText := firstRegexpMatch(leadTimePattern, text, 0)
	if match := leadDatePattern.FindStringSubmatch(text); len(match) > 3 {
		year := time.Now().Year()
		if match[1] != "" {
			if parsed, err := strconv.Atoi(match[1]); err == nil && parsed > 0 {
				year = parsed
			}
		}
		month, _ := strconv.Atoi(match[2])
		day, _ := strconv.Atoi(match[3])
		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			hour := appointmentHourFromText(timeText)
			at := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.Local)
			if timeText == "" {
				timeText = match[0]
			} else if !strings.Contains(timeText, match[0]) {
				timeText = match[0] + " " + timeText
			}
			return &at, strings.TrimSpace(timeText)
		}
	}
	return nil, strings.TrimSpace(timeText)
}

func extractAppointmentPeople(text string) int {
	if match := leadPeoplePattern.FindStringSubmatch(text); len(match) > 1 {
		if value, err := strconv.Atoi(match[1]); err == nil {
			return value
		}
		return chineseDigitValue(match[1])
	}
	return 0
}

func extractAppointmentStore(text string) string {
	if store := firstRegexpMatch(leadStorePattern, text, 1); store != "" {
		return store
	}
	return ""
}

func extractAppointmentRemark(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || !containsAnyLeadText(text, "预约", "到店", "试躺", "去店", "去门店", "体验") {
		return ""
	}
	return limitText(text, 300)
}

func appointmentHourFromText(value string) int {
	switch {
	case strings.Contains(value, "上午"), strings.Contains(value, "早上"):
		return 10
	case strings.Contains(value, "中午"):
		return 12
	case strings.Contains(value, "晚上"), strings.Contains(value, "晚间"), strings.Contains(value, "傍晚"):
		return 19
	default:
		return 14
	}
}

func parseLeadTimePtr(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.DateTime, "2006-01-02", "2006/01/02 15:04:05", "2006/01/02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &parsed
		}
	}
	return nil
}

func budgetRange(amount int64) (int64, int64) {
	if amount <= 0 {
		return 0, 0
	}
	min := amount * 8 / 10
	max := amount * 12 / 10
	return min, max
}

func chineseDigitValue(value string) int {
	switch value {
	case "一":
		return 1
	case "二", "两":
		return 2
	case "三":
		return 3
	case "四":
		return 4
	case "五":
		return 5
	case "六":
		return 6
	case "七":
		return 7
	case "八":
		return 8
	case "九":
		return 9
	case "十":
		return 10
	default:
		return 0
	}
}

func extractInterestedProducts(text string) string {
	keywords := []string{"床垫", "枕头", "乳胶枕", "护脊", "儿童床垫", "静音分区", "脊护支撑款", "云感舒睡款", "旗舰款", "1.8米", "1.5米"}
	var found []string
	seen := map[string]struct{}{}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			if _, ok := seen[keyword]; ok {
				continue
			}
			seen[keyword] = struct{}{}
			found = append(found, keyword)
		}
	}
	return strings.Join(found, ",")
}

func inferLeadBuyingStage(text string) enums.SalesLeadStage {
	switch {
	case containsAnyLeadText(text, "预约", "到店", "试躺", "周末", "明天去", "今天去"):
		return enums.SalesLeadStageAppointment
	case containsAnyLeadText(text, "售后", "安装", "质保", "异响", "退换", "投诉", "退款", "退货", "不满意", "差评"):
		return enums.SalesLeadStageAfterSales
	case containsAnyLeadText(text, "下单", "购买", "买", "报价", "付款", "定金"):
		return enums.SalesLeadStageReadyToBuy
	case containsAnyLeadText(text, "对比", "区别", "哪款", "哪种", "推荐"):
		return enums.SalesLeadStageComparing
	default:
		return enums.SalesLeadStageConsulting
	}
}

func inferLeadIntent(text string, info extractedLeadInfo) enums.SalesLeadIntent {
	if info.Phone != "" || info.WeChat != "" || info.BuyingStage == enums.SalesLeadStageAppointment || info.BuyingStage == enums.SalesLeadStageReadyToBuy {
		return enums.SalesLeadIntentHigh
	}
	if info.BudgetMax > 0 || containsAnyLeadText(text, "预算", "推荐", "哪款", "哪种", "价格", "优惠") {
		return enums.SalesLeadIntentMedium
	}
	return enums.SalesLeadIntentLow
}

func normalizeLeadIntent(value string, fallback enums.SalesLeadIntent) enums.SalesLeadIntent {
	switch enums.SalesLeadIntent(strings.TrimSpace(value)) {
	case enums.SalesLeadIntentLow, enums.SalesLeadIntentMedium, enums.SalesLeadIntentHigh, enums.SalesLeadIntentUnknown:
		return enums.SalesLeadIntent(value)
	default:
		if fallback != "" {
			return fallback
		}
		return enums.SalesLeadIntentUnknown
	}
}

func normalizeLeadStage(value string, fallback enums.SalesLeadStage) enums.SalesLeadStage {
	switch enums.SalesLeadStage(strings.TrimSpace(value)) {
	case enums.SalesLeadStageUnknown, enums.SalesLeadStageConsulting, enums.SalesLeadStageComparing, enums.SalesLeadStageAppointment, enums.SalesLeadStageReadyToBuy, enums.SalesLeadStageAfterSales:
		return enums.SalesLeadStage(value)
	default:
		if fallback != "" {
			return fallback
		}
		return enums.SalesLeadStageUnknown
	}
}

func containsAnyLeadText(value string, needles ...string) bool {
	value = strings.TrimSpace(value)
	if strs.IsBlank(value) {
		return false
	}
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
