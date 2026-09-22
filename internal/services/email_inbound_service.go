package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"agent-desk/internal/email"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
)

var (
	EmailInboundService = newEmailInboundService()
	ticketIDRegex       = regexp.MustCompile(`(?i)(?:\[(?:#|Ticket\s*#?)|(?:#|Ticket\s*#))\s*(\d+)`)
)

func newEmailInboundService() *emailInboundService {
	return &emailInboundService{}
}

type emailInboundService struct{}

// HandleWebhook processes an incoming email webhook from any supported provider (Cloudflare, Brevo, SendGrid, Postmark, Mailgun, Resend).
func (s *emailInboundService) HandleWebhook(ctx context.Context, channelID string, secretHeader string, contentType string, rawPayload []byte, form url.Values) error {
	channelID = strings.TrimSpace(channelID)
	var channel *models.Channel
	if channelID != "" {
		channel = ChannelService.Take("channel_id = ? AND channel_type = ? AND status = ?", channelID, enums.ChannelTypeEmail, enums.StatusOk)
	}

	// 1. Parse inbound email items
	inboundItems, err := email.ParseInboundWebhook(contentType, rawPayload, form)
	if err != nil {
		return fmt.Errorf("parse email webhook failed: %w", err)
	}
	if len(inboundItems) == 0 {
		return nil
	}

	systemSecret := ""
	if cfg := config.GetCurrent(); cfg != nil {
		systemSecret = cfg.Email.InboundSecret
	}

	for _, item := range inboundItems {
		targetChannel := channel
		if targetChannel == nil {
			targetChannel = ChannelService.GetEnabledEmailChannelByAddress(item.ToEmail)
		}
		if targetChannel == nil {
			targetChannel = ChannelService.Take("channel_type = ? AND status = ?", enums.ChannelTypeEmail, enums.StatusOk)
		}
		if targetChannel == nil {
			slog.Warn("no active email channel found for recipient", "to", item.ToEmail)
			return errorsx.InvalidParam("email channel not found or disabled")
		}

		cfg, err := ChannelService.ParseEmailChannelConfig(targetChannel.ConfigJSON)
		if err != nil || cfg == nil {
			return errorsx.InvalidParam("email channel config invalid")
		}

		expectedSecret := cfg.WebhookSecret
		if expectedSecret == "" {
			expectedSecret = systemSecret
		}

		if expectedSecret != "" && strings.TrimSpace(secretHeader) != expectedSecret {
			return errorsx.UnauthorizedI18n("error.auth.invalidSignature")
		}

		if err := s.processInboundItem(ctx, targetChannel, item); err != nil {
			slog.Error("process inbound email item failed", "from", item.FromEmail, "to", item.ToEmail, "error", err)
			return err
		}
	}

	return nil
}

func (s *emailInboundService) processInboundItem(ctx context.Context, channel *models.Channel, item email.InboundEmailPayload) error {
	fromEmail := strings.ToLower(strings.TrimSpace(item.FromEmail))
	if fromEmail == "" {
		return nil
	}
	fromName := strings.TrimSpace(item.FromName)
	if fromName == "" {
		parts := strings.Split(fromEmail, "@")
		fromName = parts[0]
	}

	bodyText := strings.TrimSpace(item.BodyText)
	if bodyText == "" && item.BodyHTML != "" {
		bodyText = stripHTMLTags(item.BodyHTML)
	}
	if bodyText == "" {
		bodyText = "(Empty email body)"
	}

	// Format content with subject if provided
	content := bodyText
	if item.Subject != "" {
		content = fmt.Sprintf("[%s]\n\n%s", item.Subject, bodyText)
	}

	// 1. Resolve customer identity
	externalUser := openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceEmail,
		ExternalID:     fromEmail,
		ExternalName:   fromName,
	}

	// 2. Threading Resolution: Find existing conversation by Ticket ID or In-Reply-To header
	var conversation *models.Conversation
	ticketID := s.extractTicketID(item.Subject)
	if ticketID > 0 {
		existing := ConversationService.Get(ticketID)
		if existing != nil && existing.Status != enums.IMConversationStatusClosed {
			conversation = existing
		}
	}

	if conversation == nil && item.InReplyTo != "" {
		conversation = s.findConversationByInReplyTo(item.InReplyTo)
	}

	// If no existing thread matched, create or match via standard ConversationService
	if conversation == nil {
		var err error
		conversation, err = ConversationService.Create(externalUser, channel.ID, channel.AIAgentID)
		if err != nil {
			return fmt.Errorf("create email conversation failed: %w", err)
		}
	}

	// Ensure customer primary_email is populated
	if conversation.CustomerID > 0 {
		customer := repositories.CustomerRepository.Get(sqls.DB(), conversation.CustomerID)
		if customer != nil && customer.PrimaryEmail == "" {
			_ = repositories.CustomerRepository.UpdateColumn(sqls.DB(), customer.ID, "primary_email", fromEmail)
		}
	}

	// 3. Send message through MessageService (triggers AI response loop or agent notification)
	msgHash := strs.UUID()
	if item.MessageID != "" {
		msgHash = fmt.Sprintf("email_%s", strings.Trim(item.MessageID, "<>"))
	}
	clientMsgID := fmt.Sprintf("mail_%s", msgHash)

	payloadMap := map[string]any{
		"email_from":       fromEmail,
		"email_from_name":  fromName,
		"email_to":         item.ToEmail,
		"email_subject":    item.Subject,
		"email_message_id": item.MessageID,
		"email_in_reply":   item.InReplyTo,
		"email_references": item.References,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	_, err := MessageService.SendCustomerMessage(
		conversation.ID,
		clientMsgID,
		enums.IMMessageTypeText,
		content,
		string(payloadBytes),
		externalUser,
	)
	if err != nil {
		return fmt.Errorf("send customer message failed: %w", err)
	}

	slog.Info("inbound email successfully processed",
		"from", fromEmail,
		"channel_id", channel.ChannelID,
		"conversation_id", conversation.ID,
		"subject", item.Subject,
	)
	return nil
}

func (s *emailInboundService) extractTicketID(subject string) int64 {
	matches := ticketIDRegex.FindStringSubmatch(subject)
	if len(matches) > 1 {
		if id, err := strconv.ParseInt(matches[1], 10, 64); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

func (s *emailInboundService) findConversationByInReplyTo(inReplyTo string) *models.Conversation {
	inReplyTo = strings.TrimSpace(inReplyTo)
	if inReplyTo == "" {
		return nil
	}
	// Look up recent message containing this email message ID
	likePattern := "%" + inReplyTo + "%"
	msg := repositories.MessageRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Where("payload LIKE ?", likePattern).
		Desc("id"))
	if msg != nil && msg.ConversationID > 0 {
		conv := ConversationService.Get(msg.ConversationID)
		if conv != nil && conv.Status != enums.IMConversationStatusClosed {
			return conv
		}
	}
	return nil
}

func stripHTMLTags(s string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			builder.WriteRune(r)
		}
	}
	return strings.TrimSpace(builder.String())
}
