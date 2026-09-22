package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/repositories"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"strconv"
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
	cfg := config.GetCurrent()
	if cfg == nil {
		return true
	}
	secret := strings.TrimSpace(cfg.Webhook.OrgSyncSecret)
	if secret == "" {
		secret = strings.TrimSpace(cfg.Webhook.DOSOrgSyncSecret)
	}
	if secret == "" {
		secret = strings.TrimSpace(cfg.OIDC.ClientSecret)
	}
	if secret == "" {
		return true
	}

	sigHeader := strings.TrimSpace(signature)
	if sigHeader == "" {
		return false
	}

	// 1. Check for format: t=<timestamp>,v1=<signature>
	if strings.Contains(sigHeader, "t=") && (strings.Contains(sigHeader, "v1=") || strings.Contains(sigHeader, "v0=")) {
		parts := strings.Split(sigHeader, ",")
		var tsStr, expectedSig string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "t=") {
				tsStr = strings.TrimPrefix(p, "t=")
			} else if strings.HasPrefix(p, "v1=") {
				expectedSig = strings.TrimPrefix(p, "v1=")
			} else if strings.HasPrefix(p, "v0=") && expectedSig == "" {
				expectedSig = strings.TrimPrefix(p, "v0=")
			}
		}

		if tsStr != "" && expectedSig != "" {
			// Timestamp anti-replay check (5 minutes)
			var tsInt int64
			if parsed, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
				tsInt = parsed
				now := time.Now().Unix()
				if tsInt > 1e11 { // ms
					now = time.Now().UnixMilli()
					if math.Abs(float64(now-tsInt)) > float64(5*60*1000) {
						return false
					}
				} else {
					if math.Abs(float64(now-tsInt)) > float64(5*60) {
						return false
					}
				}
			}

			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(tsStr + "." + string(payload)))
			computed := hex.EncodeToString(mac.Sum(nil))
			if hmac.Equal([]byte(expectedSig), []byte(computed)) {
				return true
			}
		}
	}

	// 2. Fallback to direct sha256= signature or raw hex signature
	sig := sigHeader
	if strings.HasPrefix(sig, "sha256=") {
		sig = strings.TrimPrefix(sig, "sha256=")
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
		orgCode = strings.TrimSpace(data.ID)
	}
	if orgCode == "" {
		orgCode = strings.TrimSpace(data.Slug)
	}
	data.OrgID = orgCode

	if data.OrgName == "" && data.Name != "" {
		data.OrgName = data.Name
	}
	if data.UserEmail == "" && data.BillingEmail != "" {
		data.UserEmail = data.BillingEmail
	}

	switch event {
	case "org.created", "org.updated", "organization.created", "organization.updated":
		if orgCode == "" {
			return errorsx.InvalidParam("org_id or id is required")
		}
		return s.handleOrgUpsert(data)
	case "org.deleted", "organization.deleted":
		if orgCode == "" {
			return errorsx.InvalidParam("org_id or id is required")
		}
		return s.handleOrgDelete(orgCode)
	case "org.member_added", "org.member_updated", "organization.member.added", "organization.member.updated", "organization.member_added":
		if orgCode == "" {
			return errorsx.InvalidParam("org_id or id is required")
		}
		return s.handleMemberUpsert(data)
	case "org.member_removed", "organization.member.removed", "organization.member_removed":
		if orgCode == "" {
			return errorsx.InvalidParam("org_id or id is required")
		}
		return s.handleMemberRemove(data)
	case "company.created", "company.updated":
		return s.handleCompanyUpsert(data)
	case "customer.created", "customer.updated":
		return s.handleCustomerUpsert(data)
	default:
		return nil
	}
}

func (s *webhookSyncService) HandleDOSOrgSync(req request.DOSOrgSyncWebhookRequest) error {
	return s.HandleOrgSync(req)
}

