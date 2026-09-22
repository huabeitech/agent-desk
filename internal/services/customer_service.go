package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var CustomerService = newCustomerService()

func newCustomerService() *customerService {
	return &customerService{}
}

type customerService struct {
}

func (s *customerService) Get(id int64) *models.Customer {
	return repositories.CustomerRepository.Get(sqls.DB(), id)
}

func (s *customerService) Take(where ...interface{}) *models.Customer {
	return repositories.CustomerRepository.Take(sqls.DB(), where...)
}

func (s *customerService) Find(cnd *sqls.Cnd) []models.Customer {
	return repositories.CustomerRepository.Find(sqls.DB(), cnd)
}

func (s *customerService) FindOne(cnd *sqls.Cnd) *models.Customer {
	return repositories.CustomerRepository.FindOne(sqls.DB(), cnd)
}

func (s *customerService) FindPageByParams(params *params.QueryParams) (list []models.Customer, paging *sqls.Paging) {
	return repositories.CustomerRepository.FindPageByParams(sqls.DB(), params)
}

func (s *customerService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Customer, paging *sqls.Paging) {
	return repositories.CustomerRepository.FindPageByCnd(sqls.DB(), cnd)
}

// ListCustomers 客户分页列表（连联系方式表，支持按非主联系方式检索）。
func (s *customerService) ListCustomers(req request.CustomerListRequest) (list []models.Customer, paging *sqls.Paging) {
	if err := s.newCustomerListQuery(req).Distinct("c.*").Offset(req.Offset()).Order("c.id DESC").Limit(req.GetLimit()).Scan(&list).Error; err != nil {
		slog.Error("customer list scan failed", slog.Any("error", err))
	}

	var total int64
	if err := s.newCustomerListQuery(req).Distinct("c.id").Count(&total).Error; err != nil {
		slog.Error("customer list count failed", slog.Any("error", err))
	}

	paging = &sqls.Paging{
		Page:  req.GetPage(),
		Limit: req.GetLimit(),
		Total: total,
	}
	return
}

func (s *customerService) newCustomerListQuery(req request.CustomerListRequest) *gorm.DB {
	deleted := int(enums.StatusDeleted)
	tx := sqls.DB().
		Table("t_customer AS c").
		Joins("LEFT JOIN t_customer_contact AS cc ON cc.customer_id = c.id AND cc.status <> ?", deleted).
		Joins("LEFT JOIN t_company AS co ON co.id = c.company_id")

	tx.Where("c.status <> ?", enums.StatusDeleted)

	if req.Status != nil {
		tx.Where("c.status = ?", *req.Status)
	}
	if req.Gender != nil {
		tx.Where("c.gender = ?", *req.Gender)
	}
	if req.CompanyID != nil && *req.CompanyID > 0 {
		tx.Where("c.company_id = ?", *req.CompanyID)
	}
	if kw := strings.TrimSpace(req.Keyword); strs.IsNotBlank(kw) {
		pat := "%" + kw + "%"
		tx.Where(`(
c.name LIKE ? OR
c.primary_mobile LIKE ? OR
c.primary_email LIKE ? OR
cc.contact_value LIKE ? OR
co.name LIKE ?
)`, pat, pat, pat, pat, pat)
	}
	return tx
}

func (s *customerService) Count(cnd *sqls.Cnd) int64 {
	return repositories.CustomerRepository.Count(sqls.DB(), cnd)
}

func (s *customerService) CountByCompanyIDs(companyIDs []int64) map[int64]int64 {
	return repositories.CustomerRepository.CountByCompanyIDs(sqls.DB(), companyIDs, int(enums.StatusDeleted))
}

