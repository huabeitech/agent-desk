package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/discord"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"agent-desk/internal/services/storage"
	"os"

	"github.com/mlogclub/simple/sqls"
)

const (
	discordOutboxBatchSize = 20
	discordOutboxMaxRetry  = 5
)

var DiscordOutboundService = newDiscordOutboundService()

func newDiscordOutboundService() *discordOutboundService {
	return &discordOutboundService{}
}

type discordOutboundService struct{}

func (s *discordOutboundService) DispatchPendingOutbox() int {
	return s.doDispatchPendingOutbox(discordOutboxBatchSize)
}

func (s *discordOutboundService) doDispatchPendingOutbox(limit int) int {
	if limit <= 0 {
		limit = discordOutboxBatchSize
	}
	items := ChannelMessageOutboxService.ListPending(enums.ChannelTypeDiscord, limit)
	if len(items) == 0 {
		return 0
	}

	successCount := 0
	for i := range items {
		if err := s.processOutbox(items[i].ID); err != nil {
			slog.Warn("process discord outbox failed",
				"outbox_id", items[i].ID,
				"error", err,
			)
			continue
		}
		successCount++
	}
	return successCount
}

func (s *discordOutboundService) processOutbox(outboxID int64) error {
	outbox := ChannelMessageOutboxService.Get(outboxID)
	if outbox == nil {
		return nil
	}
	if outbox.ChannelType != enums.ChannelTypeDiscord {
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
		return s.markOutboxFailed(outbox, "discord channel not found or disabled")
	}
	cfg, err := ChannelService.ParseDiscordChannelConfig(channel.ConfigJSON)
	if err != nil {
		return s.markOutboxFailed(outbox, "invalid discord channel config")
	}
	botToken := ""
	if cfg != nil {
		botToken = strings.TrimSpace(cfg.BotToken)
	}
	if botToken == "" {
		botToken = strings.TrimSpace(config.Current().Discord.BotToken)
	}
	if botToken == "" {
		botToken = strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
	}
	if botToken == "" {
		return s.markOutboxFailed(outbox, "discord bot token not configured")
	}

	// Resolve target Discord User ID and/or Channel ID
	var discordUserID string
	customerIdentity := repositories.CustomerIdentityRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("customer_id", conversation.CustomerID).
		Eq("external_source", enums.ExternalSourceDiscord))
	if customerIdentity != nil {
		discordUserID = strings.TrimSpace(customerIdentity.ExternalID)
	}

	// Check if there is a discord_channel_id in last message payload
	var targetChannelID string
	lastCustomerMsg := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("conversation_id", conversation.ID).
		Eq("sender_type", enums.IMSenderTypeCustomer).
		Desc("id"))
	if lastCustomerMsg != nil && lastCustomerMsg.Payload != "" {
		var payloadMap map[string]any
		if err := json.Unmarshal([]byte(lastCustomerMsg.Payload), &payloadMap); err == nil {
			if chID, ok := payloadMap["discord_channel_id"].(string); ok && chID != "" {
				targetChannelID = chID
			}
		}
	}

	client := discord.NewClient(botToken)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if targetChannelID == "" {
		if discordUserID == "" {
			return s.markOutboxFailed(outbox, "unable to resolve discord target user or channel")
		}
		dmChannel, err := client.CreateDMChannel(ctx, discordUserID)
		if err != nil {
			return s.markOutboxFailed(outbox, "create discord dm channel failed: "+err.Error())
		}
		targetChannelID = dmChannel.ID
	}

	var sendErr error
	if message.MessageType == enums.IMMessageTypeImage {
		assetPayload, err := parseIMMessageAssetPayload(message.Payload)
		var imageURL string
		if err == nil && assetPayload != nil {
			assetPayload = hydrateIMMessageAssetPayload(assetPayload)
			if assetPayload.Provider != "" && assetPayload.StorageKey != "" {
				if provider, err := storage.NewProvider(assetPayload.Provider); err == nil {
					imageURL = provider.GetSignedURL(assetPayload.StorageKey)
				}
			}
		}
		if imageURL == "" && strings.HasPrefix(strings.TrimSpace(message.Content), "http") {
			imageURL = strings.TrimSpace(message.Content)
		}

		if imageURL != "" {
			embed := discord.Embed{
				Title: "Image Attachment",
				Image: &discord.EmbedMedia{URL: imageURL},
			}
			_, sendErr = client.SendEmbedMessage(ctx, targetChannelID, message.Content, []discord.Embed{embed})
		} else {
			_, sendErr = client.SendMessage(ctx, targetChannelID, message.Content)
		}
	} else if message.MessageType == enums.IMMessageTypeAttachment {
		assetPayload, err := parseIMMessageAssetPayload(message.Payload)
		var fileURL string
		if err == nil && assetPayload != nil {
			assetPayload = hydrateIMMessageAssetPayload(assetPayload)
			if assetPayload.Provider != "" && assetPayload.StorageKey != "" {
				if provider, err := storage.NewProvider(assetPayload.Provider); err == nil {
					fileURL = provider.GetSignedURL(assetPayload.StorageKey)
				}
			}
		}
		textToSend := message.Content
		if fileURL != "" {
			if textToSend != "" {
				textToSend += "\n" + fileURL
			} else {
				textToSend = fileURL
			}
		}
		_, sendErr = client.SendMessage(ctx, targetChannelID, textToSend)
	} else {
		_, sendErr = client.SendMessage(ctx, targetChannelID, message.Content)
	}

	if sendErr != nil {
		return s.markOutboxFailed(outbox, sendErr.Error())
	}

	return ChannelMessageOutboxService.Updates(outbox.ID, map[string]any{
		"send_status": string(enums.ChannelMessageOutboxStatusSent),
		"sent_at":     time.Now(),
		"updated_at":  time.Now(),
	})
}

func (s *discordOutboundService) markOutboxFailed(outbox *models.ChannelMessageOutbox, errMsg string) error {
	if outbox == nil {
		return nil
	}
	retryCount := outbox.RetryCount + 1
	status := string(enums.ChannelMessageOutboxStatusFailed)
	if retryCount >= discordOutboxMaxRetry {
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
