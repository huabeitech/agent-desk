package event_handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/events"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestTicketAssignedInAppNotification(t *testing.T) {
	setupNotificationEventHandlerTestDB(t)

	ticket := &models.Ticket{
		TicketNo:          "TK202604280001",
		Title:             "退款处理",
		Source:            enums.TicketSourceManual,
		Status:            enums.TicketStatusPending,
		CurrentAssigneeID: 11,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.TicketRepository.Create(sqls.DB(), ticket); err != nil {
		t.Fatalf("create ticket error = %v", err)
	}

	if err := handleTicketAssignedInAppNotification(context.Background(), events.TicketAssignedEvent{
		TicketID:   ticket.ID,
		FromUserID: 0,
		ToUserID:   11,
		OperatorID: 1,
		Reason:     "需要人工跟进",
	}); err != nil {
		t.Fatalf("handler error = %v", err)
	}

	list := repositories.NotificationRepository.Find(sqls.DB(), sqls.NewCnd().Eq("recipient_user_id", 11))
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	got := list[0]
	if got.NotificationType != "ticket_assigned" || got.BizType != "ticket" || got.BizID != ticket.ID {
		t.Fatalf("unexpected notification: %+v", got)
	}
	if got.ActionURL != "/dashboard/tickets?ticketId=1" {
		t.Fatalf("unexpected action url: %q", got.ActionURL)
	}
}

func TestConversationAssignedInAppNotification(t *testing.T) {
	setupNotificationEventHandlerTestDB(t)

	conversation := &models.Conversation{
		CustomerName:      "张三",
		Status:            enums.IMConversationStatusActive,
		CurrentAssigneeID: 22,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.ConversationRepository.Create(sqls.DB(), conversation); err != nil {
		t.Fatalf("create conversation error = %v", err)
	}
	lead := &models.SalesLead{
		CustomerID:         conversation.CustomerID,
		ConversationID:     conversation.ID,
		CustomerName:       "张三",
		Phone:              "13800001111",
		InterestedProducts: "慕斯脊护支撑款",
		BudgetMin:          12000,
		BudgetMax:          18000,
		DemandSummary:      "老人腰不好，想预约周末试躺。",
		IntentLevel:        enums.SalesLeadIntentHigh,
		BuyingStage:        enums.SalesLeadStageAppointment,
		Status:             enums.SalesLeadStatusNew,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.SalesLeadRepository.Create(sqls.DB(), lead); err != nil {
		t.Fatalf("create sales lead error = %v", err)
	}
	message := &models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "我姓张，电话13800001111，想周末带老人试躺脊护支撑款。",
		SendStatus:     enums.IMMessageStatusSent,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.MessageRepository.Create(sqls.DB(), message); err != nil {
		t.Fatalf("create message error = %v", err)
	}

	if err := handleConversationAssignedInAppNotification(context.Background(), events.ConversationAssignedEvent{
		ConversationID: conversation.ID,
		FromUserID:     0,
		ToUserID:       22,
		OperatorID:     1,
		Reason:         "自动分配",
		AssignType:     events.ConversationAssignTypeAutoAssign,
	}); err != nil {
		t.Fatalf("handler error = %v", err)
	}

	list := repositories.NotificationRepository.Find(sqls.DB(), sqls.NewCnd().Eq("recipient_user_id", 22))
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	got := list[0]
	if got.NotificationType != "conversation_assigned" || got.BizType != "conversation" || got.BizID != conversation.ID {
		t.Fatalf("unexpected notification: %+v", got)
	}
	if got.ActionURL != "/dashboard/conversations?conversationId=1" {
		t.Fatalf("unexpected action url: %q", got.ActionURL)
	}
	for _, want := range []string{"13800001111", "慕斯脊护支撑款", "12000-18000元", "老人腰不好", "最近对话"} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("expected notification content to contain %q, got %q", want, got.Content)
		}
	}
}

