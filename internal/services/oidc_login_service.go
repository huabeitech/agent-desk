package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/oidcclient"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/repositories"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var OIDCLoginService = newOIDCLoginService()

type oidcLoginService struct {
}

type oidcLoginProfile = oidcclient.Profile

func newOIDCLoginService() *oidcLoginService {
	return &oidcLoginService{}
}

func (s *oidcLoginService) BuildOIDCLoginURL(next string) (string, error) {
	return oidcclient.BuildAuthCodeURL(next)
}

func (s *oidcLoginService) LoginByOIDC(ctx context.Context, code, state string, authCfg config.AuthConfig, clientIP, userAgent string) (string, string, error) {
	next, verifier, err := oidcclient.ParseState(state)
	if err != nil {
		return "", "", err
	}
	profile, err := oidcclient.ExchangeCode(ctx, code, verifier)
	if err != nil {
		return "", "", err
	}
	loginResp, err := s.loginWithOIDCProfile(profile, authCfg, clientIP, userAgent)
	if err != nil {
		return "", "", err
	}
	ticket, err := oidcclient.IssueLoginTicket(loginResp)
	if err != nil {
		return "", "", err
	}
	return ticket, next, nil
}

func (s *oidcLoginService) ExchangeOIDCLoginTicket(ticket string) (*response.LoginResponse, error) {
	return oidcclient.ConsumeLoginTicket(ticket)
}

func (s *oidcLoginService) loginWithOIDCProfile(profile *oidcLoginProfile, authCfg config.AuthConfig, clientIP, userAgent string) (*response.LoginResponse, error) {
	if profile == nil || strings.TrimSpace(profile.Subject) == "" {
		return nil, errorsx.BusinessErrorI18n(2, "error.oidc.profileMissing")
	}

	var ret *response.LoginResponse
	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		var (
			identity = repositories.UserIdentityRepository.GetBy(ctx.Tx, enums.ThirdProviderOIDC, "", profile.Subject)
			user     *models.User
			err      error
		)
		if identity == nil {
			user, identity, err = s.createOIDCUser(ctx, profile)
			if err != nil {
				return err
			}
		} else {
			if identity.Status != enums.StatusOk {
				return errorsx.BusinessErrorI18n(3, "error.oidc.bindingDisabled")
			}
			user = repositories.UserRepository.Get(ctx.Tx, identity.UserID)
			if user == nil {
				return errorsx.BusinessErrorI18n(4, "error.oidc.boundUserMissing")
			}
		}

		if user.Status != enums.StatusOk {
			return errorsx.UnauthorizedI18n("error.e0200")
		}

		userUpdates := map[string]any{
			"nickname":         s.resolveOIDCNickname(user.Nickname, profile),
			"avatar":           s.resolveOIDCAvatar(user.Avatar, profile),
			"last_login_at":    time.Now(),
			"last_login_ip":    clientIP,
			"update_user_id":   user.ID,
			"update_user_name": user.Username,
			"updated_at":       time.Now(),
		}
		if (user.Email == nil || *user.Email == "") && profile.Email != "" {
			if email := s.availableEmail(ctx.Tx, profile.Email); email != nil {
				userUpdates["email"] = *email
			}
		}

		if err = repositories.UserRepository.Updates(ctx.Tx, user.ID, userUpdates); err != nil {
			return err
		}

		s.ensureDefaultOIDCRole(ctx.Tx, user)
		s.syncOIDCUserOrganizations(ctx.Tx, user, profile)

		if err = repositories.UserIdentityRepository.Updates(ctx.Tx, identity.ID, map[string]any{
			"provider_name":    enums.GetThirdProviderLabel(enums.ThirdProviderOIDC),
			"raw_profile":      profile.RawProfile,
			"last_auth_at":     time.Now(),
			"status":           enums.StatusOk,
			"update_user_id":   user.ID,
			"update_user_name": user.Username,
			"updated_at":       time.Now(),
		}); err != nil {
			return err
		}

		ret, err = AuthService.issueTokens(ctx, user, clientIP, userAgent, authCfg)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *oidcLoginService) createOIDCUser(ctx *sqls.TxContext, profile *oidcLoginProfile) (*models.User, *models.UserIdentity, error) {
	now := time.Now()
	email := s.availableEmail(ctx.Tx, profile.Email)
	username := s.availableUsername(ctx.Tx, profile)

	user := &models.User{
		Username:     username,
		Nickname:     s.resolveOIDCNickname("", profile),
		Avatar:       s.resolveOIDCAvatar("", profile),
		Email:        email,
		Password:     "",
		PasswordSalt: "",
		UserType:     enums.UserTypeEmployee,
		Status:       enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   0,
			CreateUserName: enums.GetThirdProviderLabel(enums.ThirdProviderOIDC),
			UpdatedAt:      now,
			UpdateUserID:   0,
			UpdateUserName: enums.GetThirdProviderLabel(enums.ThirdProviderOIDC),
		},
	}
	if err := repositories.UserRepository.Create(ctx.Tx, user); err != nil {
		return nil, nil, err
	}

	identity := &models.UserIdentity{
		UserID:         user.ID,
		Provider:       enums.ThirdProviderOIDC,
		ProviderUserID: strings.TrimSpace(profile.Subject),
		ProviderCorpID: "",
		ProviderName:   enums.GetThirdProviderLabel(enums.ThirdProviderOIDC),
		RawProfile:     profile.RawProfile,
		Status:         enums.StatusOk,
		LastAuthAt:     &now,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   user.ID,
			CreateUserName: user.Username,
			UpdatedAt:      now,
			UpdateUserID:   user.ID,
			UpdateUserName: user.Username,
		},
	}
	if err := repositories.UserIdentityRepository.Create(ctx.Tx, identity); err != nil {
		return nil, nil, err
	}
	s.ensureDefaultOIDCRole(ctx.Tx, user)
	return user, identity, nil
}

