package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/repositories"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
)

var ChannelMessageOutboxService = newChannelMessageOutboxService()

func newChannelMessageOutboxService() *channelMessageOutboxService {
	return &channelMessageOutboxService{}
}

type channelMessageOutboxService struct {
}

func (s *channelMessageOutboxService) Get(id int64) *models.ChannelMessageOutbox {
	return repositories.ChannelMessageOutboxRepository.Get(sqls.DB(), id)
}

func (s *channelMessageOutboxService) Take(where ...interface{}) *models.ChannelMessageOutbox {
	return repositories.ChannelMessageOutboxRepository.Take(sqls.DB(), where...)
}

func (s *channelMessageOutboxService) Find(cnd *sqls.Cnd) []models.ChannelMessageOutbox {
	return repositories.ChannelMessageOutboxRepository.Find(sqls.DB(), cnd)
}

func (s *channelMessageOutboxService) FindOne(cnd *sqls.Cnd) *models.ChannelMessageOutbox {
	return repositories.ChannelMessageOutboxRepository.FindOne(sqls.DB(), cnd)
}

func (s *channelMessageOutboxService) FindPageByParams(params *params.QueryParams) (list []models.ChannelMessageOutbox, paging *sqls.Paging) {
	return repositories.ChannelMessageOutboxRepository.FindPageByParams(sqls.DB(), params)
}

func (s *channelMessageOutboxService) FindPageByCnd(cnd *sqls.Cnd) (list []models.ChannelMessageOutbox, paging *sqls.Paging) {
	return repositories.ChannelMessageOutboxRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *channelMessageOutboxService) Count(cnd *sqls.Cnd) int64 {
	return repositories.ChannelMessageOutboxRepository.Count(sqls.DB(), cnd)
}

func (s *channelMessageOutboxService) Create(t *models.ChannelMessageOutbox) error {
	return repositories.ChannelMessageOutboxRepository.Create(sqls.DB(), t)
}

func (s *channelMessageOutboxService) Update(t *models.ChannelMessageOutbox) error {
	return repositories.ChannelMessageOutboxRepository.Update(sqls.DB(), t)
}

func (s *channelMessageOutboxService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.ChannelMessageOutboxRepository.Updates(sqls.DB(), id, columns)
}

func (s *channelMessageOutboxService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.ChannelMessageOutboxRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *channelMessageOutboxService) Delete(id int64) {
	repositories.ChannelMessageOutboxRepository.Delete(sqls.DB(), id)
}

// GetByMessageID retrieves the outbox entry by message ID and channel type.
func (s *channelMessageOutboxService) GetByMessageID(channelType string, messageID int64) *models.ChannelMessageOutbox {
	return repositories.ChannelMessageOutboxRepository.Take(sqls.DB(), "channel_type = ? AND message_id = ?", channelType, messageID)
}

func (s *channelMessageOutboxService) EnqueueWxWorkKFMessage(conversation *models.Conversation, message *models.Message) error {
	if conversation == nil || message == nil {
		return nil
	}
	channel := ChannelService.Get(conversation.ChannelID)
	if channel == nil || channel.ChannelType != enums.ChannelTypeWxWorkKF {
		return nil
	}
	if message.SenderType != enums.IMSenderTypeAgent && message.SenderType != enums.IMSenderTypeAI {
		return nil
	}
	if message.MessageType != enums.IMMessageTypeText && message.MessageType != enums.IMMessageTypeHTML {
		return nil
	}
	if existing := s.GetByMessageID(enums.ChannelTypeWxWorkKF, message.ID); existing != nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"conversationId": conversation.ID,
		"messageId":      message.ID,
		"messageType":    message.MessageType,
		"content":        strings.TrimSpace(message.Content),
		"payload":        strings.TrimSpace(message.Payload),
		"senderId":       message.SenderID,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	return s.Create(&models.ChannelMessageOutbox{
		ChannelType:    enums.ChannelTypeWxWorkKF,
		ConversationID: conversation.ID,
		MessageID:      message.ID,
		Payload:        string(payload),
		SendStatus:     string(enums.ChannelMessageOutboxStatusPending),
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   message.UpdateUserID,
			CreateUserName: message.UpdateUserName,
			UpdatedAt:      now,
			UpdateUserID:   message.UpdateUserID,
			UpdateUserName: message.UpdateUserName,
		},
	})
}

