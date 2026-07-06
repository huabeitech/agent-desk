package migration

import "github.com/mlogclub/simple/sqls"

func init() {
	register(9, "sync product permissions", func() error {
		return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
			permissions, err := ensurePermissions(ctx.Tx)
			if err != nil {
				return err
			}

			roles, err := ensureRoles(ctx.Tx)
			if err != nil {
				return err
			}

			return ensureRolePermissions(ctx.Tx, roles, permissions)
		})
	})
}
