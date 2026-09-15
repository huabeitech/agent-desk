package models

import "strings"

// AI 回复占位与超时的系统默认文案与上限；接入渠道可通过对应字段按渠道覆盖。
const (
	DefaultAIReplyPlaceholder    = "正在查看，请稍候……"
	DefaultAIReplyTimeoutSeconds = 120
	MaxAIReplyTimeoutSeconds     = 600
	DefaultAIReplyTimeoutNotice  = "抱歉，AI 客服暂时未能处理您的问题，请重新发送问题或转人工客服。"
)

// EffectiveAIReplyPlaceholder 返回渠道生效的 AI 接待占位提示语。
func (c *Channel) EffectiveAIReplyPlaceholder() string {
	if c != nil {
		if value := strings.TrimSpace(c.AIReplyPlaceholder); value != "" {
			return value
		}
	}
	return DefaultAIReplyPlaceholder
}

// EffectiveAIReplyTimeoutNotice 返回渠道生效的 AI 回复超时/失败提示语。
func (c *Channel) EffectiveAIReplyTimeoutNotice() string {
	if c != nil {
		if value := strings.TrimSpace(c.AIReplyTimeoutNotice); value != "" {
			return value
		}
	}
	return DefaultAIReplyTimeoutNotice
}
