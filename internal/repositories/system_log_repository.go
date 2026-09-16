package repositories

import (
	"time"

	"agent-desk/internal/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var SystemLogRepository = newSystemLogRepository()

func newSystemLogRepository() *systemLogRepository {
	return &systemLogRepository{}
}

type systemLogRepository struct {
}

func (r *systemLogRepository) Get(db *gorm.DB, id int64) *models.SystemLog {
	ret := &models.SystemLog{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *systemLogRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.SystemLog, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.SystemLog{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *systemLogRepository) DeleteOlderThan(db *gorm.DB, before time.Time) (count int64) {
	res := db.Where("created_at < ?", before).Delete(&models.SystemLog{})
	if res.Error == nil {
		count = res.RowsAffected
	}
	return
}