func (s *customerService) EnsureExternalCustomer(ctx *sqls.TxContext, externalUser openidentity.ExternalUser) (int64, error) {
	if ctx == nil || ctx.Tx == nil {
		return 0, errorsx.InvalidParamI18n("error.e0086")
	}
	externalSource := externalUser.ExternalSource
	externalID := strings.TrimSpace(externalUser.ExternalID)
	if strings.TrimSpace(string(externalSource)) == "" || externalID == "" {
		return 0, errorsx.UnauthorizedI18n("error.e0149")
	}
	now := time.Now()
	if identity := repositories.CustomerIdentityRepository.GetBy(ctx.Tx, externalSource, externalID); identity != nil {
		updates := map[string]any{
			"last_active_at": now,
			"updated_at":     now,
		}
		if strs.IsNotBlank(externalUser.ExternalName) {
			updates["name"] = externalUser.ExternalName
		}
		if err := repositories.CustomerRepository.Updates(ctx.Tx, identity.CustomerID, updates); err != nil {
			return 0, err
		}

		ctx.RegisterCallback(func() {
			if strs.IsNotBlank(externalUser.ExternalName) {
				if err := s.syncConversationCustomerName(sqls.DB(), identity.CustomerID, externalUser.ExternalName, nil, now); err != nil {
					slog.Error("sync conversation customer name failed",
						"customerId", identity.CustomerID,
						"customerName", externalUser.ExternalName,
						"error", err,
					)
				}
			}
		})
		return identity.CustomerID, nil
	}

	customer := &models.Customer{
		Name:         buildExternalCustomerName(externalUser),
		LastActiveAt: &now,
		Status:       enums.StatusOk,
		AuditFields:  utils.BuildAuditFields(nil),
	}
	if err := repositories.CustomerRepository.Create(ctx.Tx, customer); err != nil {
		return 0, err
	}
	if err := repositories.CustomerIdentityRepository.Create(ctx.Tx, &models.CustomerIdentity{
		CustomerID:     customer.ID,
		ExternalSource: externalSource,
		ExternalID:     externalID,
		Status:         enums.StatusOk,
		AuditFields:    utils.BuildAuditFields(nil),
	}); err != nil {
		return 0, err
	}
	return customer.ID, nil
}

func buildExternalCustomerName(externalUser openidentity.ExternalUser) string {
	if strs.IsNotBlank(externalUser.ExternalName) {
		return externalUser.ExternalName
	}
	return "访客" + hashUUID(externalUser.ExternalID)
}

func hashUUID(uuid string) string {
	if uuid == "" {
		return "unknown"
	}

	h := md5.Sum([]byte(uuid))
	return hex.EncodeToString(h[:])[:8]
}

func (s *customerService) CreateCustomer(req request.CreateCustomerRequest, operator *dto.AuthPrincipal) (*models.Customer, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParamI18n("error.e0156")
	}

	if req.CompanyID > 0 {
		company := CompanyService.Get(req.CompanyID)
		if company == nil {
			return nil, errorsx.InvalidParamI18n("error.e0204")
		}
	}

	item := &models.Customer{
		Name:          name,
		Gender:        enums.Gender(req.Gender),
		CompanyID:     req.CompanyID,
		PrimaryMobile: strings.TrimSpace(req.PrimaryMobile),
		PrimaryEmail:  strings.TrimSpace(req.PrimaryEmail),
		Status:        enums.StatusOk,
		Remark:        strings.TrimSpace(req.Remark),
		AuditFields:   utils.BuildAuditFields(operator),
	}

	if err := repositories.CustomerRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}

	WebhookSyncService.DispatchOutboundEvent("customer.created", request.OrgSyncEventData{
		DeskCustomerID: fmt.Sprintf("cust_%d", item.ID),
		Name:           item.Name,
		Email:          item.PrimaryEmail,
		Phone:          item.PrimaryMobile,
		Source:         "crove_desk",
	})

	return item, nil
}

