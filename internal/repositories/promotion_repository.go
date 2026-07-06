package repositories

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PromotionRepository = newPromotionRepository()

func newPromotionRepository() *promotionRepository {
	return &promotionRepository{}
}

type promotionRepository struct {
}

func (r *promotionRepository) Get(db *gorm.DB, id int64) *models.Promotion {
	ret := &models.Promotion{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *promotionRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Promotion {
	ret := &models.Promotion{}
	if err := cnd.FindOne(db, ret); err != nil {
		return nil
	}
	return ret
}

func (r *promotionRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.Promotion, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *promotionRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Promotion, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Promotion{})
	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *promotionRepository) Create(db *gorm.DB, t *models.Promotion) error {
	return db.Create(t).Error
}

func (r *promotionRepository) Updates(db *gorm.DB, id int64, columns map[string]any) error {
	return db.Model(&models.Promotion{}).Where("id = ?", id).Updates(columns).Error
}
