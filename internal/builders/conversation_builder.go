package builders

import (
	"strconv"
	"strings"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/services"

	"github.com/mlogclub/simple/sqls"
)

func BuildConversation(item *models.Conversation) response.ConversationResponse {
	return BuildConversationWithLocale(item, i18nx.DefaultLocale)
}

func BuildConversationWithLocale(item *models.Conversation, locale string) response.ConversationResponse {
	agentReadState, customerReadState := services.ConversationReadStateService.GetConversationReadStates(item.ID)
	ret := response.ConversationResponse{
		ID:                        item.ID,
		AIAgentID:                 item.AIAgentID,
		ChannelID:                 item.ChannelID,
		CustomerID:                item.CustomerID,
		CustomerName:              item.CustomerName,
		Status:                    item.Status,
		ServiceMode:               item.ServiceMode,
		Priority:                  item.Priority,
		CurrentAssigneeID:         item.CurrentAssigneeID,
		CurrentTeamID:             item.CurrentTeamID,
		LastMessageID:             item.LastMessageID,
		LastMessageAt:             utils.FormatTime(item.LastMessageAt),
		LastActiveAt:              utils.FormatTime(item.LastActiveAt),
		LastMessageSummary:        localizeConversationSummary(locale, item.LastMessageSummary),
		CustomerUnreadCount:       item.CustomerUnreadCount,
		AgentUnreadCount:          item.AgentUnreadCount,
		CustomerLastReadMessageID: readStateMessageID(customerReadState),
		CustomerLastReadAt:        readStateAt(customerReadState),
		AgentLastReadMessageID:    readStateMessageID(agentReadState),
		AgentLastReadAt:           readStateAt(agentReadState),
		ClosedAt:                  utils.FormatTimePtr(item.ClosedAt),
		ClosedBy:                  item.ClosedBy,
		CloseReason:               item.CloseReason,
	}
	if identity := services.ConversationService.GetConversationExternalIdentity(item); identity != nil {
		ret.CustomerOnline = services.WsService.IsGuestOnline(identity.ExternalID)
	}
	if item.CurrentAssigneeID > 0 {
		if user := services.UserService.Get(item.CurrentAssigneeID); user != nil {
			ret.CurrentAssigneeName = user.Nickname
			if ret.CurrentAssigneeName == "" {
				ret.CurrentAssigneeName = user.Username
			}
		}
	}
	if item.CurrentTeamID > 0 {
		if team := services.AgentTeamService.Get(item.CurrentTeamID); team != nil {
			ret.CurrentTeamName = team.Name
		}
	}
	if item.ClosedBy > 0 {
		if user := services.UserService.Get(item.ClosedBy); user != nil {
			ret.ClosedByName = user.Nickname
			if ret.ClosedByName == "" {
				ret.ClosedByName = user.Username
			}
		}
	}
	return ret
}

func localizeConversationSummary(locale string, summary string) string {
	if i18nx.NormalizeLocale(locale) != i18nx.LocaleEnUS {
		return summary
	}
	switch {
	case summary == "[图片]":
		return "[Image]"
	case strings.HasPrefix(summary, "[图片] "):
		return "[Image] " + strings.TrimPrefix(summary, "[图片] ")
	case summary == "[附件]":
		return "[Attachment]"
	case strings.HasPrefix(summary, "[附件] "):
		return "[Attachment] " + strings.TrimPrefix(summary, "[附件] ")
	case summary == "该消息已撤回":
		return "This message was recalled."
	default:
		return summary
	}
}

func BuildParticipantResponses(conversationID int64) []response.ConversationParticipantResponse {
	list := services.ConversationParticipantService.Find(sqls.NewCnd().Eq("conversation_id", conversationID).Asc("id"))
	if len(list) == 0 {
		return nil
	}
	ret := make([]response.ConversationParticipantResponse, 0, len(list))
	for _, item := range list {
		ret = append(ret, response.ConversationParticipantResponse{
			ID:                    item.ID,
			ParticipantType:       item.ParticipantType,
			ParticipantID:         item.ParticipantID,
			ExternalParticipantID: item.ExternalParticipantID,
			JoinedAt:              utils.FormatTimePtr(item.JoinedAt),
			LeftAt:                utils.FormatTimePtr(item.LeftAt),
			Status:                item.Status,
		})
	}
	return ret
}

