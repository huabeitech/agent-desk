package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"agent-desk/internal/wxwork"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"github.com/silenceper/wechat/v2/work/kf/sendmsg"
)

const (
	wxWorkKFOutboxBatchSize = 20
	wxWorkKFOutboxMaxRetry  = 6
)

var WxWorkKFOutboundService = newWxWorkKFOutboundService()

func newWxWorkKFOutboundService() *wxWorkKFOutboundService {
	return &wxWorkKFOutboundService{}
}

type wxWorkKFOutboundService struct {
}

type wxWorkKFOutboundChunk struct {
	MessageType enums.IMMessageType
	Content     string
	AssetID     string
}

func (s *wxWorkKFOutboundService) DispatchPendingOutbox() int {
	if !wxwork.Enabled() {
		return 0
	}

	var totalCount int = 0
	for {
		count := s.doDispatchPendingOutbox(wxWorkKFOutboxBatchSize)

		totalCount += count
		if count > 0 {
			slog.Info("wxwork kf outbound dispatch loop",
				"batch_count", count,
				"total_count", totalCount,
			)
		}

		if count == 0 {
			break
		}
	}
	return totalCount
}

func (s *wxWorkKFOutboundService) doDispatchPendingOutbox(limit int) int {
	if !wxwork.Enabled() {
		return 0
	}
	if limit <= 0 {
		limit = wxWorkKFOutboxBatchSize
	}

	items := ChannelMessageOutboxService.ListPending(enums.ChannelTypeWxWorkKF, limit)
	if len(items) == 0 {
		return 0
	}

	successCount := 0
	for i := range items {
		if err := s.processOutbox(items[i].ID); err != nil {
			slog.Warn("process wxwork kf outbox failed",
				"outbox_id", items[i].ID,
				"conversation_id", items[i].ConversationID,
				"message_id", items[i].MessageID,
				"error", err,
			)
			continue
		}
		successCount++
	}
	return successCount
}

