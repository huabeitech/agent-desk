package services

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"

	"github.com/mlogclub/simple/common/strs"
	"github.com/silenceper/wechat/v2/work/kf"
	"github.com/silenceper/wechat/v2/work/kf/syncmsg"
)

const (
	wxWorkKFSystemOperatorName = "wxwork_kf"
	wxWorkKFSyncMsgLimit       = 1000
)

var WxWorkKFInboundService = newWxWorkKFInboundService()

func newWxWorkKFInboundService() *wxWorkKFInboundService {
	return &wxWorkKFInboundService{}
}

type wxWorkKFInboundService struct {
}

// kfClientByOpenKfID 按 openKfID 找到启用渠道，并返回该渠道绑定应用的客服客户端。
func (s *wxWorkKFInboundService) kfClientByOpenKfID(openKfID string) (*kf.Client, error) {
	channel := ChannelService.GetEnabledWxWorkKFChannelByOpenKfID(openKfID)
	if channel == nil {
		return nil, errorsx.InvalidParamI18n("error.e0231")
	}
	workCli, err := ChannelService.GetWxWorkCliByChannel(channel)
	if err != nil {
		return nil, err
	}
	return workCli.GetKF()
}

func (s *wxWorkKFInboundService) SyncCallbackMessages(message kf.CallbackMessage) error {
	cli, err := s.kfClientByOpenKfID(message.OpenKfID)
	if err != nil {
		return err
	}

	cursor := ""
	if state := WxWorkKFSyncStateService.GetByOpenKfID(message.OpenKfID); state != nil {
		cursor = strings.TrimSpace(state.NextCursor)
	}

	for {
		result, syncErr := cli.SyncMsg(kf.SyncMsgOptions{
			Cursor:   cursor,
			Token:    message.Token,
			Limit:    wxWorkKFSyncMsgLimit,
			OpenKfID: message.OpenKfID,
		})
		if syncErr != nil {
			return syncErr
		}

		for _, item := range result.MsgList {
			if err := s.consumeSyncMessage(item); err != nil {
				slog.Error("consume wxwork kf sync message failed",
					"open_kfid", item.OpenKFID,
					"external_userid", item.ExternalUserID,
					"msg_id", item.MsgID,
					"msg_type", item.MsgType,
					"event", item.EventType,
					"error", err,
				)
			}
		}

		if err := s.saveNextCursor(message.OpenKfID, result.NextCursor); err != nil {
			return err
		}

		if result.HasMore != 1 || strings.TrimSpace(result.NextCursor) == "" {
			return nil
		}
		cursor = result.NextCursor
	}
}

func (s *wxWorkKFInboundService) consumeSyncMessage(item syncmsg.Message) error {
	msgID := strings.TrimSpace(item.MsgID)
	if msgID == "" {
		return errorsx.InvalidParamI18n("error.e0101")
	}
	if WxWorkKFMessageRefService.Take("wx_msg_id = ?", msgID) != nil {
		return nil
	}

	switch strings.TrimSpace(item.MsgType) {
	case "text":
		return s.handleTextMessage(item)
	case "image":
		return s.handleImageMessage(item)
	case "file":
		return s.handleFileMessage(item)
	case "event":
		return s.handleEventMessage(item)
	default:
		return s.handleUnsupportedMessage(item)
	}
}

func (s *wxWorkKFInboundService) handleTextMessage(item syncmsg.Message) error {
	payload := syncmsg.Text{}
	if err := json.Unmarshal(item.OriginData, &payload); err != nil {
		return err
	}

	conversation, err := s.ensureConversation(payload.BaseMessage, map[string]any{
		"msgType": payload.MsgType,
		"menuId":  payload.Text.MenuID,
	})
	if err != nil {
		return err
	}
	message, err := MessageService.SendCustomerMessage(
		conversation.ID,
		s.buildInboundClientMsgID(item.MsgID),
		enums.IMMessageTypeText,
		strings.TrimSpace(payload.Text.Content),
		"",
		s.buildExternalUser(payload.ExternalUserID),
	)
	if err != nil {
		return err
	}
	return s.createMessageRef(conversation.ID, message.ID, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived)
}

