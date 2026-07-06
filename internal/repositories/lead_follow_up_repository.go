package repositories

import (
	"agent-desk/internal/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var LeadFollowUpRepository = newLeadFollowUpRepository()

func newLeadFollowUpRepository() *leadFollowUpRepository {
	return &leadFollowUpRepository{}
}

type leadFollowUpRepository struct {
}

func (r *leadFollowUpRepository) Get(db *gorm.DB, id int64) *models.LeadFollowUp {
	ret := &models.LeadFollowUp{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *leadFollowUpRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.LeadFollowUp) {
	cnd.Find(db, &list)
	return
}

func (r *leadFollowUpRepository) Create(db *gorm.DB, t *models.LeadFollowUp) error {
	return db.Create(t).Error
}
