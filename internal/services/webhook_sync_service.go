package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/repositories"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/mlogclub/simple/sqls"
)

var WebhookSyncService = newWebhookSyncService()

func newWebhookSyncService() *webhookSyncService {
	return &webhookSyncService{}
}

type webhookSyncService struct{}

func (s *webhookSyncService) VerifySignature(payload []byte, signature string) bool {
	secret := strings.TrimSpace(config.Current().Webhook.OrgSyncSecret)
	if secret == "" {
		secret = strings.TrimSpace(config.Current().Webhook.DOSOrgSyncSecret)
	}
	if secret == "" {
		secret = strings.TrimSpace(config.Current().OIDC.ClientSecret)
	}
	if secret == "" {
		// No signing secret configured: the sync endpoints cannot be
		// authenticated, so reject every request rather than accept it. An
		// empty secret used to pass here, which left the unauthenticated
		// org-sync webhook open to anyone.
		return false
	}

	sig := strings.TrimSpace(signature)
	if strings.HasPrefix(sig, "sha256=") {
		sig = strings.TrimPrefix(sig, "sha256=")
	}
	if sig == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
}

func (s *webhookSyncService) HandleOrgSync(req request.OrgSyncWebhookRequest) error {
	event := strings.TrimSpace(req.Event)
	data := req.Data

	orgCode := strings.TrimSpace(data.OrgID)
	if orgCode == "" {
		return errorsx.InvalidParam("org_id is required")
	}

	switch event {
	case "org.created", "org.updated":
		return s.handleOrgUpsert(data)
	case "org.deleted":
		return s.handleOrgDelete(orgCode)
	case "org.member_added", "org.member_updated":
		return s.handleMemberUpsert(data)
	case "org.member_removed":
		return s.handleMemberRemove(data)
	default:
		return nil
	}
}

func (s *webhookSyncService) HandleDOSOrgSync(req request.DOSOrgSyncWebhookRequest) error {
	return s.HandleOrgSync(req)
}

func (s *webhookSyncService) handleOrgUpsert(data request.OrgSyncEventData) error {
	now := time.Now()
	orgCode := strings.TrimSpace(data.OrgID)
	orgName := strings.TrimSpace(data.OrgName)
	if orgName == "" {
		orgName = orgCode
	}
	plan := strings.TrimSpace(data.Plan)
	if plan == "" {
		plan = "free"
	}

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		org := repositories.OrganizationRepository.GetByCode(ctx.Tx, orgCode)
		if org == nil {
			org = &models.Organization{
				Code:   orgCode,
				Name:   orgName,
				Plan:   plan,
				Status: enums.StatusOk,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   0,
					CreateUserName: "webhook-sync",
					UpdatedAt:      now,
					UpdateUserID:   0,
					UpdateUserName: "webhook-sync",
				},
			}
			return repositories.OrganizationRepository.Create(ctx.Tx, org)
		}

		updates := map[string]any{
			"name":             orgName,
			"status":           enums.StatusOk,
			"update_user_id":   0,
			"update_user_name": "webhook-sync",
			"updated_at":       now,
		}
		if plan != "" {
			updates["plan"] = plan
		}
		return repositories.OrganizationRepository.Updates(ctx.Tx, org.ID, updates)
	})
}

func (s *webhookSyncService) handleOrgDelete(orgCode string) error {
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		org := repositories.OrganizationRepository.GetByCode(ctx.Tx, orgCode)
		if org == nil {
			return nil
		}
		return repositories.OrganizationRepository.UpdateColumn(ctx.Tx, org.ID, "status", enums.StatusDeleted)
	})
}

