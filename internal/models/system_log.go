package models

import "time"

// SystemLog 系统日志记录，持久化 INFO/WARN/ERROR 级别的运行日志。
type SystemLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	Level      string    `gorm:"type:varchar(20);not null;default:'';index"`  // INFO / WARN / ERROR
	Message    string    `gorm:"type:text"`                                   // 日志消息
	Source     string    `gorm:"type:varchar(255);not null;default:'';index"` // file:line
	LoggerName string    `gorm:"type:varchar(128);not null;default:'';index"` // logger 名称
	Attrs      string    `gorm:"type:text"`                                   // JSON 序列化的 attrs
	CreatedAt  time.Time `gorm:"not null;index"`                              // 日志发生时间
}

func (SystemLog) TableName() string {
	return "t_system_log"
}