func (s *webhookSyncService) DispatchOutboundEvent(event string, data request.OrgSyncEventData) {
	cfg := config.GetCurrent()
	if cfg == nil {
		return
	}
	outboundURL := strings.TrimSpace(cfg.Webhook.OutboundURL)
	if outboundURL == "" {
		return
	}

	payload := request.OrgSyncWebhookRequest{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal outbound event", "event", event, "error", err)
		return
	}

	secret := strings.TrimSpace(cfg.Webhook.OrgSyncSecret)
	if secret == "" {
		secret = strings.TrimSpace(cfg.Webhook.DOSOrgSyncSecret)
	}
	if secret == "" {
		secret = strings.TrimSpace(cfg.OIDC.ClientSecret)
	}

	var signature string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(bodyBytes)
		signature = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest(http.MethodPost, outboundURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			slog.Error("failed to create outbound sync request", "url", outboundURL, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-DOS-Event", event)
		if signature != "" {
			req.Header.Set("X-DOS-Signature", signature)
			req.Header.Set("X-Webhook-Signature", signature)
		}

		resp, err := client.Do(req)
		if err != nil {
			slog.Error("failed to dispatch outbound sync event", "event", event, "url", outboundURL, "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			slog.Warn("outbound sync event returned non-2xx status", "event", event, "status", resp.StatusCode)
		} else {
			slog.Info("outbound sync event dispatched successfully", "event", event)
		}
	}()
}

func (s *webhookSyncService) DispatchOutboundOrgEvent(event string, data request.OrgSyncEventData) {
	s.DispatchOutboundEvent(event, data)
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

func (s *webhookSyncService) handleCompanyUpsert(data request.OrgSyncEventData) error {
	name := strings.TrimSpace(data.Name)
	if name == "" {
		name = strings.TrimSpace(data.OrgName)
	}
	if name == "" {
		name = strings.TrimSpace(data.CRMCompanyID)
	}
	if name == "" {
		return errorsx.InvalidParam("company name is required")
	}

	code := strings.TrimSpace(data.CRMCompanyID)
	if code == "" {
		code = strings.TrimSpace(data.ID)
	}
	if code == "" {
		code = strings.TrimSpace(data.CompanyID)
	}
	if code == "" {
		code = strings.TrimSpace(data.DeskCompanyID)
	}

	remark := strings.TrimSpace(data.DomainName)
	if remark == "" {
		remark = strings.TrimSpace(data.Domain)
	}
	if data.TaxCode != "" {
		if remark != "" {
			remark += " | Tax: " + data.TaxCode
		} else {
			remark = "Tax: " + data.TaxCode
		}
	}
	if data.Address != "" {
		if remark != "" {
			remark += " | " + data.Address
		} else {
			remark = data.Address
		}
	}
	if data.Tier != "" {
		if remark != "" {
			remark += " | Tier: " + data.Tier
		} else {
			remark = "Tier: " + data.Tier
		}
	}

	now := time.Now()
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		var company *models.Company
		if code != "" {
			company = repositories.CompanyRepository.FindOne(ctx.Tx, sqls.NewCnd().Eq("code", code))
		}
		if company == nil {
			company = repositories.CompanyRepository.GetByName(ctx.Tx, name)
		}

		if company == nil {
			company = &models.Company{
				Name:   name,
				Code:   code,
				Status: enums.StatusOk,
				Remark: remark,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   0,
					CreateUserName: "crm-sync",
					UpdatedAt:      now,
					UpdateUserID:   0,
					UpdateUserName: "crm-sync",
				},
			}
			return repositories.CompanyRepository.Create(ctx.Tx, company)
		}

		updates := map[string]any{
			"name":             name,
			"status":           enums.StatusOk,
			"update_user_id":   0,
			"update_user_name": "crm-sync",
			"updated_at":       now,
		}
		if code != "" {
			updates["code"] = code
		}
		if remark != "" {
			updates["remark"] = remark
		}
		return repositories.CompanyRepository.Updates(ctx.Tx, company.ID, updates)
	})
}

