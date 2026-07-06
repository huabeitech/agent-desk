package migration

import (
	"agent-desk/internal/models"

	"github.com/mlogclub/simple/sqls"
)

func init() {
	register(13, "add sales lead merge explanation", func() error {
		return sqls.DB().AutoMigrate(&models.SalesLead{})
	})
}
