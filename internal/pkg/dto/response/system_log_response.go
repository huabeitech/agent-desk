package response

import "time"

// SystemLogResponse 系统日志列表/详情响应。
type SystemLogResponse struct {
	ID         int64     `json:"id"`
	Level      string    `json:"level"`
	LevelName  string    `json:"levelName"`
	Message    string    `json:"message"`
	Source     string    `json:"source"`
	LoggerName string    `json:"loggerName"`
	Attrs      string    `json:"attrs"`
	CreatedAt  time.Time `json:"createdAt"`
}
