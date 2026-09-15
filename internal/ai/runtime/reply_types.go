package runtime

import "agent-desk/internal/models"

// maxAIReplyAsyncTimeoutSeconds 为 AI 回复超时的上限；默认时长由 models.DefaultAIReplyTimeoutSeconds 提供。
const maxAIReplyAsyncTimeoutSeconds = models.MaxAIReplyTimeoutSeconds