func (s *wxWorkKFInboundService) handleImageMessage(item syncmsg.Message) error {
	payload := syncmsg.Image{}
	if err := json.Unmarshal(item.OriginData, &payload); err != nil {
		return err
	}
	conversation, err := s.ensureConversation(payload.BaseMessage, map[string]any{
		"msgType": payload.MsgType,
		"mediaId": payload.Image.MediaID,
	})
	if err != nil {
		return err
	}
	canonicalPayload, content, err := s.buildInboundAssetPayload(conversation, strings.TrimSpace(payload.Image.MediaID))
	if err != nil {
		return err
	}
	message, err := MessageService.SendCustomerMessage(
		conversation.ID,
		s.buildInboundClientMsgID(item.MsgID),
		enums.IMMessageTypeImage,
		content,
		canonicalPayload,
		s.buildExternalUser(payload.ExternalUserID),
	)
	if err != nil {
		return err
	}
	return s.createMessageRef(conversation.ID, message.ID, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived)
}

func (s *wxWorkKFInboundService) handleFileMessage(item syncmsg.Message) error {
	payload := syncmsg.File{}
	if err := json.Unmarshal(item.OriginData, &payload); err != nil {
		return err
	}
	conversation, err := s.ensureConversation(payload.BaseMessage, map[string]any{
		"msgType": payload.MsgType,
		"mediaId": payload.File.MediaID,
	})
	if err != nil {
		return err
	}
	canonicalPayload, content, err := s.buildInboundAssetPayload(conversation, strings.TrimSpace(payload.File.MediaID))
	if err != nil {
		return err
	}
	message, err := MessageService.SendCustomerMessage(
		conversation.ID,
		s.buildInboundClientMsgID(item.MsgID),
		enums.IMMessageTypeAttachment,
		content,
		canonicalPayload,
		s.buildExternalUser(payload.ExternalUserID),
	)
	if err != nil {
		return err
	}
	return s.createMessageRef(conversation.ID, message.ID, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived)
}

func (s *wxWorkKFInboundService) handleUnsupportedMessage(item syncmsg.Message) error {
	base, err := s.parseBaseMessage(item.OriginData)
	if err != nil {
		return err
	}
	conversation, convErr := s.ensureConversation(base, map[string]any{"msgType": item.MsgType})
	if convErr != nil {
		return convErr
	}
	content := s.buildUnsupportedContent(item.MsgType)
	message, err := MessageService.SendCustomerMessage(
		conversation.ID,
		s.buildInboundClientMsgID(item.MsgID),
		enums.IMMessageTypeText,
		content,
		string(item.OriginData),
		s.buildExternalUser(base.ExternalUserID),
	)
	if err != nil {
		return err
	}
	return s.createMessageRef(conversation.ID, message.ID, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived)
}

func (s *wxWorkKFInboundService) handleEventMessage(item syncmsg.Message) error {
	switch strings.TrimSpace(item.EventType) {
	case enums.WxWorkKFEventTypeEnterSession:
		return s.handleEnterSessionEvent(item)
	case enums.WxWorkKFEventTypeSessionStatusChange:
		return s.handleSessionStatusChangeEvent(item)
	case enums.WxWorkKFEventTypeMsgSendFail:
		return s.handleMsgSendFailEvent(item)
	default:
		return s.recordOrphanEvent(item, "收到未处理的企业微信事件")
	}
}

func (s *wxWorkKFInboundService) handleEnterSessionEvent(item syncmsg.Message) error {
	payload := syncmsg.EnterSessionEvent{}
	if err := json.Unmarshal(item.OriginData, &payload); err != nil {
		return err
	}
	base := s.normalizeEventBaseMessage(payload.BaseMessage, payload.Event.OpenKFID, payload.Event.ExternalUserID)
	conversation, err := s.ensureConversation(base, map[string]any{
		"scene":       payload.Event.Scene,
		"sceneParam":  payload.Event.SceneParam,
		"welcomeCode": payload.Event.WelcomeCode,
	})
	if err != nil {
		return err
	}
	if err := s.createMessageRef(conversation.ID, 0, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived); err != nil {
		return err
	}
	return s.appendConversationEvent(conversation.ID, "微信客户进入会话", string(item.OriginData))
}

func (s *wxWorkKFInboundService) handleSessionStatusChangeEvent(item syncmsg.Message) error {
	payload := syncmsg.SessionStatusChangeEvent{}
	if err := json.Unmarshal(item.OriginData, &payload); err != nil {
		return err
	}
	base := s.normalizeEventBaseMessage(payload.BaseMessage, payload.Event.OpenKFID, payload.Event.ExternalUserID)
	base.ReceptionistUserID = payload.Event.NewReceptionistUserID
	conversation, err := s.ensureConversation(base, map[string]any{
		"changeType": payload.Event.ChangeType,
		"msgCode":    payload.Event.MsgCode,
	})
	if err != nil {
		return err
	}
	sessionStatus := enums.WxWorkKFSessionStatusActive
	switch payload.Event.ChangeType {
	case 2:
		sessionStatus = enums.WxWorkKFSessionStatusTransfer
	case 3:
		sessionStatus = enums.WxWorkKFSessionStatusClosed
	}
	channel, channelErr := s.getChannelByOpenKfID(payload.Event.OpenKFID)
	if channelErr != nil {
		return channelErr
	}
	if err := s.upsertConversationMapping(conversation.ID, channel.ID, payload.Event.OpenKFID, payload.Event.ExternalUserID, payload.Event.NewReceptionistUserID, sessionStatus, payload.SendTime, payload.MsgID, string(item.OriginData)); err != nil {
		return err
	}
	if err := s.createMessageRef(conversation.ID, 0, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived); err != nil {
		return err
	}
	return s.appendConversationEvent(conversation.ID, "微信会话状态变更", string(item.OriginData))
}