func (s *webhookSyncService) handleMemberUpsert(data request.OrgSyncEventData) error {
	now := time.Now()
	orgCode := strings.TrimSpace(data.OrgID)
	userSubject := strings.TrimSpace(data.UserID)
	userEmail := strings.TrimSpace(strings.ToLower(data.UserEmail))
	userName := strings.TrimSpace(data.UserName)
	role := strings.ToUpper(strings.TrimSpace(data.Role))
	if role == "" {
		role = "MEMBER"
	}

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		org := repositories.OrganizationRepository.GetByCode(ctx.Tx, orgCode)
		if org == nil {
			orgName := strings.TrimSpace(data.OrgName)
			if orgName == "" {
				orgName = orgCode
			}
			org = &models.Organization{
				Code:   orgCode,
				Name:   orgName,
				Plan:   "free",
				Status: enums.StatusOk,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   0,
					CreateUserName: "webhook-sync",
					UpdatedAt:      now,
					UpdateUserID:   0,
					UpdateUserName: "webhook-sync",
				},
			}
			if err := repositories.OrganizationRepository.Create(ctx.Tx, org); err != nil {
				return err
			}
		}

		var user *models.User
		if userSubject != "" {
			identity := repositories.UserIdentityRepository.GetBy(ctx.Tx, enums.ThirdProviderOIDC, "", userSubject)
			if identity != nil {
				user = repositories.UserRepository.Get(ctx.Tx, identity.UserID)
			}
		}
		if user == nil && userEmail != "" {
			user = repositories.UserRepository.GetByEmail(ctx.Tx, userEmail)
		}
		if user == nil {
			username := userEmail
			if username == "" {
				username = "u_" + userSubject
			}
			user = &models.User{
				Username: username,
				Nickname: userName,
				Email:    &userEmail,
				UserType: enums.UserTypeEmployee,
				Status:   enums.StatusOk,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   0,
					CreateUserName: "webhook-sync",
					UpdatedAt:      now,
					UpdateUserID:   0,
					UpdateUserName: "webhook-sync",
				},
			}
			if err := repositories.UserRepository.Create(ctx.Tx, user); err != nil {
				return err
			}
			if userSubject != "" {
				_ = repositories.UserIdentityRepository.Create(ctx.Tx, &models.UserIdentity{
					UserID:         user.ID,
					Provider:       enums.ThirdProviderOIDC,
					ProviderUserID: userSubject,
					ProviderName:   "OIDC",
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
			}
		}

		member := repositories.OrganizationMemberRepository.GetByOrgAndUser(ctx.Tx, org.ID, user.ID)
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
			if err := repositories.OrganizationMemberRepository.Create(ctx.Tx, member); err != nil {
				return err
			}
		} else {
			_ = repositories.OrganizationMemberRepository.Updates(ctx.Tx, member.ID, map[string]any{
				"role":             role,
				"status":           enums.StatusOk,
				"update_user_id":   user.ID,
				"update_user_name": user.Username,
				"updated_at":       now,
			})
		}

		if user.ActiveOrgID == 0 {
			_ = repositories.UserRepository.UpdateColumn(ctx.Tx, user.ID, "active_org_id", org.ID)
		}

		return nil
	})
}

func (s *webhookSyncService) handleMemberRemove(data request.OrgSyncEventData) error {
	orgCode := strings.TrimSpace(data.OrgID)
	userSubject := strings.TrimSpace(data.UserID)
	userEmail := strings.TrimSpace(strings.ToLower(data.UserEmail))

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		org := repositories.OrganizationRepository.GetByCode(ctx.Tx, orgCode)
		if org == nil {
			return nil
		}

		var user *models.User
		if userSubject != "" {
			identity := repositories.UserIdentityRepository.GetBy(ctx.Tx, enums.ThirdProviderOIDC, "", userSubject)
			if identity != nil {
				user = repositories.UserRepository.Get(ctx.Tx, identity.UserID)
			}
		}
		if user == nil && userEmail != "" {
			user = repositories.UserRepository.GetByEmail(ctx.Tx, userEmail)
		}
		if user == nil {
			return nil
		}

		member := repositories.OrganizationMemberRepository.GetByOrgAndUser(ctx.Tx, org.ID, user.ID)
		if member != nil {
			_ = repositories.OrganizationMemberRepository.UpdateColumn(ctx.Tx, member.ID, "status", enums.StatusDeleted)
		}

		if user.ActiveOrgID == org.ID {
			remaining := repositories.OrganizationMemberRepository.Find(ctx.Tx, sqls.NewCnd().Eq("user_id", user.ID).Eq("status", enums.StatusOk).Where("organization_id <> ?", org.ID))
			var newActiveOrgID int64 = 0
			if len(remaining) > 0 {
				newActiveOrgID = remaining[0].OrganizationID
			}
			_ = repositories.UserRepository.UpdateColumn(ctx.Tx, user.ID, "active_org_id", newActiveOrgID)
		}

		return nil
	})
}
