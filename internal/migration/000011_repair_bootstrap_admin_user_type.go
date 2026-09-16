package migration

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/enums"
	"time"

	"github.com/mlogclub/simple/sqls"
)

func init() {
	register(11, "repair bootstrap admin user type", func() error {
		db := sqls.DB()
		superAdmins := db.Table("t_user_role AS ur").Select("ur.user_id").
			Joins("JOIN t_role AS r ON r.id = ur.role_id").
			Where("r.code = ? AND r.status = ?", constants.RoleCodeSuperAdmin, enums.StatusOk)

		// Migration 2 has already run on existing installations. Repair only the
		// active bootstrap account that already has the super admin role.
		return db.Model(&models.User{}).
			Where("username = ?", constants.BootstrapAdminUsername).
			Where("user_type = ?", enums.UserTypeUser).
			Where("status = ?", enums.StatusOk).
			Where("deleted_at IS NULL").
			Where("id IN (?)", superAdmins).
			Updates(map[string]any{
				"user_type":        enums.UserTypeEmployee,
				"updated_at":       time.Now(),
				"update_user_id":   constants.SystemAuditUserID,
				"update_user_name": constants.SystemAuditUserName,
			}).Error
	})
}
