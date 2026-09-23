package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// slackTestSigningSecret is the signing secret the test channel is configured with.
const slackTestSigningSecret = "test_signing_secret_999"

// signSlackPayload builds the X-Slack-Request-Timestamp and X-Slack-Signature
// headers Slack would send for this body right now.
func signSlackPayload(t *testing.T, secret string, payload []byte) (string, string) {
	t.Helper()
	return signSlackPayloadAt(t, secret, payload, time.Now())
}

func signSlackPayloadAt(t *testing.T, secret string, payload []byte, at time.Time) (string, string) {
	t.Helper()
	timestamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v0:" + timestamp + ":" + string(payload)))
	return timestamp, "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func setupSlackTestDB(t *testing.T) *gorm.DB {
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
		t.Fatalf("migrate slack test tables: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func TestSlackInboundAndOutbound(t *testing.T) {
	db := setupSlackTestDB(t)

	now := time.Now()
	aiAgent := &models.AIAgent{
		Name:                "Slack Bot Agent",
		Status:              enums.StatusOk,
		PublishedRevisionID: 1,
		AuditFields:         models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(aiAgent).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}

	slackConfig := dto.SlackChannelConfig{
		BotToken:       "xoxb-test-bot-token-12345",
		SigningSecret:  slackTestSigningSecret,
		TeamID:         "T0123456789",
		TeamName:       "Acme Corp",
		DefaultChannel: "C9876543210",
	}
	cfgBytes, _ := json.Marshal(slackConfig)

	channel := &models.Channel{
		ChannelType:           enums.ChannelTypeSlack,
		ChannelID:             "T0123456789",
		AIAgentID:             aiAgent.ID,
		AIAgentRolloutPercent: 100,
		Name:                  "Slack Support Channel",
		ConfigJSON:            string(cfgBytes),
		Status:                enums.StatusOk,
		AuditFields:           models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create slack channel: %v", err)
	}

	payload := `{
		"token": "verification_token",
		"team_id": "T0123456789",
		"api_app_id": "A01234567",
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U12345678",
			"text": "Help with API key generation",
			"ts": "1725260000.000200",
			"channel": "C9876543210",
			"channel_type": "channel"
		}
	}`

	ctx := context.Background()
	timestamp, signature := signSlackPayload(t, slackTestSigningSecret, []byte(payload))
	_, err := SlackInboundService.HandleWebhook(ctx, "", timestamp, signature, []byte(payload))
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	// Verify customer identity
	identity := repositories.CustomerIdentityRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceSlack).
		Eq("external_id", "U12345678"))
	if identity == nil {
		t.Fatalf("expected customer identity for U12345678")
	}

	// Verify conversation
	conv := repositories.ConversationRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("customer_id", identity.CustomerID).
		Eq("channel_id", channel.ID))
	if conv == nil {
		t.Fatalf("expected conversation to be created")
	}

	// Verify message
	msg := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("conversation_id", conv.ID).
		Eq("sender_type", enums.IMSenderTypeCustomer))
	if msg == nil {
		t.Fatalf("expected message to be created")
	}
	if msg.Content != "Help with API key generation" {
		t.Fatalf("expected message content 'Help with API key generation', got %s", msg.Content)
	}

	operator := &dto.AuthPrincipal{UserID: 1, Nickname: "Agent Joy"}

	// Test Outbound enqueue
	replyMsg, err := MessageService.SendAIMessage(conv.ID, aiAgent.ID, "ai_slack_reply_1", enums.IMMessageTypeText, "You can generate your API key under Settings > API Keys.", "", operator)
	if err != nil {
		t.Fatalf("MessageService.SendAIMessage failed: %v", err)
	}

	outbox := ChannelMessageOutboxService.GetByMessageID(enums.ChannelTypeSlack, replyMsg.ID)
	if outbox == nil {
		t.Fatalf("expected outbox entry for slack message")
	}
	if outbox.ChannelType != enums.ChannelTypeSlack {
		t.Fatalf("expected outbox channel type 'slack', got %s", outbox.ChannelType)
	}
}

