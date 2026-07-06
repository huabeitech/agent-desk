package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func TestExtractLeadInfoAppointmentWithContact(t *testing.T) {
	info := extractLeadInfo("我想预约本周末到店试躺，主要看1.8米床垫。我姓李，电话13800001111，预算一万五左右。")
	if !info.HasSignal {
		t.Fatalf("expected lead signal")
	}
	if info.CustomerName != "李" {
		t.Fatalf("CustomerName=%q want 李", info.CustomerName)
	}
	if info.Phone != "13800001111" {
		t.Fatalf("Phone=%q want 13800001111", info.Phone)
	}
	if info.BudgetMin == 0 || info.BudgetMax == 0 {
		t.Fatalf("expected budget range, got %d-%d", info.BudgetMin, info.BudgetMax)
	}
	if info.IntentLevel != enums.SalesLeadIntentHigh {
		t.Fatalf("IntentLevel=%q want high", info.IntentLevel)
	}
	if info.BuyingStage != enums.SalesLeadStageAppointment {
		t.Fatalf("BuyingStage=%q want appointment", info.BuyingStage)
	}
	if info.InterestedProducts == "" {
		t.Fatalf("expected interested products")
	}
	if info.AppointmentTimeText == "" {
		t.Fatalf("expected appointment time text")
	}
}

func TestExtractLeadInfoAppointmentDetails(t *testing.T) {
	info := extractLeadInfo("我想预约7月8日下午到徐汇体验店试躺，两个人，主要看脊护支撑款。")
	if !info.HasSignal {
		t.Fatalf("expected lead signal")
	}
	if info.BuyingStage != enums.SalesLeadStageAppointment {
		t.Fatalf("BuyingStage=%q want appointment", info.BuyingStage)
	}
	if info.AppointmentAt == nil {
		t.Fatalf("expected appointment at")
	}
	if info.AppointmentTimeText == "" || info.AppointmentStore != "徐汇体验店" || info.AppointmentPeople != 2 {
		t.Fatalf("unexpected appointment details: %#v", info)
	}
	if info.AppointmentRemark == "" {
		t.Fatalf("expected appointment remark")
	}
}

func TestExtractLeadInfoBudgetQuestion(t *testing.T) {
	info := extractLeadInfo("老人腰不好，床垫是不是越硬越好？预算1.5万，推荐哪种？")
	if !info.HasSignal {
		t.Fatalf("expected lead signal")
	}
	if info.BudgetMin == 0 || info.BudgetMax == 0 {
		t.Fatalf("expected budget range, got %d-%d", info.BudgetMin, info.BudgetMax)
	}
	if info.IntentLevel != enums.SalesLeadIntentMedium {
		t.Fatalf("IntentLevel=%q want medium", info.IntentLevel)
	}
	if info.BuyingStage != enums.SalesLeadStageComparing {
		t.Fatalf("BuyingStage=%q want comparing", info.BuyingStage)
	}
}

func TestExtractLeadInfoNoSignalForThanks(t *testing.T) {
	info := extractLeadInfo("好的，谢谢")
	if info.HasSignal {
		t.Fatalf("expected no lead signal, got %#v", info)
	}
}

func TestSalesLeadExtractMergesExistingLeadByPhoneAcrossConversations(t *testing.T) {
	setupSalesLeadListTestDB(t)
	existing := models.SalesLead{
		CustomerName:   "旧客户",
		ConversationID: 11,
		Phone:          "13800001111",
		IntentLevel:    enums.SalesLeadIntentHigh,
		BuyingStage:    enums.SalesLeadStageAppointment,
		Status:         enums.SalesLeadStatusFollowing,
	}
	if err := sqls.DB().Create(&existing).Error; err != nil {
		t.Fatalf("create existing lead: %v", err)
	}

	err := SalesLeadService.ExtractFromCustomerMessage(
		models.Conversation{ID: 22, CustomerName: "王先生"},
		models.Message{
			ID:          33,
			SenderType:  enums.IMSenderTypeCustomer,
			MessageType: enums.IMMessageTypeText,
			Content:     "我是王先生，电话13800001111，想预约7月8日下午到徐汇体验店试躺1.8米床垫。",
		},
	)
	if err != nil {
		t.Fatalf("ExtractFromCustomerMessage() error = %v", err)
	}

	var count int64
	if err := sqls.DB().Model(&models.SalesLead{}).Count(&count).Error; err != nil {
		t.Fatalf("count leads: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected existing lead to be merged, got %d leads", count)
	}
	var updated models.SalesLead
	if err := sqls.DB().First(&updated, existing.ID).Error; err != nil {
		t.Fatalf("load updated lead: %v", err)
	}
	if updated.ConversationID != 22 || updated.LastMessageID != 33 {
		t.Fatalf("expected latest conversation/message to be kept, got conversation=%d message=%d", updated.ConversationID, updated.LastMessageID)
	}
	if updated.CustomerName != "旧客户" || updated.AppointmentStore == "" || updated.AppointmentAt == nil {
		t.Fatalf("expected merged lead to keep stable fields and add appointment details: %#v", updated)
	}
	if updated.MergeKey != "phone" || !strings.Contains(updated.MergeReason, "手机号 13800001111") || updated.MergedAt == nil {
		t.Fatalf("expected phone merge explanation, got key=%q reason=%q at=%v", updated.MergeKey, updated.MergeReason, updated.MergedAt)
	}
}