func (s *wxWorkKFInboundService) handleMsgSendFailEvent(item syncmsg.Message) error {
	payload := syncmsg.MsgSendFailEvent{}
	if err := json.Unmarshal(item.OriginData, &payload); err != nil {
		return err
	}
	slog.Warn("received wxwork msg_send_fail event",
		slog.String("msg", string(item.OriginData)),
	)
	base := s.normalizeEventBaseMessage(payload.BaseMessage, payload.Event.OpenKFID, payload.Event.ExternalUserID)
	conversation, err := s.ensureConversation(base, map[string]any{
		"failMsgId": payload.Event.FailMsgID,
		"failType":  payload.Event.FailType,
	})
	if err != nil {
		return err
	}
	if ref := WxWorkKFMessageRefService.GetByWxMsgID(payload.Event.FailMsgID); ref != nil {
		targetMessageID := ref.MessageID
		slog.Warn("mark wxwork message ref failed",
			"conversation_id", ref.ConversationID,
			"message_id", ref.MessageID,
			"wx_msg_id", ref.WxMsgID,
			"fail_type", payload.Event.FailType,
			"open_kfid", payload.Event.OpenKFID,
			"external_userid", payload.Event.ExternalUserID,
		)
		_ = WxWorkKFMessageRefService.Updates(ref.ID, map[string]any{
			"send_status": enums.WxWorkKFMessageSendStatusFailed,
			"fail_reason": string(item.OriginData),
			"updated_at":  time.Now(),
		})

		if outbox := ChannelMessageOutboxService.GetByMessageID(enums.ChannelTypeWxWorkKF, targetMessageID); outbox != nil {
			if err := WxWorkKFOutboundService.markOutboxFailed(outbox, string(item.OriginData)); err != nil {
				slog.Warn("mark wxwork outbox failed from callback event failed",
					"outbox_id", outbox.ID,
					"conversation_id", outbox.ConversationID,
					"message_id", outbox.MessageID,
					"fail_msg_id", payload.Event.FailMsgID,
					"fail_type", payload.Event.FailType,
					"error", err,
				)
			} else {
				slog.Warn("mark wxwork outbox failed from callback event",
					"outbox_id", outbox.ID,
					"conversation_id", outbox.ConversationID,
					"message_id", outbox.MessageID,
					"fail_msg_id", payload.Event.FailMsgID,
					"fail_type", payload.Event.FailType,
				)
			}
		} else {
			slog.Warn("wxwork msg_send_fail event missing outbox",
				"message_id", targetMessageID,
				"fail_msg_id", payload.Event.FailMsgID,
				"fail_type", payload.Event.FailType,
			)
		}
	} else {
		slog.Warn("wxwork msg_send_fail event missing message ref",
			"fail_msg_id", payload.Event.FailMsgID,
			"fail_type", payload.Event.FailType,
			"open_kfid", payload.Event.OpenKFID,
			"external_userid", payload.Event.ExternalUserID,
		)
	}

	if err := s.createMessageRef(conversation.ID, 0, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived); err != nil {
		return err
	}
	return s.appendConversationEvent(conversation.ID, "微信消息发送失败事件", string(item.OriginData))
}

func (s *wxWorkKFInboundService) recordOrphanEvent(item syncmsg.Message, content string) error {
	base, err := s.parseBaseMessage(item.OriginData)
	if err != nil {
		return err
	}
	conversation, convErr := s.ensureConversation(base, map[string]any{"eventType": item.EventType})
	if convErr != nil {
		return convErr
	}
	if err := s.createMessageRef(conversation.ID, 0, item, enums.WxWorkKFMessageDirectionIn, enums.WxWorkKFMessageSendStatusReceived); err != nil {
		return err
	}
	return s.appendConversationEvent(conversation.ID, content, string(item.OriginData))
}

