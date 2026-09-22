package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/repositories"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/mlogclub/simple/sqls"
)

var OrganizationService = newOrganizationService()

func newOrganizationService() *organizationService {
	return &organizationService{}
}

type organizationService struct{}

func (s *organizationService) GetUserOrganizations(userID int64) (*response.UserOrganizationListResponse, error) {
	user := repositories.UserRepository.Get(sqls.DB(), userID)
	if user == nil {
		return nil, errorsx.InvalidAccountI18n("error.e0260")
	}

	memberships := repositories.OrganizationMemberRepository.Find(sqls.DB(), sqls.NewCnd().Eq("user_id", userID).Eq("status", enums.StatusOk))
	if len(memberships) == 0 {
		return &response.UserOrganizationListResponse{
			CurrentOrganizationID: user.ActiveOrgID,
			Organizations:         []response.OrganizationResponse{},
		}, nil
	}

	orgIDs := make([]int64, 0, len(memberships))
	roleMap := make(map[int64]string, len(memberships))
	for _, m := range memberships {
		orgIDs = append(orgIDs, m.OrganizationID)
		roleMap[m.OrganizationID] = m.Role
	}

	orgs := repositories.OrganizationRepository.Find(sqls.DB(), sqls.NewCnd().In("id", orgIDs).Eq("status", enums.StatusOk))

	resList := make([]response.OrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		resList = append(resList, response.OrganizationResponse{
			ID:        org.ID,
			Code:      org.Code,
			Name:      org.Name,
			Logo:      org.Logo,
			Plan:      org.Plan,
			Status:    org.Status,
			Role:      roleMap[org.ID],
			IsActive:  org.ID == user.ActiveOrgID,
			CreatedAt: org.CreatedAt,
		})
	}

	return &response.UserOrganizationListResponse{
		CurrentOrganizationID: user.ActiveOrgID,
		Organizations:         resList,
	}, nil
}

func (s *organizationService) CreateOrganization(userID int64, req request.OrganizationCreateRequest) (*response.OrganizationResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParam("organization name is required")
	}

	user := repositories.UserRepository.Get(sqls.DB(), userID)
	if user == nil {
		return nil, errorsx.InvalidAccountI18n("error.e0260")
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = s.generateUniqueOrgCode(name)
	}

	var createdOrg *models.Organization
	now := time.Now()

	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if existing := repositories.OrganizationRepository.GetByCode(ctx.Tx, code); existing != nil {
			return errorsx.InvalidParam("organization code already in use")
		}

		createdOrg = &models.Organization{
			Code:   code,
			Name:   name,
			Logo:   strings.TrimSpace(req.Logo),
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

		if err := repositories.OrganizationRepository.Create(ctx.Tx, createdOrg); err != nil {
			return err
		}

		member := &models.OrganizationMember{
			OrganizationID: createdOrg.ID,
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
		}

		if err := repositories.OrganizationMemberRepository.Create(ctx.Tx, member); err != nil {
			return err
		}

		_ = repositories.UserRepository.UpdateColumn(ctx.Tx, user.ID, "active_org_id", createdOrg.ID)
		return nil
	})

	if err != nil {
		return nil, err
	}

	userEmail := ""
	if user.Email != nil {
		userEmail = *user.Email
	}
	WebhookSyncService.DispatchOutboundOrgEvent("org.created", request.OrgSyncEventData{
		OrgID:     createdOrg.Code,
		OrgName:   createdOrg.Name,
		UserID:    user.Username,
		UserEmail: userEmail,
		UserName:  user.Nickname,
		Role:      "OWNER",
		Plan:      createdOrg.Plan,
	})

	return &response.OrganizationResponse{
		ID:        createdOrg.ID,
		Code:      createdOrg.Code,
		Name:      createdOrg.Name,
		Logo:      createdOrg.Logo,
		Plan:      createdOrg.Plan,
		Status:    createdOrg.Status,
		Role:      "OWNER",
		IsActive:  true,
		CreatedAt: createdOrg.CreatedAt,
	}, nil
}