func TestSalesLeadCreatedInAppNotification(t *testing.T) {
	setupNotificationEventHandlerTestDB(t)

	if err := repositories.UserRepository.Create(sqls.DB(), &models.User{
		Username: "admin",
		Nickname: "店长",
		Status:   enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}); err != nil {
		t.Fatalf("create user error = %v", err)
	}
	lead := &models.SalesLead{
		ConversationID:      99,
		CustomerName:        "李女士",
		Phone:               "13800001111",
		InterestedProducts:  "慕斯脊护支撑款",
		BudgetMin:           12000,
		BudgetMax:           18000,
		DemandSummary:       "老人腰不好，周末想来试躺。",
		IntentLevel:         enums.SalesLeadIntentHigh,
		BuyingStage:         enums.SalesLeadStageAppointment,
		AppointmentTimeText: "本周末下午",
		AppointmentStore:    "徐汇体验店",
		AppointmentPeople:   2,
		Status:              enums.SalesLeadStatusNew,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.SalesLeadRepository.Create(sqls.DB(), lead); err != nil {
		t.Fatalf("create sales lead error = %v", err)
	}

	if err := handleSalesLeadCreatedInAppNotification(context.Background(), events.SalesLeadCreatedEvent{
		LeadID:         lead.ID,
		ConversationID: lead.ConversationID,
		Reason:         "客户已留联系方式、高意向客户、预约/到店意向",
	}); err != nil {
		t.Fatalf("handler error = %v", err)
	}

	list := repositories.NotificationRepository.Find(sqls.DB(), sqls.NewCnd().Eq("notification_type", "sales_lead_created"))
	if len(list) != 1 {
		t.Fatalf("expected 1 sales lead notification, got %d", len(list))
	}
	got := list[0]
	if got.BizType != "sales_lead" || got.BizID != lead.ID {
		t.Fatalf("unexpected notification: %+v", got)
	}
	if got.ActionURL != "/dashboard/sales-leads?leadId=1" {
		t.Fatalf("unexpected action url: %q", got.ActionURL)
	}
	for _, want := range []string{"李女士", "13800001111", "慕斯脊护支撑款", "12000-18000元", "老人腰不好", "徐汇体验店", "2人"} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("expected notification content to contain %q, got %q", want, got.Content)
		}
	}
}

func TestSalesLeadCreatedAutoSyncsQualifiedLeadToCRM(t *testing.T) {
	setupNotificationEventHandlerTestDB(t)
	lead := &models.SalesLead{
		ConversationID:      99,
		CustomerName:        "李女士",
		Phone:               "13800001111",
		InterestedProducts:  "慕斯脊护支撑款",
		BudgetMin:           12000,
		BudgetMax:           18000,
		DemandSummary:       "老人腰不好，周末想来试躺。",
		IntentLevel:         enums.SalesLeadIntentHigh,
		BuyingStage:         enums.SalesLeadStageAppointment,
		AppointmentTimeText: "本周末下午",
		AppointmentStore:    "徐汇体验店",
		Status:              enums.SalesLeadStatusNew,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.SalesLeadRepository.Create(sqls.DB(), lead); err != nil {
		t.Fatalf("create sales lead error = %v", err)
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

	if err := handleSalesLeadCreatedCRMAutoSync(context.Background(), events.SalesLeadCreatedEvent{
		LeadID:         lead.ID,
		ConversationID: lead.ConversationID,
		Reason:         "客户已留联系方式、高意向客户、预约/到店意向",
	}); err != nil {
		t.Fatalf("crm auto sync handler error = %v", err)
	}
	if got["eventType"] != "sales_lead_crm_sync" || !strings.Contains(got["text"].(string), "李女士") {
		t.Fatalf("unexpected crm webhook payload: %#v", got)
	}
	metadata := got["metadata"].(map[string]any)
	if metadata["leadId"].(float64) != float64(lead.ID) || metadata["operatorName"] != "system" {
		t.Fatalf("unexpected crm metadata: %#v", metadata)
	}
	if !strings.Contains(metadata["remark"].(string), "AI数字店长自动同步") {
		t.Fatalf("expected auto sync remark, got %#v", metadata["remark"])
	}
}

func TestSalesLeadCreatedAutoSyncSkipsLowIntentLead(t *testing.T) {
	setupNotificationEventHandlerTestDB(t)
	lead := &models.SalesLead{
		CustomerName: "普通咨询客户",
		Phone:        "13800001111",
		IntentLevel:  enums.SalesLeadIntentMedium,
		BuyingStage:  enums.SalesLeadStageConsulting,
		Status:       enums.SalesLeadStatusNew,
		AuditFields: models.AuditFields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	if err := repositories.SalesLeadRepository.Create(sqls.DB(), lead); err != nil {
		t.Fatalf("create sales lead error = %v", err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
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

	if err := handleSalesLeadCreatedCRMAutoSync(context.Background(), events.SalesLeadCreatedEvent{LeadID: lead.ID}); err != nil {
		t.Fatalf("crm auto sync handler error = %v", err)
	}
	if called {
		t.Fatal("expected low intent lead to skip CRM auto sync")
	}
}

func setupNotificationEventHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.Notification{}, &models.Ticket{}, &models.Conversation{}, &models.SalesLead{}, &models.Message{}, &models.User{}); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	sqls.SetDB(db)
	return db
}
