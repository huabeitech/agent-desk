package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/services"
)

type UserBuildOptions struct {
	Roles       bool
	Permissions bool
}

func BuildUserList(items []models.User, options UserBuildOptions) []response.UserResponse {
	results := make([]response.UserResponse, 0, len(items))
	for _, item := range items {
		results = append(results, *BuildUserResponse(&item, options))
	}
	return results
}

func BuildUserResponse(item *models.User, options UserBuildOptions) *response.UserResponse {
	if item == nil {
		return nil
	}
	ret := &response.UserResponse{
		ID:            item.ID,
		Username:      item.Username,
		Nickname:      item.Nickname,
		Avatar:        item.UserAvatarURL(),
		AvatarAssetID: item.UserAvatarAssetID(),
		UserType:      item.UserType,
		Status:        item.Status,
		LastLoginAt:   utils.FormatTimePtr(item.LastLoginAt),
		LastLoginIP:   item.LastLoginIP,
	}

	if item.Mobile != nil {
		ret.Mobile = *item.Mobile
	}
	if item.Email != nil {
		ret.Email = *item.Email
	}

	if options.Roles {
		ret.Roles = buildAssignedRoles(item.ID)
	}
	if options.Permissions {
		permissionCodes, _ := services.AuthService.GetUserPermissions(item.ID)
		ret.Permissions = permissionCodes
	}
	return ret
}

func buildAssignedRoles(userID int64) []response.RoleResponse {
	roles, _ := services.AuthService.GetUserRoles(userID)
	results := make([]response.RoleResponse, 0, len(roles))
	for _, role := range roles {
		results = append(results, response.RoleResponse{
			ID:       role.ID,
			Name:     role.Name,
			Code:     role.Code,
			Status:   role.Status,
			IsSystem: role.IsSystem,
			SortNo:   role.SortNo,
		})
	}
	return results
}
