package logx

import (
	"log/slog"
	"os"
	"strings"

	"agent-desk/internal/models"
)

// Config 定义日志初始化参数。
type Config struct {
	Level         string `yaml:"level"`
	Format        string `yaml:"format"`
	AddSource     bool   `yaml:"addSource"`
	EnableDBSink bool   `yaml:"enableDBSink"`
}

// Init 初始化全局 slog logger，并设置为默认 logger。
// 当 EnableDBSink 为 true 时，创建复合 handler（stdout + dbSink），
// dbSink 的 DB 引用在 AttachDB 调用前为 nil，不会写库。
func Init(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // 强制开启，以便 DB sink 记录 source
	}

	var stdoutHandler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		stdoutHandler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		stdoutHandler = slog.NewTextHandler(os.Stdout, opts)
	}

	var handler slog.Handler = stdoutHandler

	if cfg.EnableDBSink {
		sink = &dbSink{
			inner: stdoutHandler,
			ch:    make(chan models.SystemLog, dbSinkChannelCapacity),
			stop:  make(chan struct{}),
			done:  make(chan struct{}),
		}
		handler = sink
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
