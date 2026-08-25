package services

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/openidentity"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestAllowAIMessageOnPendingHandoff(t *testing.T) {
	conversation := &models.Conversation{
		Status:            enums.IMConversationStatusPending,
		CurrentAssigneeID: 0,
		HandoffAt:         ptrTime(time.Now()),
	}
	if !MessageService.allowAIMessageOnPendingHandoff(conversation) {
		t.Fatalf("expected pending handoff conversation to allow ai handoff notice")
	}

	conversation.Status = enums.IMConversationStatusAIServing
	if MessageService.allowAIMessageOnPendingHandoff(conversation) {
		t.Fatalf("expected ai serving conversation not to use pending handoff allowance")
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func setupMessageWelcomeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := "message_welcome_test_" + strings.NewReplacer("/", "_").Replace(t.Name())
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
		&models.Customer{},
		&models.CustomerIdentity{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.ConversationReadState{},
		&models.ConversationEventLog{},
		&models.Message{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createWelcomeTestAIAgent(t *testing.T, db *gorm.DB, welcomeMessage string) *models.AIAgent {
	t.Helper()

	now := time.Now()
	aiAgent := &models.AIAgent{
		Name:           "welcome-test-agent",
		Status:         enums.StatusOk,
		ServiceMode:    enums.IMConversationServiceModeAIOnly,
		WelcomeMessage: welcomeMessage,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	if err := db.Create(aiAgent).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}
	return aiAgent
}

func welcomeTestExternalUser(id string) openidentity.ExternalUser {
	return openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceExternal,
		ExternalID:     id,
		ExternalName:   "访客" + id,
	}
}

func createMessageTestConversation(t *testing.T, db *gorm.DB, aiAgentID int64) *models.Conversation {
	t.Helper()
	now := time.Now()
	conversation := &models.Conversation{
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
	if err := db.Create(conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	return conversation
}

func workflowTestAIPrincipal() *dto.AuthPrincipal {
	return &dto.AuthPrincipal{UserID: 0, Username: "AI", Nickname: "AI"}
}

func TestConversationCreateCreatesAIWelcomeMessage(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "  您好，请问有什么可以帮您？  ")

	conversation, err := ConversationService.Create(welcomeTestExternalUser("welcome-1"), 11, aiAgent.ID)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if conversation == nil {
		t.Fatalf("expected conversation")
	}

	var messages []models.Message
	if err := db.Find(&messages).Error; err != nil {
		t.Fatalf("find messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected exactly one welcome message, got %d", len(messages))
	}
	message := messages[0]
	if message.ConversationID != conversation.ID {
		t.Fatalf("expected conversation_id %d, got %d", conversation.ID, message.ConversationID)
	}
	if message.SenderType != enums.IMSenderTypeAI {
		t.Fatalf("expected sender type ai, got %q", message.SenderType)
	}
	if message.SenderID != aiAgent.ID {
		t.Fatalf("expected sender id %d, got %d", aiAgent.ID, message.SenderID)
	}
	if message.MessageType != enums.IMMessageTypeText {
		t.Fatalf("expected message type text, got %q", message.MessageType)
	}
	if message.Content != "您好，请问有什么可以帮您？" {
		t.Fatalf("expected trimmed welcome content, got %q", message.Content)
	}
	if message.SendStatus != enums.IMMessageStatusSent {
		t.Fatalf("expected sent status, got %d", message.SendStatus)
	}

	var updated models.Conversation
	if err := db.First(&updated, conversation.ID).Error; err != nil {
		t.Fatalf("find conversation: %v", err)
	}
	if updated.LastMessageID != message.ID {
		t.Fatalf("expected last message id %d, got %d", message.ID, updated.LastMessageID)
	}
	if updated.LastMessageSummary != "您好，请问有什么可以帮您？" {
		t.Fatalf("expected last message summary, got %q", updated.LastMessageSummary)
	}
	if updated.CustomerUnreadCount != 1 {
		t.Fatalf("expected customer unread count 1, got %d", updated.CustomerUnreadCount)
	}
	if updated.AgentUnreadCount != 0 {
		t.Fatalf("expected agent unread count 0, got %d", updated.AgentUnreadCount)
	}
}

func TestSendCustomerMessageStoresRequestIDOnMessageAndEvent(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "")
	external := welcomeTestExternalUser("trace-user")
	conversation, err := ConversationService.Create(external, 11, aiAgent.ID)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	message, err := MessageService.SendCustomerMessageWithRequestID(
		conversation.ID,
		"client-msg-trace",
		enums.IMMessageTypeText,
		"hello",
		"",
		external,
		"trace-123",
	)
	if err != nil {
		t.Fatalf("SendCustomerMessageWithRequestID() error = %v", err)
	}
	if message.RequestID != "trace-123" {
		t.Fatalf("message.RequestID=%q want %q", message.RequestID, "trace-123")
	}

	var event models.ConversationEventLog
	if err := db.Where("conversation_id = ?", conversation.ID).Order("id DESC").First(&event).Error; err != nil {
		t.Fatalf("find event: %v", err)
	}
	if event.RequestID != "trace-123" {
		t.Fatalf("event.RequestID=%q want %q", event.RequestID, "trace-123")
	}
}

func TestSendCustomerMessagesConcurrentlyAssignsUniqueIDs(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "")
	external := welcomeTestExternalUser("concurrent-user")
	conversation, err := ConversationService.Create(external, 11, aiAgent.ID)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	const messageCount = 10
	var wg sync.WaitGroup
	errCh := make(chan error, messageCount)
	for i := 0; i < messageCount; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := MessageService.SendCustomerMessageWithRequestID(
				conversation.ID,
				fmt.Sprintf("client-msg-concurrent-%d", i),
				enums.IMMessageTypeText,
				"hello concurrent",
				"",
				external,
				"trace-concurrent",
			)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("SendCustomerMessageWithRequestID() concurrent error = %v", err)
		}
	}

	var messages []models.Message
	if err := db.
		Where("conversation_id = ? AND sender_type = ?", conversation.ID, enums.IMSenderTypeCustomer).
		Order("id ASC").
		Find(&messages).Error; err != nil {
		t.Fatalf("find messages: %v", err)
	}
	if len(messages) != messageCount {
		t.Fatalf("expected %d customer messages, got %d", messageCount, len(messages))
	}
	seen := make(map[int64]struct{}, messageCount)
	for _, message := range messages {
		if message.ID <= 0 {
			t.Fatalf("expected persisted message id, got %d", message.ID)
		}
		if _, ok := seen[message.ID]; ok {
			t.Fatalf("duplicate message id %d", message.ID)
		}
		seen[message.ID] = struct{}{}
	}
}

