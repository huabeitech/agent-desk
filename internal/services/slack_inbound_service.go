package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/pkg/config"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/slack"
)

var SlackInboundService = newSlackInboundService()

func newSlackInboundService() *slackInboundService {
	return &slackInboundService{}
}

type slackInboundService struct{}

// HandleWebhook processes an incoming Events API event from Slack.
func (s *slackInboundService) HandleWebhook(ctx context.Context, channelID string, timestampHeader, signatureHeader string, rawPayload []byte) (*string, error) {
	var event slack.EventCallback
	if err := json.Unmarshal(rawPayload, &event); err != nil {
		return nil, fmt.Errorf("unmarshal slack event failed: %w", err)
	}

	// 1. URL Verification Challenge
	if event.Type == "url_verification" {
		return &event.Challenge, nil
	}

	if event.Type != "event_callback" || event.Event == nil {
		return nil, nil // Ignore non-message callbacks
	}

	teamID := strings.TrimSpace(event.TeamID)
	ev := event.Event

	if ev.BotID != "" || ev.Subtype == "bot_message" || strings.TrimSpace(ev.User) == "" {
		return nil, nil // Ignore bot loops
	}

	text := strings.TrimSpace(ev.Text)
	if text == "" {
		return nil, nil
	}

	var channel *models.Channel
	channelID = strings.TrimSpace(channelID)
	if channelID != "" {
		channel = ChannelService.Take("channel_id = ? AND channel_type = ? AND status = ?", channelID, enums.ChannelTypeSlack, enums.StatusOk)
	}
	if channel == nil && teamID != "" {
		channel = ChannelService.Take("channel_type = ? AND status = ? AND (channel_id = ? OR config_json LIKE ?)",
			enums.ChannelTypeSlack, enums.StatusOk, teamID, "%"+teamID+"%")
	}
	if channel == nil {
		channel = ChannelService.Take("channel_type = ? AND status = ?", enums.ChannelTypeSlack, enums.StatusOk)
	}
	if channel == nil {
		return nil, errorsx.InvalidParam("slack channel not found or disabled")
	}

	cfg, err := ChannelService.ParseSlackChannelConfig(channel.ConfigJSON)
	if err != nil || cfg == nil {
		return nil, errorsx.InvalidParam("slack channel config invalid")
	}

	// Verify the Slack signing secret whenever one resolves for this channel. A
	// delivery with no signature headers is rejected rather than waved through:
	// Slack always signs once a signing secret exists, so a missing header means
	// the sender is not Slack.
	slackCfg := config.ResolveSlack(cfg.BotToken, cfg.SigningSecret)
	if slackCfg.SigningSecret != "" {
		if strings.TrimSpace(signatureHeader) == "" || strings.TrimSpace(timestampHeader) == "" {
			slog.Warn("slack webhook rejected: missing signature headers",
				"channel_id", channelID, "secret_source", slackSecretSource(cfg.SigningSecret, slackCfg))
			return nil, errorsx.UnauthorizedI18n("error.auth.invalidSignature")
		}
		if !verifySlackSignature(slackCfg.SigningSecret, timestampHeader, signatureHeader, rawPayload) {
			slog.Warn("slack webhook rejected: signature verification failed",
				"channel_id", channelID, "secret_source", slackSecretSource(cfg.SigningSecret, slackCfg))
			return nil, errorsx.UnauthorizedI18n("error.auth.invalidSignature")
		}
	}

	// 1. Resolve customer identity
	senderID := strings.TrimSpace(ev.User)
	externalUser := openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceSlack,
		ExternalID:     senderID,
		ExternalName:   fmt.Sprintf("Slack User %s", senderID),
	}

	// 2. Create or match Conversation
	conversation, err := ConversationService.Create(externalUser, channel.ID, channel.AIAgentID)
	if err != nil {
		return nil, fmt.Errorf("create slack conversation failed: %w", err)
	}

	// 3. Send message through MessageService
	msgTS := strings.TrimSpace(ev.TS)
	threadTS := strings.TrimSpace(ev.ThreadTS)
	if threadTS == "" {
		threadTS = msgTS
	}
	clientMsgID := fmt.Sprintf("slack_%s_%s", ev.Channel, msgTS)
	payloadMap := map[string]any{
		"slack_channel":   ev.Channel,
		"slack_ts":        msgTS,
		"slack_thread_ts": threadTS,
		"slack_user":      senderID,
		"slack_team":      teamID,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	_, err = MessageService.SendCustomerMessage(
		conversation.ID,
		clientMsgID,
		enums.IMMessageTypeText,
		text,
		string(payloadBytes),
		externalUser,
	)
	if err != nil {
		return nil, fmt.Errorf("send customer message failed: %w", err)
	}

	return nil, nil
}

// slackSecretSource names where the verifying secret came from. A channel
// bound to a different Slack app than the deployment-wide one starts failing
// the moment the fallback secret appears, and this log field is the only way
// an operator can tell that apart from a spoofed delivery.
func slackSecretSource(channelSigningSecret string, resolved config.SlackConfig) string {
	if strings.TrimSpace(channelSigningSecret) != "" {
		return "channel"
	}
	if resolved.SigningSecret != "" {
		return "deployment_fallback"
	}
	return "none"
}

// slackTimestampTolerance is how far a request timestamp may drift from now.
//
// Slack's own verification guide requires rejecting anything older than five
// minutes. Without the check a captured request replays indefinitely: the
// signature covers the timestamp and the body, but nothing in it expires.
const slackTimestampTolerance = 5 * time.Minute

func verifySlackSignature(signingSecret, timestampHeader, signatureHeader string, payload []byte) bool {
	timestampHeader = strings.TrimSpace(timestampHeader)
	timestamp, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil || timestamp <= 0 {
		return false
	}
	if drift := time.Since(time.Unix(timestamp, 0)); drift > slackTimestampTolerance || drift < -slackTimestampTolerance {
		return false
	}

	sigBasestring := fmt.Sprintf("v0:%s:%s", timestampHeader, string(payload))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBasestring))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.TrimSpace(signatureHeader)), []byte(expectedSig))
}
