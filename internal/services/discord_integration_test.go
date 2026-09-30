package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupDiscordIntegrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Channel{},
		&models.ChannelMessageOutbox{},
		&models.Customer{},
		&models.CustomerIdentity{},
		&models.CustomerContact{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.ConversationReadState{},
		&models.ConversationInterrupt{},
		&models.ConversationEventLog{},
		&models.Message{},
		&models.AIAgent{},
		&models.AgentProfile{},
		&models.AgentTeam{},
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserPermission{},
	); err != nil {
		t.Fatalf("migrate discord integration test tables: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func TestDiscordIntegrationFullFlow(t *testing.T) {
	db := setupDiscordIntegrationTestDB(t)

	mockDiscordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"discord_msg_reply_999","channel_id":"ch_discord_general","content":"Cảm ơn bạn! Đội ngũ hỗ trợ sẽ kiểm tra ngay."}`))
	}))
	defer mockDiscordServer.Close()

	now := time.Now()
	// 1. Create AI Agent
	agent := &models.AIAgent{
		Name:                "Discord Support AI",
		ServiceMode:         enums.IMConversationServiceModeAIFirst,
		PublishedRevisionID: 1,
		WelcomeMessage:      "Chào mừng đến với máy chủ Discord Crove Desk!",
		Status:              enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	_ = db.Create(agent)

	// 2. Create Discord Channel
	discordConfig, _ := json.Marshal(dto.DiscordChannelConfig{
		GuildID:        "guild_987654321",
		GuildName:      "Crove Community Discord",
		BotToken:       "test-discord-bot-token-xyz",
		WebhookSecret:  "discord-secret-token-123",
		WelcomeMessage: "Welcome to Discord Support!",
	})

	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	channel, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		Name:                  "Crove Discord Support",
		ChannelType:           enums.ChannelTypeDiscord,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		ConfigJSON:            string(discordConfig),
		Status:                int(enums.StatusOk),
	}, operator)
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	// 3. Simulate Inbound Discord Webhook / Gateway message from user
	inboundPayload := []byte(`{
		"id": "msg_discord_user_001",
		"channel_id": "ch_discord_general",
		"guild_id": "guild_987654321",
		"content": "Tôi muốn hỏi về cách cấu hình Custom Domain cho Email Channel trên Crove Desk",
		"author": {
			"id": "discord_uid_555",
			"username": "gamer_joy",
			"global_name": "Anh Le",
			"bot": false
		}
	}`)

	ctx := context.Background()
	err = DiscordInboundService.HandleWebhook(ctx, channel.ChannelID, "discord-secret-token-123", inboundPayload)
	if err != nil {
		t.Fatalf("DiscordInboundService.HandleWebhook failed: %v", err)
	}

	// Verify Customer Identity
	identity := repositories.CustomerIdentityRepository.FindOne(db, sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceDiscord).
		Eq("external_id", "discord_uid_555"))
	if identity == nil {
		t.Fatalf("expected customer identity for discord_uid_555")
	}

	customer := repositories.CustomerRepository.Get(db, identity.CustomerID)
	if customer == nil || customer.Name != "Anh Le" {
		t.Fatalf("unexpected customer profile: %+v", customer)
	}

	// Verify Conversation created
	conv := repositories.ConversationRepository.FindOne(db, sqls.NewCnd().Eq("customer_id", customer.ID))
	if conv == nil || conv.ChannelID != channel.ID {
		t.Fatalf("unexpected conversation: %+v", conv)
	}

	// Verify Customer Message stored
	msg := repositories.MessageRepository.FindOne(db, sqls.NewCnd().
		Eq("conversation_id", conv.ID).
		Eq("sender_type", enums.IMSenderTypeCustomer))
	if msg == nil || msg.Content != "Tôi muốn hỏi về cách cấu hình Custom Domain cho Email Channel trên Crove Desk" {
		t.Fatalf("unexpected stored customer message: %+v", msg)
	}

	// 4. Simulate Agent / AI Reply and test Outbox Enqueue & Outbound Dispatch
	replyMsg, err := MessageService.SendAIMessage(conv.ID, agent.ID, "ai_reply_001", enums.IMMessageTypeText, "Cảm ơn bạn! Đội ngũ hỗ trợ sẽ kiểm tra ngay.", "", operator)
	if err != nil {
		t.Fatalf("MessageService.SendAIMessage failed: %v", err)
	}

	outbox := ChannelMessageOutboxService.GetByMessageID(enums.ChannelTypeDiscord, replyMsg.ID)
	if outbox == nil {
		t.Fatalf("expected discord outbox entry for AI message")
	}
	if outbox.ChannelType != enums.ChannelTypeDiscord {
		t.Fatalf("expected outbox channel type 'discord', got '%s'", outbox.ChannelType)
	}
}