func TestUnreadCountUsesLastReadMessageID(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "")
	conversation := createMessageTestConversation(t, db, aiAgent.ID)
	now := time.Now()

	messages := []models.Message{
		{
			ConversationID: conversation.ID,
			ClientMsgID:    "read-message",
			SenderType:     enums.IMSenderTypeCustomer,
			MessageType:    enums.IMMessageTypeText,
			Content:        "read",
			SendStatus:     enums.IMMessageStatusSent,
			SentAt:         &now,
			AuditFields:    models.AuditFields{CreatedAt: now, UpdatedAt: now},
		},
		{
			ConversationID: conversation.ID,
			ClientMsgID:    "unread-message-1",
			SenderType:     enums.IMSenderTypeCustomer,
			MessageType:    enums.IMMessageTypeText,
			Content:        "unread 1",
			SendStatus:     enums.IMMessageStatusSent,
			SentAt:         &now,
			AuditFields:    models.AuditFields{CreatedAt: now, UpdatedAt: now},
		},
		{
			ConversationID: conversation.ID,
			ClientMsgID:    "unread-message-2",
			SenderType:     enums.IMSenderTypeCustomer,
			MessageType:    enums.IMMessageTypeText,
			Content:        "unread 2",
			SendStatus:     enums.IMMessageStatusSent,
			SentAt:         &now,
			AuditFields:    models.AuditFields{CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := db.Create(&messages).Error; err != nil {
		t.Fatalf("create messages: %v", err)
	}

	readState := &models.ConversationReadState{
		ConversationID:    conversation.ID,
		ReaderType:        enums.IMSenderTypeAgent,
		ReaderID:          1,
		LastReadMessageID: messages[0].ID,
		LastReadAt:        &now,
		AuditFields:       models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(readState).Error; err != nil {
		t.Fatalf("create read state: %v", err)
	}

	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		count, err := ConversationService.countUnreadByState(ctx, conversation.ID, readState, enums.IMSenderTypeCustomer)
		if err != nil {
			return err
		}
		if count != 2 {
			t.Fatalf("unread count=%d want 2", count)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("count unread: %v", err)
	}
}

func TestSendAIMessageStoresWorkflowRunID(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "")
	conversation := createMessageTestConversation(t, db, aiAgent.ID)

	message, err := MessageService.SendAIMessageWithRequestIDAndWorkflowRunID(
		conversation.ID,
		aiAgent.ID,
		"ai-reply-workflow-1",
		enums.IMMessageTypeText,
		"AI reply",
		"",
		workflowTestAIPrincipal(),
		"trace-workflow-1",
		9988,
	)
	if err != nil {
		t.Fatalf("SendAIMessageWithRequestIDAndWorkflowRunID() error = %v", err)
	}
	if message.WorkflowRunID != 9988 {
		t.Fatalf("message.WorkflowRunID=%d want 9988", message.WorkflowRunID)
	}

	var stored models.Message
	if err := db.First(&stored, message.ID).Error; err != nil {
		t.Fatalf("find message: %v", err)
	}
	if stored.WorkflowRunID != 9988 {
		t.Fatalf("stored.WorkflowRunID=%d want 9988", stored.WorkflowRunID)
	}
}

func TestConversationCreateDoesNotDuplicateWelcomeMessageForExistingConversation(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "欢迎咨询")
	external := welcomeTestExternalUser("u-2")

	first, err := ConversationService.Create(external, 11, aiAgent.ID)
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	second, err := ConversationService.Create(external, 11, aiAgent.ID)
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected existing conversation id %d, got %d", first.ID, second.ID)
	}

	var count int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", first.ID).Count(&count).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one welcome message, got %d", count)
	}
}

