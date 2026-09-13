package enums

// ExternalSource 外部身份来源。
//
// 与 ExternalID 组合即可唯一标识某渠道下的访客身份。
type ExternalSource string

const (
	ExternalSourceGuest    ExternalSource = "guest"     // 访客
	ExternalSourceWxWorkKF ExternalSource = "wxwork_kf" // 企业微信客服
	ExternalSourceUser     ExternalSource = "user"      // 用户信息
	ExternalSourceTelegram ExternalSource = "telegram"  // Telegram
	ExternalSourceZaloOA   ExternalSource = "zalo_oa"   // Zalo OA
	ExternalSourceSlack    ExternalSource = "slack"     // Slack
)

var externalSourceLabelMap = map[ExternalSource]string{
	ExternalSourceGuest:    "访客",
	ExternalSourceWxWorkKF: "企业微信客服",
	ExternalSourceUser:     "用户",
	ExternalSourceTelegram: "Telegram",
	ExternalSourceZaloOA:   "Zalo OA",
	ExternalSourceSlack:    "Slack",
}

func GetExternalSourceLabel(v ExternalSource) string {
	if s, ok := externalSourceLabelMap[v]; ok {
		return s
	}
	return string(v)
}
