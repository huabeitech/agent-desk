package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

func TestWebhookDOSOrgSync_OrgEvents(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	svc := newWebhookSyncService()

	// 1. Test org.created
	err := svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "org.created",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			OrgID:   "org_tingee_001",
			OrgName: "Tingee Corporation",
			Plan:    "pro",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync org.created failed: %v", err)
	}

	org := repositories.OrganizationRepository.GetByCode(db, "org_tingee_001")
	if org == nil || org.Name != "Tingee Corporation" || org.Plan != "pro" || org.Status != enums.StatusOk {
		t.Fatalf("unexpected created org: %+v", org)
	}

	// 2. Test org.updated
	err = svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "org.updated",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			OrgID:   "org_tingee_001",
			OrgName: "Tingee Global Corp",
			Plan:    "enterprise",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync org.updated failed: %v", err)
	}

	org = repositories.OrganizationRepository.GetByCode(db, "org_tingee_001")
	if org == nil || org.Name != "Tingee Global Corp" || org.Plan != "enterprise" {
		t.Fatalf("unexpected updated org: %+v", org)
	}

	// 3. Test org.deleted
	err = svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "org.deleted",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			OrgID: "org_tingee_001",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync org.deleted failed: %v", err)
	}

	org = repositories.OrganizationRepository.GetByCode(db, "org_tingee_001")
	if org == nil || org.Status != enums.StatusDeleted {
		t.Fatalf("expected org to be deleted, got: %+v", org)
	}
}

func TestWebhookDOSOrgSync_MemberEvents(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	svc := newWebhookSyncService()

	// 1. Test org.member_added
	err := svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "org.member_added",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			OrgID:     "org_tingee_002",
			OrgName:   "Tingee R&D",
			UserID:    "usr_dos_001",
			UserEmail: "member@tingee.com",
			UserName:  "Nguyen Van A",
			Role:      "ADMIN",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync org.member_added failed: %v", err)
	}

	org := repositories.OrganizationRepository.GetByCode(db, "org_tingee_002")
	if org == nil {
		t.Fatalf("expected org to be auto-provisioned")
	}

	user := repositories.UserRepository.GetByEmail(db, "member@tingee.com")
	if user == nil || user.Nickname != "Nguyen Van A" {
		t.Fatalf("expected user to be created: %+v", user)
	}

	member := repositories.OrganizationMemberRepository.GetByOrgAndUser(db, org.ID, user.ID)
	if member == nil || member.Role != "ADMIN" || member.Status != enums.StatusOk {
		t.Fatalf("unexpected member record: %+v", member)
	}

	if user.ActiveOrgID != org.ID {
		t.Fatalf("expected active_org_id = %d, got %d", org.ID, user.ActiveOrgID)
	}

	// 2. Test org.member_removed
	err = svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "org.member_removed",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			OrgID:     "org_tingee_002",
			UserID:    "usr_dos_001",
			UserEmail: "member@tingee.com",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync org.member_removed failed: %v", err)
	}

	member = repositories.OrganizationMemberRepository.GetByOrgAndUser(db, org.ID, user.ID)
	if member == nil || member.Status != enums.StatusDeleted {
		t.Fatalf("expected member to be marked deleted, got: %+v", member)
	}
}

