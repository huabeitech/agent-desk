package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"crypto/rand"
	"encoding/hex"
	"net/mail"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	authPrincipalContextKey = "authPrincipal"
)

var AuthService = newAuthService()

func newAuthService() *authService {
	return &authService{}
}

type authService struct {
}

func (s *authService) GetAuthPrincipal(ctx *gin.Context) *dto.AuthPrincipal {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Get(authPrincipalContextKey)
	if principal, ok := v.(*dto.AuthPrincipal); ok {
		return principal
	}
	return nil
}

func (s *authService) setAuthPrincipal(ctx *gin.Context, user *models.User, roles, permissions []string) *dto.AuthPrincipal {
	principal := &dto.AuthPrincipal{
		UserID:      user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		UserType:    user.UserType,
		Status:      user.Status,
		Roles:       roles,
		Permissions: permissions,
	}
	ctx.Set(authPrincipalContextKey, principal)
	return principal
}

func (s *authService) RequirePermission(ctx *gin.Context, permission constants.Permission) (principal *dto.AuthPrincipal, err error) {
	if principal = s.GetAuthPrincipal(ctx); principal == nil {
		if principal, err = s.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	if principal == nil {
		return nil, errorsx.ForbiddenI18n("error.e0225")
	}

	if !s.HasPermission(ctx, permission.Code) {
		return principal, errorsx.ForbiddenI18n("error.e0225")
	}
	return principal, nil
}

func (s *authService) Login(req request.LoginRequest, authCfg config.AuthConfig, clientIP, userAgent string) (*response.LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	principal := normalizeLoginPrincipal(username)
	password := req.Password
	if username == "" || strings.TrimSpace(password) == "" {
		return nil, errorsx.InvalidParamI18n("error.e0258")
	}

	if s.isCredentialLocked(principal, authCfg) {
		_ = s.createLoginCredentialLog(principal, 0, false, clientIP, userAgent, "credential locked")
		return nil, errorsx.CredentialLockedI18n("error.e0270")
	}

	user := UserService.GetByUsername(username)
	if user == nil || user.Status != enums.StatusOk {
		_ = s.createLoginCredentialLog(principal, 0, false, clientIP, userAgent, "user not found")
		return nil, errorsx.InvalidAccountI18n("error.e0260")
	}
	if strs.IsBlank(user.Password) || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		_ = s.createLoginCredentialLog(principal, user.ID, false, clientIP, userAgent, "password mismatch")
		return nil, errorsx.InvalidAccountI18n("error.e0260")
	}

	var ret *response.LoginResponse
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		var dbErr error
		ret, dbErr = s.issueTokens(ctx, user, clientIP, userAgent, authCfg)
		if dbErr != nil {
			return dbErr
		}
		if dbErr = repositories.UserRepository.Updates(ctx.Tx, user.ID, map[string]any{
			"last_login_at":    time.Now(),
			"last_login_ip":    clientIP,
			"update_user_id":   user.ID,
			"update_user_name": user.Username,
			"updated_at":       time.Now(),
		}); dbErr != nil {
			return dbErr
		}
		return nil
	}); err != nil {
		return nil, err
	}

	_ = s.createLoginCredentialLog(principal, user.ID, true, clientIP, userAgent, "")
	return ret, nil
}

