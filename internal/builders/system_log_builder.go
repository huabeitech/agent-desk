package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
)

// BuildSystemLog 将 SystemLog 模型映射为响应 DTO。
func BuildSystemLog(item *models.SystemLog) response.SystemLogResponse {
	return response.SystemLogResponse{
		ID:         item.ID,
		Level:      item.Level,
		LevelName:  systemLogLevelName(item.Level),
		Message:    item.Message,
		Source:     item.Source,
		LoggerName: item.LoggerName,
		Attrs:      item.Attrs,
		CreatedAt:  item.CreatedAt,
	}
}

func systemLogLevelName(level string) string {
	switch level {
	case "INFO":
		return "Info"
	case "WARN":
		return "Warn"
	case "ERROR":
		return "Error"
	default:
		return level
	}
}