func TestOrganizationService_SwitchAndList(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	user := createAuthTestUser(t, db, "testuser", "secret")

	org1 := &models.Organization{Code: "org1", Name: "Org 1", Status: enums.StatusOk}
	org2 := &models.Organization{Code: "org2", Name: "Org 2", Status: enums.StatusOk}
	_ = repositories.OrganizationRepository.Create(db, org1)
	_ = repositories.OrganizationRepository.Create(db, org2)

	_ = repositories.OrganizationMemberRepository.Create(db, &models.OrganizationMember{
		OrganizationID: org1.ID,
		UserID:         user.ID,
		Role:           "OWNER",
		Status:         enums.StatusOk,
	})
	_ = repositories.OrganizationMemberRepository.Create(db, &models.OrganizationMember{
		OrganizationID: org2.ID,
		UserID:         user.ID,
		Role:           "MEMBER",
		Status:         enums.StatusOk,
	})

	_ = repositories.UserRepository.UpdateColumn(db, user.ID, "active_org_id", org1.ID)

	// List
	res, err := OrganizationService.GetUserOrganizations(user.ID)
	if err != nil {
		t.Fatalf("GetUserOrganizations failed: %v", err)
	}
	if len(res.Organizations) != 2 || res.CurrentOrganizationID != org1.ID {
		t.Fatalf("unexpected user org list: %+v", res)
	}

	// Switch
	switched, err := OrganizationService.SwitchActiveOrganization(user.ID, org2.ID)
	if err != nil {
		t.Fatalf("SwitchActiveOrganization failed: %v", err)
	}
	if switched == nil || switched.ID != org2.ID {
		t.Fatalf("unexpected switched org: %+v", switched)
	}

	updatedUser := repositories.UserRepository.Get(db, user.ID)
	if updatedUser.ActiveOrgID != org2.ID {
		t.Fatalf("expected active_org_id %d, got %d", org2.ID, updatedUser.ActiveOrgID)
	}
}

func TestOrganizationService_CreateAndManageMembers(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	owner := createAuthTestUser(t, db, "owner_user", "secret")
	memberUser := createAuthTestUser(t, db, "member_user", "secret")
	email := "member_user@example.com"
	_ = repositories.UserRepository.UpdateColumn(db, memberUser.ID, "email", email)

	// 1. Create Organization
	created, err := OrganizationService.CreateOrganization(owner.ID, request.OrganizationCreateRequest{
		Name: "Acme Support Org",
		Code: "acme-support",
	})
	if err != nil {
		t.Fatalf("CreateOrganization failed: %v", err)
	}
	if created == nil || created.Name != "Acme Support Org" || created.Role != "OWNER" {
		t.Fatalf("unexpected created org: %+v", created)
	}

	updatedOwner := repositories.UserRepository.Get(db, owner.ID)
	if updatedOwner.ActiveOrgID != created.ID {
		t.Fatalf("expected owner active_org_id = %d, got %d", created.ID, updatedOwner.ActiveOrgID)
	}

	// 2. Get Members
	members, err := OrganizationService.GetOrganizationMembers(owner.ID, created.ID)
	if err != nil {
		t.Fatalf("GetOrganizationMembers failed: %v", err)
	}
	if len(members) != 1 || members[0].UserID != owner.ID || members[0].Role != "OWNER" {
		t.Fatalf("unexpected members: %+v", members)
	}

	// 3. Add Member by email
	added, err := OrganizationService.AddMember(owner.ID, created.ID, request.OrganizationAddMemberRequest{
		EmailOrUsername: "member_user@example.com",
		Role:            "ADMIN",
	})
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if added == nil || added.UserID != memberUser.ID || added.Role != "ADMIN" {
		t.Fatalf("unexpected added member: %+v", added)
	}

	members, _ = OrganizationService.GetOrganizationMembers(owner.ID, created.ID)
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	// 4. Update Organization
	updatedOrg, err := OrganizationService.UpdateOrganization(owner.ID, created.ID, request.OrganizationUpdateRequest{
		Name: "Acme Global Support",
	})
	if err != nil {
		t.Fatalf("UpdateOrganization failed: %v", err)
	}
	if updatedOrg.Name != "Acme Global Support" {
		t.Fatalf("expected name to be Acme Global Support, got %s", updatedOrg.Name)
	}

	// 5. Remove Member
	err = OrganizationService.RemoveMember(owner.ID, created.ID, memberUser.ID)
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	members, _ = OrganizationService.GetOrganizationMembers(owner.ID, created.ID)
	if len(members) != 1 {
		t.Fatalf("expected 1 member after removal, got %d", len(members))
	}
}