func (s *wxWorkKFInboundService) ensureConversation(base syncmsg.BaseMessage, profile map[string]any) (*models.Conversation, error) {
	externalID := strings.TrimSpace(base.ExternalUserID)
	if externalID == "" {
		return nil, errorsx.InvalidParamI18n("error.e0096")
	}
	channel, err := s.getChannelByOpenKfID(base.OpenKFID)
	if err != nil {
		return nil, err
	}

	external := s.buildExternalUser(externalID)
	conversation, err := ConversationService.Create(external, channel.ID, channel.AIAgentID)
	if err != nil {
		return nil, err
	}

	if err := s.upsertConversationMapping(
		conversation.ID,
		channel.ID,
		base.OpenKFID,
		base.ExternalUserID,
		base.ReceptionistUserID,
		enums.WxWorkKFSessionStatusActive,
		base.SendTime,
		base.MsgID,
		s.mustMarshal(profile),
	); err != nil {
		return nil, err
	}
	return conversation, nil
}

func (s *wxWorkKFInboundService) upsertConversationMapping(conversationID, channelID int64, openKfID, externalUserID, servicerUserID string, sessionStatus enums.WxWorkKFSessionStatus, sendTime uint64, lastMsgID, rawProfile string) error {
	now := time.Now()
	lastMsgTime := s.parseSendTime(sendTime)
	existing := WxWorkKFConversationService.Take("conversation_id = ?", conversationID)
	if existing != nil {
		updates := map[string]any{
			"channel_id":       channelID,
			"open_kf_id":       strings.TrimSpace(openKfID),
			"external_user_id": strings.TrimSpace(externalUserID),
			"servicer_user_id": strings.TrimSpace(servicerUserID),
			"session_status":   string(sessionStatus),
			"last_wx_msg_id":   strings.TrimSpace(lastMsgID),
			"updated_at":       now,
			"status":           enums.StatusOk,
		}
		if lastMsgTime != nil {
			updates["last_wx_msg_time"] = *lastMsgTime
		}
		if strings.TrimSpace(rawProfile) != "" {
			updates["raw_profile"] = rawProfile
		}
		return WxWorkKFConversationService.Updates(existing.ID, updates)
	}

	return WxWorkKFConversationService.Create(&models.WxWorkKFConversation{
		ConversationID: conversationID,
		ChannelID:      channelID,
		OpenKfID:       strings.TrimSpace(openKfID),
		ExternalUserID: strings.TrimSpace(externalUserID),
		ServicerUserID: strings.TrimSpace(servicerUserID),
		SessionStatus:  string(sessionStatus),
		LastWxMsgID:    strings.TrimSpace(lastMsgID),
		LastWxMsgTime:  lastMsgTime,
		RawProfile:     strings.TrimSpace(rawProfile),
		Status:         enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   0,
			CreateUserName: wxWorkKFSystemOperatorName,
			UpdatedAt:      now,
			UpdateUserID:   0,
			UpdateUserName: wxWorkKFSystemOperatorName,
		},
	})
}

func (s *wxWorkKFInboundService) createMessageRef(conversationID, messageID int64, item syncmsg.Message, direction enums.WxWorkKFMessageDirection, sendStatus enums.WxWorkKFMessageSendStatus) error {
	if WxWorkKFMessageRefService.Take("wx_msg_id = ?", item.MsgID) != nil {
		return nil
	}
	now := time.Now()
	return WxWorkKFMessageRefService.Create(&models.WxWorkKFMessageRef{
		ConversationID: conversationID,
		MessageID:      messageID,
		WxMsgID:        strings.TrimSpace(item.MsgID),
		Direction:      string(direction),
		Origin:         int(item.Origin),
		OpenKfID:       strings.TrimSpace(item.OpenKFID),
		ExternalUserID: strings.TrimSpace(item.ExternalUserID),
		SendStatus:     string(sendStatus),
		RawPayload:     string(item.OriginData),
		Status:         enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   0,
			CreateUserName: wxWorkKFSystemOperatorName,
			UpdatedAt:      now,
			UpdateUserID:   0,
			UpdateUserName: wxWorkKFSystemOperatorName,
		},
	})
}