func TestSalesLeadExtractAfterSalesMessageCreatesTicketOnce(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	conversation := models.Conversation{
		CustomerName:       "售后客户",
		Status:             enums.IMConversationStatusActive,
		ServiceMode:        enums.IMConversationServiceModeAIFirst,
		LastMessageAt:      now,
		LastActiveAt:       now,
		LastMessageSummary: "客户反馈床垫异响并要求售后处理",
	}
	if err := sqls.DB().Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	message := models.Message{
		ID:          101,
		SenderType:  enums.IMSenderTypeCustomer,
		MessageType: enums.IMMessageTypeText,
		Content:     "我之前买的床垫有异响，售后一直没人处理，我要投诉。",
	}
	if err := SalesLeadService.ExtractFromCustomerMessage(conversation, message); err != nil {
		t.Fatalf("ExtractFromCustomerMessage() error = %v", err)
	}

	var lead models.SalesLead
	if err := sqls.DB().Where("conversation_id = ?", conversation.ID).First(&lead).Error; err != nil {
		t.Fatalf("load after-sales lead: %v", err)
	}
	if lead.BuyingStage != enums.SalesLeadStageAfterSales || lead.LastMessageID != message.ID {
		t.Fatalf("unexpected after-sales lead: %#v", lead)
	}

	var tickets []models.Ticket
	if err := sqls.DB().Where("conversation_id = ?", conversation.ID).Find(&tickets).Error; err != nil {
		t.Fatalf("load tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected one ticket, got %#v", tickets)
	}
	if tickets[0].Source != enums.TicketSourceConversation || tickets[0].Status != enums.TicketStatusPending {
		t.Fatalf("unexpected ticket: %#v", tickets[0])
	}
	if !strings.Contains(tickets[0].Title, "售后") || !strings.Contains(tickets[0].Description, "异响") {
		t.Fatalf("expected after-sales ticket content, got %#v", tickets[0])
	}

	message.ID = 102
	message.Content = "刚才说的售后投诉麻烦尽快处理。"
	if err := SalesLeadService.ExtractFromCustomerMessage(conversation, message); err != nil {
		t.Fatalf("second ExtractFromCustomerMessage() error = %v", err)
	}
	var ticketCount int64
	if err := sqls.DB().Model(&models.Ticket{}).Where("conversation_id = ?", conversation.ID).Count(&ticketCount).Error; err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if ticketCount != 1 {
		t.Fatalf("expected after-sales ticket to be reused, got %d", ticketCount)
	}
}

