package runtime

import (
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestReplyCommitStoresWorkflowRunIDOnAIMessage(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)

	replyMessage, err := newReplyCommitService().CommitAIReply(replyCommitInput{
		Conversation:  *conversation,
		Message:       models.Message{ID: 101, RequestID: "trace-101"},
		AIAgent:       *aiAgent,
		ReplyText:     "AI reply",
		ClientPrefix:  "ai_reply",
		WorkflowRunID: 9988,
	})
	if err != nil {
		t.Fatalf("CommitAIReply() error = %v", err)
	}
	if replyMessage == nil {
		t.Fatalf("expected reply message")
	}
	if replyMessage.WorkflowRunID != 9988 {
		t.Fatalf("replyMessage.WorkflowRunID=%d want 9988", replyMessage.WorkflowRunID)
	}

	var stored models.Message
	if err := db.First(&stored, replyMessage.ID).Error; err != nil {
		t.Fatalf("find reply message: %v", err)
	}
	if stored.WorkflowRunID != 9988 {
		t.Fatalf("stored.WorkflowRunID=%d want 9988", stored.WorkflowRunID)
	}
}

func TestSanitizeCommercialReplyTextDowngradesRiskyCommitments(t *testing.T) {
	got := sanitizeCommercialReplyText("很多重视腰背支撑的家庭都选它。已为您预留时段，周六见！预约周末试躺可以提前帮您留好专属时段。顾问会第一时间联系您，还能体验按摩功能。周六下午3点没问题，也不用担心滑落。老人一看就会按。")
	for _, banned := range []string{"很多重视腰背支撑", "已为您预留", "专属时段", "第一时间", "按摩功能", "没问题", "不用担心滑落", "一看就会按"} {
		if strings.Contains(got, banned) {
			t.Fatalf("reply still contains risky wording %q: %s", banned, got)
		}
	}
	for _, want := range []string{"产品定位", "具体时段以门店顾问确认为准", "门店顾问确认", "升降功能", "现场试一下"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reply missing %q: %s", want, got)
		}
	}
}

func TestSanitizeCommercialReplyTextWeakensEfficacyAndUnsupportedSpecs(t *testing.T) {
	got := sanitizeCommercialReplyText("单独换个合适的枕头效果就很明显，能让肩颈腰都能放松；脊护效果好，头、肩、腰、臀、腿五个区域做了不同硬度的弹簧排布，弹簧线径更粗，还有2cm厚的乳胶。到店也是一样的，不会到店再变，部分活动款支持体验期。")
	for _, banned := range []string{"效果就很明显", "肩颈腰都能放松", "脊护效果好", "五个区域做了不同硬度", "弹簧线径", "2cm厚", "不会到店再变", "支持体验期"} {
		if strings.Contains(got, banned) {
			t.Fatalf("reply still contains risky wording %q: %s", banned, got)
		}
	}
	for _, want := range []string{"可能有帮助", "承托、释压体验", "更强调支撑承托", "承托差异设计", "舒适释压层", "门店当天活动", "订单条款"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reply missing %q: %s", want, got)
		}
	}
}

func TestEnsureLowBudgetPathAddsRealisticAlternative(t *testing.T) {
	got := ensureLowBudgetPath("可以先看云感舒睡款，8000-13000元。", "我们预算最多三四千，超了就算了。")
	for _, want := range []string{"三四千", "样品", "活动款", "暂不下单"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reply missing %q: %s", want, got)
		}
	}
}

func TestTailorDigitalStoreFallbackAnswersSpecificCustomerNeed(t *testing.T) {
	fallback := "我是慕小眠，你的问题我已经记录；涉及最终价格、库存、退换货、售后争议或医疗效果时，我不能直接承诺，会安排门店顾问进一步确认。你也可以留下手机号或微信，方便顾问跟进。"
	got := tailorDigitalStoreFallback(fallback, "我最近颈肩不舒服，想了解T10释压枕和床垫怎么搭配")
	if strings.Contains(got, "你的问题我已经记录") || strings.Contains(got, "手机号或微信") {
		t.Fatalf("expected tailored answer instead of generic fallback, got %q", got)
	}
	for _, want := range []string{"T10释压枕", "不一定要和床垫成套买", "预算有限"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reply missing %q: %s", want, got)
		}
	}
}

func TestSanitizeReplyAgainstCustomerContextRemovesUnsupportedBackConcern(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)
	if err := db.Create(&models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "那你到底现在能给我什么方案？",
	}).Error; err != nil {
		t.Fatalf("create customer message: %v", err)
	}
	got := sanitizeReplyAgainstCustomerContext("腰疼的话，这款适合您这样需要针对性支撑的情况。", conversation.ID)
	if strings.Contains(got, "腰疼") || strings.Contains(got, "您这样") {
		t.Fatalf("expected unsupported back concern to be removed, got %q", got)
	}
	if !strings.Contains(got, "想要更强承托") {
		t.Fatalf("expected neutral support wording, got %q", got)
	}
}