func (s *wxWorkKFInboundService) saveNextCursor(openKfID, nextCursor string) error {
	openKfID = strings.TrimSpace(openKfID)
	if openKfID == "" {
		return errorsx.InvalidParamI18n("error.e0068")
	}
	now := time.Now()
	state := WxWorkKFSyncStateService.Take("open_kf_id = ?", openKfID)
	if state != nil {
		return WxWorkKFSyncStateService.Updates(state.ID, map[string]any{
			"next_cursor":  strings.TrimSpace(nextCursor),
			"last_sync_at": now,
			"updated_at":   now,
			"status":       enums.StatusOk,
		})
	}
	return WxWorkKFSyncStateService.Create(&models.WxWorkKFSyncState{
		OpenKfID:   openKfID,
		NextCursor: strings.TrimSpace(nextCursor),
		LastSyncAt: &now,
		Status:     enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   0,
			CreateUserName: wxWorkKFSystemOperatorName,
			UpdatedAt:      now,
			UpdateUserID:   0,
			UpdateUserName: wxWorkKFSystemOperatorName,
		},
	})
}

func (s *wxWorkKFInboundService) appendConversationEvent(conversationID int64, content, payload string) error {
	return ConversationEventLogService.Create(&models.ConversationEventLog{
		ConversationID: conversationID,
		EventType:      enums.IMEventTypeWxWorkKFEvent,
		OperatorType:   enums.IMSenderTypeSystem,
		OperatorID:     0,
		Content:        strings.TrimSpace(content),
		Payload:        strings.TrimSpace(payload),
		CreatedAt:      time.Now(),
	})
}

func (s *wxWorkKFInboundService) buildInboundAssetPayload(conversation *models.Conversation, mediaID string) (string, string, error) {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return "", "", errorsx.InvalidParamI18n("error.e0095")
	}
	channel := ChannelService.Get(conversation.ChannelID)
	workCli, err := ChannelService.GetWxWorkCliByChannel(channel)
	if err != nil {
		return "", "", err
	}
	materialCli := workCli.GetMaterial()
	data, err := materialCli.GetTempFile(mediaID)
	if err != nil {
		return "", "", err
	}
	asset, err := AssetService.UploadBytes(data, "", "", nil)
	if err != nil {
		return "", "", err
	}
	canonicalPayload, err := buildIMMessageAssetPayload(asset)
	if err != nil {
		return "", "", err
	}
	content := strings.TrimSpace(asset.Filename)
	if strs.IsBlank(content) {
		content = "[文件]"
	}
	return canonicalPayload, content, nil
}

func (s *wxWorkKFInboundService) getChannelByOpenKfID(openKfID string) (*models.Channel, error) {
	openKfID = strings.TrimSpace(openKfID)
	if openKfID == "" {
		return nil, errorsx.InvalidParamI18n("error.e0094")
	}
	channel := ChannelService.GetEnabledWxWorkKFChannelByOpenKfID(openKfID)
	if channel == nil {
		return nil, errorsx.InvalidParamI18n("error.e0231")
	}
	if channel.AIAgentID <= 0 {
		return nil, errorsx.InvalidParamI18n("error.e0098")
	}
	agent := AIAgentService.Get(channel.AIAgentID)
	if agent == nil || agent.Status != enums.StatusOk {
		return nil, errorsx.InvalidParamI18n("error.e0099")
	}
	return channel, nil
}

func (s *wxWorkKFInboundService) buildExternalUser(externalUserID string) openidentity.ExternalUser {
	return openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceWxWorkKF,
		ExternalID:     strings.TrimSpace(externalUserID),
		ExternalName:   strings.TrimSpace(externalUserID),
	}
}

func (s *wxWorkKFInboundService) buildInboundClientMsgID(wxMsgID string) string {
	return "wxwork_kf:" + strings.TrimSpace(wxMsgID)
}

func (s *wxWorkKFInboundService) buildUnsupportedContent(msgType string) string {
	switch strings.TrimSpace(msgType) {
	case "voice":
		return "[语音]"
	case "video":
		return "[视频]"
	case "location":
		return "[位置]"
	case "link":
		return "[链接]"
	case "business_card":
		return "[名片]"
	case "miniprogram":
		return "[小程序]"
	default:
		return "[" + strings.TrimSpace(msgType) + "]"
	}
}

func (s *wxWorkKFInboundService) parseSendTime(sendTime uint64) *time.Time {
	if sendTime == 0 {
		return nil
	}
	t := time.Unix(int64(sendTime), 0)
	return &t
}

func (s *wxWorkKFInboundService) mustMarshal(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (s *wxWorkKFInboundService) parseBaseMessage(raw []byte) (syncmsg.BaseMessage, error) {
	base := syncmsg.BaseMessage{}
	err := json.Unmarshal(raw, &base)
	return base, err
}

func (s *wxWorkKFInboundService) normalizeEventBaseMessage(base syncmsg.BaseMessage, openKfID, externalUserID string) syncmsg.BaseMessage {
	base.OpenKFID = strings.TrimSpace(openKfID)
	base.ExternalUserID = strings.TrimSpace(externalUserID)
	return base
}
