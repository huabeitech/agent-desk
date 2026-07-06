package repositories

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var SalesLeadRepository = newSalesLeadRepository()

func newSalesLeadRepository() *salesLeadRepository {
	return &salesLeadRepository{}
}

type salesLeadRepository struct {
}

func (r *salesLeadRepository) Get(db *gorm.DB, id int64) *models.SalesLead {
	ret := &models.SalesLead{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *salesLeadRepository) Take(db *gorm.DB, where ...interface{}) *models.SalesLead {
	ret := &models.SalesLead{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *salesLeadRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.SalesLead) {
	cnd.Find(db, &list)
	return
}

func (r *salesLeadRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.SalesLead {
	ret := &models.SalesLead{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *salesLeadRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.SalesLead, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *salesLeadRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.SalesLead, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.SalesLead{})
	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *salesLeadRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.SalesLead{})
}

func (r *salesLeadRepository) Create(db *gorm.DB, t *models.SalesLead) error {
	return db.Create(t).Error
}

func (r *salesLeadRepository) Update(db *gorm.DB, t *models.SalesLead) error {
	return db.Save(t).Error
}

func (r *salesLeadRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) error {
	return db.Model(&models.SalesLead{}).Where("id = ?", id).Updates(columns).Error
}

func (r *salesLeadRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.SalesLead{}, "id = ?", id)
}
