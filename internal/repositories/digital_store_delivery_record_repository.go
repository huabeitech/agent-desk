package repositories

import (
	"agent-desk/internal/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var DigitalStoreDeliveryRecordRepository = newDigitalStoreDeliveryRecordRepository()

func newDigitalStoreDeliveryRecordRepository() *digitalStoreDeliveryRecordRepository {
	return &digitalStoreDeliveryRecordRepository{}
}

type digitalStoreDeliveryRecordRepository struct{}

func (r *digitalStoreDeliveryRecordRepository) Get(db *gorm.DB, id int64) *models.DigitalStoreDeliveryRecord {
	ret := &models.DigitalStoreDeliveryRecord{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *digitalStoreDeliveryRecordRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.DigitalStoreDeliveryRecord {
	ret := &models.DigitalStoreDeliveryRecord{}
	if err := cnd.FindOne(db, ret); err != nil {
		return nil
	}
	return ret
}

func (r *digitalStoreDeliveryRecordRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.DigitalStoreDeliveryRecord, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.DigitalStoreDeliveryRecord{})
	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *digitalStoreDeliveryRecordRepository) Create(db *gorm.DB, item *models.DigitalStoreDeliveryRecord) error {
	return db.Create(item).Error
}