func (s *channelMessageOutboxService) EnqueueTelegramMessage(conversation *models.Conversation, message *models.Message) error {
	if conversation == nil || message == nil {
		return nil
	}
	channel := ChannelService.Get(conversation.ChannelID)
	if channel == nil || channel.ChannelType != enums.ChannelTypeTelegram {
		return nil
	}
	if message.SenderType != enums.IMSenderTypeAgent && message.SenderType != enums.IMSenderTypeAI {
		return nil
	}
	if message.MessageType != enums.IMMessageTypeText && message.MessageType != enums.IMMessageTypeHTML {
		return nil
	}
	if existing := s.GetByMessageID(enums.ChannelTypeTelegram, message.ID); existing != nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"conversationId": conversation.ID,
		"messageId":      message.ID,
		"messageType":    message.MessageType,
		"content":        strings.TrimSpace(message.Content),
		"payload":        strings.TrimSpace(message.Payload),
		"senderId":       message.SenderID,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	err = s.Create(&models.ChannelMessageOutbox{
		ChannelType:    enums.ChannelTypeTelegram,
		ConversationID: conversation.ID,
		MessageID:      message.ID,
		Payload:        string(payload),
		SendStatus:     string(enums.ChannelMessageOutboxStatusPending),
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   message.UpdateUserID,
			CreateUserName: message.UpdateUserName,
			UpdatedAt:      now,
			UpdateUserID:   message.UpdateUserID,
			UpdateUserName: message.UpdateUserName,
		},
	})
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in telegram outbound dispatch", "error", r)
			}
		}()
		TelegramOutboundService.DispatchPendingOutbox()
	}()

	return nil
}

func (s *channelMessageOutboxService) EnqueueZaloOAMessage(conversation *models.Conversation, message *models.Message) error {
	if conversation == nil || message == nil {
		return nil
	}
	channel := ChannelService.Get(conversation.ChannelID)
	if channel == nil || channel.ChannelType != enums.ChannelTypeZaloOA {
		return nil
	}
	if message.SenderType != enums.IMSenderTypeAgent && message.SenderType != enums.IMSenderTypeAI {
		return nil
	}
	if message.MessageType != enums.IMMessageTypeText && message.MessageType != enums.IMMessageTypeHTML {
		return nil
	}
	if existing := s.GetByMessageID(enums.ChannelTypeZaloOA, message.ID); existing != nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"conversationId": conversation.ID,
		"messageId":      message.ID,
		"messageType":    message.MessageType,
		"content":        strings.TrimSpace(message.Content),
		"payload":        strings.TrimSpace(message.Payload),
		"senderId":       message.SenderID,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	err = s.Create(&models.ChannelMessageOutbox{
		ChannelType:    enums.ChannelTypeZaloOA,
		ConversationID: conversation.ID,
		MessageID:      message.ID,
		Payload:        string(payload),
		SendStatus:     string(enums.ChannelMessageOutboxStatusPending),
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   message.UpdateUserID,
			CreateUserName: message.UpdateUserName,
			UpdatedAt:      now,
			UpdateUserID:   message.UpdateUserID,
			UpdateUserName: message.UpdateUserName,
		},
	})
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in zalo oa outbound dispatch", "error", r)
			}
		}()
		ZaloOAOutboundService.DispatchPendingOutbox()
	}()

	return nil
}