func (s *organizationService) SwitchActiveOrganization(userID int64, orgID int64) (*models.Organization, error) {
	user := repositories.UserRepository.Get(sqls.DB(), userID)
	if user == nil {
		return nil, errorsx.InvalidAccountI18n("error.e0260")
	}

	member := repositories.OrganizationMemberRepository.GetByOrgAndUser(sqls.DB(), orgID, userID)
	if member == nil || member.Status != enums.StatusOk {
		return nil, errorsx.ForbiddenI18n("error.e0225")
	}

	org := repositories.OrganizationRepository.Get(sqls.DB(), orgID)
	if org == nil || org.Status != enums.StatusOk {
		return nil, errorsx.InvalidParam("organization not found or disabled")
	}

	if err := repositories.UserRepository.UpdateColumn(sqls.DB(), userID, "active_org_id", orgID); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *organizationService) GetOrganizationMembers(currentUserID int64, orgID int64) ([]response.OrganizationMemberResponse, error) {
	member := repositories.OrganizationMemberRepository.GetByOrgAndUser(sqls.DB(), orgID, currentUserID)
	if member == nil || member.Status != enums.StatusOk {
		return nil, errorsx.ForbiddenI18n("error.e0225")
	}

	memberships := repositories.OrganizationMemberRepository.Find(sqls.DB(), sqls.NewCnd().Eq("organization_id", orgID).Eq("status", enums.StatusOk))
	if len(memberships) == 0 {
		return []response.OrganizationMemberResponse{}, nil
	}

	userIDs := make([]int64, 0, len(memberships))
	for _, m := range memberships {
		userIDs = append(userIDs, m.UserID)
	}

	users := repositories.UserRepository.Find(sqls.DB(), sqls.NewCnd().In("id", userIDs))
	userMap := make(map[int64]models.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	res := make([]response.OrganizationMemberResponse, 0, len(memberships))
	for _, m := range memberships {
		u := userMap[m.UserID]
		email := ""
		if u.Email != nil {
			email = *u.Email
		}
		res = append(res, response.OrganizationMemberResponse{
			ID:        m.ID,
			UserID:    m.UserID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			Email:     email,
			Avatar:    u.Avatar,
			Role:      m.Role,
			Status:    int(m.Status),
			CreatedAt: m.CreatedAt,
		})
	}

	return res, nil
}

func (s *organizationService) AddMember(currentUserID int64, orgID int64, req request.OrganizationAddMemberRequest) (*response.OrganizationMemberResponse, error) {
	currentMember := repositories.OrganizationMemberRepository.GetByOrgAndUser(sqls.DB(), orgID, currentUserID)
	if currentMember == nil || currentMember.Status != enums.StatusOk || (currentMember.Role != "OWNER" && currentMember.Role != "ADMIN") {
		return nil, errorsx.ForbiddenI18n("error.e0225")
	}

	query := strings.TrimSpace(req.EmailOrUsername)
	if query == "" {
		return nil, errorsx.InvalidParam("email or username is required")
	}

	targetUser := repositories.UserRepository.GetByUsername(sqls.DB(), query)
	if targetUser == nil {
		targetUser = repositories.UserRepository.GetByEmail(sqls.DB(), strings.ToLower(query))
	}
	if targetUser == nil {
		return nil, errorsx.InvalidParam("user not found")
	}

	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role != "ADMIN" && role != "MEMBER" {
		role = "MEMBER"
	}

	now := time.Now()
	var member *models.OrganizationMember

	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		existing := repositories.OrganizationMemberRepository.GetByOrgAndUser(ctx.Tx, orgID, targetUser.ID)
		if existing == nil {
			member = &models.OrganizationMember{
				OrganizationID: orgID,
				UserID:         targetUser.ID,
				Role:           role,
				Status:         enums.StatusOk,
				AuditFields: models.AuditFields{
					CreatedAt:      now,
					CreateUserID:   currentUserID,
					CreateUserName: "",
					UpdatedAt:      now,
					UpdateUserID:   currentUserID,
					UpdateUserName: "",
				},
			}
			if err := repositories.OrganizationMemberRepository.Create(ctx.Tx, member); err != nil {
				return err
			}
		} else {
			member = existing
			member.Role = role
			member.Status = enums.StatusOk
			_ = repositories.OrganizationMemberRepository.Updates(ctx.Tx, member.ID, map[string]any{
				"role":             role,
				"status":           enums.StatusOk,
				"update_user_id":   currentUserID,
				"update_user_name": "",
				"updated_at":       now,
			})
		}

		if targetUser.ActiveOrgID == 0 {
			_ = repositories.UserRepository.UpdateColumn(ctx.Tx, targetUser.ID, "active_org_id", orgID)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	orgCode := ""
	if targetOrg := repositories.OrganizationRepository.Get(sqls.DB(), orgID); targetOrg != nil {
		orgCode = targetOrg.Code
	}

	email := ""
	if targetUser.Email != nil {
		email = *targetUser.Email
	}

	WebhookSyncService.DispatchOutboundOrgEvent("org.member_added", request.OrgSyncEventData{
		OrgID:     orgCode,
		UserID:    targetUser.Username,
		UserEmail: email,
		UserName:  targetUser.Nickname,
		Role:      member.Role,
	})

	return &response.OrganizationMemberResponse{
		ID:        member.ID,
		UserID:    targetUser.ID,
		Username:  targetUser.Username,
		Nickname:  targetUser.Nickname,
		Email:     email,
		Avatar:    targetUser.Avatar,
		Role:      member.Role,
		Status:    int(member.Status),
		CreatedAt: member.CreatedAt,
	}, nil
}

func (s *organizationService) RemoveMember(currentUserID int64, orgID int64, targetUserID int64) error {
	currentMember := repositories.OrganizationMemberRepository.GetByOrgAndUser(sqls.DB(), orgID, currentUserID)
	if currentMember == nil || currentMember.Status != enums.StatusOk {
		return errorsx.ForbiddenI18n("error.e0225")
	}

	if currentUserID != targetUserID && currentMember.Role != "OWNER" && currentMember.Role != "ADMIN" {
		return errorsx.ForbiddenI18n("error.e0225")
	}

	targetMember := repositories.OrganizationMemberRepository.GetByOrgAndUser(sqls.DB(), orgID, targetUserID)
	if targetMember == nil || targetMember.Status != enums.StatusOk {
		return errorsx.InvalidParam("member not found in organization")
	}

	if targetMember.Role == "OWNER" {
		owners := repositories.OrganizationMemberRepository.Find(sqls.DB(), sqls.NewCnd().Eq("organization_id", orgID).Eq("role", "OWNER").Eq("status", enums.StatusOk))
		if len(owners) <= 1 {
			return errorsx.InvalidParam("cannot remove the only owner of the organization")
		}
	}

	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.OrganizationMemberRepository.UpdateColumn(ctx.Tx, targetMember.ID, "status", enums.StatusDeleted); err != nil {
			return err
		}

		targetUser := repositories.UserRepository.Get(ctx.Tx, targetUserID)
		if targetUser != nil && targetUser.ActiveOrgID == orgID {
			remaining := repositories.OrganizationMemberRepository.Find(ctx.Tx, sqls.NewCnd().Eq("user_id", targetUserID).Eq("status", enums.StatusOk).Where("organization_id <> ?", orgID))
			var newActiveOrgID int64 = 0
			if len(remaining) > 0 {
				newActiveOrgID = remaining[0].OrganizationID
			}
			_ = repositories.UserRepository.UpdateColumn(ctx.Tx, targetUserID, "active_org_id", newActiveOrgID)
		}
		return nil
	})

	if err != nil {
		return err
	}

	orgCode := ""
	if targetOrg := repositories.OrganizationRepository.Get(sqls.DB(), orgID); targetOrg != nil {
		orgCode = targetOrg.Code
	}
	email := ""
	username := ""
	if targetUser := repositories.UserRepository.Get(sqls.DB(), targetUserID); targetUser != nil {
		username = targetUser.Username
		if targetUser.Email != nil {
			email = *targetUser.Email
		}
	}
	WebhookSyncService.DispatchOutboundOrgEvent("org.member_removed", request.OrgSyncEventData{
		OrgID:     orgCode,
		UserID:    username,
		UserEmail: email,
	})

	return nil
}

