package migration

import (
	"agent-desk/internal/models"

	"github.com/mlogclub/simple/sqls"
)

func init() {
	register(14, "add product industry attributes", func() error {
		return sqls.DB().AutoMigrate(&models.Product{})
	})
}