func seedSlackChannel(t *testing.T, db *gorm.DB, signingSecret string) *models.Channel {
	t.Helper()
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
	cfgBytes, err := json.Marshal(dto.SlackChannelConfig{
		BotToken:      "xoxb-test-bot-token-12345",
		SigningSecret: signingSecret,
		TeamID:        "T0123456789",
	})
	if err != nil {
		t.Fatalf("marshal slack config: %v", err)
	}
	channel := &models.Channel{
		ChannelType:           enums.ChannelTypeSlack,
		ChannelID:             "slack_sig_channel",
		AIAgentID:             aiAgent.ID,
		AIAgentRolloutPercent: 100,
		Name:                  "Slack Support",
		ConfigJSON:            string(cfgBytes),
		Status:                enums.StatusOk,
		AuditFields:           models.AuditFields{CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create slack channel: %v", err)
	}
	return channel
}

func slackEventPayload(messageID string) []byte {
	return []byte(`{
		"team_id": "T0123456789",
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U_SIG_TEST",
			"text": "signature probe",
			"ts": "` + messageID + `",
			"channel": "C9876543210",
			"channel_type": "channel"
		}
	}`)
}

func countSlackMessages(t *testing.T, db *gorm.DB, messageTS string) int64 {
	t.Helper()
	var count int64
	if err := db.Table("t_message").Where("client_msg_id = ?", "slack_C9876543210_"+messageTS).Count(&count).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	return count
}

// A channel with a signing secret configured must reject anything Slack did not
// sign. Accepting an unsigned delivery would let anyone who learns the webhook
// URL write into a workspace conversation and trigger paid AI replies.
func TestSlackInboundRejectsUnauthenticatedDelivery(t *testing.T) {
	db := setupSlackTestDB(t)
	channel := seedSlackChannel(t, db, slackTestSigningSecret)

	cases := []struct {
		name      string
		messageTS string
		timestamp string
		signature string
	}{
		{"no signature headers at all", "1725260000.000001", "", ""},
		{"signature without a timestamp", "1725260000.000002", "", "v0=deadbeef"},
		{"timestamp without a signature", "1725260000.000003", "1725260000", ""},
		{"wrong signature", "1725260000.000004", "1725260000", "v0=deadbeef"},
		{"non-numeric timestamp", "1725260000.000005", "not-a-timestamp", "v0=deadbeef"},
	}

	for _, tc := range cases {
		payload := slackEventPayload(tc.messageTS)
		_, err := SlackInboundService.HandleWebhook(context.Background(), channel.ChannelID, tc.timestamp, tc.signature, payload)
		if err == nil {
			t.Errorf("%s: expected the delivery to be rejected", tc.name)
			continue
		}
		if countSlackMessages(t, db, tc.messageTS) != 0 {
			t.Errorf("%s: a rejected delivery stored a message", tc.name)
		}
	}
}

// Slack signs the timestamp and the body but nothing in the signature expires, so
// a captured request replays forever unless the timestamp is checked. Slack's own
// guide requires rejecting anything older than five minutes.
func TestSlackInboundRejectsReplayedTimestamp(t *testing.T) {
	db := setupSlackTestDB(t)
	channel := seedSlackChannel(t, db, slackTestSigningSecret)

	cases := []struct {
		name string
		age  time.Duration
	}{
		{"ten minutes old", 10 * time.Minute},
		{"one hour old", time.Hour},
		{"ten minutes in the future", -10 * time.Minute},
	}

	for i, tc := range cases {
		messageTS := "1725260000.0000" + strconv.Itoa(10+i)
		payload := slackEventPayload(messageTS)
		// Correctly signed for its own timestamp, which is exactly what a replayed
		// capture looks like on the wire.
		timestamp, signature := signSlackPayloadAt(t, slackTestSigningSecret, payload, time.Now().Add(-tc.age))

		if _, err := SlackInboundService.HandleWebhook(context.Background(), channel.ChannelID, timestamp, signature, payload); err == nil {
			t.Errorf("%s: expected a stale timestamp to be rejected", tc.name)
		}
		if countSlackMessages(t, db, messageTS) != 0 {
			t.Errorf("%s: a replayed delivery stored a message", tc.name)
		}
	}
}

// A correctly signed, fresh delivery is still accepted, and one signed just
// inside the tolerance window is not rejected for clock drift.
func TestSlackInboundAcceptsFreshValidSignature(t *testing.T) {
	db := setupSlackTestDB(t)
	channel := seedSlackChannel(t, db, slackTestSigningSecret)

	cases := []struct {
		name string
		age  time.Duration
	}{
		{"signed now", 0},
		{"signed four minutes ago", 4 * time.Minute},
	}

	for i, tc := range cases {
		messageTS := "1725260000.0000" + strconv.Itoa(20+i)
		payload := slackEventPayload(messageTS)
		timestamp, signature := signSlackPayloadAt(t, slackTestSigningSecret, payload, time.Now().Add(-tc.age))

		if _, err := SlackInboundService.HandleWebhook(context.Background(), channel.ChannelID, timestamp, signature, payload); err != nil {
			t.Fatalf("%s: HandleWebhook failed: %v", tc.name, err)
		}
		if countSlackMessages(t, db, messageTS) != 1 {
			t.Errorf("%s: expected the signed message to be stored", tc.name)
		}
	}
}

// The url_verification handshake has to be answered before any channel lookup or
// signature check, because Slack sends it once while the endpoint is being
// configured and will not retry.
func TestSlackInboundAnswersURLVerificationChallenge(t *testing.T) {
	setupSlackTestDB(t)

	payload := []byte(`{"type":"url_verification","challenge":"challenge_token_abc","token":"verification_token"}`)
	challenge, err := SlackInboundService.HandleWebhook(context.Background(), "", "", "", payload)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if challenge == nil || *challenge != "challenge_token_abc" {
		t.Fatalf("challenge = %v, want challenge_token_abc", challenge)
	}
}

// A channel without its own signing secret inherits the deployment-wide one,
// so a single shared Slack app can serve every channel the desk supports.
func TestSlackInboundFallsBackToDeploymentSigningSecret(t *testing.T) {
	db := setupSlackTestDB(t)
	channel := seedSlackChannel(t, db, "")
	config.SetCurrent(&config.Config{Slack: config.SlackConfig{SigningSecret: slackTestSigningSecret}})
	defer config.SetCurrent(nil)

	payload := slackEventPayload("1725260000.000300")
	timestamp, signature := signSlackPayload(t, slackTestSigningSecret, payload)
	if _, err := SlackInboundService.HandleWebhook(context.Background(), channel.ChannelID, timestamp, signature, payload); err != nil {
		t.Fatalf("expected the deployment-secret-signed delivery to be accepted: %v", err)
	}
	if countSlackMessages(t, db, "1725260000.000300") != 1 {
		t.Fatal("expected the delivery to be stored")
	}
}
