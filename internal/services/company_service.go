package services

import (
	"fmt"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
)

var CompanyService = newCompanyService()

func newCompanyService() *companyService {
	return &companyService{}
}

type companyService struct {
}

func (s *companyService) Get(id int64) *models.Company {
	if id <= 0 {
		return nil
	}
	return repositories.CompanyRepository.Get(sqls.DB(), id)
}

func (s *companyService) Take(where ...interface{}) *models.Company {
	return repositories.CompanyRepository.Take(sqls.DB(), where...)
}

func (s *companyService) Find(cnd *sqls.Cnd) []models.Company {
	return repositories.CompanyRepository.Find(sqls.DB(), cnd)
}

func (s *companyService) FindOne(cnd *sqls.Cnd) *models.Company {
	return repositories.CompanyRepository.FindOne(sqls.DB(), cnd)
}

func (s *companyService) FindPageByParams(params *params.QueryParams) (list []models.Company, paging *sqls.Paging) {
	return repositories.CompanyRepository.FindPageByParams(sqls.DB(), params)
}

func (s *companyService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Company, paging *sqls.Paging) {
	return repositories.CompanyRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *companyService) Count(cnd *sqls.Cnd) int64 {
	return repositories.CompanyRepository.Count(sqls.DB(), cnd)
}

func (s *companyService) CreateCompany(req request.CreateCompanyRequest, operator *dto.AuthPrincipal) (*models.Company, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParamI18n("error.e0125")
	}

	existing := repositories.CompanyRepository.GetByName(sqls.DB(), name)
	if existing != nil && existing.Status != enums.StatusDeleted {
		return nil, errorsx.InvalidParamI18n("error.e0126")
	}

	item := &models.Company{
		Name:        name,
		Code:        strings.TrimSpace(req.Code),
		Status:      enums.StatusOk,
		Remark:      strings.TrimSpace(req.Remark),
		AuditFields: utils.BuildAuditFields(operator),
	}
	if err := repositories.CompanyRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}

	WebhookSyncService.DispatchOutboundEvent("company.created", request.OrgSyncEventData{
		DeskCompanyID: fmt.Sprintf("comp_%d", item.ID),
		Name:          item.Name,
		DomainName:    item.Remark,
	})

	return item, nil
}

func (s *companyService) UpdateCompany(req request.UpdateCompanyRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(req.ID)
	if item == nil {
		return errorsx.InvalidParamI18n("error.e0124")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errorsx.InvalidParamI18n("error.e0125")
	}

	existing := repositories.CompanyRepository.GetByName(sqls.DB(), name)
	if existing != nil && existing.ID != req.ID {
		return errorsx.InvalidParamI18n("error.e0126")
	}

	now := time.Now()
	if err := repositories.CompanyRepository.Updates(sqls.DB(), req.ID, map[string]any{
		"name":             name,
		"code":             strings.TrimSpace(req.Code),
		"remark":           strings.TrimSpace(req.Remark),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       now,
	}); err != nil {
		return err
	}
	return nil
}

func (s *companyService) DeleteCompany(id int64, operator dto.AuthPrincipal) error {
	item := s.Get(id)
	if item == nil {
		return errorsx.InvalidParamI18n("error.e0124")
	}

	return repositories.CompanyRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           enums.StatusDeleted,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	})
}

func (s *companyService) UpdateStatus(id int64, status int, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(id)
	if item == nil {
		return errorsx.InvalidParamI18n("error.e0124")
	}
	if status != int(enums.StatusOk) && status != int(enums.StatusDisabled) {
		return errorsx.InvalidParamI18n("error.e0254")
	}
	now := time.Now()
	return repositories.CompanyRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           status,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       now,
	})
}
