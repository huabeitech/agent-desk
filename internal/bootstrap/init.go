package bootstrap

import (
	"agent-desk/internal/ai/rag/vectordb"
	"agent-desk/internal/oidcclient"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/pkg/logx"
	"agent-desk/internal/services/cronx"
	"agent-desk/internal/wxwork"
	"context"
	"log/slog"

	_ "agent-desk/internal/services/event_handlers"
)

func Init(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("init config failed", "error", err)
		return err
	}
	config.SetCurrent(cfg)
	i18nx.SetDefaultLocale(cfg.LanguageOrDefault())

	logx.Init(logx.Config{
		Level:     cfg.Logger.Level,
		Format:    cfg.Logger.Format,
		AddSource: cfg.Logger.AddSource,
	})

	if _, err := InitDB(cfg.DB); err != nil {
		slog.Error("init db failed", "error", err)
		return err
	}
	if err := InitMigrations(); err != nil {
		slog.Error("init migrations failed", "error", err)
		return err
	}
	if err := vectordb.Init(&cfg.VectorDB); err != nil {
		slog.Error("init vector db failed", "error", err)
		return err
	}
	if err := InitAI(cfg); err != nil {
		slog.Warn("init AI config failed", "error", err)
	}
	if err := InitDefaultKnowledgeBase(); err != nil {
		slog.Warn("init default knowledge base failed", "error", err)
	}

	// 启动任务调度器
	cronx.Init()

	wxwork.Init()
	if err := oidcclient.Init(context.Background()); err != nil {
		slog.Warn("init oidc at startup failed (will retry lazily on request)", "error", err)
	}
	return nil
}
