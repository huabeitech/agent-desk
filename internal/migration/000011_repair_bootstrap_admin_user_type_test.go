package migration

import (
	"path/filepath"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/enums"
	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupBootstrapAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "auth.db")), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: "t_", SingularTable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.UserRole{}); err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db
}

func TestBootstrapAdminCreatedAsEmployee(t *testing.T) {
	db := setupBootstrapAdminTestDB(t)
	role := models.Role{Code: constants.RoleCodeSuperAdmin, Status: enums.StatusOk}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureBootstrapAdmin(db, &role); err != nil {
		t.Fatal(err)
	}
	var user models.User
	if err := db.Where("username = ?", constants.BootstrapAdminUsername).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.UserType != enums.UserTypeEmployee {
		t.Fatalf("bootstrap admin type = %q, want employee", user.UserType)
	}
	var count int64
	if err := db.Model(&models.UserRole{}).Where("user_id = ? AND role_id = ?", user.ID, role.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("super admin role assignments = %d, want 1", count)
	}
}

func TestRepairBootstrapAdminUserType(t *testing.T) {
	repair, ok := migrationFuncs[11]
	if !ok {
		t.Fatal("bootstrap admin repair migration is not registered")
	}
	for _, tc := range []struct {
		name       string
		username   string
		userType   enums.UserType
		status     enums.Status
		roleCode   string
		roleStatus enums.Status
		deleted    bool
		want       enums.UserType
	}{
		{"legacy admin", "admin", enums.UserTypeUser, enums.StatusOk, constants.RoleCodeSuperAdmin, enums.StatusOk, false, enums.UserTypeEmployee},
		{"already employee", "admin", enums.UserTypeEmployee, enums.StatusOk, constants.RoleCodeSuperAdmin, enums.StatusOk, false, enums.UserTypeEmployee},
		{"ordinary user", "customer", enums.UserTypeUser, enums.StatusOk, "", enums.StatusOk, false, enums.UserTypeUser},
		{"same name without role", "admin", enums.UserTypeUser, enums.StatusOk, "", enums.StatusOk, false, enums.UserTypeUser},
		{"other super admin", "other", enums.UserTypeUser, enums.StatusOk, constants.RoleCodeSuperAdmin, enums.StatusOk, false, enums.UserTypeUser},
		{"disabled admin", "admin", enums.UserTypeUser, enums.StatusDisabled, constants.RoleCodeSuperAdmin, enums.StatusOk, false, enums.UserTypeUser},
		{"disabled role", "admin", enums.UserTypeUser, enums.StatusOk, constants.RoleCodeSuperAdmin, enums.StatusDisabled, false, enums.UserTypeUser},
		{"deleted admin", "admin", enums.UserTypeUser, enums.StatusOk, constants.RoleCodeSuperAdmin, enums.StatusOk, true, enums.UserTypeUser},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupBootstrapAdminTestDB(t)
			sqls.SetDB(db)
			t.Cleanup(func() { sqls.SetDB(nil) })
			user := models.User{Username: tc.username, Password: "unchanged-test-value", UserType: tc.userType, Status: tc.status}
			if tc.deleted {
				now := time.Now()
				user.DeletedAt = &now
			}
			if err := db.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			if tc.roleCode != "" {
				role := models.Role{Code: tc.roleCode, Status: tc.roleStatus}
				if err := db.Create(&role).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := repair.Fn(); err != nil {
				t.Fatal(err)
			}
			var got models.User
			if err := db.First(&got, user.ID).Error; err != nil {
				t.Fatal(err)
			}
			if got.UserType != tc.want {
				t.Fatalf("type = %q, want %q", got.UserType, tc.want)
			}
			if got.Password != user.Password || got.Status != user.Status {
				t.Fatal("repair changed password or account status")
			}
			updatedAt := got.UpdatedAt
			if err := repair.Fn(); err != nil {
				t.Fatal(err)
			}
			if err := db.First(&got, user.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !got.UpdatedAt.Equal(updatedAt) {
				t.Fatal("second repair changed updated_at")
			}
		})
	}
}
