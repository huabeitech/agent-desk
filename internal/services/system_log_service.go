package services

import (
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

var SystemLogService = newSystemLogService()

func newSystemLogService() *systemLogService {
	return &systemLogService{}
}

type systemLogService struct {
}

func (s *systemLogService) Get(id int64) *models.SystemLog {
	return repositories.SystemLogRepository.Get(sqls.DB(), id)
}

func (s *systemLogService) FindPageByCnd(cnd *sqls.Cnd) (list []models.SystemLog, paging *sqls.Paging) {
	return repositories.SystemLogRepository.FindPageByCnd(sqls.DB(), cnd)
}

// DeleteOlderThan 清理指定时间之前的日志，返回受影响行数。用于 cron 定时清理。
func (s *systemLogService) DeleteOlderThan(before time.Time) int64 {
	return repositories.SystemLogRepository.DeleteOlderThan(sqls.DB(), before)
}
