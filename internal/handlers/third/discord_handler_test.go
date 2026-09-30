package third

import (
	"bytes"
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
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/sqls"
)

func TestDiscordPostWebhook_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupThirdHandlerTestDB(t)

	now := time.Now()
	agent := &models.AIAgent{
		Name:                "Discord Agent",
		ServiceMode:         enums.IMConversationServiceModeAIFirst,
		PublishedRevisionID: 1,
		WelcomeMessage:      "Hello Discord!",
		Status:              enums.StatusOk,
		AuditFields:         models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	_ = db.Create(agent)

	discordConfig, _ := json.Marshal(dto.DiscordChannelConfig{
		GuildID:        "guild_999",
		BotToken:       "test_bot_token",
		WebhookSecret:  "secret_discord_123",
		WelcomeMessage: "Welcome!",
	})

	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	channel, err := services.ChannelService.CreateChannel(request.CreateChannelRequest{
		Name:                  "Discord Community",
		ChannelType:           enums.ChannelTypeDiscord,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		ConfigJSON:            string(discordConfig),
		Status:                int(enums.StatusOk),
	}, operator)
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	router := gin.New()
	router.POST("/api/third/discord/webhook/:channel_id", DiscordPostWebhook)
	router.POST("/api/third/discord/webhook", DiscordPostWebhook)

	payload := []byte(`{
		"id": "msg_001",
		"channel_id": "ch_777",
		"guild_id": "guild_999",
		"content": "Need help with setup",
		"author": {
			"id": "user_456",
			"username": "gamer_one",
			"global_name": "Gamer One",
			"bot": false
		}
	}`)

	// 1. Invalid secret
	req, _ := http.NewRequest(http.MethodPost, "/api/third/discord/webhook/"+channel.ChannelID, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Discord-Secret-Token", "wrong_secret")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK wrapper, got: %d", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["ok"] == true {
		t.Fatalf("expected error for invalid secret token")
	}

	// 2. Valid secret
	req2, _ := http.NewRequest(http.MethodPost, "/api/third/discord/webhook/"+channel.ChannelID, bytes.NewBuffer(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Discord-Secret-Token", "secret_discord_123")

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec2.Code)
	}
	var resp2 map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2["ok"] != true {
		t.Fatalf("expected ok: true, got: %+v", resp2)
	}

	// Verify identity in DB
	identity := repositories.CustomerIdentityRepository.FindOne(db, sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceDiscord).
		Eq("external_id", "user_456"))
	if identity == nil {
		t.Fatalf("expected customer identity for user_456")
	}
}
