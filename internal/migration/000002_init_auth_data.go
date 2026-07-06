package migration

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/mlogclub/simple/sqls"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func init() {
	register(2, "init auth builtin data", func() error {
		return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
			permissions, err := ensurePermissions(ctx.Tx)
			if err != nil {
				return err
			}

			roles, err := ensureRoles(ctx.Tx)
			if err != nil {
				return err
			}

			if err = ensureRolePermissions(ctx.Tx, roles, permissions); err != nil {
				return err
			}

			return ensureBootstrapAdmin(ctx.Tx, roles[constants.RoleCodeSuperAdmin])
		})
	})
}

func ensurePermissions(tx *gorm.DB) (map[string]*models.Permission, error) {
	permissions := make(map[string]*models.Permission, len(constants.Permissions))
	now := time.Now()

	for _, spec := range constants.Permissions {
		permission := repositories.PermissionRepository.FindOne(tx, sqls.NewCnd().Eq("code", spec.Code))
		if permission == nil {
			permission = &models.Permission{
				Name:      spec.Name,
				Code:      spec.Code,
				Type:      spec.Type,
				GroupName: spec.GroupName,
				Method:    spec.Method,
				APIPath:   spec.APIPath,
				SortNo:    spec.SortNo,
				Status:    enums.StatusOk,
				IsBuiltin: true,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   constants.SystemAuditUserID,
					CreateUserName: constants.SystemAuditUserName,
					UpdatedAt:      now,
					UpdateUserID:   constants.SystemAuditUserID,
					UpdateUserName: constants.SystemAuditUserName,
				},
			}
			if err := repositories.PermissionRepository.Create(tx, permission); err != nil {
				return nil, err
			}
			// slog.Info("initialized builtin permission", "code", spec.Code)
		} else {
			if err := repositories.PermissionRepository.Updates(tx, permission.ID, map[string]any{
				"name":             spec.Name,
				"type":             spec.Type,
				"group_name":       spec.GroupName,
				"method":           spec.Method,
				"api_path":         spec.APIPath,
				"sort_no":          spec.SortNo,
				"status":           enums.StatusOk,
				"is_builtin":       true,
				"update_user_id":   constants.SystemAuditUserID,
				"update_user_name": constants.SystemAuditUserName,
				"updated_at":       now,
			}); err != nil {
				return nil, err
			}
			permission = repositories.PermissionRepository.Get(tx, permission.ID)
		}
		permissions[spec.Code] = permission
	}

	return permissions, nil
}

func ensureRoles(tx *gorm.DB) (map[string]*models.Role, error) {
	roles := make(map[string]*models.Role, len(constants.Roles))
	now := time.Now()

	for _, spec := range constants.Roles {
		role := repositories.RoleRepository.GetByCode(tx, spec.Code)
		if role == nil {
			role = &models.Role{
				Name:     spec.Name,
				Code:     spec.Code,
				Status:   enums.StatusOk,
				IsSystem: true,
				SortNo:   spec.SortNo,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   constants.SystemAuditUserID,
					CreateUserName: constants.SystemAuditUserName,
					UpdatedAt:      now,
					UpdateUserID:   constants.SystemAuditUserID,
					UpdateUserName: constants.SystemAuditUserName,
				},
			}
			if err := repositories.RoleRepository.Create(tx, role); err != nil {
				return nil, err
			}
			// slog.Info("initialized builtin role", "code", spec.Code)
		} else {
			if err := repositories.RoleRepository.Updates(tx, role.ID, map[string]any{
				"name":             spec.Name,
				"sort_no":          spec.SortNo,
				"status":           enums.StatusOk,
				"is_system":        true,
				"update_user_id":   constants.SystemAuditUserID,
				"update_user_name": constants.SystemAuditUserName,
				"updated_at":       now,
			}); err != nil {
				return nil, err
			}
			role = repositories.RoleRepository.Get(tx, role.ID)
		}
		roles[spec.Code] = role
	}

	return roles, nil
}

func ensureRolePermissions(tx *gorm.DB, roles map[string]*models.Role, permissions map[string]*models.Permission) error {
	now := time.Now()

	for roleCode, rolePermissions := range constants.RolePermissions {
		role := roles[roleCode]
		if role == nil {
			return errors.New("builtin role not found: " + roleCode)
		}

		for _, permissionSpec := range rolePermissions {
			permission := permissions[permissionSpec.Code]
			if permission == nil {
				return errors.New("builtin permission not found: " + permissionSpec.Code)
			}

			exists := repositories.RolePermissionRepository.FindOne(tx, sqls.NewCnd().
				Eq("role_id", role.ID).
				Eq("permission_id", permission.ID))
			if exists != nil {
				continue
			}

			if err := repositories.RolePermissionRepository.Create(tx, &models.RolePermission{
				RoleID:       role.ID,
				PermissionID: permission.ID,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   constants.SystemAuditUserID,
					CreateUserName: constants.SystemAuditUserName,
					UpdatedAt:      now,
					UpdateUserID:   constants.SystemAuditUserID,
					UpdateUserName: constants.SystemAuditUserName,
				},
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureBootstrapAdmin(tx *gorm.DB, superAdminRole *models.Role) error {
	if superAdminRole == nil {
		return errors.New("super admin role not found")
	}

	username := constants.BootstrapAdminUsername
	nickname := constants.BootstrapAdminNickname
	password := constants.BootstrapAdminInitialPassword()

	if strings.TrimSpace(password) == "" {
		password = "ChangeMe123!"
		slog.Warn("bootstrap admin password is empty, using default password", "username", username)
	}

	user := repositories.UserRepository.FindOne(tx, sqls.NewCnd().Eq("username", username))
	now := time.Now()
	if user == nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user = &models.User{
			Username: username,
			Nickname: nickname,
			Password: string(hashedPassword),
			Status:   enums.StatusOk,
			Remark:   "bootstrap super admin",
			AuditFields: models.AuditFields{
				CreatedAt:      now,
				CreateUserID:   constants.SystemAuditUserID,
				CreateUserName: constants.SystemAuditUserName,
				UpdatedAt:      now,
				UpdateUserID:   constants.SystemAuditUserID,
				UpdateUserName: constants.SystemAuditUserName,
			},
		}
		if err := repositories.UserRepository.Create(tx, user); err != nil {
			return err
		}
		slog.Warn("initialized bootstrap admin user", "username", username)
	} else {
		if err := repositories.UserRepository.Updates(tx, user.ID, map[string]any{
			"nickname":         nickname,
			"status":           enums.StatusOk,
			"update_user_id":   constants.SystemAuditUserID,
			"update_user_name": constants.SystemAuditUserName,
			"updated_at":       now,
		}); err != nil {
			return err
		}
		user = repositories.UserRepository.Get(tx, user.ID)
	}

	exists := repositories.UserRoleRepository.FindOne(tx, sqls.NewCnd().
		Eq("user_id", user.ID).
		Eq("role_id", superAdminRole.ID))
	if exists != nil {
		return nil
	}

	return repositories.UserRoleRepository.Create(tx, &models.UserRole{
		UserID: user.ID,
		RoleID: superAdminRole.ID,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   constants.SystemAuditUserID,
			CreateUserName: constants.SystemAuditUserName,
			UpdatedAt:      now,
			UpdateUserID:   constants.SystemAuditUserID,
			UpdateUserName: constants.SystemAuditUserName,
		},
	})
}
