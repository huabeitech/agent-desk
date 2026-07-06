//go:build dev

package services

import (
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConversationBuildFollowUpAdviceUsesSalesLead(t *testing.T) {
	db := setupConversationFollowUpAdviceTestDB(t)
	now := time.Now()
	conversation := models.Conversation{
		CustomerID:         8,
		CustomerName:       "王先生",
		Status:             enums.IMConversationStatusActive,
		LastMessageSummary: "客户想买偏硬床垫，预算两万左右。",
		LastMessageAt:      now,
		LastActiveAt:       now,
	}
	if err := db.Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	message := models.Message{
		ConversationID: conversation.ID,
		ClientMsgID:    "lead-customer-1",
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "我周末能去上海旗舰店试一下吗？",
		SentAt:         &now,
	}
	if err := db.Create(&message).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}
	lead := models.SalesLead{
		CustomerID:         conversation.CustomerID,
		ConversationID:     conversation.ID,
		CustomerName:       "王先生",
		Phone:              "13800000000",
		City:               "上海",
		BudgetMin:          18000,
		BudgetMax:          22000,
		InterestedProducts: "T9 旗舰床垫",
		DemandSummary:      "偏硬支撑，周末想预约到店试躺",
		IntentLevel:        enums.SalesLeadIntentHigh,
		BuyingStage:        enums.SalesLeadStageAppointment,
		Status:             enums.SalesLeadStatusNew,
		LastMessageID:      message.ID,
	}
	if err := db.Create(&lead).Error; err != nil {
		t.Fatalf("create lead: %v", err)
	}
	followUp := models.LeadFollowUp{
		LeadID:       lead.ID,
		OperatorName: "顾问A",
		Content:      "已确认客户关注偏硬支撑。",
		NextAction:   "确认到店时间",
		CreatedAt:    now,
	}
	if err := db.Create(&followUp).Error; err != nil {
		t.Fatalf("create follow-up: %v", err)
	}

	advice, err := ConversationService.BuildFollowUpAdvice(conversation.ID)
	if err != nil {
		t.Fatalf("BuildFollowUpAdvice() error = %v", err)
	}
	if advice.Source != "sales_lead" || advice.LeadID != lead.ID {
		t.Fatalf("unexpected source/lead: source=%q lead=%d", advice.Source, advice.LeadID)
	}
	for _, want := range []string{"王先生", "T9 旗舰床垫", "最近对话", "周末能去上海旗舰店"} {
		if !strings.Contains(advice.CopyText, want) {
			t.Fatalf("CopyText missing %q: %s", want, advice.CopyText)
		}
	}
}

func TestConversationBuildFollowUpAdviceWithoutSalesLead(t *testing.T) {
	db := setupConversationFollowUpAdviceTestDB(t)
	now := time.Now()
	conversation := models.Conversation{
		CustomerName:       "李女士",
		Status:             enums.IMConversationStatusAIServing,
		LastMessageSummary: "客户询问儿童房床垫和除螨面料。",
		LastMessageAt:      now,
		LastActiveAt:       now,
	}
	if err := db.Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	messages := []models.Message{
		{ConversationID: conversation.ID, ClientMsgID: "conversation-customer-1", SenderType: enums.IMSenderTypeCustomer, MessageType: enums.IMMessageTypeText, Content: "小朋友睡，想要护脊一点的。", SentAt: &now},
		{ConversationID: conversation.ID, ClientMsgID: "conversation-ai-1", SenderType: enums.IMSenderTypeAI, MessageType: enums.IMMessageTypeText, Content: "可以先看儿童护脊系列。", SentAt: &now},
	}
	if err := db.Create(&messages).Error; err != nil {
		t.Fatalf("create messages: %v", err)
	}

	advice, err := ConversationService.BuildFollowUpAdvice(conversation.ID)
	if err != nil {
		t.Fatalf("BuildFollowUpAdvice() error = %v", err)
	}
	if advice.Source != "conversation" || advice.LeadID != 0 {
		t.Fatalf("unexpected source/lead: source=%q lead=%d", advice.Source, advice.LeadID)
	}
	for _, want := range []string{"【会话跟进摘要】", "李女士", "小朋友睡", "尚未形成销售线索"} {
		if !strings.Contains(advice.CopyText, want) {
			t.Fatalf("CopyText missing %q: %s", want, advice.CopyText)
		}
	}
	if len(advice.RiskHints) == 0 {
		t.Fatalf("expected risk hints")
	}
}

func setupConversationFollowUpAdviceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.Message{},
		&models.SalesLead{},
		&models.LeadFollowUp{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
