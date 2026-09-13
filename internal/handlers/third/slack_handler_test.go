package third

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

const slackHandlerTestSigningSecret = "test_signing_secret"

// signSlackHandlerPayload builds the X-Slack-Request-Timestamp and
// X-Slack-Signature headers Slack sends for this body right now.
func signSlackHandlerPayload(payload []byte) (string, string) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(slackHandlerTestSigningSecret))
	mac.Write([]byte("v0:" + timestamp + ":" + string(payload)))
	return timestamp, "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestSlackWebhook_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupThirdHandlerTestDB(t)

	now := time.Now()
	agent := &models.AIAgent{
		Name:                "Slack Agent",
		ServiceMode:         enums.IMConversationServiceModeAIFirst,
		PublishedRevisionID: 1,
		WelcomeMessage:      "Hello Slack User!",
		Status:              enums.StatusOk,
		AuditFields:         models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(agent).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}

	slackConfig, err := json.Marshal(dto.SlackChannelConfig{
		BotToken:       "xoxb-test-token",
		SigningSecret:  slackHandlerTestSigningSecret,
		TeamID:         "T_SLACK_100",
		DefaultChannel: "C_GENERAL",
	})
	if err != nil {
		t.Fatalf("marshal slack config: %v", err)
	}

	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	channel, err := services.ChannelService.CreateChannel(request.CreateChannelRequest{
		Name:                  "Slack Channel",
		ChannelType:           enums.ChannelTypeSlack,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		ConfigJSON:            string(slackConfig),
		Status:                int(enums.StatusOk),
	}, operator)
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	router := gin.New()
	router.POST("/api/third/slack/webhook/:channel_id", SlackPostWebhook)
	router.POST("/api/third/slack/webhook", SlackPostWebhook)

	post := func(path string, payload []byte, timestamp, signature string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		if timestamp != "" {
			req.Header.Set("X-Slack-Request-Timestamp", timestamp)
		}
		if signature != "" {
			req.Header.Set("X-Slack-Signature", signature)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	webhookPath := "/api/third/slack/webhook/" + channel.ChannelID

	// 1. The url_verification handshake is answered without a signature, because
	//    Slack sends it once while the endpoint is being configured.
	challengePayload := []byte(`{
		"token": "token123",
		"challenge": "slack_challenge_string_999",
		"type": "url_verification"
	}`)
	recChallenge := post(webhookPath, challengePayload, "", "")
	if recChallenge.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for challenge, got: %d", recChallenge.Code)
	}
	var challengeResp map[string]any
	if err := json.Unmarshal(recChallenge.Body.Bytes(), &challengeResp); err != nil {
		t.Fatalf("unmarshal challenge response: %v", err)
	}
	if challengeResp["challenge"] != "slack_challenge_string_999" {
		t.Fatalf("expected challenge in body, got: %+v", challengeResp)
	}

	// 2. An event the channel's signing secret did not produce is rejected, and
	//    must not create a customer identity.
	eventPayload := []byte(`{
		"token": "token123",
		"team_id": "T_SLACK_100",
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U_USER_777",
			"text": "Hello support team on Slack!",
			"ts": "1725260000.000100",
			"channel": "C_GENERAL"
		}
	}`)
	recUnsigned := post(webhookPath, eventPayload, "", "")
	if recUnsigned.Code != http.StatusOK {
		t.Fatalf("expected the handler to answer 200 with an error body, got: %d", recUnsigned.Code)
	}
	if repositories.CustomerIdentityRepository.FindOne(db, sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceSlack).
		Eq("external_id", "U_USER_777")) != nil {
		t.Fatalf("an unsigned event created a customer identity")
	}

	// 3. A correctly signed event is accepted and resolves the sender.
	timestamp, signature := signSlackHandlerPayload(eventPayload)
	recEvent := post(webhookPath, eventPayload, timestamp, signature)
	if recEvent.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for event, got: %d", recEvent.Code)
	}

	identity := repositories.CustomerIdentityRepository.FindOne(db, sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceSlack).
		Eq("external_id", "U_USER_777"))
	if identity == nil {
		t.Fatalf("expected customer identity for U_USER_777")
	}
}
