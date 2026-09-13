package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"agent-desk/internal/services/storage"
	"agent-desk/internal/slack"

	"github.com/mlogclub/simple/sqls"
)

const (
	slackOutboxBatchSize = 20
	slackOutboxMaxRetry  = 5
)

var SlackOutboundService = newSlackOutboundService()

func newSlackOutboundService() *slackOutboundService {
	return &slackOutboundService{}
}

type slackOutboundService struct{}

func (s *slackOutboundService) DispatchPendingOutbox() int {
	return s.doDispatchPendingOutbox(slackOutboxBatchSize)
}

func (s *slackOutboundService) doDispatchPendingOutbox(limit int) int {
	if limit <= 0 {
		limit = slackOutboxBatchSize
	}
	items := ChannelMessageOutboxService.ListPending(enums.ChannelTypeSlack, limit)
	if len(items) == 0 {
		return 0
	}

	successCount := 0
	for i := range items {
		if err := s.processOutbox(items[i].ID); err != nil {
			slog.Warn("process slack outbox failed",
				"outbox_id", items[i].ID,
				"error", err,
			)
			continue
		}
		successCount++
	}
	return successCount
}

func (s *slackOutboundService) processOutbox(outboxID int64) error {
	outbox := ChannelMessageOutboxService.Get(outboxID)
	if outbox == nil {
		return nil
	}
	if outbox.ChannelType != enums.ChannelTypeSlack {
		return nil
	}
	if outbox.SendStatus == string(enums.ChannelMessageOutboxStatusSent) {
		return nil
	}
	if outbox.NextRetryAt != nil && outbox.NextRetryAt.After(time.Now()) {
		return nil
	}

	if err := ChannelMessageOutboxService.Updates(outbox.ID, map[string]any{
		"send_status": string(enums.ChannelMessageOutboxStatusSending),
		"updated_at":  time.Now(),
	}); err != nil {
		return err
	}

	message := MessageService.Get(outbox.MessageID)
	if message == nil {
		return s.markOutboxFailed(outbox, "message not found")
	}
	conversation := ConversationService.Get(outbox.ConversationID)
	if conversation == nil {
		return s.markOutboxFailed(outbox, "conversation not found")
	}

	channel := ChannelService.Get(conversation.ChannelID)
	if channel == nil || channel.Status != enums.StatusOk {
		return s.markOutboxFailed(outbox, "slack channel not found or disabled")
	}
	cfg, err := ChannelService.ParseSlackChannelConfig(channel.ConfigJSON)
	if err != nil || cfg == nil || strings.TrimSpace(cfg.BotToken) == "" {
		return s.markOutboxFailed(outbox, "slack bot token not configured")
	}

	// Resolve target Slack Channel ID and Thread TS
	var targetChannel string
	var threadTS string

	lastCustomerMsg := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("conversation_id", conversation.ID).
		Eq("sender_type", enums.IMSenderTypeCustomer).
		Desc("id"))
	if lastCustomerMsg != nil && lastCustomerMsg.Payload != "" {
		var payloadMap map[string]any
		if err := json.Unmarshal([]byte(lastCustomerMsg.Payload), &payloadMap); err == nil {
			if ch, ok := payloadMap["slack_channel"].(string); ok && ch != "" {
				targetChannel = ch
			}
			if ts, ok := payloadMap["slack_thread_ts"].(string); ok && ts != "" {
				threadTS = ts
			}
		}
	}

	if targetChannel == "" {
		targetChannel = cfg.DefaultChannel
	}
	if targetChannel == "" {
		return s.markOutboxFailed(outbox, "unable to resolve target slack channel")
	}

	client := slack.NewClient(cfg.BotToken)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	textToSend := message.Content
	if message.MessageType == enums.IMMessageTypeImage || message.MessageType == enums.IMMessageTypeAttachment {
		assetPayload, err := parseIMMessageAssetPayload(message.Payload)
		if err == nil && assetPayload != nil {
			assetPayload = hydrateIMMessageAssetPayload(assetPayload)
			if assetPayload.Provider != "" && assetPayload.StorageKey != "" {
				if provider, err := storage.NewProvider(assetPayload.Provider); err == nil {
					fileURL := provider.GetSignedURL(assetPayload.StorageKey)
					if fileURL != "" {
						if textToSend != "" {
							textToSend += "\n" + fileURL
						} else {
							textToSend = fileURL
						}
					}
				}
			}
		}
	}

	_, sendErr := client.PostMessage(ctx, targetChannel, textToSend, threadTS)
	if sendErr != nil {
		return s.markOutboxFailed(outbox, sendErr.Error())
	}

	return ChannelMessageOutboxService.Updates(outbox.ID, map[string]any{
		"send_status": string(enums.ChannelMessageOutboxStatusSent),
		"sent_at":     time.Now(),
		"updated_at":  time.Now(),
	})
}

func (s *slackOutboundService) markOutboxFailed(outbox *models.ChannelMessageOutbox, errMsg string) error {
	if outbox == nil {
		return nil
	}
	retryCount := outbox.RetryCount + 1
	status := string(enums.ChannelMessageOutboxStatusFailed)
	if retryCount >= slackOutboxMaxRetry {
		status = string(enums.ChannelMessageOutboxStatusIgnored)
	}
	nextRetryAt := time.Now().Add(time.Duration(retryCount*30) * time.Second)

	return ChannelMessageOutboxService.Updates(outbox.ID, map[string]any{
		"send_status":   status,
		"retry_count":   retryCount,
		"next_retry_at": &nextRetryAt,
		"last_error":    errMsg,
		"updated_at":    time.Now(),
	})
}