func (s *organizationService) UpdateOrganization(currentUserID int64, orgID int64, req request.OrganizationUpdateRequest) (*response.OrganizationResponse, error) {
	currentMember := repositories.OrganizationMemberRepository.GetByOrgAndUser(sqls.DB(), orgID, currentUserID)
	if currentMember == nil || currentMember.Status != enums.StatusOk || (currentMember.Role != "OWNER" && currentMember.Role != "ADMIN") {
		return nil, errorsx.ForbiddenI18n("error.e0225")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParam("organization name cannot be empty")
	}

	org := repositories.OrganizationRepository.Get(sqls.DB(), orgID)
	if org == nil || org.Status != enums.StatusOk {
		return nil, errorsx.InvalidParam("organization not found")
	}

	updates := map[string]any{
		"name":             name,
		"logo":             strings.TrimSpace(req.Logo),
		"update_user_id":   currentUserID,
		"update_user_name": "",
		"updated_at":       time.Now(),
	}

	if err := repositories.OrganizationRepository.Updates(sqls.DB(), orgID, updates); err != nil {
		return nil, err
	}

	org = repositories.OrganizationRepository.Get(sqls.DB(), orgID)

	WebhookSyncService.DispatchOutboundOrgEvent("org.updated", request.OrgSyncEventData{
		OrgID:   org.Code,
		OrgName: org.Name,
		Plan:    org.Plan,
	})

	return &response.OrganizationResponse{
		ID:        org.ID,
		Code:      org.Code,
		Name:      org.Name,
		Logo:      org.Logo,
		Plan:      org.Plan,
		Status:    org.Status,
		Role:      currentMember.Role,
		IsActive:  true,
		CreatedAt: org.CreatedAt,
	}, nil
}

func (s *organizationService) GetActiveOrganization(principal *dto.AuthPrincipal) *models.Organization {
	if principal == nil || principal.UserID <= 0 {
		return nil
	}
	user := repositories.UserRepository.Get(sqls.DB(), principal.UserID)
	if user == nil || user.ActiveOrgID <= 0 {
		return nil
	}
	return repositories.OrganizationRepository.Get(sqls.DB(), user.ActiveOrgID)
}

func (s *organizationService) generateUniqueOrgCode(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteString("-")
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "org"
	}
	if len(slug) > 30 {
		slug = slug[:30]
	}

	if repositories.OrganizationRepository.GetByCode(sqls.DB(), slug) == nil {
		return slug
	}

	for i := 1; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", slug, i)
		if repositories.OrganizationRepository.GetByCode(sqls.DB(), candidate) == nil {
			return candidate
		}
	}

	randBuf := make([]byte, 4)
	_, _ = rand.Read(randBuf)
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(randBuf))
}
