package repositories

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var ProductRepository = newProductRepository()

func newProductRepository() *productRepository {
	return &productRepository{}
}

type productRepository struct {
}

func (r *productRepository) Get(db *gorm.DB, id int64) *models.Product {
	ret := &models.Product{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *productRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Product) {
	cnd.Find(db, &list)
	return
}

func (r *productRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Product {
	ret := &models.Product{}
	if err := cnd.FindOne(db, ret); err != nil {
		return nil
	}
	return ret
}

func (r *productRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.Product, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *productRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Product, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Product{})
	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *productRepository) Create(db *gorm.DB, t *models.Product) error {
	return db.Create(t).Error
}

func (r *productRepository) Update(db *gorm.DB, t *models.Product) error {
	return db.Save(t).Error
}

func (r *productRepository) Updates(db *gorm.DB, id int64, columns map[string]any) error {
	return db.Model(&models.Product{}).Where("id = ?", id).Updates(columns).Error
}

func (r *productRepository) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&models.Product{}, "id = ?", id).Error
}
