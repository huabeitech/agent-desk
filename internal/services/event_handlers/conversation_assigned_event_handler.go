package event_handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-desk/internal/events"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/eventbus"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/services"

	"github.com/mlogclub/simple/common/strs"
)

func init() {
	eventbus.
		Register[events.ConversationAssignedEvent]().
		Subscribe(handleConversationAssignedNotify)
	eventbus.
		Register[events.ConversationAssignedEvent]().
		Subscribe(handleConversationAssignedWebhookNotify)
}

func handleConversationAssignedNotify(ctx context.Context, event events.ConversationAssignedEvent) error {
	if event.ConversationID <= 0 || event.ToUserID <= 0 {
		return nil
	}
	conversation := services.ConversationService.Get(event.ConversationID)
	if conversation == nil {
		return nil
	}
	return services.WxWorkNotifyService.SendTextToAssigneeOrDefault(event.ToUserID,
		conversationAssignedNotifyTitle(event.AssignType),
		buildConversationAssignedNotifyBody(conversation, event.ToUserID, event.Reason, event.AssignType, event.ContextText))
}

func handleConversationAssignedWebhookNotify(ctx context.Context, event events.ConversationAssignedEvent) error {
	if event.ConversationID <= 0 {
		return nil
	}
	conversation := services.ConversationService.Get(event.ConversationID)
	if conversation == nil {
		return nil
	}
	return services.WebhookNotifyService.SendText("conversation_assigned",
		conversationAssignedNotifyTitle(event.AssignType),
		buildConversationAssignedNotifyBody(conversation, event.ToUserID, event.Reason, event.AssignType, event.ContextText),
		map[string]any{
			"conversationId": event.ConversationID,
			"toUserId":       event.ToUserID,
			"assignType":     event.AssignType,
			"actionUrl":      fmt.Sprintf("/dashboard/conversations?conversationId=%d", conversation.ID),
		})
}

func conversationAssignedNotifyTitle(assignType string) string {
	switch strings.TrimSpace(assignType) {
	case events.ConversationAssignTypeTransfer:
		return i18nx.Get("notification.conversationTransferred.title")
	case events.ConversationAssignTypeAutoAssign:
		return i18nx.Get("notification.conversationAutoAssigned.title")
	default:
		return i18nx.Get("notification.conversationAssigned.title")
	}
}

func buildConversationAssignedNotifyBody(conversation *models.Conversation, assigneeID int64, reason string, assignType string, eventContext string) string {
	if conversation == nil {
		return ""
	}
	reasonKey := "notification.conversationAssigned.reason"
	if strings.TrimSpace(assignType) == events.ConversationAssignTypeTransfer {
		reasonKey = "notification.conversationTransferred.reason"
	}
	lines := []string{
		i18nx.Getf(i18nx.DefaultLocale, "notification.conversationAssigned.wxwork.id", conversation.ID),
		i18nx.Getf(i18nx.DefaultLocale, "notification.conversationAssigned.wxwork.summary", strs.DefaultIfBlank(services.ConversationService.BuildConversationSummary(conversation), "-")),
		i18nx.Getf(i18nx.DefaultLocale, "notification.conversationAssigned.wxwork.channel", resolveConversationChannelLabel(conversation)),
		i18nx.Getf(i18nx.DefaultLocale, "notification.conversationAssigned.wxwork.status", enums.GetIMConversationStatusLabel(conversation.Status)),
		i18nx.Getf(i18nx.DefaultLocale, "notification.assignee", resolveNotifyUserLabel(assigneeID)),
	}
	if strings.TrimSpace(reason) != "" {
		lines = append(lines, i18nx.Getf(i18nx.DefaultLocale, reasonKey, strings.TrimSpace(reason)))
	}
	if contextText := conversationHandoffContext(conversation, eventContext); contextText != "" {
		lines = append(lines, contextText)
	}
	lines = append(lines, i18nx.Getf(i18nx.DefaultLocale, "notification.time", time.Now().Format("2006-01-02 15:04:05")))
	return strings.Join(lines, "\n")
}

func resolveConversationChannelLabel(conversation *models.Conversation) string {
	if conversation == nil || conversation.ChannelID <= 0 {
		return "-"
	}
	if channel := services.ChannelService.Get(conversation.ChannelID); channel != nil {
		return strs.DefaultIfBlank(channel.Name, channel.ChannelType)
	}
	return "-"
}

func resolveNotifyUserLabel(userID int64) string {
	if userID <= 0 {
		return "-"
	}
	user := services.UserService.Get(userID)
	if user == nil {
		return fmt.Sprintf("用户#%d", userID)
	}
	if nickname := strings.TrimSpace(user.Nickname); nickname != "" {
		return nickname
	}
	if username := strings.TrimSpace(user.Username); username != "" {
		return username
	}
	return fmt.Sprintf("用户#%d", userID)
}
