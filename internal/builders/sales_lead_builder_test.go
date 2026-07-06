package builders

import (
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func TestBuildSalesLeadAutoTags(t *testing.T) {
	overdue := time.Now().AddDate(0, 0, -1)
	lead := &models.SalesLead{
		Phone:               "13800000000",
		BudgetMin:           20000,
		InterestedProducts:  "智能床垫",
		IntentLevel:         enums.SalesLeadIntentHigh,
		BuyingStage:         enums.SalesLeadStageAppointment,
		AppointmentTimeText: "周末",
		SourceChannel:       "官网",
		Status:              enums.SalesLeadStatusFollowing,
		NextFollowUpAt:      &overdue,
	}

	resp := BuildSalesLead(lead)
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "高意向")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "已预约")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "未分配")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "逾期跟进")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "已留联系方式")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "有预算")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "高预算")
	assertHasSalesLeadAutoTag(t, resp.AutoTags, "渠道:官网")
	assertHasSalesLeadAutoTagDetail(t, resp.AutoTagDetails, "高意向", "hot", "优先跟进")
	assertHasSalesLeadAutoTagDetail(t, resp.AutoTagDetails, "逾期跟进", "danger", "立即联系客户")
	assertHasSalesLeadAutoTagDetail(t, resp.AutoTagDetails, "未分配", "warning", "认领或分配顾问")

	lead.Status = enums.SalesLeadStatusVisited
	visitedResp := BuildSalesLead(lead)
	assertHasSalesLeadAutoTag(t, visitedResp.AutoTags, "已到店")
}

func TestBuildSalesLeadIncludesRecentMessageSummary(t *testing.T) {
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
	if err := db.AutoMigrate(&models.Conversation{}, &models.Message{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	conversation := models.Conversation{
		CustomerName:       "李静",
		Status:             enums.IMConversationStatusAIServing,
		LastMessageSummary: "客户问老人腰不好怎么选床垫，AI 推荐分区支撑款。",
	}
	if err := db.Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	message := models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "我想给爸妈选床垫，老人腰不好，预算两万以内，周末想去店里试一下。",
	}
	if err := db.Create(&message).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}
	lead := &models.SalesLead{
		ConversationID: conversation.ID,
		LastMessageID:  message.ID,
		Status:         enums.SalesLeadStatusNew,
	}

	resp := BuildSalesLead(lead)
	if resp.LastMessageSummary != conversation.LastMessageSummary {
		t.Fatalf("LastMessageSummary = %q, want %q", resp.LastMessageSummary, conversation.LastMessageSummary)
	}
	if resp.LastCustomerMessage != message.Content {
		t.Fatalf("LastCustomerMessage = %q, want %q", resp.LastCustomerMessage, message.Content)
	}
}

func assertHasSalesLeadAutoTag(t *testing.T, tags []string, want string) {
	t.Helper()
	for _, tag := range tags {
		if tag == want {
			return
		}
	}
	t.Fatalf("auto tags %#v missing %q", tags, want)
}

func assertHasSalesLeadAutoTagDetail(t *testing.T, tags []response.SalesLeadAutoTag, label string, level string, action string) {
	t.Helper()
	for _, tag := range tags {
		if tag.Label == label {
			if tag.Level != level || tag.ActionLabel != action || tag.Reason == "" {
				t.Fatalf("unexpected auto tag detail for %q: %#v", label, tag)
			}
			return
		}
	}
	t.Fatalf("auto tag details %#v missing %q", tags, label)
}