func (s *authService) Logout(accessToken string) error {
	accessToken = s.extractBearerToken(accessToken)
	now := time.Now()
	if accessToken != "" {
		if session := LoginSessionService.FindOne(sqls.NewCnd().Eq("token", accessToken)); session != nil && session.RevokedAt == nil {
			if err := LoginSessionService.Updates(session.ID, map[string]any{
				"revoked_at": now,
				"updated_at": now,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *authService) Authenticate(ctx *gin.Context) (*dto.AuthPrincipal, error) {
	if principal := s.GetAuthPrincipal(ctx); principal != nil {
		return principal, nil
	}

	token := s.extractBearerToken(ctx.GetHeader("Authorization"))
	if token == "" {
		token = strings.TrimSpace(ctx.Query("accessToken"))
	}
	if token == "" {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}

	session, err := s.validateSessionToken(token)
	if err != nil {
		return nil, err
	}

	user := UserService.Get(session.UserID)
	if user == nil || user.Status != enums.StatusOk {
		return nil, errorsx.UnauthorizedI18n("error.e0256")
	}

	roles, permissions, err := s.loadUserAuthScope(sqls.DB(), user.ID)
	if err != nil {
		return nil, err
	}
	principal := s.setAuthPrincipal(ctx, user, roles, permissions)

	now := time.Now()
	_ = LoginSessionService.Updates(session.ID, map[string]any{
		"last_seen_at": now,
		"updated_at":   now,
	})

	return principal, nil
}

func (s *authService) HasPermission(ctx *gin.Context, permissionCode string) bool {
	principal := s.GetAuthPrincipal(ctx)
	if principal == nil {
		return false
	}
	return slices.Contains(principal.Permissions, permissionCode)
}

func (s *authService) CurrentProfile(ctx *gin.Context) (*response.LoginResponse, error) {
	principal, err := s.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	user := UserService.Get(principal.UserID)
	if user == nil || user.Status != enums.StatusOk {
		return nil, errorsx.UnauthorizedI18n("error.e0256")
	}

	return &response.LoginResponse{
		User: &response.AuthUserResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   utils.BuildAvatarURL(user.Avatar),
			Email:    derefString(user.Email),
			UserType: user.UserType,
			Status:   user.Status,
			Roles:    principal.Roles,
		},
		Permissions: principal.Permissions,
		Roles:       principal.Roles,
	}, nil
}

func (s *authService) UpdateProfile(ctx *gin.Context, req request.UpdateProfileRequest) (*response.LoginResponse, error) {
	principal, err := s.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	user := UserService.Get(principal.UserID)
	if user == nil || user.DeletedAt != nil || user.Status != enums.StatusOk {
		return nil, errorsx.UnauthorizedI18n("error.e0256")
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		return nil, errorsx.InvalidParamI18n("error.profile.nicknameRequired")
	}
	if len([]rune(nickname)) > 100 {
		return nil, errorsx.InvalidParamI18n("error.profile.nicknameTooLong")
	}
	avatar := utils.NormalizeAvatarValue(req.Avatar)
	if len(avatar) > 255 {
		return nil, errorsx.InvalidParamI18n("error.profile.avatarTooLong")
	}
	email := utils.NormalizeNullableString(req.Email)
	if email != nil {
		if _, parseErr := mail.ParseAddress(*email); parseErr != nil {
			return nil, errorsx.InvalidParamI18n("error.profile.emailInvalid")
		}
		if existed := UserService.GetByEmail(*email); existed != nil && existed.ID != user.ID {
			return nil, errorsx.InvalidParamI18n("error.e0338")
		}
	}

	if err := repositories.UserRepository.Updates(sqls.DB(), user.ID, map[string]any{
		"nickname":         nickname,
		"avatar":           avatar,
		"email":            email,
		"update_user_id":   user.ID,
		"update_user_name": user.Username,
		"updated_at":       time.Now(),
	}); err != nil {
		return nil, err
	}
	return s.CurrentProfile(ctx)
}

func (s *authService) GetUserRoles(userID int64) ([]models.Role, error) {
	return s.loadUserRoles(sqls.DB(), userID)
}

func (s *authService) GetUserPermissions(userID int64) ([]string, error) {
	return s.loadUserPermissionCodes(sqls.DB(), userID)
}

func (s *authService) issueTokens(ctx *sqls.TxContext, user *models.User, clientIP, userAgent string, authCfg config.AuthConfig) (*response.LoginResponse, error) {
	roles, permissions, err := s.loadUserAuthScope(ctx.Tx, user.ID)
	if err != nil {
		return nil, err
	}

	tokenTTL := s.resolveTokenTTL(authCfg)
	accessToken, err := randomToken(constants.AuthTokenPrefix)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := repositories.LoginSessionRepository.Create(ctx.Tx, &models.LoginSession{
		UserID:     user.ID,
		Token:      accessToken,
		ClientType: constants.ClientTypeAdminWeb,
		ClientIP:   clientIP,
		UserAgent:  userAgent,
		ExpiredAt:  now.Add(tokenTTL),
		LastSeenAt: &now,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   user.ID,
			CreateUserName: user.Username,
			UpdatedAt:      now,
			UpdateUserID:   user.ID,
			UpdateUserName: user.Username,
		},
	}); err != nil {
		return nil, err
	}

	return &response.LoginResponse{
		AccessToken: accessToken,
		ExpiresAt:   now.Add(tokenTTL).Format(time.DateTime),
		User: &response.AuthUserResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   utils.BuildAvatarURL(user.Avatar),
			Email:    derefString(user.Email),
			UserType: user.UserType,
			Status:   user.Status,
			Roles:    roles,
		},
		Permissions: permissions,
		Roles:       roles,
	}, nil
}

func (s *authService) resolveTokenTTL(authCfg config.AuthConfig) time.Duration {
	tokenTTL := 12 * time.Hour
	if authCfg.TokenTTLHours > 0 {
		tokenTTL = time.Duration(authCfg.TokenTTLHours) * time.Hour
	}
	return tokenTTL
}

func (s *authService) validateSessionToken(token string) (*models.LoginSession, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	session := LoginSessionService.FindOne(sqls.NewCnd().Eq("token", token))
	if session == nil {
		return nil, errorsx.InvalidTokenI18n("error.e0269")
	}
	if session.RevokedAt != nil {
		return nil, errorsx.InvalidTokenI18n("error.e0267")
	}
	if time.Now().After(session.ExpiredAt) {
		return nil, errorsx.InvalidTokenI18n("error.e0268")
	}
	return session, nil
}

func (s *authService) loadUserAuthScope(tx *gorm.DB, userID int64) ([]string, []string, error) {
	roleCodes, err := s.loadUserRoleCodes(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	permissionCodes, err := s.loadUserPermissionCodes(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	return roleCodes, permissionCodes, nil
}

func (s *authService) loadUserRoleCodes(tx *gorm.DB, userID int64) ([]string, error) {
	roles, err := s.loadUserRoles(tx, userID)
	if err != nil {
		return nil, err
	}
	roleCodes := make([]string, 0, len(roles))
	for _, role := range roles {
		roleCodes = append(roleCodes, role.Code)
	}
	return roleCodes, nil
}

func (s *authService) loadUserRoles(tx *gorm.DB, userID int64) ([]models.Role, error) {
	roles := make([]models.Role, 0)
	if err := tx.
		Table("t_role AS r").
		Select("r.*").
		Joins("JOIN t_user_role AS ur ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.status = ?", userID, enums.StatusOk).
		Order("r.sort_no ASC, r.id ASC").
		Scan(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

func (s *authService) loadUserPermissionCodes(tx *gorm.DB, userID int64) ([]string, error) {
	permissionRows := make([]struct {
		Code string
	}, 0)
	db := tx.Table("t_permission AS p").
		Select("DISTINCT p.code").
		Joins("JOIN t_role_permission AS rp ON rp.permission_id = p.id").
		Joins("JOIN t_user_role AS ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", userID).
		Where("p.status = ?", enums.StatusOk)
	if err := db.Scan(&permissionRows).Error; err != nil {
		return nil, err
	}

	permissionCodes := make([]string, 0, len(permissionRows))
	for _, permission := range permissionRows {
		permissionCodes = append(permissionCodes, permission.Code)
	}

	overrideRows := make([]struct {
		Code   string
		Effect int
	}, 0)
	if err := tx.
		Table("t_user_permission AS up").
		Select("p.code, up.effect").
		Joins("JOIN t_permission AS p ON p.id = up.permission_id").
		Where("up.user_id = ? AND (up.expired_at IS NULL OR up.expired_at > ?)", userID, time.Now()).
		Scan(&overrideRows).Error; err != nil {
		return nil, err
	}

	permissionSet := make(map[string]bool, len(permissionCodes))
	for _, code := range permissionCodes {
		permissionSet[code] = true
	}
	for _, override := range overrideRows {
		if override.Effect < 0 {
			delete(permissionSet, override.Code)
			continue
		}
		permissionSet[override.Code] = true
	}

	permissionCodes = permissionCodes[:0]
	for code := range permissionSet {
		permissionCodes = append(permissionCodes, code)
	}
	sort.Strings(permissionCodes)
	return permissionCodes, nil
}

func (s *authService) extractBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (s *authService) createLoginCredentialLog(principal string, userID int64, success bool, clientIP, userAgent, reason string) error {
	return LoginCredentialLogService.Create(&models.LoginCredentialLog{
		Principal: principal,
		UserID:    userID,
		Success:   success,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Reason:    reason,
		CreatedAt: time.Now(),
	})
}

func (s *authService) isCredentialLocked(principal string, authCfg config.AuthConfig) bool {
	maxFailedAttempts := authCfg.MaxFailedAttempts
	if maxFailedAttempts <= 0 {
		return false
	}
	lockMinute := authCfg.CredentialLockMinute
	if lockMinute <= 0 {
		lockMinute = 15
	}
	since := time.Now().Add(-time.Duration(lockMinute) * time.Minute)
	return LoginCredentialLogService.Count(sqls.NewCnd().
		Eq("principal", normalizeLoginPrincipal(principal)).
		Eq("success", false).
		NotEq("reason", "credential locked").
		Where("created_at >= ?", since)) >= int64(maxFailedAttempts)
}

func normalizeLoginPrincipal(principal string) string {
	return strings.ToLower(strings.TrimSpace(principal))
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func randomToken(prefix string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}