func (s *oidcLoginService) availableEmail(tx *gorm.DB, email string) *string {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || repositories.UserRepository.GetByEmail(tx, email) != nil {
		return nil
	}
	return &email
}

func (s *oidcLoginService) availableUsername(tx *gorm.DB, profile *oidcLoginProfile) string {
	for _, candidate := range []string{
		profile.PreferredUsername,
		strings.Split(strings.TrimSpace(profile.Email), "@")[0],
	} {
		username := normalizeOIDCUsername(candidate)
		if username != "" && repositories.UserRepository.GetByUsername(tx, username) == nil {
			return username
		}
	}
	base := "oidc_" + shortSubjectHash(profile.Subject)
	if repositories.UserRepository.GetByUsername(tx, base) == nil {
		return base
	}
	for i := 1; i < 100; i++ {
		username := base + "_" + strconv.Itoa(i)
		if repositories.UserRepository.GetByUsername(tx, username) == nil {
			return username
		}
	}
	return base + "_" + shortSubjectHash(time.Now().String())
}

func (s *oidcLoginService) resolveOIDCNickname(current string, profile *oidcLoginProfile) string {
	if profile != nil {
		for _, candidate := range []string{profile.Name, profile.PreferredUsername, profile.Email, profile.Subject} {
			if candidate = strings.TrimSpace(candidate); candidate != "" {
				return candidate
			}
		}
	}
	return strings.TrimSpace(current)
}

func (s *oidcLoginService) resolveOIDCAvatar(current string, profile *oidcLoginProfile) string {
	if profile != nil {
		if picture := strings.TrimSpace(profile.Picture); picture != "" {
			return picture
		}
	}
	return strings.TrimSpace(current)
}

func normalizeOIDCUsername(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		}
	}
	ret := strings.Trim(b.String(), "._-")
	if len(ret) > 100 {
		ret = ret[:100]
	}
	return ret
}

func shortSubjectHash(subject string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(subject)))
	return hex.EncodeToString(sum[:])[:16]
}

// ensureDefaultOIDCRole gives a first-time OIDC user the lowest staff role.
// The provider vouches for who the user is, not what they may administer, so
// a user arriving without any role must never escalate to an administrative
// one.
func (s *oidcLoginService) ensureDefaultOIDCRole(tx *gorm.DB, user *models.User) {
	if user == nil || user.ID <= 0 {
		return
	}
	existingRole := repositories.UserRoleRepository.FindOne(tx, sqls.NewCnd().Eq("user_id", user.ID))
	if existingRole != nil {
		return
	}
	defaultRole := repositories.RoleRepository.GetByCode(tx, constants.RoleCodeCsUser)
	if defaultRole == nil {
		return
	}
	now := time.Now()
	_ = repositories.UserRoleRepository.Create(tx, &models.UserRole{
		UserID: user.ID,
		RoleID: defaultRole.ID,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   user.ID,
			CreateUserName: user.Username,
			UpdatedAt:      now,
			UpdateUserID:   user.ID,
			UpdateUserName: user.Username,
		},
	})
}