func BuildMessages(list []models.Message) []response.MessageResponse {
	return BuildMessagesWithLocale(list, i18nx.DefaultLocale)
}

func BuildMessagesWithLocale(list []models.Message, locale string) []response.MessageResponse {
	if len(list) == 0 {
		return nil
	}
	agentReadState, customerReadState := services.ConversationReadStateService.GetConversationReadStates(list[0].ConversationID)
	aiSenderNames, userSenderNames := collectMessageSenderNameMaps(list)
	agentProfiles := collectAgentProfilesByMessages(list)
	ret := make([]response.MessageResponse, 0, len(list))
	for i := range list {
		ret = append(ret, BuildMessageWithReadStatesAndLocale(&list[i], agentReadState, customerReadState, aiSenderNames, userSenderNames, agentProfiles, locale))
	}
	return ret
}

func BuildMessage(item *models.Message) response.MessageResponse {
	return BuildMessageWithLocale(item, i18nx.DefaultLocale)
}

func BuildMessageWithLocale(item *models.Message, locale string) response.MessageResponse {
	agentReadState, customerReadState := services.ConversationReadStateService.GetConversationReadStates(item.ConversationID)
	return BuildMessageWithReadStatesAndLocale(item, agentReadState, customerReadState, nil, nil, nil, locale)
}

func BuildMessageWithReadStates(item *models.Message, agentReadState, customerReadState *models.ConversationReadState, aiSenderNames, userSenderNames map[int64]string, agentProfiles map[int64]*models.AgentProfile) response.MessageResponse {
	return BuildMessageWithReadStatesAndLocale(item, agentReadState, customerReadState, aiSenderNames, userSenderNames, agentProfiles, i18nx.DefaultLocale)
}

func BuildMessageWithReadStatesAndLocale(item *models.Message, agentReadState, customerReadState *models.ConversationReadState, aiSenderNames, userSenderNames map[int64]string, agentProfiles map[int64]*models.AgentProfile, locale string) response.MessageResponse {
	content, payload := utils.BuildRenderableMessage(item)
	ret := response.MessageResponse{
		ID:              item.ID,
		ConversationID:  item.ConversationID,
		RequestID:       item.RequestID,
		WorkflowRunID:   item.WorkflowRunID,
		ClientMsgID:     item.ClientMsgID,
		SenderType:      item.SenderType,
		SenderID:        item.SenderID,
		MessageType:     item.MessageType,
		Content:         localizeRenderableMessageContent(locale, content),
		Payload:         payload,
		SendStatus:      item.SendStatus,
		SentAt:          utils.FormatTimePtr(item.SentAt),
		DeliveredAt:     utils.FormatTimePtr(item.DeliveredAt),
		ReadAt:          utils.FormatTimePtr(item.ReadAt),
		CustomerRead:    isMessageRead(item, customerReadState),
		CustomerReadAt:  readMessageAt(item, customerReadState),
		AgentRead:       isMessageRead(item, agentReadState),
		AgentReadAt:     readMessageAt(item, agentReadState),
		RecalledAt:      utils.FormatTimePtr(item.RecalledAt),
		QuotedMessageID: item.QuotedMessageID,
	}
	if item.SenderID > 0 {
		if item.SenderType == enums.IMSenderTypeAI {
			if aiSenderNames != nil {
				ret.SenderName = aiSenderNames[item.SenderID]
			} else if aiAgent := services.AIAgentService.Get(item.SenderID); aiAgent != nil {
				ret.SenderName = aiAgent.Name
			}
		} else if item.SenderType == enums.IMSenderTypeAgent {
			profile := agentProfiles[item.SenderID]
			if profile != nil {
				if dn := strings.TrimSpace(profile.DisplayName); dn != "" {
					ret.SenderName = dn
				}
				if av := utils.BuildAvatarURL(profile.Avatar, "/api/avatar/agent/"+strconv.FormatInt(profile.ID, 10)); av != "" {
					ret.SenderAvatar = av
				}
			}
			if ret.SenderName == "" {
				if userSenderNames != nil {
					ret.SenderName = userSenderNames[item.SenderID]
				} else if user := services.UserService.Get(item.SenderID); user != nil {
					ret.SenderName = user.Nickname
					if ret.SenderName == "" {
						ret.SenderName = user.Username
					}
				}
			}
		} else if userSenderNames != nil {
			ret.SenderName = userSenderNames[item.SenderID]
		} else if user := services.UserService.Get(item.SenderID); user != nil {
			ret.SenderName = user.Nickname
			if ret.SenderName == "" {
				ret.SenderName = user.Username
			}
		}
	}
	return ret
}