func TestSalesLeadExtractCreatesCustomerProfileWhenConversationUnbound(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	conversation := models.Conversation{
		CustomerName:       "",
		Status:             enums.IMConversationStatusAIServing,
		ServiceMode:        enums.IMConversationServiceModeAIFirst,
		LastMessageAt:      now,
		LastActiveAt:       now,
		LastMessageSummary: "客户留下手机号预约试躺",
	}
	if err := sqls.DB().Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	err := SalesLeadService.ExtractFromCustomerMessage(
		conversation,
		models.Message{
			ID:          201,
			SenderType:  enums.IMSenderTypeCustomer,
			MessageType: enums.IMMessageTypeText,
			Content:     "我姓赵，电话13900001111，预算两万，想周末到徐汇店试躺脊护支撑款。",
		},
	)
	if err != nil {
		t.Fatalf("ExtractFromCustomerMessage() error = %v", err)
	}

	var customer models.Customer
	if err := sqls.DB().Where("name = ?", "赵").First(&customer).Error; err != nil {
		t.Fatalf("load created customer: %v", err)
	}
	if customer.PrimaryMobile != "13900001111" {
		t.Fatalf("expected primary mobile synced, got %#v", customer)
	}
	var updatedConversation models.Conversation
	if err := sqls.DB().First(&updatedConversation, conversation.ID).Error; err != nil {
		t.Fatalf("load updated conversation: %v", err)
	}
	if updatedConversation.CustomerID != customer.ID || updatedConversation.CustomerName != "赵" {
		t.Fatalf("expected conversation bound to customer, got %#v", updatedConversation)
	}
	var lead models.SalesLead
	if err := sqls.DB().Where("conversation_id = ?", conversation.ID).First(&lead).Error; err != nil {
		t.Fatalf("load created lead: %v", err)
	}
	if lead.CustomerID != customer.ID || lead.Phone != "13900001111" {
		t.Fatalf("expected lead bound to customer, got %#v", lead)
	}
	var contact models.CustomerContact
	if err := sqls.DB().Where("customer_id = ? AND contact_type = ? AND contact_value = ?", customer.ID, enums.ContactTypeMobile, "13900001111").First(&contact).Error; err != nil {
		t.Fatalf("load created contact: %v", err)
	}
	if !contact.IsPrimary || contact.Source != "ai_lead" {
		t.Fatalf("unexpected contact: %#v", contact)
	}
}

func TestSalesLeadExtractReusesCustomerProfileByExistingContact(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	existing := models.Customer{
		Name:         "已有客户",
		LastActiveAt: &now,
		Status:       enums.StatusOk,
	}
	if err := sqls.DB().Create(&existing).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}
	if err := sqls.DB().Create(&models.CustomerContact{
		CustomerID:   existing.ID,
		ContactType:  enums.ContactTypeMobile,
		ContactValue: "13900002222",
		IsPrimary:    true,
		Source:       "manual",
		Status:       enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create customer contact: %v", err)
	}
	if err := CustomerContactService.syncCustomerPrimaryFromContacts(sqls.DB(), existing.ID); err != nil {
		t.Fatalf("sync customer primary: %v", err)
	}
	conversation := models.Conversation{
		Status:        enums.IMConversationStatusAIServing,
		ServiceMode:   enums.IMConversationServiceModeAIFirst,
		LastMessageAt: now,
		LastActiveAt:  now,
	}
	if err := sqls.DB().Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	err := SalesLeadService.ExtractFromCustomerMessage(
		conversation,
		models.Message{
			ID:          202,
			SenderType:  enums.IMSenderTypeCustomer,
			MessageType: enums.IMMessageTypeText,
			Content:     "电话13900002222，我想看智能电动床组合。",
		},
	)
	if err != nil {
		t.Fatalf("ExtractFromCustomerMessage() error = %v", err)
	}

	var customerCount int64
	if err := sqls.DB().Model(&models.Customer{}).Count(&customerCount).Error; err != nil {
		t.Fatalf("count customers: %v", err)
	}
	if customerCount != 1 {
		t.Fatalf("expected existing customer reused, got %d customers", customerCount)
	}
	var lead models.SalesLead
	if err := sqls.DB().Where("conversation_id = ?", conversation.ID).First(&lead).Error; err != nil {
		t.Fatalf("load lead: %v", err)
	}
	if lead.CustomerID != existing.ID {
		t.Fatalf("expected lead customer %d, got %#v", existing.ID, lead)
	}
	if lead.MergeKey != "new" || !strings.Contains(lead.MergeReason, "已创建新线索") {
		t.Fatalf("expected new lead explanation, got key=%q reason=%q", lead.MergeKey, lead.MergeReason)
	}
	var updatedConversation models.Conversation
	if err := sqls.DB().First(&updatedConversation, conversation.ID).Error; err != nil {
		t.Fatalf("load updated conversation: %v", err)
	}
	if updatedConversation.CustomerID != existing.ID || updatedConversation.CustomerName != "已有客户" {
		t.Fatalf("expected conversation bound to existing customer, got %#v", updatedConversation)
	}
}