func (s *oidcLoginService) syncOIDCUserOrganizations(tx *gorm.DB, user *models.User, profile *oidcLoginProfile) {
	if user == nil || user.ID <= 0 {
		return
	}
	now := time.Now()
	var activeOrgID int64 = user.ActiveOrgID

	if len(profile.Organizations) > 0 {
		for _, orgClaim := range profile.Organizations {
			orgCode := strings.TrimSpace(orgClaim.ID)
			if orgCode == "" {
				continue
			}
			orgName := strings.TrimSpace(orgClaim.Name)
			if orgName == "" {
				orgName = orgCode
			}
			role := strings.ToUpper(strings.TrimSpace(orgClaim.Role))
			if role == "" {
				role = "MEMBER"
			}

			org := repositories.OrganizationRepository.GetByCode(tx, orgCode)
			if org == nil {
				org = &models.Organization{
					Code:   orgCode,
					Name:   orgName,
					Plan:   "free",
					Status: enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   user.ID,
						CreateUserName: user.Username,
						UpdatedAt:      now,
						UpdateUserID:   user.ID,
						UpdateUserName: user.Username,
					},
				}
				if err := repositories.OrganizationRepository.Create(tx, org); err != nil {
					continue
				}
			} else if orgName != "" && org.Name != orgName {
				_ = repositories.OrganizationRepository.UpdateColumn(tx, org.ID, "name", orgName)
			}

			member := repositories.OrganizationMemberRepository.GetByOrgAndUser(tx, org.ID, user.ID)
			if member == nil {
				member = &models.OrganizationMember{
					OrganizationID: org.ID,
					UserID:         user.ID,
					Role:           role,
					Status:         enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   user.ID,
						CreateUserName: user.Username,
						UpdatedAt:      now,
						UpdateUserID:   user.ID,
						UpdateUserName: user.Username,
					},
				}
				_ = repositories.OrganizationMemberRepository.Create(tx, member)
			} else if member.Role != role || member.Status != enums.StatusOk {
				_ = repositories.OrganizationMemberRepository.Updates(tx, member.ID, map[string]any{
					"role":             role,
					"status":           enums.StatusOk,
					"update_user_id":   user.ID,
					"update_user_name": user.Username,
					"updated_at":       now,
				})
			}

			if activeOrgID == 0 || (profile.ActiveOrgID != "" && (profile.ActiveOrgID == orgCode || profile.ActiveOrgID == orgClaim.Slug)) {
				activeOrgID = org.ID
			}
		}
	} else {
		existingMemberships := repositories.OrganizationMemberRepository.Find(tx, sqls.NewCnd().Eq("user_id", user.ID).Eq("status", enums.StatusOk))
		if len(existingMemberships) == 0 {
			defaultOrgCode := "org_" + shortSubjectHash(profile.Subject)
			defaultOrgName := s.resolveOIDCNickname(user.Nickname, profile) + "'s Workspace"
			org := repositories.OrganizationRepository.GetByCode(tx, defaultOrgCode)
			if org == nil {
				org = &models.Organization{
					Code:   defaultOrgCode,
					Name:   defaultOrgName,
					Plan:   "free",
					Status: enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   user.ID,
						CreateUserName: user.Username,
						UpdatedAt:      now,
						UpdateUserID:   user.ID,
						UpdateUserName: user.Username,
					},
				}
				_ = repositories.OrganizationRepository.Create(tx, org)
			}
			if org.ID > 0 {
				_ = repositories.OrganizationMemberRepository.Create(tx, &models.OrganizationMember{
					OrganizationID: org.ID,
					UserID:         user.ID,
					Role:           "OWNER",
					Status:         enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   user.ID,
						CreateUserName: user.Username,
						UpdatedAt:      now,
						UpdateUserID:   user.ID,
						UpdateUserName: user.Username,
					},
				})
				activeOrgID = org.ID
			}
		} else if activeOrgID == 0 {
			activeOrgID = existingMemberships[0].OrganizationID
		}
	}

	if activeOrgID > 0 && user.ActiveOrgID != activeOrgID {
		user.ActiveOrgID = activeOrgID
		_ = repositories.UserRepository.UpdateColumn(tx, user.ID, "active_org_id", activeOrgID)
	}
}
