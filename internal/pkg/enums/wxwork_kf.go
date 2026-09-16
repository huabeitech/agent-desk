package enums

type WxWorkKFMessageDirection string

const (
	WxWorkKFMessageDirectionIn  WxWorkKFMessageDirection = "in"
	WxWorkKFMessageDirectionOut WxWorkKFMessageDirection = "out"
)

type ChannelMessageOutboxStatus string

const (
	ChannelMessageOutboxStatusPending ChannelMessageOutboxStatus = "pending"
	ChannelMessageOutboxStatusSending ChannelMessageOutboxStatus = "sending"
	ChannelMessageOutboxStatusSent    ChannelMessageOutboxStatus = "sent"
	ChannelMessageOutboxStatusFailed  ChannelMessageOutboxStatus = "failed"
	ChannelMessageOutboxStatusIgnored ChannelMessageOutboxStatus = "ignored"
)

const (
	ChannelTypeWeb      = "web"
	ChannelTypeWechatMP = "wechat_mp"
	ChannelTypeWxWorkKF = "wxwork_kf"
	ChannelTypeTelegram = "telegram"
	ChannelTypeZaloOA   = "zalo_oa"
	ChannelTypeDiscord  = "discord"
)

type WxWorkKFMessageSendStatus string

const (
	WxWorkKFMessageSendStatusReceived WxWorkKFMessageSendStatus = "received"
	WxWorkKFMessageSendStatusSent     WxWorkKFMessageSendStatus = "sent"
	WxWorkKFMessageSendStatusFailed   WxWorkKFMessageSendStatus = "failed"
)

type WxWorkKFSessionStatus string

const (
	WxWorkKFSessionStatusActive   WxWorkKFSessionStatus = "active"
	WxWorkKFSessionStatusTransfer WxWorkKFSessionStatus = "transfer"
	WxWorkKFSessionStatusClosed   WxWorkKFSessionStatus = "closed"
)

const (
	WxWorkKFEventTypeEnterSession        = "enter_session"
	WxWorkKFEventTypeSessionStatusChange = "session_status_change"
	WxWorkKFEventTypeMsgSendFail         = "msg_send_fail"
)

const (
	IMEventTypeWxWorkKFSync     IMEventType = "wxwork_kf_sync"
	IMEventTypeWxWorkKFEvent    IMEventType = "wxwork_kf_event"
	IMEventTypeWxWorkKFOutbound IMEventType = "wxwork_kf_outbound"
)