func TestSalesLeadListFiltersByFollowUpStatus(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	overdue := today.AddDate(0, 0, -1)
	scheduled := today.AddDate(0, 0, 2)
	seeds := []models.SalesLead{
		{CustomerName: "逾期客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, NextFollowUpAt: &overdue},
		{CustomerName: "今日客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, NextFollowUpAt: &today},
		{CustomerName: "未来客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, NextFollowUpAt: &scheduled},
		{CustomerName: "未设置客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	cases := []struct {
		status string
		want   string
	}{
		{status: "overdue", want: "逾期客户"},
		{status: "today", want: "今日客户"},
		{status: "scheduled", want: "未来客户"},
		{status: "none", want: "未设置客户"},
	}
	for _, tc := range cases {
		list, paging := SalesLeadService.List(requestForFollowUpStatus(tc.status))
		if paging.Total != 1 || len(list) != 1 || list[0].CustomerName != tc.want {
			t.Fatalf("status %s got total=%d list=%#v want %s", tc.status, paging.Total, list, tc.want)
		}
	}
}

func TestSalesLeadListFiltersByAppointmentStatus(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	overdue := today.AddDate(0, 0, -1)
	upcoming := today.AddDate(0, 0, 2)
	seeds := []models.SalesLead{
		{CustomerName: "逾期预约", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &overdue},
		{CustomerName: "今日预约", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &today},
		{CustomerName: "未来预约", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &upcoming},
		{CustomerName: "未定时间预约", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentTimeText: "周末方便"},
		{CustomerName: "无预约客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	cases := []struct {
		status string
		want   string
	}{
		{status: "overdue", want: "逾期预约"},
		{status: "today", want: "今日预约"},
		{status: "upcoming", want: "未来预约"},
		{status: "unscheduled", want: "未定时间预约"},
	}
	for _, tc := range cases {
		list, paging := SalesLeadService.List(requestForAppointmentStatus(tc.status))
		if paging.Total != 1 || len(list) != 1 || list[0].CustomerName != tc.want {
			t.Fatalf("appointment status %s got total=%d list=%#v want %s", tc.status, paging.Total, list, tc.want)
		}
	}

	list, paging := SalesLeadService.List(requestForAppointmentStatus("all"))
	if paging.Total != 4 || len(list) != 4 {
		t.Fatalf("appointment all got total=%d list=%#v want 4 appointment leads", paging.Total, list)
	}
}

func TestSalesLeadListFiltersByOwner(t *testing.T) {
	setupSalesLeadListTestDB(t)
	seeds := []models.SalesLead{
		{CustomerName: "顾问A客户", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, OwnerUserID: 7},
		{CustomerName: "顾问B客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, OwnerUserID: 9},
		{CustomerName: "未分配客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	ownerID := int64(7)
	list, paging := SalesLeadService.List(request.SalesLeadListRequest{Page: 1, Limit: 20, OwnerUserID: &ownerID})
	if paging.Total != 1 || len(list) != 1 || list[0].CustomerName != "顾问A客户" {
		t.Fatalf("owner filter got total=%d list=%#v", paging.Total, list)
	}

	unassigned := int64(-1)
	list, paging = SalesLeadService.List(request.SalesLeadListRequest{Page: 1, Limit: 20, OwnerUserID: &unassigned})
	if paging.Total != 1 || len(list) != 1 || list[0].CustomerName != "未分配客户" {
		t.Fatalf("unassigned owner filter got total=%d list=%#v", paging.Total, list)
	}
}

func TestSalesLeadListFiltersByTaskView(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	overdue := today.AddDate(0, 0, -1)
	future := today.AddDate(0, 0, 2)
	seeds := []models.SalesLead{
		{CustomerName: "今日任务", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentMedium, NextFollowUpAt: &today},
		{CustomerName: "逾期任务", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentMedium, NextFollowUpAt: &overdue},
		{CustomerName: "高意向任务", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, NextFollowUpAt: &future},
		{CustomerName: "预约任务", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentTimeText: "周末"},
		{CustomerName: "售后任务", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAfterSales},
		{CustomerName: "已转化高意向不算任务", Status: enums.SalesLeadStatusConverted, IntentLevel: enums.SalesLeadIntentHigh},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	cases := []struct {
		taskView string
		want     string
	}{
		{taskView: "today", want: "今日任务"},
		{taskView: "overdue", want: "逾期任务"},
		{taskView: "high_intent", want: "高意向任务"},
		{taskView: "appointment", want: "预约任务"},
		{taskView: "after_sales", want: "售后任务"},
	}
	for _, tc := range cases {
		list, paging := SalesLeadService.List(request.SalesLeadListRequest{Page: 1, Limit: 20, TaskView: tc.taskView})
		if paging.Total != 1 || len(list) != 1 || list[0].CustomerName != tc.want {
			t.Fatalf("task view %s got total=%d list=%#v want %s", tc.taskView, paging.Total, list, tc.want)
		}
	}
}

func TestSalesLeadClaimUnassignedUsesCurrentFilters(t *testing.T) {
	setupSalesLeadListTestDB(t)
	seeds := []models.SalesLead{
		{CustomerName: "可领取高意向", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh},
		{CustomerName: "可领取预约", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentTimeText: "周末"},
		{CustomerName: "低意向未分配", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentLow},
		{CustomerName: "已有负责人", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, OwnerUserID: 9},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	result, err := SalesLeadService.ClaimUnassigned(
		request.ClaimUnassignedSalesLeadsRequest{Intent: string(enums.SalesLeadIntentHigh), Limit: 100},
		&dto.AuthPrincipal{UserID: 7, Username: "advisor"},
	)
	if err != nil {
		t.Fatalf("ClaimUnassigned() error = %v", err)
	}
	if result.ClaimedCount != 1 || len(result.LeadIDs) != 1 {
		t.Fatalf("unexpected claim result: %#v", result)
	}

	var claimed models.SalesLead
	if err := sqls.DB().First(&claimed, result.LeadIDs[0]).Error; err != nil {
		t.Fatalf("load claimed lead: %v", err)
	}
	if claimed.CustomerName != "可领取高意向" || claimed.OwnerUserID != 7 || claimed.Status != enums.SalesLeadStatusFollowing {
		t.Fatalf("unexpected claimed lead: %#v", claimed)
	}

	var untouched models.SalesLead
	if err := sqls.DB().Where("customer_name = ?", "已有负责人").First(&untouched).Error; err != nil {
		t.Fatalf("load untouched lead: %v", err)
	}
	if untouched.OwnerUserID != 9 {
		t.Fatalf("expected existing owner to be preserved, got %#v", untouched)
	}
}

func TestSalesLeadUpdateStatusAppendsRemark(t *testing.T) {
	setupSalesLeadListTestDB(t)
	lead := models.SalesLead{
		CustomerName: "待成交客户",
		Phone:        "13800001111",
		Status:       enums.SalesLeadStatusFollowing,
		Remark:       "原备注",
	}
	if err := sqls.DB().Create(&lead).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}

	err := SalesLeadService.UpdateStatus(
		request.UpdateSalesLeadStatusRequest{ID: lead.ID, Status: string(enums.SalesLeadStatusConverted), Remark: "列表快捷标记：已转化"},
		&dto.AuthPrincipal{UserID: 7, Username: "advisor"},
	)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	var updated models.SalesLead
	if err := sqls.DB().First(&updated, lead.ID).Error; err != nil {
		t.Fatalf("load updated lead: %v", err)
	}
	if updated.Status != enums.SalesLeadStatusConverted || updated.CustomerName != "待成交客户" || updated.Phone != "13800001111" {
		t.Fatalf("unexpected updated lead: %#v", updated)
	}
	if !strings.Contains(updated.Remark, "原备注") || !strings.Contains(updated.Remark, "列表快捷标记：已转化") {
		t.Fatalf("expected appended remark, got %q", updated.Remark)
	}
}

func TestSalesLeadUpdateStatusSupportsVisited(t *testing.T) {
	setupSalesLeadListTestDB(t)
	lead := models.SalesLead{
		CustomerName: "到店客户",
		Phone:        "13800002222",
		Status:       enums.SalesLeadStatusFollowing,
	}
	if err := sqls.DB().Create(&lead).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}

	err := SalesLeadService.UpdateStatus(
		request.UpdateSalesLeadStatusRequest{ID: lead.ID, Status: string(enums.SalesLeadStatusVisited), Remark: "列表快捷标记：客户已到店"},
		&dto.AuthPrincipal{UserID: 7, Username: "advisor"},
	)
	if err != nil {
		t.Fatalf("UpdateStatus() visited error = %v", err)
	}
	var updated models.SalesLead
	if err := sqls.DB().First(&updated, lead.ID).Error; err != nil {
		t.Fatalf("load updated lead: %v", err)
	}
	if updated.Status != enums.SalesLeadStatusVisited || !strings.Contains(updated.Remark, "客户已到店") {
		t.Fatalf("unexpected visited lead: %#v", updated)
	}
}

func TestSalesLeadBuildFollowUpAdviceForHighIntentAppointment(t *testing.T) {
	appointmentAt := time.Date(2026, 7, 6, 15, 0, 0, 0, time.Local)
	lead := &models.SalesLead{
		CustomerName:       "李静",
		Phone:              "13900001111",
		City:               "浦东",
		BudgetMin:          15000,
		BudgetMax:          20000,
		InterestedProducts: "慕斯脊护支撑款",
		DemandSummary:      "给爸妈选护腰床垫，想周末试躺。",
		IntentLevel:        enums.SalesLeadIntentHigh,
		BuyingStage:        enums.SalesLeadStageAppointment,
		AppointmentAt:      &appointmentAt,
		AppointmentStore:   "徐汇店",
		AppointmentPeople:  3,
		Status:             enums.SalesLeadStatusFollowing,
	}
	advice := SalesLeadService.BuildFollowUpAdvice(lead, nil)
	if !strings.Contains(advice.CustomerSummary, "李静") ||
		!strings.Contains(advice.CustomerSummary, "13900001111") ||
		!strings.Contains(advice.CustomerSummary, "预算15000-20000 元") ||
		!strings.Contains(advice.CustomerSummary, "慕斯脊护支撑款") {
		t.Fatalf("unexpected customer summary: %#v", advice)
	}
	if !strings.Contains(advice.NextAction, "确认预约时间") {
		t.Fatalf("unexpected next action: %#v", advice.NextAction)
	}
	if !strings.Contains(advice.Script, "徐汇店") || !strings.Contains(advice.Script, "慕斯脊护支撑款") {
		t.Fatalf("unexpected script: %#v", advice.Script)
	}
	if !strings.Contains(advice.CopyText, "【客户跟进摘要】") ||
		!strings.Contains(advice.CopyText, "建议话术") {
		t.Fatalf("unexpected copy text: %#v", advice.CopyText)
	}
	if len(advice.RiskHints) == 0 || !strings.Contains(strings.Join(advice.RiskHints, "\n"), "未分配负责人") {
		t.Fatalf("unexpected risk hints: %#v", advice.RiskHints)
	}
}

func TestSalesLeadSyncToCRMSendsWebhookPayload(t *testing.T) {
	setupSalesLeadListTestDB(t)
	appointmentAt := time.Date(2026, 7, 8, 15, 0, 0, 0, time.Local)
	lead := models.SalesLead{
		CustomerName:        "李静",
		Phone:               "13900001111",
		WeChat:              "muse-lijing",
		City:                "上海",
		BudgetMin:           15000,
		BudgetMax:           22000,
		InterestedProducts:  "慕斯脊护支撑款",
		DemandSummary:       "给父母选护腰床垫，周末想试躺。",
		IntentLevel:         enums.SalesLeadIntentHigh,
		BuyingStage:         enums.SalesLeadStageAppointment,
		AppointmentAt:       &appointmentAt,
		AppointmentStore:    "徐汇店",
		AppointmentPeople:   2,
		AppointmentTimeText: "周末下午",
		SourceChannel:       "官网",
		Status:              enums.SalesLeadStatusFollowing,
	}
	if err := sqls.DB().Create(&lead).Error; err != nil {
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
		},
	})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := SalesLeadService.SyncToCRM(request.SyncSalesLeadToCRMRequest{
		ID:     lead.ID,
		Remark: "同步到商家 CRM",
	}, &dto.AuthPrincipal{UserID: 88, Username: "manager"})
	if err != nil {
		t.Fatalf("SyncToCRM() error = %v", err)
	}
	if !resp.Sent || !resp.WebhookEnabled || resp.WebhookEventType != "sales_lead_crm_sync" {
		t.Fatalf("unexpected crm sync response: %#v", resp)
	}
	if got["eventType"] != "sales_lead_crm_sync" || !strings.Contains(got["text"].(string), "李静") {
		t.Fatalf("unexpected webhook payload: %#v", got)
	}
	metadata := got["metadata"].(map[string]any)
	if metadata["leadId"].(float64) != float64(lead.ID) ||
		metadata["customerName"] != "李静" ||
		metadata["phone"] != "13900001111" ||
		metadata["sourceChannel"] != "官网" ||
		metadata["operatorId"].(float64) != 88 {
		t.Fatalf("unexpected crm metadata: %#v", metadata)
	}
	tags := metadata["autoTags"].([]any)
	if len(tags) == 0 {
		t.Fatalf("expected auto tags in crm metadata: %#v", metadata)
	}
	tagDetails := metadata["autoTagDetails"].([]any)
	if len(tagDetails) == 0 {
		t.Fatalf("expected auto tag details in crm metadata: %#v", metadata)
	}
	firstTagDetail := tagDetails[0].(map[string]any)
	if firstTagDetail["label"] == "" || firstTagDetail["reason"] == "" || firstTagDetail["actionLabel"] == "" {
		t.Fatalf("unexpected auto tag detail metadata: %#v", tagDetails)
	}
}

func TestSalesLeadSyncToCRMReportsDisabledWebhook(t *testing.T) {
	setupSalesLeadListTestDB(t)
	lead := models.SalesLead{
		CustomerName: "王先生",
		Phone:        "13800002222",
		Status:       enums.SalesLeadStatusNew,
	}
	if err := sqls.DB().Create(&lead).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := SalesLeadService.SyncToCRM(request.SyncSalesLeadToCRMRequest{ID: lead.ID}, &dto.AuthPrincipal{UserID: 1, Username: "admin"})
	if err != nil {
		t.Fatalf("SyncToCRM() disabled error = %v", err)
	}
	if resp.Sent || resp.WebhookEnabled || !strings.Contains(resp.Message, "未启用") {
		t.Fatalf("unexpected disabled response: %#v", resp)
	}
}

func TestSalesLeadFollowUpReminderSummaryCountsDueLeads(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	overdue := today.AddDate(0, 0, -1)
	future := today.AddDate(0, 0, 2)
	seeds := []models.SalesLead{
		{CustomerName: "逾期客户", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, OwnerUserID: 7, NextFollowUpAt: &overdue},
		{CustomerName: "今日未分配客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, NextFollowUpAt: &today},
		{CustomerName: "未来客户", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentMedium, OwnerUserID: 7, NextFollowUpAt: &future},
		{CustomerName: "未设置客户", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh},
		{CustomerName: "已转化客户", Status: enums.SalesLeadStatusConverted, IntentLevel: enums.SalesLeadIntentHigh, NextFollowUpAt: &overdue},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	summary := SalesLeadService.GetFollowUpReminderSummary(request.SalesLeadFollowUpReminderRequest{Limit: 5})
	if summary.OverdueCount != 1 || summary.TodayCount != 1 || summary.DueCount != 2 || summary.UnassignedDueCount != 1 || summary.MissingScheduleCount != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(summary.PreviewLeads) != 2 {
		t.Fatalf("expected 2 preview leads, got %#v", summary.PreviewLeads)
	}
	if summary.PreviewLeads[0].FollowUpState != "overdue" || summary.PreviewLeads[1].FollowUpState != "today" {
		t.Fatalf("unexpected preview states: %#v", summary.PreviewLeads)
	}
	if summary.Message == "" ||
		!strings.Contains(summary.Message, "逾期未跟进：1") ||
		!strings.Contains(summary.Message, "今日待跟进：1") ||
		!strings.Contains(summary.Message, "重点线索") {
		t.Fatalf("unexpected reminder message: %s", summary.Message)
	}
}

func TestSalesLeadAppointmentSummaryCountsAppointments(t *testing.T) {
	setupSalesLeadListTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	overdue := today.AddDate(0, 0, -1)
	future := today.AddDate(0, 0, 2)
	seeds := []models.SalesLead{
		{CustomerName: "逾期预约", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentHigh, OwnerUserID: 7, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &overdue, AppointmentStore: "徐汇店"},
		{CustomerName: "今日预约", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &today, AppointmentStore: "静安店"},
		{CustomerName: "未来预约", Status: enums.SalesLeadStatusFollowing, IntentLevel: enums.SalesLeadIntentMedium, OwnerUserID: 7, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &future, AppointmentStore: "浦东店"},
		{CustomerName: "未定时间预约", Status: enums.SalesLeadStatusNew, IntentLevel: enums.SalesLeadIntentMedium, BuyingStage: enums.SalesLeadStageAppointment, AppointmentTimeText: "周末方便", AppointmentStore: "待确认"},
		{CustomerName: "已转化预约", Status: enums.SalesLeadStatusConverted, IntentLevel: enums.SalesLeadIntentHigh, BuyingStage: enums.SalesLeadStageAppointment, AppointmentAt: &today},
	}
	for i := range seeds {
		if err := sqls.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create lead: %v", err)
		}
	}

	summary := SalesLeadService.GetAppointmentSummary(request.SalesLeadAppointmentSummaryRequest{Days: 7, Limit: 5})
	if summary.OverdueCount != 1 || summary.TodayCount != 1 || summary.UpcomingCount != 1 || summary.UnscheduledCount != 1 || summary.UnassignedCount != 2 {
		t.Fatalf("unexpected appointment summary: %#v", summary)
	}
	if len(summary.PreviewAppointments) != 4 {
		t.Fatalf("expected 4 preview appointments, got %#v", summary.PreviewAppointments)
	}
	if summary.PreviewAppointments[0].AppointmentState != "overdue" ||
		summary.PreviewAppointments[1].AppointmentState != "today" ||
		summary.PreviewAppointments[2].AppointmentState != "upcoming" ||
		summary.PreviewAppointments[3].AppointmentState != "unscheduled" {
		t.Fatalf("unexpected appointment preview states: %#v", summary.PreviewAppointments)
	}
	if summary.Message == "" ||
		!strings.Contains(summary.Message, "今日预约：1") ||
		!strings.Contains(summary.Message, "逾期未到店：1") ||
		!strings.Contains(summary.Message, "重点预约") {
		t.Fatalf("unexpected appointment message: %s", summary.Message)
	}
}

func TestSalesLeadSendAppointmentReminderCreatesNotifications(t *testing.T) {
	setupSalesLeadListTestDB(t)
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })
	owner := models.User{Username: "appointment-owner", Status: enums.StatusOk}
	operator := models.User{Username: "appointment-manager", Status: enums.StatusOk}
	if err := sqls.DB().Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := sqls.DB().Create(&operator).Error; err != nil {
		t.Fatalf("create operator: %v", err)
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())
	if err := sqls.DB().Create(&models.SalesLead{
		CustomerName:     "今日试躺客户",
		Status:           enums.SalesLeadStatusFollowing,
		IntentLevel:      enums.SalesLeadIntentHigh,
		OwnerUserID:      owner.ID,
		BuyingStage:      enums.SalesLeadStageAppointment,
		AppointmentAt:    &today,
		AppointmentStore: "徐汇店",
	}).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}

	summary, err := SalesLeadService.SendAppointmentReminder(request.SalesLeadAppointmentSummaryRequest{Days: 7, Limit: 5}, &dto.AuthPrincipal{UserID: operator.ID, Username: operator.Username})
	if err != nil {
		t.Fatalf("SendAppointmentReminder() error = %v", err)
	}
	if !summary.NotificationSent || summary.TodayCount != 1 {
		t.Fatalf("unexpected appointment reminder result: %#v", summary)
	}
	var count int64
	if err := sqls.DB().Model(&models.Notification{}).
		Where("notification_type = ?", "sales_lead_appointment_reminder").
		Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected owner and operator notifications, got %d", count)
	}
}

func TestSalesLeadSendFollowUpReminderCreatesNotifications(t *testing.T) {
	setupSalesLeadListTestDB(t)
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })
	owner := models.User{Username: "owner", Status: enums.StatusOk}
	operator := models.User{Username: "manager", Status: enums.StatusOk}
	if err := sqls.DB().Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := sqls.DB().Create(&operator).Error; err != nil {
		t.Fatalf("create operator: %v", err)
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, now.Location())
	if err := sqls.DB().Create(&models.SalesLead{
		CustomerName:   "今日客户",
		Status:         enums.SalesLeadStatusFollowing,
		IntentLevel:    enums.SalesLeadIntentHigh,
		OwnerUserID:    owner.ID,
		NextFollowUpAt: &today,
	}).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}

	summary, err := SalesLeadService.SendFollowUpReminder(request.SalesLeadFollowUpReminderRequest{Limit: 5}, &dto.AuthPrincipal{UserID: operator.ID, Username: operator.Username})
	if err != nil {
		t.Fatalf("SendFollowUpReminder() error = %v", err)
	}
	if !summary.NotificationSent || summary.TodayCount != 1 {
		t.Fatalf("unexpected reminder result: %#v", summary)
	}
	var count int64
	if err := sqls.DB().Model(&models.Notification{}).
		Where("notification_type = ?", "sales_lead_follow_up_reminder").
		Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected owner and operator notifications, got %d", count)
	}
}

func requestForFollowUpStatus(status string) request.SalesLeadListRequest {
	return request.SalesLeadListRequest{Page: 1, Limit: 20, FollowUpStatus: status}
}

func requestForAppointmentStatus(status string) request.SalesLeadListRequest {
	return request.SalesLeadListRequest{Page: 1, Limit: 20, AppointmentStatus: status}
}

func setupSalesLeadListTestDB(t *testing.T) {
	t.Helper()
	config.SetCurrent(&config.Config{})
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
		&models.SalesLead{},
		&models.User{},
		&models.Notification{},
		&models.Customer{},
		&models.CustomerContact{},
		&models.Conversation{},
		&models.Channel{},
		&models.Ticket{},
		&models.TicketProgress{},
		&models.TicketTag{},
		&models.TicketNoSequence{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
}