func (s *wxWorkKFOutboundService) processOutbox(outboxID int64) error {
	outbox := ChannelMessageOutboxService.Get(outboxID)
	if outbox == nil {
		return nil
	}
	if outbox.ChannelType != enums.ChannelTypeWxWorkKF {
		return nil
	}
	if outbox.SendStatus == string(enums.ChannelMessageOutboxStatusSent) {
		return nil
	}
	if outbox.NextRetryAt != nil && outbox.NextRetryAt.After(time.Now()) {
		return nil
	}

	slog.Info("processing wxwork kf outbox",
		"outbox_id", outbox.ID,
		"conversation_id", outbox.ConversationID,
		"message_id", outbox.MessageID,
		"send_status", outbox.SendStatus,
		"retry_count", outbox.RetryCount,
	)

	now := time.Now()
	if err := ChannelMessageOutboxService.Updates(outbox.ID, map[string]any{
		"send_status":      string(enums.ChannelMessageOutboxStatusSending),
		"updated_at":       now,
		"update_user_id":   outbox.UpdateUserID,
		"update_user_name": outbox.UpdateUserName,
	}); err != nil {
		return err
	}

	message := MessageService.Get(outbox.MessageID)
	if message == nil {
		return s.markOutboxFailed(outbox, "平台消息不存在")
	}
	conversation := ConversationService.Get(outbox.ConversationID)
	if conversation == nil {
		return s.markOutboxFailed(outbox, "平台会话不存在")
	}
	mapping := WxWorkKFConversationService.Take("conversation_id = ?", conversation.ID)
	if mapping == nil {
		return s.markOutboxFailed(outbox, "企业微信会话映射不存在")
	}
	if mapping.ChannelID <= 0 {
		return s.markOutboxFailed(outbox, "企业微信会话映射缺少渠道ID")
	}
	channel := ChannelService.Get(mapping.ChannelID)
	if channel == nil || channel.Status != enums.StatusOk || channel.ChannelType != enums.ChannelTypeWxWorkKF {
		return s.markOutboxFailed(outbox, "企业微信接入渠道不存在、未启用或类型不匹配")
	}
	if strings.TrimSpace(mapping.OpenKfID) == "" || strings.TrimSpace(mapping.ExternalUserID) == "" {
		return s.markOutboxFailed(outbox, "企业微信会话映射缺少发送必要参数")
	}
	chunks, buildErr := s.buildOutboundChunks(message)
	if buildErr != nil {
		return s.markOutboxFailed(outbox, buildErr.Error())
	}
	if len(chunks) == 0 {
		return s.markOutboxFailed(outbox, "当前消息无法转换为企业微信下行消息")
	}

	slog.Info("built wxwork outbound chunks",
		"outbox_id", outbox.ID,
		"conversation_id", conversation.ID,
		"message_id", message.ID,
		"sender_type", message.SenderType,
		"message_type", message.MessageType,
		"chunk_count", len(chunks),
		"open_kfid", mapping.OpenKfID,
		"external_userid", mapping.ExternalUserID,
	)

	wxMsgIDs := make([]string, 0, len(chunks))
	for i := range chunks {
		wxMsgID, sendErr := s.sendOutboundChunk(channel, mapping, message, chunks[i], i)
		if sendErr != nil {
			return s.markOutboxFailed(outbox, sendErr.Error())
		}
		wxMsgIDs = append(wxMsgIDs, wxMsgID)
	}

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		now = time.Now()
		if err := repositories.ChannelMessageOutboxRepository.Updates(ctx.Tx, outbox.ID, map[string]any{
			"send_status":      string(enums.ChannelMessageOutboxStatusSent),
			"sent_at":          now,
			"last_error":       "",
			"updated_at":       now,
			"update_user_id":   outbox.UpdateUserID,
			"update_user_name": outbox.UpdateUserName,
		}); err != nil {
			return err
		}
		if existing := repositories.WxWorkKFMessageRefRepository.Take(ctx.Tx, "message_id = ? AND direction = ?", message.ID, string(enums.WxWorkKFMessageDirectionOut)); existing == nil {
			for i := range wxMsgIDs {
				rawPayload := strings.TrimSpace(outbox.Payload)
				if len(chunks) > i {
					if payload, err := json.Marshal(map[string]any{
						"messageId":    message.ID,
						"chunkIndex":   i,
						"chunkType":    chunks[i].MessageType,
						"chunkText":    strings.TrimSpace(chunks[i].Content),
						"chunkAssetId": strings.TrimSpace(chunks[i].AssetID),
					}); err == nil {
						rawPayload = string(payload)
					}
				}
				if err := repositories.WxWorkKFMessageRefRepository.Create(ctx.Tx, &models.WxWorkKFMessageRef{
					ConversationID: conversation.ID,
					MessageID:      message.ID,
					WxMsgID:        strings.TrimSpace(wxMsgIDs[i]),
					Direction:      string(enums.WxWorkKFMessageDirectionOut),
					Origin:         0,
					OpenKfID:       mapping.OpenKfID,
					ExternalUserID: mapping.ExternalUserID,
					SendStatus:     string(enums.WxWorkKFMessageSendStatusSent),
					RawPayload:     rawPayload,
					Status:         enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   outbox.UpdateUserID,
						CreateUserName: outbox.UpdateUserName,
						UpdatedAt:      now,
						UpdateUserID:   outbox.UpdateUserID,
						UpdateUserName: outbox.UpdateUserName,
					},
				}); err != nil {
					return err
				}
			}
		}
		return ConversationEventLogService.CreateEvent(ctx, conversation.ID, enums.IMEventTypeWxWorkKFOutbound, message.SenderType, message.SenderID, fmt.Sprintf("企业微信消息发送成功，共%d条", len(wxMsgIDs)), "")
	})
}

func (s *wxWorkKFOutboundService) sendOutboundChunk(channel *models.Channel, mapping *models.WxWorkKFConversation, message *models.Message, chunk wxWorkKFOutboundChunk, chunkIndex int) (string, error) {
	switch chunk.MessageType {
	case enums.IMMessageTypeText:
		return s.sendTextMessage(channel, mapping, message, chunk.Content, chunkIndex)
	case enums.IMMessageTypeImage:
		return s.sendImageMessage(channel, mapping, message, chunk, chunkIndex)
	default:
		return "", i18nx.Errorf("error.wxwork.unsupportedOutboundMessageType", chunk.MessageType)
	}
}

