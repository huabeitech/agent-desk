package services

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"agent-desk/internal/discord"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"
)

var DiscordInboundService = newDiscordInboundService()

func newDiscordInboundService() *discordInboundService {
	return &discordInboundService{}
}

type discordInboundService struct{}

// HandleWebhook processes an incoming webhook or gateway payload from Discord.
func (s *discordInboundService) HandleWebhook(ctx context.Context, channelID string, secretHeader string, rawPayload []byte) error {
	channelID = strings.TrimSpace(channelID)
	var channel *models.Channel
	if channelID != "" {
		channel = ChannelService.Take("channel_id = ? AND channel_type = ? AND status = ?", channelID, enums.ChannelTypeDiscord, enums.StatusOk)
	}
	if channel == nil {
		channel = ChannelService.Take("channel_type = ? AND status = ?", enums.ChannelTypeDiscord, enums.StatusOk)
	}
	if channel == nil {
		return errorsx.InvalidParam("discord channel not found or disabled")
	}

	cfg, err := ChannelService.ParseDiscordChannelConfig(channel.ConfigJSON)
	if err != nil || cfg == nil {
		return errorsx.InvalidParam("discord channel config invalid")
	}

	// Compared in constant time: a byte-wise != leaks how much of the prefix
	// matched through response timing.
	if cfg.WebhookSecret != "" &&
		subtle.ConstantTimeCompare([]byte(strings.TrimSpace(secretHeader)), []byte(cfg.WebhookSecret)) != 1 {
		return errorsx.UnauthorizedI18n("error.auth.invalidSignature")
	}

	var payload discord.WebhookPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal discord payload failed: %w", err)
	}

	author := payload.Author
	text := strings.TrimSpace(payload.Content)
	msgID := payload.ID
	targetChannelID := payload.ChannelID
	guildID := payload.GuildID
	attachments := payload.Attachments
	embeds := payload.Embeds

	if payload.Message != nil {
		if author == nil {
			author = &payload.Message.Author
		}
		if text == "" {
			text = strings.TrimSpace(payload.Message.Content)
		}
		if msgID == "" {
			msgID = payload.Message.ID
		}
		if targetChannelID == "" {
			targetChannelID = payload.Message.ChannelID
		}
		if guildID == "" {
			guildID = payload.Message.GuildID
		}
		if len(attachments) == 0 && len(payload.Message.Attachments) > 0 {
			attachments = payload.Message.Attachments
		}
		if len(embeds) == 0 && len(payload.Message.Embeds) > 0 {
			embeds = payload.Message.Embeds
		}
	}

	if author == nil || author.Bot || strings.TrimSpace(author.ID) == "" {
		return nil // Ignore bot messages or invalid authors
	}

	// Honour the channel's guild scope. A bot can be invited to several servers,
	// and without these checks GuildID and ChannelScope would be stored
	// configuration that silently does nothing.
	if cfg.GuildID != "" && guildID != cfg.GuildID {
		slog.Debug("ignoring discord message from an out-of-scope guild",
			"guild_id", guildID,
			"channel", channel.ID,
		)
		return nil
	}
	if cfg.ChannelScope == "dm_only" && guildID != "" {
		slog.Debug("ignoring discord guild message, channel is dm_only",
			"guild_id", guildID,
			"channel", channel.ID,
		)
		return nil
	}

	if text == "" && len(attachments) > 0 {
		firstAtt := attachments[0]
		if firstAtt.Filename != "" {
			text = fmt.Sprintf("[%s] %s", firstAtt.Filename, firstAtt.URL)
		} else {
			text = firstAtt.URL
		}
	}

	if text == "" && len(attachments) == 0 && len(embeds) == 0 {
		return nil // Ignore empty messages
	}
	if text == "" && len(embeds) > 0 {
		text = embeds[0].Description
		if text == "" {
			text = embeds[0].Title
		}
	}

	// 1. Resolve customer identity
	externalID := author.ID
	name := strings.TrimSpace(author.GlobalName)
	if name == "" {
		name = strings.TrimSpace(author.Username)
	}
	if name == "" {
		name = fmt.Sprintf("Discord User %s", author.ID)
	}

	externalUser := openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceDiscord,
		ExternalID:     externalID,
		ExternalName:   name,
	}

	// 2. Create or match Conversation
	conversation, err := ConversationService.Create(externalUser, channel.ID, channel.AIAgentID)
	if err != nil {
		return fmt.Errorf("create discord conversation failed: %w", err)
	}

	// 3. Send message through MessageService
	clientMsgID := fmt.Sprintf("discord_%s_%s", targetChannelID, msgID)
	payloadMap := map[string]any{
		"discord_message_id":  msgID,
		"discord_channel_id":  targetChannelID,
		"discord_guild_id":    guildID,
		"discord_user_id":     author.ID,
		"discord_attachments": attachments,
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
		return fmt.Errorf("send customer message failed: %w", err)
	}

	return nil
}
