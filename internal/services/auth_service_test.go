package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestExtractBearerToken(t *testing.T) {
	svc := newAuthService()

	if got := svc.extractBearerToken("Bearer token_123"); got != "token_123" {
		t.Fatalf("expected bearer token to be extracted, got %q", got)
	}

	if got := svc.extractBearerToken("token_123"); got != "" {
		t.Fatalf("expected raw token to be rejected by bearer extractor, got %q", got)
	}
}

func TestAuthServiceLoginCreatesSingleAccessSession(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	user := createAuthTestUser(t, db, "admin", "secret")
	svc := newAuthService()

	ret, err := svc.Login(request.LoginRequest{
		Username: " admin ",
		Password: "secret",
	}, config.AuthConfig{TokenTTLHours: 2, MaxFailedAttempts: 5, CredentialLockMinute: 15}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if ret.AccessToken == "" || !strings.HasPrefix(ret.AccessToken, "ak_") {
		t.Fatalf("expected ak_ access token, got %q", ret.AccessToken)
	}
	if ret.ExpiresAt == "" {
		t.Fatal("expected expiresAt to be returned")
	}

	var sessions []models.LoginSession
	if err := db.Find(&sessions).Error; err != nil {
		t.Fatalf("query login sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected exactly one session, got %d", len(sessions))
	}
	if sessions[0].Token != ret.AccessToken {
		t.Fatalf("expected session token %q, got %q", ret.AccessToken, sessions[0].Token)
	}
	if sessions[0].UserID != user.ID {
		t.Fatalf("expected session user %d, got %d", user.ID, sessions[0].UserID)
	}
	if sessions[0].ClientType != "admin_web" {
		t.Fatalf("expected admin_web client type, got %q", sessions[0].ClientType)
	}

	logs := findCredentialLogs(t, db)
	if len(logs) != 1 {
		t.Fatalf("expected one credential log, got %d", len(logs))
	}
	if !logs[0].Success || logs[0].Principal != "admin" || logs[0].UserID != user.ID {
		t.Fatalf("unexpected success credential log: %+v", logs[0])
	}
}

func TestAuthServiceLoginFailureWritesCredentialLogs(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	createAuthTestUser(t, db, "admin", "secret")
	svc := newAuthService()
	authCfg := config.AuthConfig{TokenTTLHours: 2, MaxFailedAttempts: 5, CredentialLockMinute: 15}

	if _, err := svc.Login(request.LoginRequest{Username: "missing", Password: "secret"}, authCfg, "127.0.0.1", "go-test"); !hasCode(err, errorsx.CodeAuthInvalidAccount) {
		t.Fatalf("expected invalid account for missing user, got %v", err)
	}
	if _, err := svc.Login(request.LoginRequest{Username: "admin", Password: "wrong"}, authCfg, "127.0.0.1", "go-test"); !hasCode(err, errorsx.CodeAuthInvalidAccount) {
		t.Fatalf("expected invalid account for password mismatch, got %v", err)
	}

	logs := findCredentialLogs(t, db)
	if len(logs) != 2 {
		t.Fatalf("expected two credential logs, got %d", len(logs))
	}
	if logs[0].Reason != "user not found" || logs[0].Success {
		t.Fatalf("unexpected missing-user log: %+v", logs[0])
	}
	if logs[1].Reason != "password mismatch" || logs[1].Success {
		t.Fatalf("unexpected password-mismatch log: %+v", logs[1])
	}
}

func TestAuthServiceLoginCredentialLockout(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	user := createAuthTestUser(t, db, "admin", "secret")
	now := time.Now()
	for i := 0; i < 2; i++ {
		if err := db.Create(&models.LoginCredentialLog{
			Principal: "admin",
			UserID:    user.ID,
			Success:   false,
			ClientIP:  "127.0.0.1",
			Reason:    "password mismatch",
			CreatedAt: now.Add(-time.Duration(i+1) * time.Minute),
		}).Error; err != nil {
			t.Fatalf("seed credential log: %v", err)
		}
	}
	if err := db.Create(&models.LoginCredentialLog{
		Principal: "admin",
		UserID:    user.ID,
		Success:   false,
		ClientIP:  "127.0.0.1",
		Reason:    "password mismatch",
		CreatedAt: now.Add(-30 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed old credential log: %v", err)
	}

	svc := newAuthService()
	_, err := svc.Login(request.LoginRequest{Username: "admin", Password: "secret"}, config.AuthConfig{
		TokenTTLHours:        2,
		MaxFailedAttempts:    2,
		CredentialLockMinute: 15,
	}, "127.0.0.1", "go-test")
	if !hasCode(err, errorsx.CodeAuthCredentialLocked) {
		t.Fatalf("expected credential locked error, got %v", err)
	}

	var lockedLog models.LoginCredentialLog
	if err := db.Order("id DESC").Take(&lockedLog).Error; err != nil {
		t.Fatalf("query latest credential log: %v", err)
	}
	if lockedLog.Reason != "credential locked" || lockedLog.Success {
		t.Fatalf("unexpected locked credential log: %+v", lockedLog)
	}

	var sessionCount int64
	if err := db.Model(&models.LoginSession{}).Count(&sessionCount).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("expected no session while credential locked, got %d", sessionCount)
	}
}

func TestAuthServiceCredentialLockoutDoesNotExtendWhileLocked(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	createAuthTestUser(t, db, "admin", "secret")
	now := time.Now()
	entries := []models.LoginCredentialLog{
		{
			Principal: "admin",
			UserID:    1,
			Success:   false,
			ClientIP:  "127.0.0.1",
			Reason:    "password mismatch",
			CreatedAt: now.Add(-2 * time.Minute),
		},
		{
			Principal: "admin",
			UserID:    0,
			Success:   false,
			ClientIP:  "127.0.0.1",
			Reason:    "credential locked",
			CreatedAt: now.Add(-1 * time.Minute),
		},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("seed credential logs: %v", err)
	}

	ret, err := newAuthService().Login(request.LoginRequest{Username: "admin", Password: "secret"}, config.AuthConfig{
		TokenTTLHours:        2,
		MaxFailedAttempts:    2,
		CredentialLockMinute: 15,
	}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatalf("expected locked attempt logs not to extend lockout, got %v", err)
	}
	if ret == nil || !strings.HasPrefix(ret.AccessToken, "ak_") {
		t.Fatalf("expected login response with ak_ token, got %+v", ret)
	}
}

func TestAuthServiceCredentialLockoutNormalizesPrincipalCase(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	createAuthTestUser(t, db, "admin", "secret")
	if err := db.Create(&models.LoginCredentialLog{
		Principal: "admin",
		UserID:    1,
		Success:   false,
		ClientIP:  "127.0.0.1",
		Reason:    "password mismatch",
		CreatedAt: time.Now().Add(-time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed credential log: %v", err)
	}

	_, err := newAuthService().Login(request.LoginRequest{Username: "ADMIN", Password: "secret"}, config.AuthConfig{
		TokenTTLHours:        2,
		MaxFailedAttempts:    1,
		CredentialLockMinute: 15,
	}, "127.0.0.1", "go-test")
	if !hasCode(err, errorsx.CodeAuthCredentialLocked) {
		t.Fatalf("expected normalized principal to be locked, got %v", err)
	}

	var lockedLog models.LoginCredentialLog
	if err := db.Order("id DESC").Take(&lockedLog).Error; err != nil {
		t.Fatalf("query latest credential log: %v", err)
	}
	if lockedLog.Principal != "admin" || lockedLog.Reason != "credential locked" {
		t.Fatalf("unexpected locked log: %+v", lockedLog)
	}
}

// TestAuthServiceCredentialLockoutIsScopedToTheClientAddress is the regression
// test for the denial of service. Keyed on the username alone, anybody who knew a
// username could lock the real account out for the whole window, from anywhere,
// as often as they liked.
func TestAuthServiceCredentialLockoutIsScopedToTheClientAddress(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	user := createAuthTestUser(t, db, "admin", "secret")
	now := time.Now()
	for i := 0; i < 5; i++ {
		if err := db.Create(&models.LoginCredentialLog{
			Principal: "admin",
			UserID:    user.ID,
			Success:   false,
			ClientIP:  "203.0.113.66",
			Reason:    "password mismatch",
			CreatedAt: now.Add(-time.Duration(i+1) * time.Minute),
		}).Error; err != nil {
			t.Fatalf("seed credential log: %v", err)
		}
	}

	authCfg := config.AuthConfig{TokenTTLHours: 2, MaxFailedAttempts: 3, CredentialLockMinute: 15}

	if _, err := newAuthService().Login(request.LoginRequest{Username: "admin", Password: "secret"}, authCfg, "203.0.113.66", "go-test"); !hasCode(err, errorsx.CodeAuthCredentialLocked) {
		t.Fatalf("expected the attacking address to be locked, got %v", err)
	}

	ret, err := newAuthService().Login(request.LoginRequest{Username: "admin", Password: "secret"}, authCfg, "198.51.100.7", "go-test")
	if err != nil {
		t.Fatalf("the legitimate owner was locked out by somebody else's failures: %v", err)
	}
	if ret == nil || ret.AccessToken == "" {
		t.Fatalf("expected a session for the legitimate owner, got %+v", ret)
	}
}

// TestAuthServiceCredentialLockoutByAddressAcrossPrincipals covers the window that
// replaces the removed account-wide lock: one address spraying many usernames.
func TestAuthServiceCredentialLockoutByAddressAcrossPrincipals(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	createAuthTestUser(t, db, "admin", "secret")
	now := time.Now()
	for i, principal := range []string{"admin", "root", "operator", "support"} {
		if err := db.Create(&models.LoginCredentialLog{
			Principal: principal,
			UserID:    0,
			Success:   false,
			ClientIP:  "203.0.113.66",
			Reason:    "user not found",
			CreatedAt: now.Add(-time.Duration(i+1) * time.Minute),
		}).Error; err != nil {
			t.Fatalf("seed credential log: %v", err)
		}
	}

	authCfg := config.AuthConfig{
		TokenTTLHours:          2,
		MaxFailedAttempts:      10,
		MaxFailedAttemptsPerIP: 4,
		CredentialLockMinute:   15,
	}

	// No single username has reached MaxFailedAttempts, so only the per-address
	// window can catch this.
	if _, err := newAuthService().Login(request.LoginRequest{Username: "admin", Password: "secret"}, authCfg, "203.0.113.66", "go-test"); !hasCode(err, errorsx.CodeAuthCredentialLocked) {
		t.Fatalf("expected credential stuffing from one address to be locked, got %v", err)
	}

	if _, err := newAuthService().Login(request.LoginRequest{Username: "admin", Password: "secret"}, authCfg, "198.51.100.7", "go-test"); err != nil {
		t.Fatalf("an unrelated address was locked by another address's failures: %v", err)
	}
}

func TestAuthServiceCredentialLockoutDisabledWhenMaxAttemptsNonPositive(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	user := createAuthTestUser(t, db, "admin", "secret")
	now := time.Now()
	for i := 0; i < 3; i++ {
		if err := db.Create(&models.LoginCredentialLog{
			Principal: "admin",
			UserID:    user.ID,
			Success:   false,
			ClientIP:  "127.0.0.1",
			Reason:    "credential locked",
			CreatedAt: now.Add(-time.Duration(i+1) * time.Minute),
		}).Error; err != nil {
			t.Fatalf("seed credential log: %v", err)
		}
	}

	ret, err := newAuthService().Login(request.LoginRequest{Username: "admin", Password: "secret"}, config.AuthConfig{
		TokenTTLHours:        2,
		MaxFailedAttempts:    0,
		CredentialLockMinute: 15,
	}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatalf("expected lockout to be disabled, got %v", err)
	}
	if ret == nil || ret.AccessToken == "" {
		t.Fatalf("expected login response with access token, got %+v", ret)
	}
}

func TestValidateSessionTokenStates(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	svc := newAuthService()
	now := time.Now()

	if _, err := svc.validateSessionToken("  "); !hasCode(err, errorsx.CodeAuthUnauthorized) {
		t.Fatalf("expected unauthorized for empty token, got %v", err)
	}
	if _, err := svc.validateSessionToken("missing"); !hasCode(err, errorsx.CodeAuthInvalidToken) {
		t.Fatalf("expected invalid token for missing session, got %v", err)
	}

	revokedAt := now
	if err := db.Create(&models.LoginSession{
		UserID:     1,
		Token:      "ak_revoked",
		ClientType: "admin_web",
		ExpiredAt:  now.Add(time.Hour),
		RevokedAt:  &revokedAt,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}).Error; err != nil {
		t.Fatalf("seed revoked session: %v", err)
	}
	if _, err := svc.validateSessionToken("ak_revoked"); !hasCode(err, errorsx.CodeAuthInvalidToken) {
		t.Fatalf("expected invalid token for revoked session, got %v", err)
	}

	if err := db.Create(&models.LoginSession{
		UserID:     1,
		Token:      "ak_expired",
		ClientType: "admin_web",
		ExpiredAt:  now.Add(-time.Hour),
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}).Error; err != nil {
		t.Fatalf("seed expired session: %v", err)
	}
	if _, err := svc.validateSessionToken("ak_expired"); !hasCode(err, errorsx.CodeAuthInvalidToken) {
		t.Fatalf("expected invalid token for expired session, got %v", err)
	}

	if err := db.Create(&models.LoginSession{
		UserID:     1,
		Token:      "ak_valid",
		ClientType: "admin_web",
		ExpiredAt:  now.Add(time.Hour),
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}).Error; err != nil {
		t.Fatalf("seed valid session: %v", err)
	}
	session, err := svc.validateSessionToken("ak_valid")
	if err != nil {
		t.Fatalf("expected valid session token, got %v", err)
	}
	if session.Token != "ak_valid" {
		t.Fatalf("expected valid session token ak_valid, got %q", session.Token)
	}
}

func TestAuthServiceLogoutRevokesCurrentTokenOnly(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	now := time.Now()
	sessions := []models.LoginSession{
		{
			UserID:     1,
			Token:      "ak_current",
			ClientType: "admin_web",
			ExpiredAt:  now.Add(time.Hour),
			AuditFields: models.AuditFields{
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			UserID:     1,
			Token:      "ak_other",
			ClientType: "admin_web",
			ExpiredAt:  now.Add(time.Hour),
			AuditFields: models.AuditFields{
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	if err := newAuthService().Logout("Bearer ak_current"); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	var current models.LoginSession
	if err := db.Take(&current, "token = ?", "ak_current").Error; err != nil {
		t.Fatalf("query current session: %v", err)
	}
	if current.RevokedAt == nil {
		t.Fatal("expected current session to be revoked")
	}
	var other models.LoginSession
	if err := db.Take(&other, "token = ?", "ak_other").Error; err != nil {
		t.Fatalf("query other session: %v", err)
	}
	if other.RevokedAt != nil {
		t.Fatal("expected other session to remain active")
	}
}

func setupAuthServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Company{},
		&models.Customer{},
		&models.CustomerIdentity{},
		&models.CustomerContact{},
		&models.User{},
		&models.UserIdentity{},
		&models.Role{},
		&models.Permission{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.LoginSession{},
		&models.LoginCredentialLog{},
	); err != nil {
		t.Fatalf("migrate auth tables: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createAuthTestUser(t *testing.T, db *gorm.DB, username, password string) *models.User {
	t.Helper()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	now := time.Now()
	user := &models.User{
		Username: username,
		Nickname: username,
		Password: string(passwordHash),
		Status:   enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create auth test user: %v", err)
	}
	return user
}

func findCredentialLogs(t *testing.T, db *gorm.DB) []models.LoginCredentialLog {
	t.Helper()
	var logs []models.LoginCredentialLog
	if err := db.Order("id ASC").Find(&logs).Error; err != nil {
		t.Fatalf("query credential logs: %v", err)
	}
	return logs
}

func hasCode(err error, code int) bool {
	if err == nil {
		return false
	}
	var codeErr *web.CodeError
	if errors.As(err, &codeErr) {
		return codeErr.Code == code
	}
	return false
}