func localizeRenderableMessageContent(locale string, content string) string {
	if i18nx.NormalizeLocale(locale) != i18nx.LocaleEnUS {
		return content
	}
	if content == "该消息已撤回" {
		return "This message was recalled."
	}
	return content
}

func collectAgentProfilesByMessages(list []models.Message) map[int64]*models.AgentProfile {
	var agentUserIDs []int64
	seen := make(map[int64]struct{})
	for i := range list {
		m := &list[i]
		if m.SenderType != enums.IMSenderTypeAgent || m.SenderID <= 0 {
			continue
		}
		if _, ok := seen[m.SenderID]; ok {
			continue
		}
		seen[m.SenderID] = struct{}{}
		agentUserIDs = append(agentUserIDs, m.SenderID)
	}
	if len(agentUserIDs) == 0 {
		return nil
	}
	profiles := services.AgentProfileService.Find(sqls.NewCnd().In("user_id", agentUserIDs))
	out := make(map[int64]*models.AgentProfile, len(profiles))
	for i := range profiles {
		out[profiles[i].UserID] = &profiles[i]
	}
	return out
}

func collectMessageSenderNameMaps(list []models.Message) (aiNames map[int64]string, userNames map[int64]string) {
	aiNames = make(map[int64]string)
	userNames = make(map[int64]string)
	var aiIDs, userIDs []int64
	seenAI := make(map[int64]struct{})
	seenUser := make(map[int64]struct{})
	for i := range list {
		m := &list[i]
		if m.SenderID <= 0 {
			continue
		}
		if m.SenderType == enums.IMSenderTypeAI {
			if _, ok := seenAI[m.SenderID]; ok {
				continue
			}
			seenAI[m.SenderID] = struct{}{}
			aiIDs = append(aiIDs, m.SenderID)
			continue
		}
		if _, ok := seenUser[m.SenderID]; ok {
			continue
		}
		seenUser[m.SenderID] = struct{}{}
		userIDs = append(userIDs, m.SenderID)
	}
	for _, a := range services.AIAgentService.FindByIds(aiIDs) {
		aiNames[a.ID] = a.Name
	}
	for _, u := range services.UserService.FindByIds(userIDs) {
		name := u.Nickname
		if name == "" {
			name = u.Username
		}
		userNames[u.ID] = name
	}
	return aiNames, userNames
}

func isMessageRead(item *models.Message, state *models.ConversationReadState) bool {
	return item != nil && state != nil && state.LastReadMessageID >= item.ID
}

func readMessageAt(item *models.Message, state *models.ConversationReadState) string {
	if !isMessageRead(item, state) {
		return ""
	}
	return utils.FormatTimePtr(state.LastReadAt)
}

func readStateMessageID(state *models.ConversationReadState) int64 {
	if state == nil {
		return 0
	}
	return state.LastReadMessageID
}

func readStateAt(state *models.ConversationReadState) string {
	if state == nil {
		return ""
	}
	return utils.FormatTimePtr(state.LastReadAt)
}
