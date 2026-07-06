package migration

import (
	"agent-desk/internal/models"

	"github.com/mlogclub/simple/sqls"
)

func init() {
	register(12, "create digital store delivery records", func() error {
		return sqls.DB().AutoMigrate(&models.DigitalStoreDeliveryRecord{})
	})
}