func TestConversationCreateSkipsBlankWelcomeMessage(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "   ")

	conversation, err := ConversationService.Create(welcomeTestExternalUser("blank-welcome-1"), 11, aiAgent.ID)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if conversation == nil {
		t.Fatalf("expected conversation")
	}

	var count int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", conversation.ID).Count(&count).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no welcome messages, got %d", count)
	}

	var updated models.Conversation
	if err := db.First(&updated, conversation.ID).Error; err != nil {
		t.Fatalf("find conversation: %v", err)
	}
	if updated.LastMessageID != 0 {
		t.Fatalf("expected last message id 0, got %d", updated.LastMessageID)
	}
	if updated.CustomerUnreadCount != 0 {
		t.Fatalf("expected customer unread count 0, got %d", updated.CustomerUnreadCount)
	}
}

func TestConversationCreateWelcomeMessageDoesNotTriggerAIReplyHook(t *testing.T) {
	db := setupMessageWelcomeTestDB(t)
	aiAgent := createWelcomeTestAIAgent(t, db, "欢迎咨询")

	previousHook := TriggerAIReplyAsyncHook
	called := false
	TriggerAIReplyAsyncHook = func(conversation models.Conversation, message models.Message) {
		called = true
	}
	t.Cleanup(func() {
		TriggerAIReplyAsyncHook = previousHook
	})

	if _, err := ConversationService.Create(welcomeTestExternalUser("hook-welcome-1"), 11, aiAgent.ID); err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if called {
		t.Fatalf("expected welcome message not to trigger ai reply hook")
	}
}
