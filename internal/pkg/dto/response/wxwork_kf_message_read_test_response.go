package response

// WxWorkKFMessageReadTestResult 为“测试读取企业微信客服消息”的结果。
// 该接口是只读连通性测试：不消费消息、不创建会话、不更新同步游标；
// 调用失败也以 HTTP 200 返回结构化结果，由前端按错误码给出排查建议。
type WxWorkKFMessageReadTestResult struct {
	Success      bool   `json:"success"`                // 是否成功读取
	Stage        string `json:"stage,omitempty"`        // 失败环节：gettoken / syncmsg
	ErrorCode    string `json:"errorCode,omitempty"`    // 企业微信原始 errcode，或本地错误码
	ErrorMessage string `json:"errorMessage,omitempty"` // 企业微信原始 errmsg 或网络错误原文
	OpenKfID     string `json:"openKfId,omitempty"`     // 测试使用的客服账号 ID

	TotalScanned int                         `json:"totalScanned"` // 本次扫描到的消息/事件总条数（受分页上限约束）
	MessageCount int                         `json:"messageCount"` // 普通消息条数
	EventCount   int                         `json:"eventCount"`   // 事件条数
	Truncated    bool                        `json:"truncated"`    // 是否因达到分页扫描上限而提前停止
	EarliestTime string                      `json:"earliestTime,omitempty"`
	LatestTime   string                      `json:"latestTime,omitempty"`
	Samples      []WxWorkKFMessageReadSample `json:"samples"` // 最近若干条样例（新消息在前）
}

// WxWorkKFMessageReadSample 为测试结果中的单条消息/事件样例，仅携带展示所需的原始字段，本地化在前端完成。
type WxWorkKFMessageReadSample struct {
	MsgID          string `json:"msgId,omitempty"`
	SendTime       string `json:"sendTime,omitempty"`
	Origin         int    `json:"origin"`                   // 3-客户发送 4-系统事件 5-接待人员发送
	MsgType        string `json:"msgType,omitempty"`        // text/image/file/voice/video/event 等
	EventType      string `json:"eventType,omitempty"`      // msgtype=event 时的事件类型
	TextContent    string `json:"textContent,omitempty"`    // 文本消息原文（已截断）
	ExternalUserID string `json:"externalUserId,omitempty"` // 客户 external_userid
	ServicerUserID string `json:"servicerUserId,omitempty"` // 接待人员 userid
}