func TestSanitizeReplyAgainstCustomerContextKeepsSupportedBackConcern(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)
	if err := db.Create(&models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "我腰疼，想看护脊床垫。",
	}).Error; err != nil {
		t.Fatalf("create customer message: %v", err)
	}
	input := "腰背不适的话，这款可以重点试躺。"
	if got := sanitizeReplyAgainstCustomerContext(input, conversation.ID); got != input {
		t.Fatalf("expected supported back concern to stay unchanged, got %q", got)
	}
}

func TestSanitizeReplyAgainstKnownLeadInfoAvoidsAskingPhoneAgain(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)
	if err := db.Create(&models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "我姓林，手机 13800138000，周六下午三点去徐汇店。",
	}).Error; err != nil {
		t.Fatalf("create customer message: %v", err)
	}
	got := sanitizeReplyAgainstKnownLeadInfo("麻烦您再留一下您的姓名和手机号，方便顾问提前帮您安排体验和礼包～", conversation.ID)
	if strings.Contains(got, "再留") || strings.Contains(got, "姓名和手机号") {
		t.Fatalf("expected duplicate phone ask to be removed, got %q", got)
	}
	if !strings.Contains(got, "转给门店顾问") {
		t.Fatalf("expected handoff confirmation, got %q", got)
	}
}

func TestSanitizeStoreMentionAgainstCustomerContextRemovesUnmentionedArea(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)
	if err := db.Create(&models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "家里老人腰不舒服，有没有偏硬一点的？",
	}).Error; err != nil {
		t.Fatalf("create customer message: %v", err)
	}
	got := sanitizeStoreMentionAgainstCustomerContext("欢迎周末带老人来徐汇门店试躺。", conversation.ID)
	if strings.Contains(got, "徐汇") {
		t.Fatalf("expected unmentioned store area to be removed, got %q", got)
	}
	if !strings.Contains(got, "门店") {
		t.Fatalf("expected generic store mention, got %q", got)
	}
}

func TestSanitizeStoreMentionAgainstCustomerContextKeepsMentionedArea(t *testing.T) {
	db := setupReplyCommitTestDB(t)
	aiAgent := createReplyCommitTestAIAgent(t, db)
	conversation := createReplyCommitTestConversation(t, db, aiAgent.ID)
	if err := db.Create(&models.Message{
		ConversationID: conversation.ID,
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "我周六想去徐汇店试躺。",
	}).Error; err != nil {
		t.Fatalf("create customer message: %v", err)
	}
	input := "欢迎周末来徐汇门店试躺。"
	if got := sanitizeStoreMentionAgainstCustomerContext(input, conversation.ID); got != input {
		t.Fatalf("expected mentioned store area to stay, got %q", got)
	}
}

func TestEnsureConcretePlanWhenAskedAddsRetailPlan(t *testing.T) {
	got := ensureConcretePlanWhenAsked("您是给自己用还是给长辈用？", "那你到底现在能给我什么方案？")
	for _, want := range []string{"云感舒睡款", "脊护支撑款", "8000-13000", "12000-18000"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected concrete plan to include %q, got %q", want, got)
		}
	}
}

func TestEnsureConcretePlanWhenAskedKeepsExistingPlan(t *testing.T) {
	input := "可以先看云感舒睡款，偏柔和包裹。"
	if got := ensureConcretePlanWhenAsked(input, "那你到底现在能给我什么方案？"); got != input {
		t.Fatalf("expected existing plan unchanged, got %q", got)
	}
}

func TestSanitizeChitchatReplyAvoidsInventingPainPoint(t *testing.T) {
	got := sanitizeChitchatReply("哈哈，写诗我可不太擅长。您最近睡得好吗？有没有腰酸背累或者床垫不舒服的情况？跟我说说，我帮您分析分析。", "你会写诗吗？")
	if strings.Contains(got, "腰酸背累") || strings.Contains(got, "不太擅长") {
		t.Fatalf("expected chitchat reply to avoid invented pain point, got %q", got)
	}
	if !strings.Contains(got, "可以简单来一句") || !strings.Contains(got, "随便看看") {
		t.Fatalf("expected warmer chitchat redirect, got %q", got)
	}
}

func setupReplyCommitTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := "reply_commit_test_" + strings.NewReplacer("/", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})
	if err := db.AutoMigrate(
		&models.AIAgent{},
		&models.Channel{},
		&models.ChannelMessageOutbox{},
		&models.Conversation{},
		&models.ConversationReadState{},
		&models.ConversationEventLog{},
		&models.Message{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createReplyCommitTestAIAgent(t *testing.T, db *gorm.DB) *models.AIAgent {
	t.Helper()
	now := time.Now()
	item := &models.AIAgent{
		Name:   "reply-agent",
		Status: enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}
	return item
}

func createReplyCommitTestConversation(t *testing.T, db *gorm.DB, aiAgentID int64) *models.Conversation {
	t.Helper()
	now := time.Now()
	item := &models.Conversation{
		CustomerID:   1,
		ChannelID:    11,
		AIAgentID:    aiAgentID,
		Status:       enums.IMConversationStatusAIServing,
		LastActiveAt: now,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	return item
}