func TestWebhookSync_CompanyAndCustomerEvents(t *testing.T) {
	db := setupAuthServiceTestDB(t)
	svc := newWebhookSyncService()

	// 1. Test company.created
	err := svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "company.created",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			CRMCompanyID:      "comp_crm_001",
			Name:              "MetaDOS LLC",
			DomainName:        "metados.com",
			Address:           "Ho Chi Minh City, Vietnam",
			Tier:              "enterprise",
			AccountOwnerEmail: "sales@crove.com",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync company.created failed: %v", err)
	}

	comp := repositories.CompanyRepository.GetByName(db, "MetaDOS LLC")
	if comp == nil || comp.Code != "comp_crm_001" || comp.Status != enums.StatusOk {
		t.Fatalf("unexpected created company: %+v", comp)
	}

	// 2. Test customer.created
	err = svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "customer.created",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			CRMPersonID:  "pers_crm_001",
			CRMCompanyID: "comp_crm_001",
			CompanyName:  "MetaDOS LLC",
			Name:         "Nguyen Van A",
			Email:        "customer_a@metados.com",
			Phone:        "+84901234567",
			JobTitle:     "CTO",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync customer.created failed: %v", err)
	}

	cust := repositories.CustomerRepository.FindOne(db, sqls.NewCnd().Eq("primary_email", "customer_a@metados.com"))
	if cust == nil || cust.Name != "Nguyen Van A" || cust.PrimaryMobile != "+84901234567" || cust.CompanyID != comp.ID {
		t.Fatalf("unexpected created customer: %+v", cust)
	}

	identity := repositories.CustomerIdentityRepository.FindOne(db, sqls.NewCnd().
		Eq("external_source", enums.ExternalSourceTwentyCRM).
		Eq("external_id", "pers_crm_001"))
	if identity == nil || identity.CustomerID != cust.ID {
		t.Fatalf("unexpected customer identity: %+v", identity)
	}

	// 3. Test customer.updated
	err = svc.HandleOrgSync(request.OrgSyncWebhookRequest{
		Event:     "customer.updated",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: request.OrgSyncEventData{
			CRMPersonID: "pers_crm_001",
			Name:        "Nguyen Van A (Updated)",
			Email:       "customer_a@metados.com",
			JobTitle:    "VP of Engineering",
		},
	})
	if err != nil {
		t.Fatalf("HandleOrgSync customer.updated failed: %v", err)
	}

	cust = repositories.CustomerRepository.Get(db, cust.ID)
	if cust == nil || cust.Name != "Nguyen Van A (Updated)" || cust.Remark != "VP of Engineering" {
		t.Fatalf("unexpected updated customer: %+v", cust)
	}
}

func TestWebhookSignatureVerification_TimestampFormat(t *testing.T) {
	svc := newWebhookSyncService()
	secret := "test-secret-key-12345"

	// Mock config
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			OrgSyncSecret: secret,
		},
	}
	config.SetCurrent(cfg)

	payload := []byte(`{"event":"customer.created","data":{"name":"John Doe"}}`)
	nowMs := time.Now().UnixMilli()
	tsStr := fmt.Sprintf("%d", nowMs)

	// Compute HMAC-SHA256 of timestamp.payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(tsStr + "." + string(payload)))
	sigHex := hex.EncodeToString(mac.Sum(nil))

	header := fmt.Sprintf("t=%s,v1=%s", tsStr, sigHex)

	if !svc.VerifySignature(payload, header) {
		t.Fatalf("expected signature %q to be verified successfully", header)
	}

	// Test expired timestamp (> 5 minutes ago)
	oldMs := time.Now().Add(-10 * time.Minute).UnixMilli()
	oldTsStr := fmt.Sprintf("%d", oldMs)
	macOld := hmac.New(sha256.New, []byte(secret))
	macOld.Write([]byte(oldTsStr + "." + string(payload)))
	oldSigHex := hex.EncodeToString(macOld.Sum(nil))
	oldHeader := fmt.Sprintf("t=%s,v1=%s", oldTsStr, oldSigHex)

	if svc.VerifySignature(payload, oldHeader) {
		t.Fatalf("expected expired signature %q to fail verification", oldHeader)
	}

	// Test sha256= format
	macRaw := hmac.New(sha256.New, []byte(secret))
	macRaw.Write(payload)
	rawSigHex := hex.EncodeToString(macRaw.Sum(nil))

	if !svc.VerifySignature(payload, "sha256="+rawSigHex) {
		t.Fatalf("expected sha256= signature to be verified successfully")
	}
}
