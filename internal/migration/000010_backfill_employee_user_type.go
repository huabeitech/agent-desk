package migration

import (
	"agent-desk/internal/pkg/enums"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func init() {
	register(10, "backfill employee user_type for staff users", func() error {
		return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
			// t_user stores both staff accounts (user_type=employee, allowed on /api/dashboard)
			// and support-portal customers (user_type=user). Existing staff accounts created
			// before the user_type column was introduced kept the column default "user" and
			// were rejected by the dashboard auth middleware. Staff accounts are identified
			// by having at least one role binding in t_user_role.
			res := ctx.Tx.Exec(
				"UPDATE t_user SET user_type = ? WHERE user_type = ? "+
					"AND EXISTS (SELECT 1 FROM t_user_role ur WHERE ur.user_id = t_user.id)",
				enums.UserTypeEmployee, enums.UserTypeUser,
			)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				slog.Info("backfilled employee user_type for staff users", "count", res.RowsAffected)
			}
			return nil
		})
	})
}
