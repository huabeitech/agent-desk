package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"slices"
	"strings"
	"time"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/mlogclub/simple/sqls"
)

var RoleService = newRoleService()

func newRoleService() *roleService {
	return &roleService{}
}

type roleService struct {
}

func (s *roleService) Get(id int64) *models.Role {
	return repositories.RoleRepository.Get(sqls.DB(), id)
}

func (s *roleService) Take(where ...interface{}) *models.Role {
	return repositories.RoleRepository.Take(sqls.DB(), where...)
}

func (s *roleService) Find(cnd *sqls.Cnd) []models.Role {
	return repositories.RoleRepository.Find(sqls.DB(), cnd)
}

func (s *roleService) FindOne(cnd *sqls.Cnd) *models.Role {
	return repositories.RoleRepository.FindOne(sqls.DB(), cnd)
}

func (s *roleService) FindPageByParams(params *params.QueryParams) (list []models.Role, paging *sqls.Paging) {
	return repositories.RoleRepository.FindPageByParams(sqls.DB(), params)
}

func (s *roleService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Role, paging *sqls.Paging) {
	return repositories.RoleRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *roleService) Count(cnd *sqls.Cnd) int64 {
	return repositories.RoleRepository.Count(sqls.DB(), cnd)
}

func (s *roleService) Create(t *models.Role) error {
	return repositories.RoleRepository.Create(sqls.DB(), t)
}

func (s *roleService) Update(t *models.Role) error {
	return repositories.RoleRepository.Update(sqls.DB(), t)
}

func (s *roleService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.RoleRepository.Updates(sqls.DB(), id, columns)
}

func (s *roleService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.RoleRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *roleService) Delete(id int64) {
	repositories.RoleRepository.Delete(sqls.DB(), id)
}

func (s *roleService) CreateRole(req request.CreateRoleRequest, operator *dto.AuthPrincipal) (*models.Role, error) {
	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if name == "" || code == "" {
		return nil, errorsx.InvalidParamI18n("error.e0306")
	}
	if s.Take("code = ?", code) != nil {
		return nil, errorsx.InvalidParamI18n("error.e0308")
	}

	role := &models.Role{
		Name:        name,
		Code:        code,
		Status:      enums.StatusOk,
		IsSystem:    false,
		SortNo:      s.NextSortNo(),
		Remark:      strings.TrimSpace(req.Remark),
		AuditFields: utils.BuildAuditFields(operator),
	}
	if err := s.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) UpdateRole(req request.UpdateRoleRequest, operator *dto.AuthPrincipal) error {
	role := s.Get(req.ID)
	if role == nil {
		return errorsx.InvalidParamI18n("error.e0305")
	}
	now := time.Now()
	return s.Updates(req.ID, map[string]any{
		"name":             strings.TrimSpace(req.Name),
		"sort_no":          req.SortNo,
		"remark":           strings.TrimSpace(req.Remark),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       now,
	})
}

func (s *roleService) NextSortNo() int {
	if latest := s.FindOne(sqls.NewCnd().Desc("sort_no").Desc("id")); latest != nil {
		return latest.SortNo + 1
	}
	return 0
}

func (s *roleService) UpdateSort(ids []int64) error {
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		for i, id := range ids {
			if err := repositories.RoleRepository.UpdateColumn(ctx.Tx, id, "sort_no", i); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *roleService) DeleteRole(id int64) error {
	role := s.Get(id)
	if role == nil {
		return errorsx.InvalidParamI18n("error.e0305")
	}
	if role.IsSystem {
		return errorsx.ForbiddenI18n("error.e0293")
	}
	if UserRoleService.Take("role_id = ?", id) != nil {
		return errorsx.ForbiddenI18n("error.e0307")
	}
	s.Delete(id)
	return nil
}

func (s *roleService) UpdateStatus(id int64, status enums.Status, operator *dto.AuthPrincipal) error {
	role := s.Get(id)
	if role == nil {
		return errorsx.InvalidParamI18n("error.e0305")
	}
	if !slices.Contains(enums.StatusValues, status) {
		return errorsx.InvalidParamI18n("error.e0254")
	}
	if err := s.Updates(id, map[string]any{
		"status":           status,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	}); err != nil {
		return err
	}
	return nil
}

func (s *roleService) AssignPermissions(roleID int64, permissionIDs []int64, operator *dto.AuthPrincipal) error {
	role := s.Get(roleID)
	if role == nil {
		return errorsx.InvalidParamI18n("error.e0305")
	}
	if role.IsSystem && (operator == nil || !slices.Contains(operator.Roles, string(constants.RoleCodeSuperAdmin))) {
		return errorsx.ForbiddenI18n("error.e0293")
	}

	return s.replaceRolePermissions(roleID, permissionIDs, operator)
}

func (s *roleService) replaceRolePermissions(roleID int64, permissionIDs []int64, operator *dto.AuthPrincipal) error {
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := ctx.Tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		for _, permissionID := range permissionIDs {
			permission := PermissionService.Get(permissionID)
			if permission == nil {
				return errorsx.InvalidParamI18n("error.e0236")
			}
			relation := &models.RolePermission{
				RoleID:       roleID,
				PermissionID: permissionID,
				AuditFields:  utils.BuildAuditFields(operator),
			}
			if err := ctx.Tx.Create(relation).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