func (s *webhookSyncService) handleCustomerUpsert(data request.OrgSyncEventData) error {
	name := strings.TrimSpace(data.Name)
	if name == "" {
		name = strings.TrimSpace(data.UserName)
	}
	email := strings.TrimSpace(strings.ToLower(data.Email))
	if email == "" {
		email = strings.TrimSpace(strings.ToLower(data.UserEmail))
	}
	phone := strings.TrimSpace(data.Phone)
	crmPersonID := strings.TrimSpace(data.CRMPersonID)
	if crmPersonID == "" {
		crmPersonID = strings.TrimSpace(data.ID)
	}
	crmCompanyID := strings.TrimSpace(data.CRMCompanyID)
	if crmCompanyID == "" {
		crmCompanyID = strings.TrimSpace(data.CompanyID)
	}
	companyName := strings.TrimSpace(data.CompanyName)

	if name == "" && email == "" && phone == "" && crmPersonID == "" {
		return errorsx.InvalidParam("at least one customer identifier is required")
	}
	if name == "" {
		if email != "" {
			name = email
		} else if phone != "" {
			name = phone
		} else {
			name = "Customer " + crmPersonID
		}
	}

	now := time.Now()
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		var companyID int64 = 0
		if crmCompanyID != "" {
			company := repositories.CompanyRepository.FindOne(ctx.Tx, sqls.NewCnd().Eq("code", crmCompanyID))
			if company != nil {
				companyID = company.ID
			}
		}
		if companyID == 0 && companyName != "" {
			company := repositories.CompanyRepository.GetByName(ctx.Tx, companyName)
			if company != nil {
				companyID = company.ID
			} else {
				newComp := &models.Company{
					Name:   companyName,
					Code:   crmCompanyID,
					Status: enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   0,
						CreateUserName: "crm-sync",
						UpdatedAt:      now,
						UpdateUserID:   0,
						UpdateUserName: "crm-sync",
					},
				}
				if err := repositories.CompanyRepository.Create(ctx.Tx, newComp); err == nil {
					companyID = newComp.ID
				}
			}
		}

		var customer *models.Customer
		if crmPersonID != "" {
			identity := repositories.CustomerIdentityRepository.FindOne(ctx.Tx, sqls.NewCnd().
				Eq("external_source", enums.ExternalSourceTwentyCRM).
				Eq("external_id", crmPersonID))
			if identity != nil {
				customer = repositories.CustomerRepository.Get(ctx.Tx, identity.CustomerID)
			}
		}
		if customer == nil && email != "" {
			customer = repositories.CustomerRepository.FindOne(ctx.Tx, sqls.NewCnd().Eq("primary_email", email))
		}
		if customer == nil && phone != "" {
			customer = repositories.CustomerRepository.FindOne(ctx.Tx, sqls.NewCnd().Eq("primary_mobile", phone))
		}

		if customer == nil {
			customer = &models.Customer{
				Name:          name,
				PrimaryEmail:  email,
				PrimaryMobile: phone,
				CompanyID:     companyID,
				Status:        enums.StatusOk,
				Remark:        data.JobTitle,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   0,
					CreateUserName: "crm-sync",
					UpdatedAt:      now,
					UpdateUserID:   0,
					UpdateUserName: "crm-sync",
				},
			}
			if err := repositories.CustomerRepository.Create(ctx.Tx, customer); err != nil {
				return err
			}
		} else {
			updates := map[string]any{
				"name":             name,
				"status":           enums.StatusOk,
				"update_user_id":   0,
				"update_user_name": "crm-sync",
				"updated_at":       now,
			}
			if email != "" {
				updates["primary_email"] = email
			}
			if phone != "" {
				updates["primary_mobile"] = phone
			}
			if companyID > 0 {
				updates["company_id"] = companyID
			}
			if data.JobTitle != "" {
				updates["remark"] = data.JobTitle
			}
			if err := repositories.CustomerRepository.Updates(ctx.Tx, customer.ID, updates); err != nil {
				return err
			}
		}

		if crmPersonID != "" {
			identity := repositories.CustomerIdentityRepository.FindOne(ctx.Tx, sqls.NewCnd().
				Eq("external_source", enums.ExternalSourceTwentyCRM).
				Eq("external_id", crmPersonID))
			if identity == nil {
				_ = repositories.CustomerIdentityRepository.Create(ctx.Tx, &models.CustomerIdentity{
					CustomerID:     customer.ID,
					ExternalSource: enums.ExternalSourceTwentyCRM,
					ExternalID:     crmPersonID,
					Status:         enums.StatusOk,
					AuditFields: models.AuditFields{
						CreatedAt:      now,
						CreateUserID:   0,
						CreateUserName: "crm-sync",
						UpdatedAt:      now,
						UpdateUserID:   0,
						UpdateUserName: "crm-sync",
					},
				})
			}
		}

		return nil
	})
}