func (s *customerService) UpdateCustomer(req request.UpdateCustomerRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(req.ID)
	if item == nil {
		return errorsx.InvalidParamI18n("error.e0155")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errorsx.InvalidParamI18n("error.e0156")
	}

	if req.CompanyID > 0 {
		company := CompanyService.Get(req.CompanyID)
		if company == nil {
			return errorsx.InvalidParamI18n("error.e0204")
		}
	}

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		now := time.Now()
		if err := repositories.CustomerRepository.Updates(ctx.Tx, req.ID, map[string]any{
			"name":             name,
			"gender":           req.Gender,
			"company_id":       req.CompanyID,
			"primary_mobile":   strings.TrimSpace(req.PrimaryMobile),
			"primary_email":    strings.TrimSpace(req.PrimaryEmail),
			"remark":           strings.TrimSpace(req.Remark),
			"update_user_id":   operator.UserID,
			"update_user_name": operator.Username,
			"updated_at":       now,
		}); err != nil {
			return err
		}
		return s.syncConversationCustomerName(ctx.Tx, req.ID, name, operator, now)
	})
}

func (s *customerService) DeleteCustomer(id int64, operator dto.AuthPrincipal) error {
	item := s.Get(id)
	if item == nil {
		return errorsx.InvalidParamI18n("error.e0155")
	}
	return repositories.CustomerRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           enums.StatusDeleted,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	})
}

func (s *customerService) syncConversationCustomerName(db *gorm.DB, customerID int64, name string, operator *dto.AuthPrincipal, now time.Time) error {
	if customerID <= 0 {
		return nil
	}
	updates := map[string]any{
		"customer_name": name,
		"updated_at":    now,
	}
	if operator != nil {
		updates["update_user_id"] = operator.UserID
		updates["update_user_name"] = operator.Username
	}
	return repositories.ConversationRepository.UpdatesByCustomerID(db, customerID, updates)
}

func (s *customerService) UpdateStatus(id int64, status int, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(id)
	if item == nil {
		return errorsx.InvalidParamI18n("error.e0155")
	}
	if status != int(enums.StatusOk) && status != int(enums.StatusDisabled) {
		return errorsx.InvalidParamI18n("error.e0254")
	}
	return repositories.CustomerRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           status,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	})
}

// SaveCustomerProfile 单事务保存客户主信息与联系方式全量（新建或更新）。
func (s *customerService) SaveCustomerProfile(req request.SaveCustomerProfileRequest, operator *dto.AuthPrincipal) (*models.Customer, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParamI18n("error.e0156")
	}
	if req.CompanyID > 0 {
		if CompanyService.Get(req.CompanyID) == nil {
			return nil, errorsx.InvalidParamI18n("error.e0204")
		}
	}
	createMode := req.ID == nil || *req.ID <= 0

	var out *models.Customer
	err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		var customerID int64
		if createMode {
			c := &models.Customer{
				Name:          name,
				Gender:        enums.Gender(req.Gender),
				CompanyID:     req.CompanyID,
				PrimaryMobile: "",
				PrimaryEmail:  "",
				Status:        enums.StatusOk,
				Remark:        strings.TrimSpace(req.Remark),
				AuditFields:   utils.BuildAuditFields(operator),
			}
			if err := repositories.CustomerRepository.Create(ctx.Tx, c); err != nil {
				return err
			}
			customerID = c.ID
			out = c
		} else {
			customerID = *req.ID
			cur := repositories.CustomerRepository.Get(ctx.Tx, customerID)
			if cur == nil {
				return errorsx.InvalidParamI18n("error.e0155")
			}
			now := time.Now()
			if err := repositories.CustomerRepository.Updates(ctx.Tx, customerID, map[string]any{
				"name":             name,
				"gender":           req.Gender,
				"company_id":       req.CompanyID,
				"remark":           strings.TrimSpace(req.Remark),
				"update_user_id":   operator.UserID,
				"update_user_name": operator.Username,
				"updated_at":       now,
			}); err != nil {
				return err
			}
			if err := s.syncConversationCustomerName(ctx.Tx, customerID, name, operator, now); err != nil {
				return err
			}
			out = repositories.CustomerRepository.Get(ctx.Tx, customerID)
		}
		return CustomerContactService.ReplaceAllForCustomerInTx(ctx, customerID, req.Contacts, operator)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
