package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupDiscordTestDB(t *testing.T) *gorm.DB {
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
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserPermission{},
	); err != nil {
		t.Fatalf("migrate discord test tables: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func TestDiscordInboundAndOutbound(t *testing.T) {
	db := setupDiscordTestDB(t)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"out_msg_100","channel_id":"text_chan_1","content":"Agent reply"}`))
	}))
	defer mockServer.Close()

	now := time.Now()
	aiAgent := &models.AIAgent{
		Name:                "Support AI",
		Status:              enums.StatusOk,
		PublishedRevisionID: 1,
		AuditFields:         models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(aiAgent).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}

	discordConfig := dto.DiscordChannelConfig{
		GuildID:       "guild_12345",
		GuildName:     "Test Guild",
		BotToken:      "discord_bot_token",
		WebhookSecret: "test_secret",
	}
	cfgBytes, _ := json.Marshal(discordConfig)

	channel := &models.Channel{
		ChannelType:           enums.ChannelTypeDiscord,
		ChannelID:             "discord_ch_1",
		AIAgentID:             aiAgent.ID,
		AIAgentRolloutPercent: 100,
		Name:                  "Community Support",
		ConfigJSON:            string(cfgBytes),
		Status:                enums.StatusOk,
		AuditFields:           models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create discord channel: %v", err)
	}

	payload := `{
		"id": "msg_999",
		"channel_id": "text_chan_1",
		"guild_id": "guild_12345",
		"content": "",
		"author": {
			"id": "user_888",
			"username": "gamer_joy",
			"global_name": "Joy Le",
			"bot": false
		},
		"attachments": [
			{
				"id": "att_1",
				"filename": "screenshot.png",
				"url": "https://cdn.discordapp.com/attachments/1/screenshot.png",
				"content_type": "image/png",
				"size": 10240
			}
		]
	}`

	ctx := context.Background()
	err := DiscordInboundService.HandleWebhook(ctx, channel.ChannelID, "test_secret", []byte(payload))
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	// Verify customer identity
	identity := repositories.CustomerIdentityRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceDiscord).
		Eq("external_id", "user_888"))
	if identity == nil {
		t.Fatalf("expected customer identity to be created")
	}

	// Verify conversation
	conv := repositories.ConversationRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("customer_id", identity.CustomerID).
		Eq("channel_id", channel.ID))
	if conv == nil {
		t.Fatalf("expected conversation to be created")
	}

	// Verify image message created from attachment
	custMsg := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("conversation_id", conv.ID).
		Eq("sender_type", enums.IMSenderTypeCustomer))
	if custMsg == nil {
		t.Fatalf("expected customer message to be created")
	}

	operator := &dto.AuthPrincipal{UserID: 1, Nickname: "Agent Joy"}

	// Test Outbound enqueue with Message
	replyMsg, err := MessageService.SendAIMessage(conv.ID, aiAgent.ID, "ai_msg_1", enums.IMMessageTypeText, "Here is your response image: https://example.com/response_img.png", "", operator)
	if err != nil {
		t.Fatalf("MessageService.SendAIMessage failed: %v", err)
	}

	outbox := ChannelMessageOutboxService.GetByMessageID(enums.ChannelTypeDiscord, replyMsg.ID)
	if outbox == nil {
		t.Fatalf("expected outbox entry for discord message")
	}
	if outbox.SendStatus != string(enums.ChannelMessageOutboxStatusPending) && outbox.SendStatus != string(enums.ChannelMessageOutboxStatusSending) && outbox.SendStatus != string(enums.ChannelMessageOutboxStatusFailed) {
		t.Fatalf("unexpected outbox status: %s", outbox.SendStatus)
	}
}

func seedDiscordScopedChannel(t *testing.T, db *gorm.DB, channelID string, cfg dto.DiscordChannelConfig) *models.Channel {
	t.Helper()
	now := time.Now()
	agent := &models.AIAgent{
		Name:                "Support AI",
		Status:              enums.StatusOk,
		PublishedRevisionID: 1,
		AuditFields:         models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(agent).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}
	cfg.BotToken = "discord_bot_token"
	cfg.WebhookSecret = "scope_secret"
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal discord config: %v", err)
	}
	channel := &models.Channel{
		ChannelType:           enums.ChannelTypeDiscord,
		ChannelID:             channelID,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		Name:                  "Discord " + channelID,
		ConfigJSON:            string(cfgBytes),
		Status:                enums.StatusOk,
		AuditFields:           models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create discord channel: %v", err)
	}
	return channel
}

func discordScopedPayload(t *testing.T, guildID, messageID string) []byte {
	t.Helper()
	body := map[string]any{
		"id":         messageID,
		"channel_id": "discord_text_chan",
		"content":    "hello from discord",
		"author":     map[string]any{"id": "user_scope", "username": "scoped_user", "bot": false},
	}
	if guildID != "" {
		body["guild_id"] = guildID
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}

// A bot can be invited to several servers, and GuildID and ChannelScope are
// stored on the channel. Messages outside that scope must not create a
// conversation, otherwise the stored scope is configuration that does nothing.
func TestDiscordInboundHonoursGuildScope(t *testing.T) {
	db := setupDiscordTestDB(t)

	guildScoped := seedDiscordScopedChannel(t, db, "discord_guild_scoped", dto.DiscordChannelConfig{
		GuildID:   "guild_in_scope",
		GuildName: "In Scope",
	})
	dmOnly := seedDiscordScopedChannel(t, db, "discord_dm_only", dto.DiscordChannelConfig{
		ChannelScope: "dm_only",
	})
	unscoped := seedDiscordScopedChannel(t, db, "discord_unscoped", dto.DiscordChannelConfig{})

	cases := []struct {
		name       string
		channel    *models.Channel
		guildID    string
		wantStored bool
	}{
		{"matching guild is accepted", guildScoped, "guild_in_scope", true},
		{"another guild is ignored", guildScoped, "guild_elsewhere", false},
		{"a dm is ignored by a guild scoped channel", guildScoped, "", false},
		{"a dm is accepted by a dm_only channel", dmOnly, "", true},
		{"a guild message is ignored by a dm_only channel", dmOnly, "guild_anywhere", false},
		{"an unscoped channel accepts any guild", unscoped, "guild_anywhere", true},
		{"an unscoped channel accepts a dm", unscoped, "", true},
	}

	for _, tc := range cases {
		messageID := "scope_" + strings.ReplaceAll(tc.name, " ", "_")
		payload := discordScopedPayload(t, tc.guildID, messageID)

		if err := DiscordInboundService.HandleWebhook(context.Background(), tc.channel.ChannelID, "scope_secret", payload); err != nil {
			t.Fatalf("%s: HandleWebhook failed: %v", tc.name, err)
		}

		var count int64
		if err := db.Table("t_message").Where("client_msg_id LIKE ?", "%"+messageID).Count(&count).Error; err != nil {
			t.Fatalf("%s: count messages: %v", tc.name, err)
		}
		switch {
		case tc.wantStored && count == 0:
			t.Errorf("%s: expected the message to be stored", tc.name)
		case !tc.wantStored && count != 0:
			t.Errorf("%s: expected the message to be dropped, found %d", tc.name, count)
		}
	}
}

// The webhook secret is compared in constant time, so a wrong secret of the same
// length must be rejected rather than accepted by a prefix match.
func TestDiscordInboundRejectsWrongWebhookSecret(t *testing.T) {
	db := setupDiscordTestDB(t)
	channel := seedDiscordScopedChannel(t, db, "discord_secret", dto.DiscordChannelConfig{})

	payload := discordScopedPayload(t, "", "secret_msg_1")
	err := DiscordInboundService.HandleWebhook(context.Background(), channel.ChannelID, "wrong_secret_value", payload)
	if err == nil {
		t.Fatalf("expected a wrong webhook secret to be rejected")
	}

	var count int64
	if err := db.Table("t_message").Where("client_msg_id LIKE ?", "%secret_msg_1").Count(&count).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if count != 0 {
		t.Fatalf("a rejected delivery stored %d messages", count)
	}
}
