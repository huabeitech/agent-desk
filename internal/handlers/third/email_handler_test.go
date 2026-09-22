package third

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestEmailPostWebhook_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhook", EmailPostWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString("invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestEmailPostWebhook_FullFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupThirdHandlerTestDB(t)

	now := time.Now()
	agent := &models.AIAgent{
		Name:                "Email Support Agent",
		ServiceMode:         enums.IMConversationServiceModeAIFirst,
		PublishedRevisionID: 1,
		WelcomeMessage:      "Thanks for emailing support.",
		Status:              enums.StatusOk,
		AuditFields:         models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	_ = db.Create(agent)

	emailConfig, _ := json.Marshal(dto.EmailChannelConfig{
		EmailAddress:   "help@crove.com",
		SenderName:     "Crove Desk Support",
		Provider:       "brevo",
		WebhookSecret:  "email_secret_token_123",
		WelcomeMessage: "We have received your email.",
	})

	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	channel, err := services.ChannelService.CreateChannel(request.CreateChannelRequest{
		Name:                  "Email Support Channel",
		ChannelType:           enums.ChannelTypeEmail,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		ConfigJSON:            string(emailConfig),
		Status:                int(enums.StatusOk),
	}, operator)
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	router := gin.New()
	router.POST("/api/third/email/webhook/:channel_id", EmailPostWebhook)
	router.POST("/api/third/email/webhook", EmailPostWebhook)

	// 1. Test unauthorized when secret doesn't match
	genericPayload := []byte(`{
		"from": "alice@customer.com",
		"from_name": "Alice Customer",
		"to": "help@crove.com",
		"subject": "Need help with Crove Desk",
		"text": "Hello, I have an issue with my login credentials.",
		"message_id": "<msg-001@mail.customer.com>"
	}`)

	req, _ := http.NewRequest(http.MethodPost, "/api/third/email/webhook/"+channel.ChannelID, bytes.NewBuffer(genericPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", "wrong_secret")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK wrapper, got: %d", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["ok"] != false {
		t.Fatalf("expected ok: false on wrong secret token, got: %v", resp)
	}

	// 2. Test successful processing with Generic / Cloudflare payload
	req2, _ := http.NewRequest(http.MethodPost, "/api/third/email/webhook/"+channel.ChannelID, bytes.NewBuffer(genericPayload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Webhook-Secret", "email_secret_token_123")

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	var resp2 map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2["ok"] != true {
		t.Fatalf("expected ok: true, got: %v", resp2)
	}

	// Verify Customer was created with Email source
	identity := repositories.CustomerIdentityRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceEmail).
		Eq("external_id", "alice@customer.com"))
	if identity == nil {
		t.Fatalf("expected customer identity for alice@customer.com to be created")
	}

	customer := repositories.CustomerRepository.Get(sqls.DB(), identity.CustomerID)
	if customer == nil || customer.PrimaryEmail != "alice@customer.com" {
		t.Fatalf("expected customer primary_email 'alice@customer.com', got %+v", customer)
	}

	// Verify Conversation was created
	conversations := repositories.ConversationRepository.Find(sqls.DB(), sqls.NewCnd().
		Eq("customer_id", customer.ID).
		Eq("channel_id", channel.ID))
	if len(conversations) == 0 {
		t.Fatalf("expected conversation to be created")
	}
	convID := conversations[0].ID

	// Verify Message was saved
	messages := repositories.MessageRepository.Find(sqls.DB(), sqls.NewCnd().
		Eq("conversation_id", convID))
	if len(messages) == 0 {
		t.Fatalf("expected message to be stored")
	}

	// 3. Test Threading: Send reply with ticket ID in subject
	threadedPayload := []byte(strings.ReplaceAll(`{
		"from": "alice@customer.com",
		"to": "help@crove.com",
		"subject": "Re: [#CONV_ID] Need help with Crove Desk",
		"text": "Thanks! Here is additional information.",
		"message_id": "<msg-002@mail.customer.com>",
		"in_reply_to": "<msg-001@mail.customer.com>"
	}`, "CONV_ID", string(rune('0'+convID))))

	reqThread, _ := http.NewRequest(http.MethodPost, "/api/third/email/webhook", bytes.NewBuffer(threadedPayload))
	reqThread.Header.Set("Content-Type", "application/json")
	reqThread.Header.Set("X-Webhook-Secret", "email_secret_token_123")

	recThread := httptest.NewRecorder()
	router.ServeHTTP(recThread, reqThread)

	var respThread map[string]any
	_ = json.Unmarshal(recThread.Body.Bytes(), &respThread)
	if respThread["ok"] != true {
		t.Fatalf("expected threaded ok: true, got: %v", respThread)
	}

	// Verify message was attached to existing conversation rather than creating a duplicate
	convCount := len(repositories.ConversationRepository.Find(sqls.DB(), sqls.NewCnd().Eq("customer_id", customer.ID)))
	if convCount != 1 {
		t.Fatalf("expected conversation count to remain 1, got %d", convCount)
	}

	// 4. Test Postmark Inbound Webhook format
	postmarkPayload := []byte(`{
		"From": "developer@company.org",
		"FromName": "Dev Team",
		"To": "help@crove.com",
		"Subject": "API Inquiry",
		"TextBody": "How do I call MCP tools?",
		"MessageID": "postmark-uuid-001"
	}`)

	reqPM, _ := http.NewRequest(http.MethodPost, "/api/third/email/webhook", bytes.NewBuffer(postmarkPayload))
	reqPM.Header.Set("Content-Type", "application/json")
	reqPM.Header.Set("X-Webhook-Secret", "email_secret_token_123")

	recPM := httptest.NewRecorder()
	router.ServeHTTP(recPM, reqPM)

	var respPM map[string]any
	_ = json.Unmarshal(recPM.Body.Bytes(), &respPM)
	if respPM["ok"] != true {
		t.Fatalf("expected postmark format ok: true, got: %v", respPM)
	}

	// 5. Test Mailgun Form Data Webhook format
	mgForm := url.Values{}
	mgForm.Set("sender", "mailgunner@test.com")
	mgForm.Set("from", "Mailgun User <mailgunner@test.com>")
	mgForm.Set("recipient", "help@crove.com")
	mgForm.Set("subject", "Mailgun Inbound Ticket")
	mgForm.Set("body-plain", "Testing mailgun webhook support.")
	mgForm.Set("Message-Id", "<mg-998877@mailgun.org>")

	reqMG, _ := http.NewRequest(http.MethodPost, "/api/third/email/webhook", strings.NewReader(mgForm.Encode()))
	reqMG.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqMG.Header.Set("X-Webhook-Secret", "email_secret_token_123")

	recMG := httptest.NewRecorder()
	router.ServeHTTP(recMG, reqMG)

	var respMG map[string]any
	_ = json.Unmarshal(recMG.Body.Bytes(), &respMG)
	if respMG["ok"] != true {
		t.Fatalf("expected mailgun format ok: true, got: %v", respMG)
	}

	mgIdentity := repositories.CustomerIdentityRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceEmail).
		Eq("external_id", "mailgunner@test.com"))
	if mgIdentity == nil {
		t.Fatalf("expected customer identity for mailgunner@test.com")
	}
}