func (s *wxWorkKFOutboundService) sendTextMessage(channel *models.Channel, mapping *models.WxWorkKFConversation, message *models.Message, content string, chunkIndex int) (string, error) {
	workCli, err := ChannelService.GetWxWorkCliByChannel(channel)
	if err != nil {
		return "", err
	}
	cli, err := workCli.GetKF()
	if err != nil {
		return "", err
	}

	req := sendmsg.Text{}
	req.Message.ToUser = strings.TrimSpace(mapping.ExternalUserID)
	req.Message.OpenKFID = strings.TrimSpace(mapping.OpenKfID)
	req.Message.MsgID = s.buildOutboundClientMsgID(message.ID, chunkIndex)
	req.MsgType = "text"
	req.Text.Content = strings.TrimSpace(content)

	slog.Info("sending wxwork text message",
		"conversation_id", message.ConversationID,
		"message_id", message.ID,
		"chunk_index", chunkIndex,
		"client_msg_id", req.Message.MsgID,
		"open_kfid", req.Message.OpenKFID,
		"external_userid", req.Message.ToUser,
		"content_length", len([]rune(req.Text.Content)),
	)

	resp, err := cli.SendMsg(req)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resp.MsgID) == "" {
		return "", i18nx.Errorf("error.e0114")
	}
	slog.Info("wxwork text message accepted",
		"conversation_id", message.ConversationID,
		"message_id", message.ID,
		"chunk_index", chunkIndex,
		"client_msg_id", req.Message.MsgID,
		"wx_msg_id", strings.TrimSpace(resp.MsgID),
		"open_kfid", req.Message.OpenKFID,
		"external_userid", req.Message.ToUser,
	)
	return strings.TrimSpace(resp.MsgID), nil
}

func (s *wxWorkKFOutboundService) sendImageMessage(channel *models.Channel, mapping *models.WxWorkKFConversation, message *models.Message, chunk wxWorkKFOutboundChunk, chunkIndex int) (string, error) {
	if strings.TrimSpace(chunk.AssetID) == "" {
		return "", i18nx.Errorf("error.e0145")
	}

	asset := AssetService.GetByAssetID(chunk.AssetID)
	if asset == nil {
		return "", i18nx.Errorf("error.e0146")
	}
	fileReader, err := AssetService.OpenReader(asset)
	if err != nil {
		return "", err
	}
	defer func() {
		if fileReader != nil {
			_ = fileReader.Close()
		}
	}()

	slog.Info("sending wxwork image message",
		"conversation_id", message.ConversationID,
		"message_id", message.ID,
		"chunk_index", chunkIndex,
		"asset_id", asset.AssetID,
		"filename", asset.Filename,
		"storage_key", asset.StorageKey,
		"open_kfid", mapping.OpenKfID,
		"external_userid", mapping.ExternalUserID,
	)

	workCli, err := ChannelService.GetWxWorkCliByChannel(channel)
	if err != nil {
		return "", err
	}
	materialCli := workCli.GetMaterial()
	uploadResp, err := materialCli.UploadTempFileFromReader(asset.Filename, "image", fileReader)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(uploadResp.MediaID) == "" {
		return "", i18nx.Errorf("error.e0113")
	}

	kfCli, err := workCli.GetKF()
	if err != nil {
		return "", err
	}
	req := sendmsg.Image{
		Message: sendmsg.Message{
			ToUser:   strings.TrimSpace(mapping.ExternalUserID),
			OpenKFID: strings.TrimSpace(mapping.OpenKfID),
			MsgID:    s.buildOutboundClientMsgID(message.ID, chunkIndex),
		},
		MsgType: "image",
	}
	req.Image.MediaID = strings.TrimSpace(uploadResp.MediaID)

	resp, err := kfCli.SendMsg(req)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resp.MsgID) == "" {
		return "", i18nx.Errorf("error.e0114")
	}
	slog.Info("wxwork image message accepted",
		"conversation_id", message.ConversationID,
		"message_id", message.ID,
		"chunk_index", chunkIndex,
		"client_msg_id", req.Message.MsgID,
		"wx_msg_id", strings.TrimSpace(resp.MsgID),
		"media_id", req.Image.MediaID,
		"asset_id", asset.AssetID,
		"open_kfid", req.Message.OpenKFID,
		"external_userid", req.Message.ToUser,
	)
	return strings.TrimSpace(resp.MsgID), nil
}