func (s *channelMessageOutboxService) EnqueueDiscordMessage(conversation *models.Conversation, message *models.Message) error {
	if conversation == nil || message == nil {
		return nil
	}
	channel := ChannelService.Get(conversation.ChannelID)
	if channel == nil || channel.ChannelType != enums.ChannelTypeDiscord {
		return nil
	}
	if message.SenderType != enums.IMSenderTypeAgent && message.SenderType != enums.IMSenderTypeAI {
		return nil
	}
	if message.MessageType != enums.IMMessageTypeText && message.MessageType != enums.IMMessageTypeHTML && message.MessageType != enums.IMMessageTypeImage && message.MessageType != enums.IMMessageTypeAttachment {
		return nil
	}
	if existing := s.GetByMessageID(enums.ChannelTypeDiscord, message.ID); existing != nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"conversationId": conversation.ID,
		"messageId":      message.ID,
		"messageType":    message.MessageType,
		"content":        strings.TrimSpace(message.Content),
		"payload":        strings.TrimSpace(message.Payload),
		"senderId":       message.SenderID,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	err = s.Create(&models.ChannelMessageOutbox{
		ChannelType:    enums.ChannelTypeDiscord,
		ConversationID: conversation.ID,
		MessageID:      message.ID,
		Payload:        string(payload),
		SendStatus:     string(enums.ChannelMessageOutboxStatusPending),
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   message.UpdateUserID,
			CreateUserName: message.UpdateUserName,
			UpdatedAt:      now,
			UpdateUserID:   message.UpdateUserID,
			UpdateUserName: message.UpdateUserName,
		},
	})
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in discord outbound dispatch", "error", r)
			}
		}()
		DiscordOutboundService.DispatchPendingOutbox()
	}()

	return nil
}

func (s *channelMessageOutboxService) ListPending(channelType string, limit int) []models.ChannelMessageOutbox {
	if limit <= 0 {
		limit = 20
	}
	// 只取"现在就可以尝试发送"的记录，保证每批取出的记录都会被真正尝试、投递循环必然收敛：
	//   pending：无重试计划限制（新入队或人工重试后 next_retry_at 为空）；
	//   failed：必须存在重试计划且已到期。达到最大重试次数（next_retry_at 为空）的记录
	//   停止自动重试，等待管理端人工处置；退避窗口内的记录不占用批次。
	now := time.Now()
	cnd := sqls.NewCnd().
		Eq("channel_type", strings.TrimSpace(channelType)).
		Where(
			"((send_status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) "+
				"OR (send_status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?))",
			string(enums.ChannelMessageOutboxStatusPending), now,
			string(enums.ChannelMessageOutboxStatusFailed), now,
		).
		// Only rows whose backoff has elapsed are eligible; ordering by
		// next_retry_at keeps a backlog of not-yet-due retries from starving
		// newer pending sends.
		Lte("next_retry_at", now).
		Asc("next_retry_at").
		Asc("id").
		Limit(limit)
	return s.Find(cnd)
}

func (s *channelMessageOutboxService) RetryWxWorkFailure(id int64, operator *dto.AuthPrincipal) error {
	item := s.Get(id)
	if item == nil || item.ChannelType != enums.ChannelTypeWxWorkKF {
		return errorsx.InvalidParam("outbox record does not exist")
	}
	if item.SendStatus != string(enums.ChannelMessageOutboxStatusFailed) &&
		item.SendStatus != string(enums.ChannelMessageOutboxStatusIgnored) {
		return errorsx.InvalidParam("only failed or ignored outbox records can be retried")
	}
	now := time.Now()
	columns := map[string]interface{}{
		"send_status":   string(enums.ChannelMessageOutboxStatusPending),
		"next_retry_at": nil,
		"updated_at":    now,
	}
	if operator != nil {
		columns["update_user_id"] = operator.UserID
		columns["update_user_name"] = operator.Username
	}
	return s.Updates(id, columns)
}

func (s *channelMessageOutboxService) IgnoreWxWorkFailure(id int64, operator *dto.AuthPrincipal) error {
	item := s.Get(id)
	if item == nil || item.ChannelType != enums.ChannelTypeWxWorkKF {
		return errorsx.InvalidParam("outbox record does not exist")
	}
	if item.SendStatus != string(enums.ChannelMessageOutboxStatusFailed) {
		return errorsx.InvalidParam("only failed outbox records can be ignored")
	}
	now := time.Now()
	columns := map[string]interface{}{
		"send_status":   string(enums.ChannelMessageOutboxStatusIgnored),
		"next_retry_at": nil,
		"updated_at":    now,
	}
	if operator != nil {
		columns["update_user_id"] = operator.UserID
		columns["update_user_name"] = operator.Username
	}
	return s.Updates(id, columns)
}