func (s *wxWorkKFOutboundService) markOutboxFailed(outbox *models.ChannelMessageOutbox, errMsg string) error {
	if outbox == nil {
		return nil
	}
	slog.Warn("mark wxwork kf outbox failed",
		"outbox_id", outbox.ID,
		"conversation_id", outbox.ConversationID,
		"message_id", outbox.MessageID,
		"retry_count", outbox.RetryCount+1,
		"error", strings.TrimSpace(errMsg),
	)
	now := time.Now()
	retryCount := outbox.RetryCount + 1
	nextRetryAt := s.nextRetryAt(retryCount)
	status := string(enums.ChannelMessageOutboxStatusFailed)
	if retryCount >= wxWorkKFOutboxMaxRetry {
		nextRetryAt = nil
	}
	return ChannelMessageOutboxService.Updates(outbox.ID, map[string]any{
		"send_status":      status,
		"retry_count":      retryCount,
		"next_retry_at":    nextRetryAt,
		"last_error":       strings.TrimSpace(errMsg),
		"updated_at":       now,
		"update_user_id":   outbox.UpdateUserID,
		"update_user_name": outbox.UpdateUserName,
	})
}

func (s *wxWorkKFOutboundService) nextRetryAt(retryCount int) *time.Time {
	delay := time.Minute
	switch {
	case retryCount <= 1:
		delay = 30 * time.Second
	case retryCount == 2:
		delay = time.Minute
	case retryCount == 3:
		delay = 2 * time.Minute
	default:
		delay = 5 * time.Minute
	}
	t := time.Now().Add(delay)
	return &t
}

func (s *wxWorkKFOutboundService) buildOutboundClientMsgID(messageID int64, chunkIndex int) string {
	return fmt.Sprintf("outbox_wxwork_kf_%d_%d", messageID, chunkIndex)
}

type wxWorkKFOutboundPayload struct {
	ConversationID int64               `json:"conversationId"`
	MessageID      int64               `json:"messageId"`
	MessageType    enums.IMMessageType `json:"messageType"`
	Content        string              `json:"content"`
	Payload        string              `json:"payload"`
	SenderID       int64               `json:"senderId"`
}

func (s *wxWorkKFOutboundService) parseOutboxPayload(raw string) (*wxWorkKFOutboundPayload, error) {
	payload := &wxWorkKFOutboundPayload{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *wxWorkKFOutboundService) buildOutboundChunks(message *models.Message) ([]wxWorkKFOutboundChunk, error) {
	if message == nil {
		return nil, i18nx.Errorf("error.e0186")
	}
	switch message.MessageType {
	case enums.IMMessageTypeText:
		content := strings.TrimSpace(message.Content)
		if content == "" {
			return nil, i18nx.Errorf("error.e0217")
		}
		return []wxWorkKFOutboundChunk{{MessageType: enums.IMMessageTypeText, Content: content}}, nil
	case enums.IMMessageTypeHTML:
		return s.buildHTMLChunks(message.Content)
	default:
		return nil, i18nx.Errorf("error.wxwork.currentUnsupportedOutboundMessageType", message.MessageType)
	}
}

func (s *wxWorkKFOutboundService) buildHTMLChunks(content string) ([]wxWorkKFOutboundChunk, error) {
	contentChunks, err := utils.SplitHTMLContentChunks(content)
	if err != nil {
		return nil, err
	}
	chunks := make([]wxWorkKFOutboundChunk, 0, len(contentChunks))
	for _, chunk := range contentChunks {
		switch chunk.Type {
		case utils.ContentChunkTypeText:
			if strs.IsNotBlank(chunk.Content) {
				chunks = append(chunks, wxWorkKFOutboundChunk{
					MessageType: enums.IMMessageTypeText,
					Content:     chunk.Content,
				})
			}
		case utils.ContentChunkTypeImage:
			if strs.IsBlank(chunk.AssetID) {
				chunks = append(chunks, wxWorkKFOutboundChunk{
					MessageType: enums.IMMessageTypeText,
					Content:     "[图片]",
				})
			} else {
				chunks = append(chunks, wxWorkKFOutboundChunk{
					MessageType: enums.IMMessageTypeImage,
					AssetID:     chunk.AssetID,
				})
			}
		}
	}
	if len(chunks) == 0 {
		return nil, i18nx.Errorf("error.e0029")
	}
	return chunks, nil
}
